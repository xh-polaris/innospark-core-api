package interaction

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/cloudwego/eino/schema"
	"github.com/cloudwego/hertz/pkg/protocol/sse"
	"github.com/xh-polaris/innospark-core-api/biz/conf"
	ss "github.com/xh-polaris/innospark-core-api/biz/domain/interaction/sse"
	"github.com/xh-polaris/innospark-core-api/biz/domain/state"
	"github.com/xh-polaris/innospark-core-api/biz/domain/state/event"
	"github.com/xh-polaris/innospark-core-api/biz/domain/state/info"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	mmsg "github.com/xh-polaris/innospark-core-api/biz/infra/mapper/message"
)

var Interrupt = errors.New("interrupt")

// Interaction 交互域, 负责下游消息组装与转换, 并响应给前端
type Interaction struct {
	sse   *ss.SSEStream       // SSE流
	event *event.EventStream  // 事件流
	st    *state.RelayContext // 状态上下文

	containers map[int]*strings.Builder // 记录不同类型内容
	code       []*strings.Builder       // 记录代码内容
	codeTyp    []string                 // 记录代码类型
}

// NewInteraction 创建交互
func NewInteraction(st *state.RelayContext) (i *Interaction) {
	i = &Interaction{st: st,
		sse:   ss.NewSSEStream(st.Info.RequestContext),
		event: st.EventStream,
		containers: map[int]*strings.Builder{
			cst.EventMessageContentTypeText:    {}, // 文本消息
			cst.EventMessageContentTypeThink:   {}, // 思考消息
			cst.EventMessageContentTypeSuggest: {}, // 建议消息
		}}
	return
}
func (i *Interaction) Close() error {
	return i.sse.Close()
}
func (i *Interaction) HandleEvent(ctx context.Context) (err error) {
	defer i.collect() // 收集各类型消息
	var e *event.Event
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if e, err = i.event.R.Recv(); err != nil {
				if errors.Is(err, io.EOF) { // 正常结束
					return Interrupt
				}
				return
			}
			switch e.Type {
			case event.SSE:
				if err = i.handleSSE(e.SSEEvent); err != nil {
					return
				}
			case event.ChatModel:
				if err = i.handleChatModel(e.Message); err != nil {
					return
				}
			case event.Suggest:
				if err = i.handleSuggest(e.Message); err != nil {
					return
				}
			}
		}
	}
}

func (i *Interaction) handleSSE(e *sse.Event) error {
	if err := i.sse.Write(e); err != nil {
		return Interrupt
	}
	return nil
}

func (i *Interaction) handleSuggest(msg *schema.Message) (err error) {
	// 精化消息
	refine := &info.RefineContent{}
	content, typ := refine.SetContentWithTyp(msg.Content, cst.EventMessageContentTypeSuggest)

	i.containers[cst.EventMessageContentTypeSuggest].WriteString(content) // 收集 suggest

	inf := i.st.Info
	ce, err := ChatEvent(inf.ConversationId.Hex(), inf.SectionId.Hex(), inf.ReplyId,
		inf.MessageInfo.AssistantMessage.Index, inf.ModelInfo.BotId, refine, typ)
	if err != nil {
		return
	}
	if err = i.sse.Write(ce.SSEEvent); err != nil {
		return Interrupt
	}
	return nil
}

// handleChatModel 组装模型事件, 将模型消息转换为ChatEvent
// 同时兼顾消息内容收集和敏感词检测
func (i *Interaction) handleChatModel(msg *schema.Message) (err error) {
	if msg.ResponseMeta != nil { // 收集用量信息
		i.st.Info.ResponseMeta = msg.ResponseMeta
	}
	// 精化消息
	refine := &info.RefineContent{}
	content, typ := refine.SetContentWithTyp(msg.Content, msg.Extra[cst.EventMessageContentType].(int))
	// 收集信息
	if typ == cst.EventMessageContentTypeCodeType {
		i.codeTyp = append(i.codeTyp, content)
		i.code = append(i.code, &strings.Builder{})
	} else if typ == cst.EventMessageContentTypeCode {
		i.code[len(i.code)-1].WriteString(content)
	} else {
		i.containers[typ].WriteString(content)
	}
	inf := i.st.Info
	ce, err := ChatEvent(inf.ConversationId.Hex(), inf.SectionId.Hex(), inf.ReplyId,
		inf.MessageInfo.AssistantMessage.Index, inf.ModelInfo.BotId, refine, typ)
	if err != nil {
		return
	}
	if err = i.sse.Write(ce.SSEEvent); err != nil {
		return Interrupt
	}
	return nil
}

// collect 收集各类消息内容
func (i *Interaction) collect() {
	// 存储各类消息内容
	i.st.Info.MessageInfo.Text = i.containers[cst.EventMessageContentTypeText].String()       // 文本
	i.st.Info.MessageInfo.Think = i.containers[cst.EventMessageContentTypeThink].String()     // 思考
	i.st.Info.MessageInfo.Suggest = i.containers[cst.EventMessageContentTypeSuggest].String() // 建议

	// 构造代码段信息
	codes := make([]*mmsg.Code, len(i.code))
	for index, code := range i.code {
		codes[index] = &mmsg.Code{
			Index:    int32(index),
			CodeType: i.codeTyp[index],
			Code:     code.String(),
		}
	}
	i.st.Info.MessageInfo.Code = codes
}

//// sensitive 检查是否有违禁词
//func (i *Interaction) sensitive(sb *strings.Builder) bool {
//	sensitive, hits := ac.AcSearch(sb.String(), true, cst.SensitivePost)
//	if sensitive {
//		i.st.Info.Sensitive.Hits = hits
//		i.st.Cancel() // TODO
//		if err := user.Mapper.Warn(context.Background(), i.st.Info.UserId.Hex()); err != nil {
//			logs.Errorf("[interaction] warn err: %v", err)
//		}
//	}
//	return sensitive
//}

// 判断是否需要检查违禁词
func needSensitiveCheck(cnt int) bool {
	return cnt%conf.GetConfig().Sensitive.SensitiveStreamGap == 0
}
