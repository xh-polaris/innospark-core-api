package model

import (
	"context"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/coze-dev/coze-go"
	"github.com/xh-polaris/innospark-core-api/biz/infra/config"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util/errorx"
)

func init() {
	RegisterModel(SelfCoze, NewSelfCozeModel)
}

const (
	SelfCoze = "self-coze"
	BaseURL  = "https://coze.aiecnu.net"
)

var autoSaveHistory = false
var isStream = true

type SelfCozeModel struct {
	model string
	cli   *coze.CozeAPI
	uid   string
	botId string
}

func NewSelfCozeModel(ctx context.Context, uid, botId string) (_ model.ToolCallingChatModel, err error) {
	cozeCli := coze.NewCozeAPI(coze.NewTokenAuth(config.GetConfig().Coze.PAT),
		coze.WithBaseURL(BaseURL),
		coze.WithHttpClient(util.NewDebugClient()))
	return &SelfCozeModel{SelfCoze, &cozeCli, uid, botId}, nil
}

func (c *SelfCozeModel) Generate(ctx context.Context, in []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	return nil, errorx.New(cst.UnImplementErrCode)
}

func (c *SelfCozeModel) Stream(ctx context.Context, in []*schema.Message, opts ...model.Option) (sr *schema.StreamReader[*schema.Message], err error) {
	// messages翻转顺序, 调用模型时消息应该正序
	var reverse []*schema.Message
	for i := len(in) - 1; i >= 0; i-- {
		in[i].Name = ""
		reverse = append(reverse, in[i])
	}
	sr, sw := schema.Pipe[*schema.Message](5)
	request := &coze.CreateChatsReq{
		BotID:           c.botId,
		UserID:          c.uid,
		Messages:        e2c(reverse),
		AutoSaveHistory: &autoSaveHistory,
		Stream:          &isStream,
		ConnectorID:     "1024",
		ConversationID:  "0",
	}
	var stream coze.Stream[coze.ChatEvent]
	if stream, err = c.cli.Chat.Stream(ctx, request); err != nil {
		return nil, err
	}
	go process(ctx, stream, sw)
	return sr, nil
}

func process(ctx context.Context, reader coze.Stream[coze.ChatEvent], writer *schema.StreamWriter[*schema.Message]) {
	defer func() { _ = reader.Close() }()
	defer writer.Close()

	var err error
	var event *coze.ChatEvent
	var msg *schema.Message

	var pass int       // 跳过次数
	var collect string // 收集跳过的内容
	var status = cst.EventMessageContentTypeText
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if event, err = reader.Recv(); err != nil {
				writer.Send(nil, err)
				return
			}
			if event.Message == nil || event.Event != coze.ChatEventConversationMessageDelta {
				if event.Message != nil && event.Message.Type == coze.MessageTypeFollowUp {
					msg = ce2e(event)
					util.AddExtra(msg, cst.EventMessageContentType, cst.EventMessageContentTypeSuggest)
					writer.Send(msg, nil)
				}
				continue
			}
			msg = ce2e(event)

			if pass > 0 { // 跳过指定个数
				pass, collect = pass-1, collect+msg.Content
				continue
			}
			// 深度思考需要处理 Think标签
			if len(msg.Content) > 0 && msg.Content[0] == '<' { // 如果是 < 开头, 可能为深度思考<think>标签, 考虑到都是三个, 所以收集三个
				pass, collect = 2, msg.Content
				continue
			}
			// 处理消息
			switch strings.Trim(collect, "\n") {
			case cst.ThinkStart:
				collect = ""
				status = cst.EventMessageContentTypeThink
			case cst.ThinkEnd:
				collect = ""
				status = cst.EventMessageContentTypeText
			default:
			}
			if collect != "" {
				msg.Content = collect + msg.Content
			}
			util.AddExtra(msg, cst.EventMessageContentType, status)
			writer.Send(msg, nil)
		}
	}
}

func (c *SelfCozeModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return c, nil
}

func e2c(in []*schema.Message) (c []*coze.Message) {
	for _, i := range in {
		m := &coze.Message{
			Role:             coze.MessageRole(i.Role),
			Content:          i.Content,
			ReasoningContent: i.ReasoningContent,
			Type:             "question",
			ContentType:      "text",
		}
		c = append(c, m)
	}
	return
}

func ce2e(e *coze.ChatEvent) *schema.Message {
	return c2e(e.Message)
}

func c2e(c *coze.Message) *schema.Message {
	return &schema.Message{
		Role:    schema.Assistant,
		Content: c.Content,
	}
}
