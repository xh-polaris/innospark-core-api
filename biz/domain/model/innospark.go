package model

import (
	"context"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/xh-polaris/innospark-core-api/biz/conf"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util"
)

func init() {
	RegisterModel(DefaultModel, NewDefaultChatModel)
	RegisterModel(DeepThinkModel, NewDeepThinkChatModel)
}

const (
	DefaultModel   = "InnoSpark"
	DeepThinkModel = "InnoSpark-R"
	APIVersion     = "v1"
)

type InnosparkChatModel struct {
	cli   *openai.ChatModel
	model string
}

func NewDefaultChatModel(ctx context.Context, uid, _ string) (_ model.ToolCallingChatModel, err error) {
	var cli *openai.ChatModel
	cli, err = openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:     conf.GetConfig().InnoSpark.DefaultAPIKey,
		BaseURL:    conf.GetConfig().InnoSpark.DefaultBaseURL,
		APIVersion: APIVersion,
		Model:      DefaultModel,
		User:       &uid,
		HTTPClient: util.NewDebugClient(),
	})
	if err != nil {
		return nil, err
	}

	return &InnosparkChatModel{cli: cli, model: DefaultModel}, nil
}

func NewDeepThinkChatModel(ctx context.Context, uid, _ string) (_ model.ToolCallingChatModel, err error) {
	var cli *openai.ChatModel
	cli, err = openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:     conf.GetConfig().InnoSpark.DeepThinkAPIKey,
		BaseURL:    conf.GetConfig().InnoSpark.DeepThinkBaseURL,
		APIVersion: APIVersion,
		Model:      DeepThinkModel,
		User:       &uid,
		HTTPClient: util.NewDebugClient(),
	})
	if err != nil {
		return nil, err
	}
	return &InnosparkChatModel{cli: cli, model: DeepThinkModel}, nil
}

func (c *InnosparkChatModel) Generate(ctx context.Context, in []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	return c.cli.Generate(ctx, in, opts...)
}

func (c *InnosparkChatModel) Stream(ctx context.Context, in []*schema.Message, opts ...model.Option) (processReader *schema.StreamReader[*schema.Message], err error) {
	var raw *schema.StreamReader[*schema.Message]
	var single []*schema.Message
	for _, i := range in {
		newMsg := &schema.Message{
			Content:               i.Content,
			UserInputMultiContent: i.UserInputMultiContent,
			Extra:                 make(map[string]any),
		}
		// 复制 Extra 中的内容
		for k, v := range i.Extra {
			newMsg.Extra[k] = v
		}
		if len(newMsg.UserInputMultiContent) > 0 {
			newMsg.Content = util.GetInputText(newMsg)
			newMsg.UserInputMultiContent = nil
			if ocr, ok := newMsg.Extra["ocr"]; ok {
				newMsg.Content = "ocr的识别结果是" + ocr.(string) + "\n用户问题\n" + newMsg.Content
			}
		}
		single = append(single, newMsg)
	}
	if raw, err = c.cli.Stream(ctx, single, opts...); err != nil {
		return nil, err
	}
	processReader, processWriter := schema.Pipe[*schema.Message](5)
	if c.model == DefaultModel {
		go c.process(ctx, raw, processWriter)
	} else if c.model == DeepThinkModel {
		go c.deepThinkProcess(ctx, raw, processWriter)
	} else {
		raw.Close()
	}
	return processReader, nil
}

func (c *InnosparkChatModel) process(ctx context.Context, reader *schema.StreamReader[*schema.Message], writer *schema.StreamWriter[*schema.Message]) {
	defer reader.Close()
	defer writer.Close()

	var err error
	var msg *schema.Message

	var status = cst.EventMessageContentTypeText
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if msg, err = reader.Recv(); err != nil {
				writer.Send(nil, err)
				return
			}
			// 处理消息
			switch msg.Content {
			case cst.SuggestStart: // 建议内容开始
				status = cst.EventMessageContentTypeSuggest
				continue
			case cst.SuggestEnd: // 切回普通内容
				status = cst.EventMessageContentTypeText
				continue
			}
			util.AddExtra(msg, cst.EventMessageContentType, status)
			writer.Send(msg, nil)
		}
	}
}

func (c *InnosparkChatModel) deepThinkProcess(ctx context.Context, reader *schema.StreamReader[*schema.Message], writer *schema.StreamWriter[*schema.Message]) {
	defer reader.Close()
	defer writer.Close()

	var err error
	var msg *schema.Message

	var pass int       // 跳过次数
	var collect string // 收集跳过的内容
	var status = cst.EventMessageContentTypeText
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if msg, err = reader.Recv(); err != nil {
				writer.Send(nil, err)
				return
			}
			if pass > 0 { // 跳过指定个数
				pass, collect = pass-1, collect+msg.Content
				continue
			}

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
			default: // 都不是, 保持原状态
			}
			if collect != "" {
				msg.Content = collect + msg.Content
			}
			util.AddExtra(msg, cst.EventMessageContentType, status)
			writer.Send(msg, nil)
		}
	}
}

func (c *InnosparkChatModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return c.cli.WithTools(tools)
}
