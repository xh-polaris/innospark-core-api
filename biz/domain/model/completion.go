package model

import (
	"context"
	"encoding/json"
	"io"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/cloudwego/hertz/pkg/protocol/sse"
	"github.com/xh-polaris/innospark-core-api/biz/adaptor"
	"github.com/xh-polaris/innospark-core-api/biz/application/dto/core_api"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util/logx"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CompletionInfo 本轮对话的上下文信息
type CompletionInfo struct {
	ConversationId string
	SectionId      string
	MessageId      string
	MessageIndex   int
	ReplyId        string
	Model          string
	BotId          string
	BotName        string
	UserId         string
	SSE            *adaptor.SSEStream
}

type RefineContent struct {
	Think   string `json:"think,omitempty"`
	Text    string `json:"text,omitempty"`
	Suggest string `json:"suggest,omitempty"`
}

// Completion 调用对话接口, 根据配置项选择流式或非流式
func Completion(ctx context.Context, uid string, req *core_api.CompletionsReq, messages []*schema.Message) (any, error) {
	info := &CompletionInfo{
		ConversationId: req.ConversationId,
		SectionId:      req.ConversationId,
		MessageId:      primitive.NewObjectID().Hex(),
		MessageIndex:   len(messages),
		ReplyId:        messages[0].Name,
		Model:          req.Model,
		BotId:          req.BotId,
		UserId:         uid,
	}
	m := getModel(ctx, uid, req)
	// 调用模型
	if req.CompletionsOption.Stream {
		return doStream(ctx, info, m, uid, req, messages)
	}
	return doGenerate(ctx, info, m, uid, req, messages)
}

// 非流式
func doGenerate(ctx context.Context, info *CompletionInfo, m model.ToolCallingChatModel, uid string, req *core_api.CompletionsReq, messages []*schema.Message) (any, error) {
	// 注入基本消息
	ctx = context.WithValue(ctx, cst.CompletionInfo, info)
	resp, err := m.Generate(ctx, messages, getOpts(req.CompletionsOption)...)
	if err != nil {
		return nil, err
	}
	return resp, cst.UnImplementErr // undo 非流式对话
}

// 流式
func doStream(ctx context.Context, info *CompletionInfo, m model.ToolCallingChatModel, uid string, req *core_api.CompletionsReq, messages []*schema.Message) (_ any, err error) {
	// 注入基本消息, 提前创建sse流, 并注入到ctx中
	info.SSE = adaptor.NewSSEStream()
	ctx = context.WithValue(ctx, cst.CompletionInfo, info)

	var reader *schema.StreamReader[*schema.Message]
	if reader, err = m.Stream(ctx, messages, getOpts(req.CompletionsOption)...); err != nil {
		logx.Error("[domain model] do stream error: %v", err)
		return nil, err
	}
	go doSSE(ctx, reader, info.SSE)
	return info.SSE, nil
}

// 实际sse转换
func doSSE(ctx context.Context, reader *schema.StreamReader[*schema.Message], s *adaptor.SSEStream) {
	var err error
	var idx int
	var msg *schema.Message
	defer reader.Close()
	defer close(s.C)

	info := ctx.Value(cst.CompletionInfo).(*CompletionInfo)
	s.C <- eventMeta(info)  // 对话元数据事件
	s.C <- eventModel(info) // 模型信息事件

	for {
		msg, err = reader.Recv()
		if err != nil { // optimize 错误处理
			logx.CondError(err != io.EOF, "[domain model] do conv error: %v", err)
			s.C <- eventEnd() // 结束事件
			return
		}
		var typ = cst.MessageContentTypeText
		if msg.Extra != nil { // 存在额外消息
			if t, ok := msg.Extra[cst.MessageContentType].(int); ok { // 消息类型
				typ = t
			}
			// TODO 历史记录
		}

		s.C <- eventChat(idx, info, msg, typ) // 模型消息事件
	}
}

func eventMeta(info *CompletionInfo) *sse.Event {
	meta := &adaptor.EventMeta{
		MessageId:        info.MessageId,
		ConversationId:   info.ConversationId,
		SectionId:        info.SectionId,
		MessageIndex:     info.MessageIndex,
		ConversationType: cst.ConversationTypeText,
	}
	return event(meta, cst.EventMeta)
}

func eventModel(info *CompletionInfo) *sse.Event {
	m := &adaptor.EventModel{Model: info.Model, BotId: info.BotId, BotName: info.BotName}
	return event(m, cst.EventModel)
}

// 将模型流式响应转换为sse事件
func eventChat(idx int, info *CompletionInfo, msg *schema.Message, typ int) *sse.Event {
	chat := &adaptor.EventChat{
		Message:          &adaptor.ChatMessage{Content: msg.Content, ContentType: typ},
		ConversationId:   info.ConversationId,
		SectionId:        info.SectionId,
		ReplyId:          info.ReplyId,
		IsDeleted:        true,
		Status:           cst.MessageStatus,
		InputContentType: cst.InputContentTypeText,
		MessageIndex:     idx,
		BotId:            info.BotId,
	}
	return event(chat, cst.EventChat)
}

func eventEnd() *sse.Event {
	return &sse.Event{Type: cst.EventEnd, Data: []byte(cst.EventEndValue)}
}

func event(obj any, typ string) *sse.Event {
	var err error
	var data []byte
	if data, err = json.Marshal(obj); err != nil {
		logx.Error("[domain model] event marshal error: %v", err)
	}
	return &sse.Event{Type: typ, Data: data}
}
