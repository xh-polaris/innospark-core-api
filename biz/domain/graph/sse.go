package graph

import (
	"errors"
	"io"
	"strings"

	"github.com/cloudwego/eino/schema"
	"github.com/cloudwego/hertz/pkg/protocol/sse"
	info "github.com/xh-polaris/innospark-core-api/biz/domain/info"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util/logx"
)

type Transformer struct {
	relay                *info.RelayContext
	text, think, suggest *strings.Builder
}

func NewTransformer(relay *info.RelayContext) (t *Transformer) {
	t = &Transformer{relay: relay, text: &strings.Builder{}, think: &strings.Builder{}, suggest: &strings.Builder{}}
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
	// 收集信息
	switch typ {
	case cst.EventMessageContentTypeText:
		t.text.WriteString(content)
	case cst.EventMessageContentTypeThink:
		t.think.WriteString(content)
	case cst.EventMessageContentTypeSuggest:
		t.suggest.WriteString(content)
	}

	return t.relay.ChatEvent(msg, typ), nil
}

func (t *Transformer) meta() (*sse.Event, error) {
	return t.relay.MetaEvent(), nil
}

func (t *Transformer) model() (*sse.Event, error) {
	return t.relay.ModelEvent(), nil
}

func (t *Transformer) end(err error) (*sse.Event, error) {
	return t.relay.EndEvent(), err
}

func (t *Transformer) collect() {
	t.relay.MessageInfo.Text = t.text.String()
	t.relay.MessageInfo.Think = t.think.String()
	t.relay.MessageInfo.Suggest = t.suggest.String()
}

func SSE(relay *info.RelayContext, input *schema.StreamReader[*sse.Event]) (_ *info.RelayContext, err error) {
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
