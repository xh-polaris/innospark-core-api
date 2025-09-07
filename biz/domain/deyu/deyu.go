package deyu

import (
	"context"
	"sync"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/xh-polaris/innospark-core-api/biz/application/dto/core_api"
	dm "github.com/xh-polaris/innospark-core-api/biz/domain/model"
	"github.com/xh-polaris/innospark-core-api/biz/infra/config"
)

func init() {
	dm.RegisterModel(DefaultModel, NewChatModel)
}

var (
	cli  *openai.ChatModel
	once sync.Once

	DefaultModel = "deyu-default"
	APIVersion   = "v1"
	BaseURL      = "https://edusys1.sii.edu.cn/deyu/14b/bzr_only/v1"
)

// ChatModel 德育大模型
// 在openai模型基础上封装
type ChatModel struct {
	cli *openai.ChatModel
}

func NewChatModel(ctx context.Context, uid string, req *core_api.CompletionsReq) model.ToolCallingChatModel {
	once.Do(func() {
		var err error
		cli, err = openai.NewChatModel(ctx, &openai.ChatModelConfig{
			APIKey:     config.GetConfig().DeyuAPIKey,
			BaseURL:    BaseURL,
			APIVersion: APIVersion,
			Model:      DefaultModel,
			User:       &uid,
		})
		if err != nil {
			panic(err)
		}
	})
	return &ChatModel{cli: cli}
}

func (c *ChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	return c.cli.Generate(ctx, input, opts...)
}

func (c *ChatModel) Stream(ctx context.Context, in []*schema.Message, opts ...model.Option) (outStream *schema.StreamReader[*schema.Message], err error) {
	return c.cli.Stream(ctx, in, opts...)
}

func (c *ChatModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return c.cli.WithTools(tools)
}
