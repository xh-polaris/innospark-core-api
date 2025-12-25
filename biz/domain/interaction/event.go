package interaction

import (
	"encoding/json"
	"strconv"

	"github.com/cloudwego/hertz/pkg/protocol/sse"
	"github.com/xh-polaris/innospark-core-api/biz/adaptor"
	"github.com/xh-polaris/innospark-core-api/biz/domain/state/event"
	"github.com/xh-polaris/innospark-core-api/biz/domain/state/info"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	mmsg "github.com/xh-polaris/innospark-core-api/biz/infra/mapper/message"
	"github.com/xh-polaris/innospark-core-api/pkg/errorx"
	"github.com/xh-polaris/innospark-core-api/pkg/logs"
)

// MetaEvent 组装元数据事件
func MetaEvent(mid, cid, sid string, midx int32, rid string) (*event.Event, error) {
	meta := &adaptor.EventMeta{
		MessageId:        mid,                      // 当前响应消息的id
		ConversationId:   cid,                      // 当前会话id
		SectionId:        sid,                      // 当前段落id
		MessageIndex:     int(midx),                // 当前消息索引
		ConversationType: cst.ConversationTypeText, // 对话类型
		ReplyId:          rid,                      // 回复的用户消息id
	}
	return MarshEvent(cst.EventMeta, meta)
}

// ChatEvent 组装Chat事件
func ChatEvent(cid, sid, rid string, midx int32, bid string, refine *info.RefineContent, typ int) (*event.Event, error) {
	data, err := json.Marshal(refine) // 序列化精化内容
	if err != nil {
		return nil, err
	}

	chat := &adaptor.EventChat{
		Message:          &adaptor.ChatMessage{Content: string(data), ContentType: typ}, // 消息内容
		ConversationId:   cid,                                                           // 当前会话id
		SectionId:        sid,                                                           // 当前段落id
		ReplyId:          rid,                                                           // 回复的用户消息id
		IsDelta:          true,                                                          // 是否增量
		Status:           cst.MessageStatus,                                             // 消息状态
		InputContentType: cst.InputContentTypeText,                                      // 输入内容类型
		MessageIndex:     int(midx),                                                     // 当前消息索引
		BotId:            bid,                                                           // agent名称
	}
	return MarshEvent(cst.EventChat, chat)
}

// ModelEvent 组装模型事件
func ModelEvent(model, bid, bname string) (*event.Event, error) {
	m := &adaptor.EventModel{
		Model:   model, // 模型名称
		BotId:   bid,   // agent id
		BotName: bname, // agent 名称
	}
	return MarshEvent(cst.EventModel, m)
}

// EndEvent 结束事件
func (i *Interaction) EndEvent() error {
	return i.sse.Write(&sse.Event{Type: cst.EventEnd, Data: []byte(cst.EventNotifyValue)})
}

//// ErrorEvent 错误事件
//func (i *Interaction) ErrorEvent(code int, msg string) (*event.Event, error) {
//	return EventWithoutMarshal(cst.EventError, []byte(fmt.Sprintf("{\"code\":%d,\"message\":\"%s\"}", code, msg))), nil
//}

// SearchStartEvent 搜索开始事件, 标识搜索过程开始
func SearchStartEvent() (*event.Event, error) {
	return EventWithoutMarshal(cst.EventSearchStart, []byte(cst.EventNotifyValue)), nil
}

// SearchEndEvent 搜索结束事件, 标识搜索事件结束
func SearchEndEvent() (*event.Event, error) {
	return EventWithoutMarshal(cst.EventSearchEnd, []byte(cst.EventNotifyValue)), nil
}

// SearchFindEvent 搜索总数事件, 标识搜索到的总数目
func SearchFindEvent(n int) (*event.Event, error) {
	return EventWithoutMarshal(cst.EventSearchFind, []byte(strconv.Itoa(n))), nil
}

// SearchChooseEvent 搜索选中事件, 标识选中的数目
func SearchChooseEvent(n int) (*event.Event, error) {
	return EventWithoutMarshal(cst.EventSearchChoose, []byte(strconv.Itoa(n))), nil
}

// SearchCiteEvent 搜索引用事件, 标识选中的具体内容
func SearchCiteEvent(cite *mmsg.Cite) (*event.Event, error) {
	c := &adaptor.EventSearchCite{
		Index:         cite.Index,         // 引用索引
		Name:          cite.Name,          // 引用名称
		URL:           cite.URL,           // 引用链接
		Snippet:       cite.Snippet,       // 引用内容
		SiteName:      cite.SiteName,      // 引用站点名称
		SiteIcon:      cite.SiteIcon,      // 引用站点图标
		DatePublished: cite.DatePublished} // 引用发布时间
	return MarshEvent(cst.EventSearchCite, c)
}

// MarshEvent 序列化一个消息
func MarshEvent(typ string, obj any) (_ *event.Event, err error) {
	var data []byte
	if data, err = json.Marshal(obj); err != nil {
		logs.Errorf("[interaction] event marshal error: %s", errorx.ErrorWithoutStack(err))
		return nil, err
	}
	se := &sse.Event{Type: typ, Data: data}
	return &event.Event{Type: event.SSE, SSEEvent: se}, nil
}

// EventWithoutMarshal 不序列化直接组装消息
func EventWithoutMarshal(typ string, data []byte) *event.Event {
	se := &sse.Event{Type: typ, Data: data}
	return &event.Event{Type: event.SSE, SSEEvent: se}
}
