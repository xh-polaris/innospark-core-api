package model

import (
	"github.com/cloudwego/eino/components/model"
	"github.com/xh-polaris/innospark-core-api/biz/application/dto/core_api"
)

// getOpts TODO 处理模型 配置
func getOpts(option *core_api.CompletionsOption) []model.Option {
	// 自定义配置
	if option.UseDeepThink { // optimize 深度思考
	}
	if option.WithSuggest { // optimize 生成建议
	}
	// 模型调用配置
	var opts []model.Option
	return opts
}
