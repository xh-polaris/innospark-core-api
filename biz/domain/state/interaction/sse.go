package interaction

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/cloudwego/eino/schema"
	"github.com/cloudwego/hertz/pkg/protocol/sse"
	"github.com/xh-polaris/innospark-core-api/biz/adaptor"
	"github.com/xh-polaris/innospark-core-api/biz/domain/state/info"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	mmsg "github.com/xh-polaris/innospark-core-api/biz/infra/mapper/message"
	"github.com/xh-polaris/innospark-core-api/pkg/errorx"
	"github.com/xh-polaris/innospark-core-api/pkg/logs"
)

// WriteSSE 写入SSE事件至SSEStream, 由adaptor.sse实际响应给前端
func WriteSSE(s *info.Info, input *schema.StreamReader[*sse.Event]) error {
	defer input.Close() // 关闭input流读入
	defer s.SSE.Close() // 关闭SSE流写入

	for {
		e, err := input.Recv()
		if e != nil {
			if err = s.SSE.W.Write(e); err != nil { // 写入事件
				logs.CondErrorf(!errors.Is(err, io.EOF), "[interaction] write see err: %s", errorx.ErrorWithoutStack(err))
				break
			}
		}
		if err != nil {
			logs.CondErrorf(!errors.Is(err, io.EOF), "[interaction] write see err: %s", errorx.ErrorWithoutStack(err))
			break
		}
	}
	return nil // !!此处忽略错误, 以便进入后续记录处理节点
}

// id 获取并自增id
func (i *Interaction) id() string {
	id := strconv.Itoa(i.info.SSEIndex)
	i.info.SSEIndex++
	return id
}

// MetaEvent 组装元数据事件
func (i *Interaction) MetaEvent() (*sse.Event, error) {
	meta := &adaptor.EventMeta{
		MessageId:        i.info.MessageInfo.AssistantMessage.MessageId.Hex(), // 当前响应消息的id
		ConversationId:   i.info.ConversationId.Hex(),                         // 当前会话id
		SectionId:        i.info.SectionId.Hex(),                              // 当前段落id
		MessageIndex:     int(i.info.MessageInfo.AssistantMessage.Index),      // 当前消息索引
		ConversationType: cst.ConversationTypeText,                            // 对话类型
		ReplyId:          i.info.ReplyId,                                      // 回复的用户消息id
	}
	return MarshEvent(i.id(), cst.EventMeta, meta)
}

// ChatEvent 组装Chat事件
func (i *Interaction) ChatEvent(refine *info.RefineContent, typ int) (*sse.Event, error) {
	data, err := json.Marshal(refine) // 序列化精化内容
	if err != nil {
		return nil, err
	}

	chat := &adaptor.EventChat{
		Message:          &adaptor.ChatMessage{Content: string(data), ContentType: typ}, // 消息内容
		ConversationId:   i.info.ConversationId.Hex(),                                   // 当前会话id
		SectionId:        i.info.SectionId.Hex(),                                        // 当前段落id
		ReplyId:          i.info.ReplyId,                                                // 回复的用户消息id
		IsDelta:          true,                                                          // 是否增量
		Status:           cst.MessageStatus,                                             // 消息状态
		InputContentType: cst.InputContentTypeText,                                      // 输入内容类型
		MessageIndex:     int(i.info.MessageInfo.AssistantMessage.Index),                // 当前消息索引
		BotId:            i.info.ModelInfo.BotId,                                        // agent名称
	}
	return MarshEvent(i.id(), cst.EventChat, chat)
}

// ModelEvent 组装模型事件
func (i *Interaction) ModelEvent() (*sse.Event, error) {
	m := &adaptor.EventModel{
		Model:   i.info.ModelInfo.Model,   // 模型名称
		BotId:   i.info.ModelInfo.BotId,   // agent id
		BotName: i.info.ModelInfo.BotName, // agent 名称
	}
	return MarshEvent(i.id(), cst.EventModel, m)
}

// EndEvent 结束事件
func (i *Interaction) EndEvent(err error) (*sse.Event, error) {
	return EventWithoutMarshal(i.id(), cst.EventEnd, []byte(cst.EventNotifyValue)), err
}

// ErrorEvent 错误事件
func (i *Interaction) ErrorEvent(code int, msg string) (*sse.Event, error) {
	return EventWithoutMarshal(i.id(), cst.EventError, []byte(fmt.Sprintf("{\"code\":%d,\"message\":\"%s\"}", code, msg))), nil
}

// SearchStartEvent 搜索开始事件, 标识搜索过程开始
func (i *Interaction) SearchStartEvent() (*sse.Event, error) {
	return EventWithoutMarshal(i.id(), cst.EventSearchStart, []byte(cst.EventNotifyValue)), nil
}

// SearchEndEvent 搜索结束事件, 标识搜索事件结束
func (i *Interaction) SearchEndEvent() (*sse.Event, error) {
	return EventWithoutMarshal(i.id(), cst.EventSearchEnd, []byte(cst.EventNotifyValue)), nil
}

// SearchFindEvent 搜索总数事件, 标识搜索到的总数目
func (i *Interaction) SearchFindEvent(n int) (*sse.Event, error) {
	return EventWithoutMarshal(i.id(), cst.EventSearchFind, []byte(strconv.Itoa(n))), nil
}

// SearchChooseEvent 搜索选中事件, 标识选中的数目
func (i *Interaction) SearchChooseEvent(n int) (*sse.Event, error) {
	return EventWithoutMarshal(i.id(), cst.EventSearchChoose, []byte(strconv.Itoa(n))), nil
}

// SearchCiteEvent 搜索引用事件, 标识选中的具体内容
func (i *Interaction) SearchCiteEvent(cite *mmsg.Cite) (*sse.Event, error) {
	c := &adaptor.EventSearchCite{
		Index:         cite.Index,         // 引用索引
		Name:          cite.Name,          // 引用名称
		URL:           cite.URL,           // 引用链接
		Snippet:       cite.Snippet,       // 引用内容
		SiteName:      cite.SiteName,      // 引用站点名称
		SiteIcon:      cite.SiteIcon,      // 引用站点图标
		DatePublished: cite.DatePublished} // 引用发布时间
	return MarshEvent(i.id(), cst.EventSearchCite, c)
}

// MarshEvent 序列化一个消息
func MarshEvent(index string, typ string, obj any) (_ *sse.Event, err error) {
	var data []byte
	if data, err = json.Marshal(obj); err != nil {
		logs.Errorf("[interaction] event marshal error: %s", errorx.ErrorWithoutStack(err))
	}
	return &sse.Event{ID: index, Type: typ, Data: data}, err
}

// EventWithoutMarshal 不序列化直接组装消息
func EventWithoutMarshal(index string, typ string, data []byte) *sse.Event {
	return &sse.Event{ID: index, Type: typ, Data: data}
}

// WriteEvent 写入一个sse事件
func (i *Interaction) WriteEvent(e *sse.Event, err error) error {
	i.info.SSE.C <- e
	return err
}
