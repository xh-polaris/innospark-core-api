package graph

import (
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"

	"github.com/cloudwego/eino/schema"
	"github.com/cloudwego/hertz/pkg/protocol/sse"
	"github.com/xh-polaris/innospark-core-api/biz/adaptor"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util/logx"
)

type Transformer struct {
	relay                *RelayContext
	index                int
	text, think, suggest *strings.Builder
}

func NewTransformer(relay *RelayContext) (t *Transformer) {
	t = &Transformer{relay: relay, index: 0, text: &strings.Builder{}, think: &strings.Builder{}, suggest: &strings.Builder{}}
	return
}

func (t *Transformer) TransformToEvent(mr *schema.StreamReader[*schema.Message], sw *schema.StreamWriter[*sse.Event]) {
	defer mr.Close() // 关闭模型读
	defer sw.Close() // 关闭sse写

	sw.Send(t.meta())  // 元数据事件
	sw.Send(t.model()) // 模型信息事件

	var err error
	var msg *schema.Message
	defer t.collect() // 收集各类型消息
	for {
		select {
		case <-t.relay.SSE.Done: // sse中断
			t.relay.ModelCancel()
			return
		default:
			msg, err = mr.Recv()
			if err != nil {
				logx.CondError(!errors.Is(err, io.EOF), "[graph transformer] recv err:", err)
				sw.Send(t.end(err))
				return
			}
			sw.Send(t.chat(msg)) // chat 事件
		}
	}
}

func (t *Transformer) chat(msg *schema.Message) (*sse.Event, error) {
	var typ = cst.EventMessageContentTypeText
	content := msg.Content
	if msg.Extra != nil { // 存在额外消息
		if tp, ok := msg.Extra[cst.EventMessageContentType].(int); ok { // 消息类型
			typ = tp
		}
		content = msg.Extra[cst.RawMessage].(string)
	}
	switch typ {
	case cst.EventMessageContentTypeText:
		t.text.WriteString(content)
	case cst.EventMessageContentTypeThink:
		t.think.WriteString(content)
	case cst.EventMessageContentTypeSuggest:
		t.suggest.WriteString(content)
	}

	chat := &adaptor.EventChat{
		Message:          &adaptor.ChatMessage{Content: msg.Content, ContentType: typ},
		ConversationId:   t.relay.ConversationId.Hex(),
		SectionId:        t.relay.SectionId.Hex(),
		ReplyId:          t.relay.ReplyId,
		IsDelta:          true,
		Status:           cst.MessageStatus,
		InputContentType: cst.InputContentTypeText,
		MessageIndex:     int(t.relay.MessageInfo.AssistantMessage.Index),
		BotId:            t.relay.ModelInfo.BotId,
	}
	return event(t.id(), chat, cst.EventChat), nil
}

func (t *Transformer) meta() (*sse.Event, error) {
	meta := &adaptor.EventMeta{
		MessageId:        t.relay.MessageInfo.AssistantMessage.MessageId.Hex(),
		ConversationId:   t.relay.ConversationId.Hex(),
		SectionId:        t.relay.SectionId.Hex(),
		MessageIndex:     int(t.relay.MessageInfo.AssistantMessage.Index),
		ConversationType: cst.ConversationTypeText,
		ReplyId:          t.relay.ReplyId,
	}
	return event(t.id(), meta, cst.EventMeta), nil
}

func (t *Transformer) model() (*sse.Event, error) {
	m := &adaptor.EventModel{Model: t.relay.ModelInfo.Model, BotId: t.relay.ModelInfo.BotId, BotName: t.relay.ModelInfo.BotName}
	return event(t.id(), m, cst.EventModel), nil
}

func (t *Transformer) end(err error) (*sse.Event, error) {
	return &sse.Event{Type: cst.EventEnd, Data: []byte(cst.EventEndValue)}, err
}

func (t *Transformer) id() string {
	i := strconv.Itoa(t.index)
	t.index++
	return i
}

func event(index string, obj any, typ string) *sse.Event {
	var err error
	var data []byte
	if data, err = json.Marshal(obj); err != nil {
		logx.Error("[graph sse] event marshal error: %v", err)
	}
	return &sse.Event{ID: index, Type: typ, Data: data}
}

func (t *Transformer) collect() {
	t.relay.MessageInfo.Text = t.text.String()
	t.relay.MessageInfo.Think = t.think.String()
	t.relay.MessageInfo.Suggest = t.suggest.String()
}

func SSE(relay *RelayContext, input *schema.StreamReader[*sse.Event]) (_ *RelayContext, err error) {
	var et *sse.Event
	sw := sse.NewWriter(relay.RequestContext)
	for {
		et, err = input.Recv()
		if err != nil { // sse 提前结束
			logx.CondError(!errors.Is(err, io.EOF), "[sse] recv err: %v", err)
			break
		}
		err = sw.Write(et) // 写入事件
		if err != nil {
			logx.CondError(!errors.Is(err, io.EOF), "[sse] write err: %v", err)
			break
		}
	}
	input.Close()
	_ = sw.Close()

	if err == io.EOF {
		err = nil
	}
	return relay, err
}
