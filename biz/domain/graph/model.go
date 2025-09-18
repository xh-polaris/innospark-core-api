package graph

import (
	"context"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

type getModelFunc func(ctx context.Context, uid string) (model.ToolCallingChatModel, error)

var models = map[string]getModelFunc{}

func RegisterModel(name string, f getModelFunc) {
	models[name] = f
}

// getModel 获取模型
func getModel(ctx context.Context, model, uid string) (model.ToolCallingChatModel, error) {
	return models[model](ctx, uid)
}

type ModelFactory struct{}

func (m *ModelFactory) Generate(ctx context.Context, in []*schema.Message, opts ...model.Option) (out *schema.Message, err error) {
	var relay *RelayContext
	if relay, err = GetState(ctx); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(ctx)
	relay.ModelCancel = cancel

	var cm model.ToolCallingChatModel
	if cm, err = m.get(ctx); err != nil {
		return nil, err
	}
	return cm.Generate(ctx, in, opts...)
}
func (m *ModelFactory) Stream(ctx context.Context, in []*schema.Message, opts ...model.Option) (r *schema.StreamReader[*schema.Message], err error) {
	var relay *RelayContext
	if relay, err = GetState(ctx); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(ctx)
	relay.ModelCancel = cancel

	var cm model.ToolCallingChatModel
	if cm, err = m.get(ctx); err != nil {
		return nil, err
	}
	return cm.Stream(ctx, in, opts...)
}

func (m *ModelFactory) get(ctx context.Context) (cm model.ToolCallingChatModel, err error) {
	err = compose.ProcessState(ctx, func(ctx context.Context, s *RelayContext) (err error) {
		cm, err = getModel(ctx, s.ModelInfo.Model, s.UserId.Hex())
		return
	})
	return cm, err
}
