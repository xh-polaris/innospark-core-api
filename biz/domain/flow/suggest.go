package flow

import (
	"context"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/xh-polaris/innospark-core-api/biz/conf"
	"github.com/xh-polaris/innospark-core-api/biz/domain/model"
	"github.com/xh-polaris/innospark-core-api/biz/domain/state"
	"github.com/xh-polaris/innospark-core-api/biz/domain/state/event"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	"github.com/xh-polaris/innospark-core-api/pkg/ctxcache"
)

type Suggest = *compose.Graph[*state.RelayContext, *event.Event]

func StreamExecuteSuggest(f Suggest, ctx context.Context, input *state.RelayContext) (*schema.StreamReader[*event.Event], error) {
	compiled, err := f.Compile(ctx)
	if err != nil {
		return nil, err
	}
	return compiled.Stream(ctx, input)
}

const (
	SuggestExtract   = "suggest-extract"
	SuggestChatModel = "suggest-chat-model"
	SuggestAssembly  = "suggest-assembly"
)

func BuildSuggest(st *state.RelayContext) Suggest {
	gls := func(ctx context.Context) (s *state.RelayContext) {
		v, _ := ctxcache.Get[*state.RelayContext](ctx, cst.CtxState)
		return v
	}
	suggest := compose.NewGraph[*state.RelayContext, *event.Event](compose.WithGenLocalState(gls))

	// 提取输入
	extract := compose.InvokableLambda(func(ctx context.Context, input *state.RelayContext) (output []*schema.Message, err error) {
		// 提示词模板
		template := prompt.FromMessages(schema.FString, schema.UserMessage(conf.GetConfig().Suggest.Template))
		return template.Format(ctx, map[string]any{"input": st.Info.OriginMessage.Content, // 原始输入
			"answer": st.Info.MessageInfo.RuntimeAssistantMessage.String(), // 模型输出
		})
	})
	_ = suggest.AddLambdaNode(SuggestExtract, extract, compose.WithNodeName(SuggestExtract))

	// 调用模型
	cm := model.NewModelFactory(model.WithModel(conf.GetConfig().Suggest.Model))
	_ = suggest.AddChatModelNode(SuggestChatModel, cm, compose.WithNodeName(SuggestChatModel))

	// 将建议事件写入事件流
	assembleSuggestEvents := compose.CollectableLambda(func(ctx context.Context, input *schema.StreamReader[*schema.Message]) (_ *state.RelayContext, err error) {
		go func() {
			var m *schema.Message
			for {
				m, err = input.Recv()
				st.EventStream.W.Send(&event.Event{Type: event.Suggest, Message: m}, err)
				if err != nil {
					return
				}
			}
		}()
		return st, nil
	})
	_ = suggest.AddLambdaNode(SuggestAssembly, assembleSuggestEvents, compose.WithNodeName(SuggestAssembly))

	output := compose.StreamableLambda(func(ctx context.Context, st *state.RelayContext) (_ *schema.StreamReader[*event.Event], err error) {
		return st.EventStream.R, nil
	})
	_ = suggest.AddLambdaNode(Output, output, compose.WithNodeName(Output))

	// 编排
	_ = suggest.AddEdge(compose.START, SuggestExtract)
	_ = suggest.AddEdge(SuggestExtract, SuggestChatModel)
	_ = suggest.AddEdge(SuggestChatModel, SuggestAssembly)
	_ = suggest.AddEdge(SuggestAssembly, Output)
	_ = suggest.AddEdge(Output, compose.END)
	return suggest
}
