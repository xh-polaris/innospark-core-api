package flow

import (
	"context"
	"strings"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/xh-polaris/innospark-core-api/biz/conf"
	"github.com/xh-polaris/innospark-core-api/biz/domain/interaction"
	"github.com/xh-polaris/innospark-core-api/biz/domain/message/prompt_inject"
	"github.com/xh-polaris/innospark-core-api/biz/domain/model"
	"github.com/xh-polaris/innospark-core-api/biz/domain/state"
	"github.com/xh-polaris/innospark-core-api/biz/domain/state/event"
	tool "github.com/xh-polaris/innospark-core-api/biz/domain/tool"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	"github.com/xh-polaris/innospark-core-api/pkg/ctxcache"
)

type Flow = *compose.Graph[[]*schema.Message, *event.Event]

func StreamExecute(f Flow, ctx context.Context, input []*schema.Message) (output *schema.StreamReader[*event.Event], err error) {
	compiled, err := f.Compile(ctx)
	if err != nil {
		return nil, err
	}
	return compiled.Stream(ctx, input)
}

func BuildFlow(st *state.RelayContext) Flow {
	// 初始化状态
	gls := func(ctx context.Context) (s *state.RelayContext) {
		v, _ := ctxcache.Get[*state.RelayContext](ctx, cst.CtxState)
		return v
	}
	flow := compose.NewGraph[[]*schema.Message, *event.Event](compose.WithGenLocalState(gls))

	// 搜索
	if st.Info.ModelInfo.WebSearch {
		search := compose.InvokableLambda(func(ctx context.Context, input []*schema.Message) (_ []*schema.Message, err error) {
			return tool.Search(ctx, "bocha", conf.GetConfig().Bocha.APIKey, conf.GetConfig().Bocha.Template, input)
		})
		_ = flow.AddLambdaNode(WebSearch, search, compose.WithNodeName(WebSearch))
	}

	// TODO 建议

	// 模型节点
	cm := &model.ModelFactory{}
	_ = flow.AddChatModelNode(ChatModel, cm, compose.WithNodeName(ChatModel))

	// 将模型消息写入事件流
	assembleModelEvents := compose.TransformableLambda(func(ctx context.Context, input *schema.StreamReader[*schema.Message]) (_ *schema.StreamReader[*event.Event], err error) {
		go func() {
			var m *schema.Message
			for {
				m, err = input.Recv()
				st.EventStream.W.Send(&event.Event{Type: event.ChatModel, Message: m}, err)
			}
		}()
		return nil, nil
	})
	_ = flow.AddLambdaNode(ChatModelEventSend, assembleModelEvents, compose.WithNodeName(ChatModelEventSend))

	output := compose.TransformableLambda(func(ctx context.Context, input *schema.StreamReader[*event.Event]) (_ *schema.StreamReader[*event.Event], err error) {
		return st.EventStream.R, nil
	})
	_ = flow.AddLambdaNode(Output, output, compose.WithNodeName(Output))

	var pre = compose.START
	if st.Info.ModelInfo.WebSearch {
		_ = flow.AddEdge(compose.START, WebSearch)
		pre = WebSearch
	}
	// TODO 建议
	_ = flow.AddEdge(pre, ChatModel)
	_ = flow.AddEdge(ChatModel, ChatModelEventSend)
	_ = flow.AddEdge(ChatModelEventSend, Output)
	_ = flow.AddEdge(Output, compose.END)
	return flow
}

const (
	WebSearch          = "web-search"
	ChatModel          = "chat-model"
	ChatModelEventSend = "chat-model-event-send"
	Output             = "output"
)

// BuildChatModel 构建不同模型
func BuildChatModel(ctx context.Context, st *state.RelayContext, in []*schema.Message) (err error) {
	info := st.Info
	if info.ModelInfo.BotId == "code-gen" {
		info.ModelInfo.Model = model.Claude4Sonnet
		format, err := prompt.FromMessages(schema.FString, &schema.Message{Role: cst.User, Content: conf.GetConfig().ARK.CodeGenTemplate}).Format(ctx,
			map[string]any{"userQuery": info.OriginMessage.Content})
		if err != nil {
			return err
		}
		// 找到最近一条有效的用户消息, 主要是为了适配regen的情况
		for _, m := range in {
			if m.Role == schema.User && m.Content != "" {
				m.Content = format[0].Content
				break
			}
		}
	} else if strings.HasPrefix(info.ModelInfo.BotId, "intelligence-") { // coze 智能体
		info.ModelInfo.Model, info.ModelInfo.BotId = model.SelfCoze, info.ModelInfo.BotId[13:]
	} else if needVL(in) { // 需要视觉模型
		if !strings.HasSuffix(info.ModelInfo.Model, "-VL") {
			info.ModelInfo.Model += "-VL"
		}
	}
	if strings.HasPrefix(info.ModelInfo.BotId, "cotea-") {
		in, err = prompt_inject.CoTeaSysInject(ctx, in, st)
		if err != nil {
			return err
		}
	}
	// 写入模型事件
	if err = st.EventStream.Write(interaction.ModelEvent(
		info.ModelInfo.Model,
		info.ModelInfo.BotId,
		info.ModelInfo.BotName,
	)); err != nil {
		return err
	}
	return err
}

func needVL(in []*schema.Message) bool {
	for _, m := range in {
		if len(m.UserInputMultiContent) > 0 {
			return true
		}
	}
	return false
}
