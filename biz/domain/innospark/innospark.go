package innospark

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
	dm.RegisterModel(DefaultModel, NewDefaultChatModel)
	dm.RegisterModel(DeepThinkModel, NewDeepThinkChatModel)
}

var (
	cli           *openai.ChatModel
	defaultOnce   sync.Once
	deepThinkOnce sync.Once

	DefaultModel   = "InnoSpark"
	DeepThinkModel = "InnoSpark-R"
	APIVersion     = "v1"
)

type ChatModel struct {
	cli   *openai.ChatModel
	model string
}

func NewDefaultChatModel(ctx context.Context, uid string, req *core_api.CompletionsReq) model.ToolCallingChatModel {
	defaultOnce.Do(func() {
		var err error
		cli, err = openai.NewChatModel(ctx, &openai.ChatModelConfig{
			APIKey:     config.GetConfig().InnoSpark.DefaultAPIKey,
			BaseURL:    config.GetConfig().InnoSpark.DefaultBaseURL,
			APIVersion: APIVersion,
			Model:      DefaultModel,
			User:       &uid,
		})
		if err != nil {
			panic(err)
		}
	})
	return &ChatModel{cli: cli, model: DefaultModel}
}

func NewDeepThinkChatModel(ctx context.Context, uid string, req *core_api.CompletionsReq) model.ToolCallingChatModel {
	deepThinkOnce.Do(func() {
		var err error
		cli, err = openai.NewChatModel(ctx, &openai.ChatModelConfig{
			APIKey:     config.GetConfig().InnoSpark.DeepThinkAPIKey,
			BaseURL:    config.GetConfig().InnoSpark.DeepThinkBaseURL,
			APIVersion: APIVersion,
			Model:      DeepThinkModel,
			User:       &uid,
		})
		if err != nil {
			panic(err)
		}
	})
	return &ChatModel{cli: cli, model: DeepThinkModel}
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
	if c.model == DefaultModel {
		go process(ctx, reader, sw)
	} else {
		go deepThinkProcess(ctx, reader, sw)
	}
	return sr, nil
}

func process(ctx context.Context, reader *schema.StreamReader[*schema.Message], writer *schema.StreamWriter[*schema.Message]) {
	defer reader.Close()
	defer writer.Close()

	var err error
	var data []byte
	var msg *schema.Message
	var text, suggest strings.Builder
	defer func() {
		info := ctx.Value(cst.CompletionInfo).(*dm.CompletionInfo)
		info.Text, info.Suggest = text.String(), suggest.String()
	}()

	var status = cst.EventMessageContentTypeText
	for {
		if msg, err = reader.Recv(); err != nil {
			writer.Send(nil, err)
			return
		}

		refine := &dm.RefineContent{}
		// 处理消息
		switch msg.Content {
		case cst.SuggestStart: // 建议内容开始
			status = cst.EventMessageContentTypeSuggest
			continue
		case cst.SuggestEnd: // 切回普通内容
			status = cst.EventMessageContentTypeText
			continue
		}
		switch status {
		case cst.EventMessageContentTypeText:
			refine.Text = msg.Content
			text.WriteString(msg.Content)
		case cst.EventMessageContentTypeSuggest:
			refine.Suggest = msg.Content // optimize 建议内容处理
			suggest.WriteString(msg.Content)
		}
		if data, err = json.Marshal(&refine); err != nil {
			continue
		}
		msg.Content, msg.Extra = string(data), map[string]any{cst.EventMessageContentType: status}
		writer.Send(msg, nil)
	}
}

func deepThinkProcess(ctx context.Context, reader *schema.StreamReader[*schema.Message], writer *schema.StreamWriter[*schema.Message]) {
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

	var pass int       // 跳过次数
	var collect string // 收集跳过的内容
	var status = cst.EventMessageContentTypeText
	for {
		if msg, err = reader.Recv(); err != nil {
			writer.Send(nil, err)
			return
		}
		if pass > 0 { // 跳过指定个数
			pass, collect = pass-1, collect+msg.Content
			continue
		}

		refine := &dm.RefineContent{}
		// 深度思考需要处理 Think标签
		if len(msg.Content) > 0 && msg.Content[0] == '<' { // 如果是 < 开头, 可能为深度思考<think>标签, 考虑到都是三个, 所以收集三个
			pass, collect = 2, msg.Content
			continue
		}
		switch strings.Trim(collect, "\n") {
		case cst.ThinkStart:
			collect = ""
			status = cst.EventMessageContentTypeThink
		case cst.ThinkEnd:
			collect = ""
			status = cst.EventMessageContentTypeText
		}

		switch status {
		case cst.EventMessageContentTypeText:
			refine.Text = msg.Content
			text.WriteString(msg.Content)
		case cst.EventMessageContentTypeSuggest: // optimize 需要处理建议
			refine.Suggest = msg.Content
			suggest.WriteString(msg.Content)
		case cst.EventMessageContentTypeThink:
			refine.Think = msg.Content
			think.WriteString(msg.Content)
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
