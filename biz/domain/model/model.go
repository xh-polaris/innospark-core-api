package model

import (
	"context"
	"errors"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/xh-polaris/innospark-core-api/biz/domain/state"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util"
)

type getModelFunc func(ctx context.Context, uid, botId string) (model.ToolCallingChatModel, error)

var models = map[string]getModelFunc{}
var NoSuchModel = errors.New("no such model")

func RegisterModel(name string, f getModelFunc) {
	models[name] = f
}

// getModel 获取模型
func getModel(ctx context.Context, model, uid, botId string) (model.ToolCallingChatModel, error) {
	fn, ok := models[model]
	if !ok {
		return nil, NoSuchModel
	}
	return fn(ctx, uid, botId)
}

type ModelFactory struct {
	// 覆盖消息, 优先级高于全局消息
	model string
	botId string
}

func NewModelFactory(opts ...ModelFactoryOpt) model.ToolCallingChatModel {
	m := &ModelFactory{}
	for _, f := range opts {
		f(m)
	}
	return m
}

type ModelFactoryOpt func(*ModelFactory)

func WithModel(model string) ModelFactoryOpt {
	return func(m *ModelFactory) {
		m.model = model
	}
}

func WithBotId(botId string) ModelFactoryOpt {
	return func(m *ModelFactory) {
		m.botId = botId
	}
}

func (m *ModelFactory) Generate(ctx context.Context, in []*schema.Message, opts ...model.Option) (_ *schema.Message, err error) {
	// messages翻转顺序, 调用模型时消息应该正序
	var reverse []*schema.Message
	for i := len(in) - 1; i >= 0; i-- {
		in[i].Name = ""
		reverse = append(reverse, in[i])
	}
	var cm model.ToolCallingChatModel
	if cm, err = m.get(ctx); err != nil {
		return nil, err
	}
	return cm.Generate(ctx, reverse, opts...)
}
func (m *ModelFactory) Stream(ctx context.Context, in []*schema.Message, opts ...model.Option) (_ *schema.StreamReader[*schema.Message], err error) {
	// messages翻转顺序, 调用模型时消息应该正序
	var reverse []*schema.Message
	for i := len(in) - 1; i >= 0; i-- {
		if in[i].Content != "" || len(in[i].UserInputMultiContent) != 0 || len(in[i].AssistantGenMultiContent) != 0 {
			in[i].Name = ""
			reverse = append(reverse, in[i])
		}
	}
	var cm model.ToolCallingChatModel
	if cm, err = m.get(ctx); err != nil {
		return nil, err
	}
	return cm.Stream(ctx, reverse, opts...)
}

func (m *ModelFactory) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return nil, nil
}

func (m *ModelFactory) get(ctx context.Context) (cm model.ToolCallingChatModel, err error) {
	err = compose.ProcessState(ctx, func(ctx context.Context, s *state.RelayContext) (err error) {
		mo := util.ZeroDefault(m.model, s.Info.ModelInfo.Model)
		botId := util.ZeroDefault(m.botId, s.Info.ModelInfo.BotId)
		cm, err = getModel(ctx, mo, s.Info.UserId.Hex(), botId)
		return
	})
	return cm, err
}
