package graph

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/cloudwego/eino/schema"
	"github.com/cloudwego/hertz/pkg/protocol/sse"
	"github.com/xh-polaris/innospark-core-api/biz/domain/info"
	"github.com/xh-polaris/innospark-core-api/biz/infra/config"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	mmsg "github.com/xh-polaris/innospark-core-api/biz/infra/mapper/message"
	"github.com/xh-polaris/innospark-core-api/biz/infra/mapper/user"
	"github.com/xh-polaris/innospark-core-api/pkg/ac"
	"github.com/xh-polaris/innospark-core-api/pkg/errorx"
	"github.com/xh-polaris/innospark-core-api/pkg/logs"
	"github.com/xh-polaris/innospark-core-api/types/errno"
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
	var sb strings.Builder
	var cnt int
	defer t.collect()         // 收集各类型消息
	defer sw.Send(t.end(err)) // 发送结束消息
	defer func(sb *strings.Builder) { // 最终校验是否有违禁词
		if t.checkSensitive(sb) {
			sw.Send(t.error(fmt.Sprintf("这个话题暂时还不能聊哦, 也请不要引导我聊敏感话题否则会被封禁哦")))
		}
	}(&sb)
	for {
		select {
		case <-t.relay.SSE.Done: // sse中断
			t.relay.ModelCancel()
			return
		default:
			msg, err = mr.Recv()
			if err != nil {
				logs.CondErrorf(!errors.Is(err, io.EOF), "[graph transformer] recv err:", errorx.ErrorWithoutStack(err))
				return
			}
			refine := &info.RefineContent{}
			content, typ := refine.SetContentWithTyp(msg.Content, msg.Extra[cst.EventMessageContentType].(int))
			sb.WriteString(content)
			cnt++
			if cnt%config.GetConfig().SensitiveStreamGap == 0 {
				if t.checkSensitive(&sb) {
					sw.Send(t.error(fmt.Sprintf("这个话题暂时还不能聊哦, 也请不要引导我聊敏感话题否则会被封禁哦")))
					return
				}
			}
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

func (t *Transformer) error(msg string) (*sse.Event, error) {
	return t.relay.ErrorEvent(errno.ErrSensitive, msg), nil
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

func (t *Transformer) checkSensitive(sb *strings.Builder) bool {
	sensitive, hits := ac.AcSearch(sb.String(), true)
	if sensitive {
		t.relay.Sensitive.Hits = hits
		t.relay.ModelCancel()
		if err := user.Mapper.Warn(context.Background(), t.relay.UserId.Hex()); err != nil {
			logs.Errorf("warn err: %v", err)
		}
	}
	return sensitive
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
			logs.CondErrorf(!errors.Is(err, io.EOF), "[sse] write err: %s", errorx.ErrorWithoutStack(err))
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
