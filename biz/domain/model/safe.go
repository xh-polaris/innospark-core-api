package model

import (
	"context"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/xh-polaris/innospark-core-api/biz/conf"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util"
)

func init() {
	RegisterModel(SafeModel, NewSafeChatModel)
}

const (
	SafeModel = "Safe-InnoSpark"
)

type SafeInnosparkChatModel struct {
	cli   *openai.ChatModel
	model string
}

func NewSafeChatModel(ctx context.Context, uid, _ string) (_ model.ToolCallingChatModel, err error) {
	var cli *openai.ChatModel
	cli, err = openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:     conf.GetConfig().InnoSpark.DefaultAPIKey,
		BaseURL:    conf.GetConfig().InnoSpark.DefaultBaseURL,
		APIVersion: APIVersion,
		Model:      SafeModel,
		User:       &uid,
		HTTPClient: util.NewDebugClient(),
	})
	if err != nil {
		return nil, err
	}

	return &SafeInnosparkChatModel{cli: cli, model: SafeModel}, nil
}

func (c *SafeInnosparkChatModel) Generate(ctx context.Context, in []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	return c.cli.Generate(ctx, in, opts...)
}

func (c *SafeInnosparkChatModel) Stream(ctx context.Context, in []*schema.Message, opts ...model.Option) (processReader *schema.StreamReader[*schema.Message], err error) {
	processReader, processWriter := schema.Pipe[*schema.Message](2)
	go func() {
		defer processWriter.Close()
		message, err := c.cli.Generate(ctx, in, opts...)
		util.AddExtra(message, cst.EventMessageContentType, cst.EventMessageContentTypeText)
		processWriter.Send(message, err)
	}()
	return processReader, nil
}

func (c *SafeInnosparkChatModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return c.cli.WithTools(tools)
}
