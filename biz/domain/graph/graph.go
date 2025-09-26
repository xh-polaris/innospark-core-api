package graph

import (
	"context"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/cloudwego/hertz/pkg/protocol/sse"
	"github.com/xh-polaris/innospark-core-api/biz/adaptor"
	"github.com/xh-polaris/innospark-core-api/biz/domain/info"
	"github.com/xh-polaris/innospark-core-api/biz/domain/model"
	"github.com/xh-polaris/innospark-core-api/biz/domain/msg"
	tool "github.com/xh-polaris/innospark-core-api/biz/domain/tool"
	"github.com/xh-polaris/innospark-core-api/biz/infra/config"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	mmsg "github.com/xh-polaris/innospark-core-api/biz/infra/mapper/message"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util"
)

type Input = *info.RelayContext
type Output = *info.RelayContext
type Context = *info.RelayContext

type CompletionGraph struct {
	*compose.Graph[Input, Output]
}

const (
	GenLocalState        = "gen-local-state"
	LoadHistory          = "load-history"
	ProcessCompletionOpt = "process-completion-opt"
	MsgMToE              = "msg-m2e"
	WebSearch            = "web-search"
	WebSearchBranch      = "web-search-branch"
	ChatModel            = "chat-model"
	TransformToSSEEvent  = "transform-to-sse-event"
	SSEStream            = "sse-stream"
	StoreHis             = "store-his"
)

// DrawCompletionGraph 定义Completion 图的结构
// to optimize state存在并发风险, 需要加锁
func DrawCompletionGraph(hd *HistoryDomain) *CompletionGraph {
	// 将RelayContext作为状态的Graph
	gls := func(ctx context.Context) (state Context) { return &info.RelayContext{} }
	cg := compose.NewGraph[Input, Output](compose.WithGenLocalState(gls))

	// 初始化GenLocalState
	genGLS := compose.InvokableLambda(func(ctx context.Context, input Input) (_ Input, err error) {
		input.SSEWriter = sse.NewWriter(input.RequestContext) // 提前创建SSE Writer, 以便中间节点能响应sse事件
		err = compose.ProcessState(ctx, func(ctx context.Context, s Context) (err error) { *s = *input; return })
		return input, err
	})
	util.MustAddLambdaNode(cg, GenLocalState, genGLS)

	// 历史记录节点
	loadHis := compose.InvokableLambda(func(ctx context.Context, input Input) (output []*mmsg.Message, err error) {
		return hd.RetrieveHistory(ctx, input)
	})
	util.MustAddLambdaNode(cg, LoadHistory, loadHis)

	// 处理对话配置项
	completionOpt := compose.InvokableLambda(func(ctx context.Context, input []*mmsg.Message) (output []*mmsg.Message, err error) {
		err = compose.ProcessState(ctx, func(ctx context.Context, s Context) (err error) {
			output, err = DoCompletionOption(s, input)

			return
		})
		return
	})
	util.MustAddLambdaNode(cg, ProcessCompletionOpt, completionOpt)

	// 搜索
	webSearch := compose.InvokableLambda(func(ctx context.Context, input []*mmsg.Message) (_ []*mmsg.Message, err error) {
		return tool.Search(ctx, "bocha", config.GetConfig().Bocha.APIKey, config.GetConfig().Bocha.Template, input)
	})
	util.MustAddLambdaNode(cg, WebSearch, webSearch)

	// 存储域到模型域
	mMsgToEMsg := compose.InvokableLambda(func(ctx context.Context, input []*mmsg.Message) (output []*schema.Message, err error) {
		return msg.MMsgToEMsgList(input), nil
	})
	util.MustAddLambdaNode(cg, MsgMToE, mMsgToEMsg)

	// 联网搜索分支
	condition := func(ctx context.Context, _ []*mmsg.Message) (out string, err error) {
		out = MsgMToE
		err = compose.ProcessState(ctx, func(ctx context.Context, s Context) (err error) {
			if s.ModelInfo.WebSearch { // 开了联网搜索
				out = WebSearch
			}
			return
		})
		return
	}
	endNodes := map[string]bool{MsgMToE: true, WebSearch: true}
	webSearchBranch := compose.NewGraphBranch(condition, endNodes)
	util.MustAddGraphBranch(cg, ProcessCompletionOpt, webSearchBranch)

	// to optimize 根据模型配置, 路由到不同的分支
	// 调用模型
	modelOpt := compose.WithStatePreHandler(func(ctx context.Context, in []*schema.Message, state Context) ([]*schema.Message, error) {
		if state.ModelInfo.BotId == "code-gen" {
			state.ModelInfo.Model = model.Claude4Sonnet
			// 填充模板
			format, err := prompt.FromMessages(schema.FString, &schema.Message{Role: cst.User, Content: config.GetConfig().ARK.CodeGenTemplate}).Format(ctx,
				map[string]any{"userQuery": state.OriginMessage.Content})
			if err != nil {
				return nil, err
			}
			// 找到最近一条有效的用户消息, 主要是为了适配regen的情况
			for _, m := range in {
				if m.Role == cst.User && m.Content != "" {
					m.Content = format[0].Content
					break
				}
			}
		}
		return in, nil
	})
	cm := &model.ModelFactory{}
	util.MustAddChatModelNode(cg, ChatModel, cm, modelOpt)

	// 构建事件
	transformToEvent := compose.TransformableLambda(func(ctx context.Context, input *schema.StreamReader[*schema.Message]) (_ *schema.StreamReader[*sse.Event], err error) {
		var relay *info.RelayContext
		if relay, err = util.GetState[*info.RelayContext](ctx); err != nil {
			return
		}
		relay.SSE = adaptor.NewSSEStream() // 提前创建sse流, 以便组装事件时可以监听中断
		r, w := schema.Pipe[*sse.Event](3)
		go NewTransformer(relay).TransformToEvent(input, w)
		return r, nil
	})
	util.MustAddLambdaNode(cg, TransformToSSEEvent, transformToEvent)

	// 返回SSE
	sseStream := compose.CollectableLambda(func(ctx context.Context, input *schema.StreamReader[*sse.Event]) (relay *info.RelayContext, err error) {
		if relay, err = util.GetState[*info.RelayContext](ctx); err != nil {
			return
		}
		return SSE(relay, input)
	})
	util.MustAddLambdaNode(cg, SSEStream, sseStream)

	// 存储历史记录
	storeHis := compose.InvokableLambda(func(ctx context.Context, input *info.RelayContext) (_ Output, err error) {
		err = hd.StoreHistory(ctx, input)
		return input, err
	})
	util.MustAddLambdaNode(cg, StoreHis, storeHis)

	// 链式连接
	util.MustChain(cg, compose.START, GenLocalState, LoadHistory, ProcessCompletionOpt)
	util.MustChain(cg, WebSearch, MsgMToE)
	util.MustChain(cg, MsgMToE, ChatModel, TransformToSSEEvent, SSEStream, StoreHis, compose.END)
	return &CompletionGraph{cg}
}

func (g *CompletionGraph) CompileAndInvoke(ctx context.Context, input Input) (_ Output, _ error) {
	r, err := g.Compile(ctx)
	if err != nil {
		return nil, err
	}
	defer close(input.SSE.C)                       // 关闭sse 事件流
	defer func() { _ = input.SSEWriter.Close() }() // 关闭sse 响应流
	return r.Invoke(ctx, input)
}

func (g *CompletionGraph) CompileAndStream(ctx context.Context, input Input) (_ *schema.StreamReader[Output], _ error) {
	r, err := g.Compile(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = input.SSEWriter.Close() }() // 关闭sse 响应流
	return r.Stream(ctx, input)
}
