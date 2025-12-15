package graph

import (
	"context"
	"strings"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/cloudwego/hertz/pkg/protocol/sse"
	"github.com/xh-polaris/innospark-core-api/biz/conf"
	"github.com/xh-polaris/innospark-core-api/biz/domain/memory"
	"github.com/xh-polaris/innospark-core-api/biz/domain/message"
	"github.com/xh-polaris/innospark-core-api/biz/domain/message/prompt_inject"
	"github.com/xh-polaris/innospark-core-api/biz/domain/model"
	"github.com/xh-polaris/innospark-core-api/biz/domain/state"
	"github.com/xh-polaris/innospark-core-api/biz/domain/state/interaction"
	tool "github.com/xh-polaris/innospark-core-api/biz/domain/tool"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	mmsg "github.com/xh-polaris/innospark-core-api/biz/infra/mapper/message"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util"
)

type Input = *state.RelayContext
type Output = *state.RelayContext
type Context = *state.RelayContext

type CompletionGraph struct {
	*compose.Graph[Input, Output]
}

const (
	GenLocalState                   = "gen-local-state"
	LoadHistory                     = "load-history"
	ProcessCompletionOpt            = "process-completion-opt"
	MsgMToE                         = "message-m2e"
	WebSearch                       = "web-search"
	ChatModel                       = "chat-model"
	InteractionAssemblyModelMessage = "interaction-assembly-model-message"
	SSEStream                       = "sse-stream"
	StoreHis                        = "store-history"
)

// DrawCompletionGraph 定义Completion 图的结构
// to optimize state存在并发风险, 需要加锁
func DrawCompletionGraph(memory *memory.MemoryManager) *CompletionGraph {
	// 将RelayContext作为状态的Graph
	gls := func(ctx context.Context) (s Context) { return &state.RelayContext{} }
	cg := compose.NewGraph[Input, Output](compose.WithGenLocalState(gls))

	// 初始化GenLocalState
	genGLS := compose.InvokableLambda(func(ctx context.Context, input Input) (_ Input, err error) {
		err = compose.ProcessState(ctx, func(ctx context.Context, s Context) (err error) { *s = *input; return })
		return input, err
	})
	util.MustAddLambdaNode(cg, GenLocalState, genGLS)

	// 获取记忆
	loadHis := compose.InvokableLambda(func(ctx context.Context, input Input) (output []*mmsg.Message, err error) {
		return memory.RetrieveMemory(ctx, input)
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
		return tool.Search(ctx, "bocha", conf.GetConfig().Bocha.APIKey, conf.GetConfig().Bocha.Template, input)
	})
	util.MustAddLambdaNode(cg, WebSearch, webSearch)

	// 存储域到模型域
	mMsgToEMsg := compose.InvokableLambda(func(ctx context.Context, input []*mmsg.Message) (output []*schema.Message, err error) {
		return message.MMsgToEMsgList(input), nil
	})
	util.MustAddLambdaNode(cg, MsgMToE, mMsgToEMsg)

	// 联网搜索分支
	condition := func(ctx context.Context, _ []*mmsg.Message) (out string, err error) {
		out = MsgMToE
		err = compose.ProcessState(ctx, func(ctx context.Context, s Context) (err error) {
			if s.Info.ModelInfo.WebSearch { // 开了联网搜索
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
	modelOpt := compose.WithStatePreHandler(func(ctx context.Context, in []*schema.Message, state Context) (_ []*schema.Message, err error) {
		if state.Info.ModelInfo.BotId == "code-gen" {
			state.Info.ModelInfo.Model = model.Claude4Sonnet
			// 填充模板
			format, err := prompt.FromMessages(schema.FString, &schema.Message{Role: cst.User, Content: conf.GetConfig().ARK.CodeGenTemplate}).Format(ctx,
				map[string]any{"userQuery": state.Info.OriginMessage.Content})
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
		} else if strings.HasPrefix(state.Info.ModelInfo.BotId, "intelligence-") { // coze 智能体
			state.Info.ModelInfo.Model, state.Info.ModelInfo.BotId = model.SelfCoze, state.Info.ModelInfo.BotId[13:]
		} else if needVL(in) { // 需要视觉模型
			if !strings.HasSuffix(state.Info.ModelInfo.Model, "-VL") {
				state.Info.ModelInfo.Model += "-VL"
			}
		}
		if strings.HasPrefix(state.Info.ModelInfo.BotId, "cotea-") {
			in, err = prompt_inject.CoTeaSysInject(ctx, in, state)
			if err != nil {
				return nil, err
			}
		}
		return in, nil
	})
	cm := &model.ModelFactory{}
	util.MustAddChatModelNode(cg, ChatModel, cm, modelOpt)

	// 组装模型消息
	assembleModelEvents := compose.TransformableLambda(func(ctx context.Context, input *schema.StreamReader[*schema.Message]) (_ *schema.StreamReader[*sse.Event], err error) {
		var relay *state.RelayContext
		if relay, err = util.GetState[*state.RelayContext](ctx); err != nil {
			return
		}
		r, w := schema.Pipe[*sse.Event](3)
		go relay.Interaction.AssembleModelEvents(input, w)
		return r, nil
	})
	util.MustAddLambdaNode(cg, InteractionAssemblyModelMessage, assembleModelEvents)

	// 返回SSE
	sseStream := compose.CollectableLambda(func(ctx context.Context, input *schema.StreamReader[*sse.Event]) (relay *state.RelayContext, err error) {
		if relay, err = util.GetState[*state.RelayContext](ctx); err != nil {
			return
		}
		return relay, interaction.WriteSSE(relay.Info, input)
	})
	util.MustAddLambdaNode(cg, SSEStream, sseStream)

	// 存储历史记录
	storeHis := compose.InvokableLambda(func(ctx context.Context, input *state.RelayContext) (_ Output, err error) {
		err = memory.StoreHistory(ctx, input)
		return input, err
	})
	util.MustAddLambdaNode(cg, StoreHis, storeHis)

	// 链式连接 optimize 满足Cond执行的节点
	util.MustChain(cg, compose.START, GenLocalState, LoadHistory, ProcessCompletionOpt)
	util.MustChain(cg, WebSearch, MsgMToE)
	util.MustChain(cg, MsgMToE, ChatModel, InteractionAssemblyModelMessage, SSEStream, StoreHis, compose.END)
	return &CompletionGraph{cg}
}

func (g *CompletionGraph) CompileAndInvoke(ctx context.Context, input Input) (_ Output, _ error) {
	r, err := g.Compile(ctx)
	if err != nil {
		return nil, err
	}
	defer close(input.Info.SSE.C)             // 关闭sse 事件流
	defer func() { input.Info.SSE.Close() }() // 关闭sse 响应流
	return r.Invoke(ctx, input)
}

func (g *CompletionGraph) CompileAndStream(ctx context.Context, input Input) (_ *schema.StreamReader[Output], _ error) {
	r, err := g.Compile(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { input.Info.SSE.Close() }() // 关闭sse 响应流
	return r.Stream(ctx, input)
}

func needVL(in []*schema.Message) bool {
	for _, m := range in {
		if len(m.UserInputMultiContent) > 0 {
			return true
		}
	}
	return false
}
