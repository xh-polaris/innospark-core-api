package graph

import (
	"encoding/json"
	"errors"
	"io"
	"strings"

	"github.com/cloudwego/eino/schema"
	"github.com/cloudwego/hertz/pkg/protocol/sse"
	info "github.com/xh-polaris/innospark-core-api/biz/domain/info"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	mmsg "github.com/xh-polaris/innospark-core-api/biz/infra/mapper/message"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util/logx"
)

type Transformer struct {
	relay      *info.RelayContext
	containers map[int]*strings.Builder
	code       []*strings.Builder
	codeTyp    []string
}

func NewTransformer(relay *info.RelayContext) (t *Transformer) {
	t = &Transformer{relay: relay, containers: map[int]*strings.Builder{
		cst.EventMessageContentTypeText: {}, cst.EventMessageContentTypeThink: {},
		cst.EventMessageContentTypeSuggest: {},
	}}
	return
}

func (t *Transformer) TransformToEvent(mr *schema.StreamReader[*schema.Message], sw *schema.StreamWriter[*sse.Event]) {
	defer mr.Close() // 关闭模型读
	defer sw.Close() // 关闭sse写

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
			refine := &info.RefineContent{}
			content, typ := refine.SetContentWithTyp(msg.Content, msg.Extra[cst.EventMessageContentType].(int))
			if typ == cst.EventMessageContentTypeCodeType {
				t.codeTyp = append(t.codeTyp, content)
				t.code = append(t.code, &strings.Builder{})
			} else if typ == cst.EventMessageContentTypeCode {
				t.code[len(t.code)-1].WriteString(content)
			} else {
				t.containers[typ].WriteString(content)
			}
			sw.Send(t.chat(refine, typ)) // chat 事件
		}
	}
}

func (t *Transformer) chat(refine *info.RefineContent, typ int) (*sse.Event, error) {
	data, err := json.Marshal(refine)
	if err != nil {
		return nil, err
	}
	return t.relay.ChatEvent(string(data), typ), nil
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
	t.relay.MessageInfo.Text = t.containers[cst.EventMessageContentTypeText].String()
	t.relay.MessageInfo.Think = t.containers[cst.EventMessageContentTypeThink].String()
	t.relay.MessageInfo.Suggest = t.containers[cst.EventMessageContentTypeSuggest].String()
	var codes []*mmsg.Code
	for i, code := range t.code {
		codes = append(codes, &mmsg.Code{
			Index:    int32(i),
			CodeType: t.codeTyp[i],
			Code:     code.String(),
		})
	}
	t.relay.MessageInfo.Code = codes
}

func SSE(relay *info.RelayContext, input *schema.StreamReader[*sse.Event]) (_ *info.RelayContext, err error) {
	var et *sse.Event
	sw := sse.NewWriter(relay.RequestContext)
	for {
		et, err = input.Recv()
		if et != nil {
			err = sw.Write(et) // 写入事件
		}
		if err != nil {
			logx.CondError(!errors.Is(err, io.EOF), "[sse] write err: %v", err)
			err = nil // 为了能进入后续的存储历史记录节点
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
