package model

import (
	"context"
	"encoding/json"
	"io"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/cloudwego/hertz/pkg/protocol/sse"
	"github.com/google/wire"
	"github.com/xh-polaris/innospark-core-api/biz/adaptor"
	"github.com/xh-polaris/innospark-core-api/biz/application/dto/core_api"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util/logx"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CompletionInfo 本轮对话的上下文信息
type CompletionInfo struct {
	ConversationId string             // 对话id
	SectionId      string             // 段落id
	MessageId      string             // 消息id
	MessageIndex   int                // 消息索引
	ReplyId        string             // 回复id
	Model          string             // 模型名称
	BotId          string             // 智能体id
	BotName        string             // 智能体名称
	UserId         string             // 用户id
	ContentType    int                // 内容类型
	MessageType    int                // 消息类型
	Text           string             // 对话内容
	Think          string             // 思考内容
	Suggest        string             // 建议内容
	SSE            *adaptor.SSEStream // SSE流
}

type RefineContent struct {
	Think   string `json:"think,omitempty"`
	Text    string `json:"text,omitempty"`
	Suggest string `json:"suggest,omitempty"`
}

type CompletionDomain struct {
	MsgDomain *MessageDomain
}

var CompletionDomainSet = wire.NewSet(wire.Struct(new(CompletionDomain), "*"))

// Completion 调用对话接口, 根据配置项选择流式或非流式
func (d *CompletionDomain) Completion(ctx context.Context, uid string, req *core_api.CompletionsReq, messages []*schema.Message) (any, error) {
	info := &CompletionInfo{
		ConversationId: req.ConversationId,
		SectionId:      req.ConversationId,
		MessageId:      primitive.NewObjectID().Hex(),
		MessageIndex:   len(messages),
		ReplyId:        messages[0].Name,
		Model:          req.Model,
		BotId:          req.BotId,
		UserId:         uid,
		ContentType:    cst.ContentTypeText,
		MessageType:    cst.MessageTypeText,
	}
	m, err := getModel(ctx, uid, req)
	if err != nil {
		return nil, err
	}
	// 调用模型
	if req.CompletionsOption.Stream {
		return d.doStream(ctx, info, m, uid, req, messages)
	}
	return d.doGenerate(ctx, info, m, uid, req, messages)
}

// 非流式
func (d *CompletionDomain) doGenerate(ctx context.Context, info *CompletionInfo, m model.ToolCallingChatModel, uid string, req *core_api.CompletionsReq, messages []*schema.Message) (any, error) {
	// 注入基本消息
	ctx = context.WithValue(ctx, cst.CompletionInfo, info)
	resp, err := m.Generate(ctx, messages, getOpts(req.CompletionsOption)...)
	if err != nil {
		return nil, err
	}
	return resp, cst.UnImplementErr // undo 非流式对话
}

// 流式
func (d *CompletionDomain) doStream(ctx context.Context, info *CompletionInfo, m model.ToolCallingChatModel, uid string, req *core_api.CompletionsReq, messages []*schema.Message) (_ any, err error) {
	// 注入基本消息, 提前创建sse流, 并注入到ctx中
	info.SSE = adaptor.NewSSEStream()
	ctx = context.WithValue(ctx, cst.CompletionInfo, info)
	ctx, cancel := context.WithCancelCause(ctx)

	var reader *schema.StreamReader[*schema.Message]
	if reader, err = m.Stream(ctx, messages, getOpts(req.CompletionsOption)...); err != nil {
		logx.Error("[domain model] do stream error: %v", err)
		cancel(err)
		return nil, err
	}
	go d.doSSE(ctx, cancel, reader, info.SSE)
	return info.SSE, nil
}

// 实际sse转换
func (d *CompletionDomain) doSSE(ctx context.Context, cancel context.CancelCauseFunc, reader *schema.StreamReader[*schema.Message], s *adaptor.SSEStream) {
	var err error
	var idx int
	var msg *schema.Message
	defer reader.Close()
	defer close(s.C)

	info := ctx.Value(cst.CompletionInfo).(*CompletionInfo)
	var text, think, suggest strings.Builder
	s.C <- eventMeta(info)  // 对话元数据事件
	s.C <- eventModel(info) // 模型信息事件

	for {
		select {
		case <-s.Done: // 提前结束
			cancel(cst.Interrupt)
			info.Text, info.Think, info.Suggest = text.String(), think.String(), suggest.String() // 记录各类型信息
			d.MsgDomain.ProcessHistory(ctx, info)                                                 // 处理历史记录
			return
		default: // 正常情况
			msg, err = reader.Recv() // 获取到的是refine后的msg
			if err != nil {          // optimize 错误处理
				logx.CondError(err != io.EOF, "[domain model] do conv error: %v", err)
				info.Text, info.Think, info.Suggest = text.String(), think.String(), suggest.String() // 记录各类型信息
				d.MsgDomain.ProcessHistory(ctx, info)                                                 // 处理历史记录, 这里不异步, 是考虑到如果异步, 可能历史记录还没存, 用户就发下一条, 导致历史记录不对
				s.C <- eventEnd()                                                                     // 结束事件
				return
			}
			var typ = cst.EventMessageContentTypeText
			if msg.Extra != nil { // 存在额外消息
				if t, ok := msg.Extra[cst.EventMessageContentType].(int); ok { // 消息类型
					typ = t
				}
			}
			s.C <- eventChat(idx, info, msg, typ)         // 模型消息事件
			collectMsg(&text, &think, &suggest, msg, typ) // 收集各类型消息
		}
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

func collectMsg(text, think, suggest *strings.Builder, msg *schema.Message, typ int) {
	content := msg.Extra[cst.RawMessage].(string)
	switch typ {
	case cst.EventMessageContentTypeText:
		text.WriteString(content)
	case cst.EventMessageContentTypeSuggest:
		suggest.WriteString(content)
	case cst.EventMessageContentTypeThink:
		think.WriteString(content)
	}
}
