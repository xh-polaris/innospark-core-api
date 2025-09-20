package model

import (
	"context"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/xh-polaris/innospark-core-api/biz/domain/info"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util"
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

func (m *ModelFactory) Generate(ctx context.Context, in []*schema.Message, opts ...model.Option) (_ *schema.Message, err error) {
	var r *info.RelayContext
	if r, err = util.GetState[*info.RelayContext](ctx); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(ctx)
	r.ModelCancel = cancel

	var cm model.ToolCallingChatModel
	if cm, err = m.get(ctx); err != nil {
		return nil, err
	}
	return cm.Generate(ctx, in, opts...)
}
func (m *ModelFactory) Stream(ctx context.Context, in []*schema.Message, opts ...model.Option) (_ *schema.StreamReader[*schema.Message], err error) {
	var r *info.RelayContext
	if r, err = util.GetState[*info.RelayContext](ctx); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(ctx)
	r.ModelCancel = cancel

	var cm model.ToolCallingChatModel
	if cm, err = m.get(ctx); err != nil {
		return nil, err
	}
	return cm.Stream(ctx, in, opts...)
}

func (m *ModelFactory) get(ctx context.Context) (cm model.ToolCallingChatModel, err error) {
	err = compose.ProcessState(ctx, func(ctx context.Context, s *info.RelayContext) (err error) {
		cm, err = getModel(ctx, s.ModelInfo.Model, s.UserId.Hex())
		return
	})
	return cm, err
}
