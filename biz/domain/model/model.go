package model

import (
	"context"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/xh-polaris/innospark-core-api/biz/application/dto/core_api"
	_ "github.com/xh-polaris/innospark-core-api/biz/domain/deyu"
	"github.com/xh-polaris/innospark-core-api/biz/domain/msg"
)

func Completion(ctx context.Context, uid string, req *core_api.CompletionsReq, messages []*schema.Message) (any, error) {
	m := getModel(uid, req)
	if req.CompletionsOption.Stream {
		return doStream(ctx, m, req, messages)
	}
	return doGenerate(ctx, m, req, messages)
}

// 非流式
func doGenerate(ctx context.Context, m model.ToolCallingChatModel, req *core_api.CompletionsReq, messages []*schema.Message) (any, error) {
	resp, err := m.Generate(ctx, messages, getOpts(req.CompletionsOption)...)
	if err != nil {
		return nil, err
	}
	return msg.ConvFromEino(resp), nil
}

// 流式
func doStream(ctx context.Context, m model.ToolCallingChatModel, req *core_api.CompletionsReq, messages []*schema.Message) (any, error) {
	return nil, nil
}

// getModel 获取模型
func getModel(uid string, req *core_api.CompletionsReq) model.ToolCallingChatModel {
	return models[req.Model](uid, req)
}

type getModelFunc func(uid string, req *core_api.CompletionsReq) model.ToolCallingChatModel

var models = map[string]getModelFunc{}

func RegisterModel(name string, f getModelFunc) {
	models[name] = f
}

// getOpts TODO 处理模型 配置
func getOpts(option *core_api.CompletionsOption) []model.Option {
	// 自定义配置
	if option.UseDeepThink { // TODO 深度思考
	}
	if option.WithSuggest { // TODO 生成建议
	}
	// 模型调用配置
	var opts []model.Option
	return opts
}
