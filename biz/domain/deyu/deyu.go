package deyu

import (
	"context"
	"encoding/json"
	"strings"
	"sync"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/xh-polaris/innospark-core-api/biz/application/dto/core_api"
	dm "github.com/xh-polaris/innospark-core-api/biz/domain/model"
	"github.com/xh-polaris/innospark-core-api/biz/infra/config"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
)

func init() {
	dm.RegisterModel(DefaultModel, NewChatModel)
}

var (
	cli  *openai.ChatModel
	once sync.Once

	DefaultModel = "deyu-default"
	APIVersion   = "v1"
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
			APIKey:     config.GetConfig().Deyu.APIKey,
			BaseURL:    config.GetConfig().Deyu.BaseURL,
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

func (c *ChatModel) Stream(ctx context.Context, in []*schema.Message, opts ...model.Option) (sr *schema.StreamReader[*schema.Message], err error) {
	var reader *schema.StreamReader[*schema.Message]
	if reader, err = c.cli.Stream(ctx, in, opts...); err != nil {
		return nil, err
	}
	sr, sw := schema.Pipe[*schema.Message](5)
	go process(ctx, reader, sw)
	return sr, nil
}

func process(ctx context.Context, reader *schema.StreamReader[*schema.Message], writer *schema.StreamWriter[*schema.Message]) {
	defer reader.Close()
	defer writer.Close()

	var err error
	var data []byte
	var msg *schema.Message
	var text, think, suggest strings.Builder
	defer func() {
		info := ctx.Value(cst.CompletionInfo).(*dm.CompletionInfo)
		info.Text, info.Think, info.Suggest = text.String(), think.String(), suggest.String()
	}()

	var pass bool // 跳过一个\n\n
	var status = cst.EventMessageContentTypeText
	for {
		if msg, err = reader.Recv(); err != nil {
			writer.Send(nil, err)
			return
		}
		if pass && msg.Content == "\n\n" {
			pass = false
			continue
		}

		refine := &dm.RefineContent{}
		// 处理消息
		switch msg.Content {
		case cst.ThinkStart: // 深度思考内容开始
			status, pass = cst.EventMessageContentTypeThink, true
			continue
		case cst.SuggestStart: // 建议内容开始
			status, pass = cst.EventMessageContentTypeSuggest, true
			continue
		case cst.ThinkEnd:
			fallthrough // 切回普通内容
		case cst.SuggestEnd:
			status, pass = cst.EventMessageContentTypeText, true
			continue
		}
		switch status {
		case cst.EventMessageContentTypeText:
			refine.Text = msg.Content
			text.WriteString(msg.Content)
		case cst.EventMessageContentTypeThink:
			refine.Think = msg.Content
			think.WriteString(msg.Content)
		case cst.EventMessageContentTypeSuggest:
			refine.Suggest = msg.Content
			suggest.WriteString(msg.Content)
		}
		if data, err = json.Marshal(&refine); err != nil {
			continue
		}
		msg.Content, msg.Extra = string(data), map[string]any{cst.EventMessageContentType: status}
		writer.Send(msg, nil)
	}
}

func (c *ChatModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return c.cli.WithTools(tools)
}
