package model

import (
	"context"

	"github.com/cloudwego/eino-ext/components/model/qwen"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/xh-polaris/innospark-core-api/biz/infra/config"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util"
)

func init() {
	RegisterModel(InnoSparkVL, NewQwenChatModel)
	RegisterModel(InnoSparkRVL, NewQwenChatModelWithThinking)
}

const (
	InnoSparkVL  = "InnoSpark-VL"
	InnoSparkRVL = "InnoSpark-R-VL"
	VLMFlash     = "qwen3-vl-flash"
)

type QwenChatModel struct {
	cli   *qwen.ChatModel
	model string
}

func NewQwenChatModel(ctx context.Context, uid, _ string) (_ model.ToolCallingChatModel, err error) {
	var cli *qwen.ChatModel
	cli, err = qwen.NewChatModel(ctx, &qwen.ChatModelConfig{
		APIKey:     config.GetConfig().InnoSpark.VlmAPIKey,
		BaseURL:    config.GetConfig().InnoSpark.VlmURL,
		Model:      VLMFlash,
		User:       &uid,
		HTTPClient: util.NewDebugClient(),
	})
	if err != nil {
		return nil, err
	}

	return &QwenChatModel{cli: cli, model: InnoSparkVL}, nil
}

func NewQwenChatModelWithThinking(ctx context.Context, uid, _ string) (_ model.ToolCallingChatModel, err error) {
	var cli *qwen.ChatModel
	cli, err = qwen.NewChatModel(ctx, &qwen.ChatModelConfig{
		APIKey:         config.GetConfig().InnoSpark.VlmAPIKey,
		BaseURL:        config.GetConfig().InnoSpark.VlmURL,
		Model:          VLMFlash,
		User:           &uid,
		HTTPClient:     util.NewDebugClient(),
		EnableThinking: util.Of(true),
	})
	if err != nil {
		return nil, err
	}

	return &QwenChatModel{cli: cli, model: InnoSparkRVL}, nil
}

func (c *QwenChatModel) Generate(ctx context.Context, in []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	return c.cli.Generate(ctx, in, opts...)
}

func (c *QwenChatModel) Stream(ctx context.Context, in []*schema.Message, opts ...model.Option) (processReader *schema.StreamReader[*schema.Message], err error) {
	var raw *schema.StreamReader[*schema.Message]
	if raw, err = c.cli.Stream(ctx, in, opts...); err != nil {
		return nil, err
	}
	processReader, processWriter := schema.Pipe[*schema.Message](5)
	go c.process(ctx, raw, processWriter)
	return processReader, nil
}

func (c *QwenChatModel) process(ctx context.Context, reader *schema.StreamReader[*schema.Message], writer *schema.StreamWriter[*schema.Message]) {
	defer reader.Close()
	defer writer.Close()

	var err error
	var msg *schema.Message
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if msg, err = reader.Recv(); err != nil {
				writer.Send(nil, err)
				return
			}
			if msg.ReasoningContent != "" {
				msg.Content = msg.ReasoningContent
				util.AddExtra(msg, cst.EventMessageContentType, cst.EventMessageContentTypeThink)
			} else {
				util.AddExtra(msg, cst.EventMessageContentType, cst.EventMessageContentTypeText)
			}
			writer.Send(msg, nil)
		}
	}
}

func (c *QwenChatModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return c.cli.WithTools(tools)
}
