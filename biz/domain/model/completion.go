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

// Completion 调用对话接口, 根据配置项选择流式或非流式
func Completion(ctx context.Context, uid string, req *core_api.CompletionsReq, messages []*schema.Message) (any, error) {
	info := &CompletionInfo{
		ConversationId: req.ConversationId,
		SectionId:      req.ConversationId,
		MessageId:      primitive.NewObjectID().Hex(),
		MessageIndex:   len(messages),
		ReplyId:        messages[0].Name,
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
	var chunk *schema.Message
	var idx int
	defer reader.Close()
	defer close(s.C)

	info := ctx.Value(cst.CompletionInfo).(*CompletionInfo)
	s.C <- eventMeta(info)  // 对话元数据事件
	s.C <- eventModel(info) // 模型信息事件

	for {
		chunk, err = reader.Recv()
		if err != nil {
			if err != io.EOF { // optimize 错误处理
				logx.Error("[domain model] do conv error: %v", err)
			}
			s.C <- eventEnd() // 结束事件
			return
		}
		s.C <- eventChat(idx, info, chunk) // 模型消息事件
	}
}

func eventMeta(info *CompletionInfo) *sse.Event {
	var err error
	var data []byte
	meta := &adaptor.EventMeta{
		MessageId:        info.MessageId,
		ConversationId:   info.ConversationId,
		SectionId:        info.SectionId,
		MessageIndex:     info.MessageIndex,
		ConversationType: cst.ConversationTypeText,
	}
	if data, err = json.Marshal(meta); err != nil {
		logx.Error("[domain model] event meta marshal error: %v", err)
	}
	return &sse.Event{Type: cst.EventMeta, Data: data}
}

func eventModel(info *CompletionInfo) *sse.Event {
	var err error
	var data []byte
	m := &adaptor.EventModel{Model: info.Model, BotId: info.BotId, BotName: info.BotName}
	if data, err = json.Marshal(m); err != nil {
		logx.Error("[domain m] event m marshal error: %v", err)
	}
	return &sse.Event{Type: cst.EventModel, Data: data}
}

// 将模型流式响应转换为sse事件
func eventChat(idx int, info *CompletionInfo, msg *schema.Message) *sse.Event {
	var err error
	var data []byte

	event := &adaptor.EventChat{
		Message:          &adaptor.ChatMessage{Content: msg.Content, ContentType: cst.MessageContentType},
		ConversationId:   info.ConversationId,
		SectionId:        info.SectionId,
		ReplyId:          info.ReplyId,
		IsDeleted:        true,
		Status:           cst.MessageStatus,
		InputContentType: cst.InputContentTypeText,
		MessageIndex:     idx,
		BotId:            info.BotId,
	}
	if data, err = json.Marshal(event); err != nil {
		logx.Error("[domain model] event chat marshal error: %v", err)
	}
	return &sse.Event{Type: cst.EventChat, Data: data}
}

func eventEnd() *sse.Event {
	return &sse.Event{Type: cst.EventEnd, Data: []byte(cst.EventEndValue)}
}
