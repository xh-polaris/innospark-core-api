package model

import (
	"context"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/xh-polaris/innospark-core-api/biz/domain/info"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util"
)

type getModelFunc func(ctx context.Context, uid, botId string) (model.ToolCallingChatModel, error)

var models = map[string]getModelFunc{}

func RegisterModel(name string, f getModelFunc) {
	models[name] = f
}

// getModel 获取模型
func getModel(ctx context.Context, model, uid, botId string) (model.ToolCallingChatModel, error) {
	return models[model](ctx, uid, botId)
}

type ModelFactory struct{}

func (m *ModelFactory) Generate(ctx context.Context, in []*schema.Message, opts ...model.Option) (_ *schema.Message, err error) {
	var r *info.RelayContext
	if r, err = util.GetState[*info.RelayContext](ctx); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(ctx)
	r.ModelCancel = cancel

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
	var r *info.RelayContext
	if r, err = util.GetState[*info.RelayContext](ctx); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(ctx)
	r.ModelCancel = cancel

	// messages翻转顺序, 调用模型时消息应该正序
	var reverse []*schema.Message
	for i := len(in) - 1; i >= 0; i-- {
		if in[i].Content != "" {
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

func (m *ModelFactory) get(ctx context.Context) (cm model.ToolCallingChatModel, err error) {
	err = compose.ProcessState(ctx, func(ctx context.Context, s *info.RelayContext) (err error) {
		cm, err = getModel(ctx, s.ModelInfo.Model, s.UserId.Hex(), s.ModelInfo.BotId)
		return
	})
	return cm, err
}
