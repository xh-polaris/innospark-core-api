package graph

import (
	"context"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/cloudwego/hertz/pkg/protocol/sse"
	"github.com/xh-polaris/innospark-core-api/biz/adaptor"
	mmsg "github.com/xh-polaris/innospark-core-api/biz/infra/mapper/message"
)

type Input *RelayContext
type Output *RelayContext

type CompletionGraph struct {
	*compose.Graph[Input, Output]
}

const (
	GenLocalState        = "gen-local-state"
	LoadHistory          = "load-history"
	ProcessCompletionOpt = "process-completion-opt"
	MsgMToE              = "msg-m2e"
	ChatModel            = "chat-model"
	TransformToSSEEvent  = "transform-to-sse-event"
	SSEStream            = "sse-stream"
	StoreHis             = "store-his"
)

// DrawCompletionGraph 定义Completion 图的结构
// to optimize state存在并发风险, 需要加锁
func DrawCompletionGraph(hd *HistoryDomain) *CompletionGraph {
	// 将RelayContext作为状态的Graph
	gls := func(ctx context.Context) (state *RelayContext) { return &RelayContext{} }
	cg := compose.NewGraph[Input, Output](compose.WithGenLocalState(gls))

	// 初始化GenLocalState
	genGLS := compose.InvokableLambda(func(ctx context.Context, input Input) (_ *RelayContext, err error) {
		input.SSEWriter = sse.NewWriter(input.RequestContext) // 提前创建SSE Writer, 以便中间节点能响应sse事件
		err = compose.ProcessState(ctx, func(ctx context.Context, s *RelayContext) (err error) { *s = *input; return })
		return input, err
	})
	MustAddLambdaNode(cg, GenLocalState, genGLS)

	// 历史记录节点
	loadHis := compose.InvokableLambda(func(ctx context.Context, input *RelayContext) (output []*mmsg.Message, err error) {
		return hd.RetrieveHistory(ctx, input)
	})
	MustAddLambdaNode(cg, LoadHistory, loadHis)

	// 处理对话配置项
	completionOpt := compose.InvokableLambda(func(ctx context.Context, input []*mmsg.Message) (output []*mmsg.Message, err error) {
		err = compose.ProcessState(ctx, func(ctx context.Context, s *RelayContext) (err error) {
			output, err = DoCompletionOption(s, input)
			return
		})
		return
	})
	MustAddLambdaNode(cg, ProcessCompletionOpt, completionOpt)

	// 存储域到模型域
	mMsgToEMsg := compose.InvokableLambda(func(ctx context.Context, input []*mmsg.Message) (output []*schema.Message, err error) {
		return MMsgToEMsgList(input), nil
	})
	MustAddLambdaNode(cg, MsgMToE, mMsgToEMsg)

	// to optimize 根据模型配置, 路由到不同的分支
	// 调用模型
	cm := &ModelFactory{}
	MustAddChatModelNode(cg, ChatModel, cm)

	// 构建事件
	transformToEvent := compose.TransformableLambda(func(ctx context.Context, input *schema.StreamReader[*schema.Message]) (_ *schema.StreamReader[*sse.Event], err error) {
		var relay *RelayContext
		if relay, err = GetState(ctx); err != nil {
			return
		}
		relay.SSE = adaptor.NewSSEStream() // 提前创建sse流, 以便组装事件时可以监听中断
		r, w := schema.Pipe[*sse.Event](3)
		go NewTransformer(relay).TransformToEvent(input, w)
		return r, nil
	})
	MustAddLambdaNode(cg, TransformToSSEEvent, transformToEvent)

	// 返回SSE
	sseStream := compose.CollectableLambda(func(ctx context.Context, input *schema.StreamReader[*sse.Event]) (relay *RelayContext, err error) {
		if relay, err = GetState(ctx); err != nil {
			return
		}
		return SSE(relay, input)
	})
	MustAddLambdaNode(cg, SSEStream, sseStream)

	// 存储历史记录
	storeHis := compose.InvokableLambda(func(ctx context.Context, input *RelayContext) (_ Output, err error) {
		err = hd.StoreHistory(ctx, input)
		return input, err
	})
	MustAddLambdaNode(cg, StoreHis, storeHis)

	// 链式连接
	MustChain(cg, compose.START, GenLocalState, LoadHistory, ProcessCompletionOpt, MsgMToE, ChatModel, TransformToSSEEvent, SSEStream, StoreHis, compose.END)
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

func GetState(ctx context.Context) (relay *RelayContext, err error) {
	err = compose.ProcessState(ctx, func(ctx context.Context, s *RelayContext) (err error) {
		relay = s
		return
	})
	return
}

func MustAddLambdaNode[I any, O any](g *compose.Graph[I, O], key string, node *compose.Lambda, opts ...compose.GraphAddNodeOpt) {
	if err := g.AddLambdaNode(key, node, opts...); err != nil {
		panic(err)
	}
}

func MustAddChatModelNode[I any, O any](g *compose.Graph[I, O], key string, node model.BaseChatModel, opts ...compose.GraphAddNodeOpt) {
	if err := g.AddChatModelNode(key, node, opts...); err != nil {
		panic(err)
	}
}

func MustChain[I any, O any](g *compose.Graph[I, O], nodes ...string) {
	if len(nodes) < 2 {
		return
	}
	for i := 1; i < len(nodes); i++ {
		MustAddEdge(g, nodes[i-1], nodes[i])
	}
}

func MustAddEdge[I any, O any](g *compose.Graph[I, O], start, end string) {
	if err := g.AddEdge(start, end); err != nil {
		panic(err)
	}
}
