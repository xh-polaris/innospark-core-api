package adaptor

import (
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/sse"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util/logx"
)

// SSEStream SSE事件流
type SSEStream struct {
	C  chan *sse.Event
	id int
}

// NewSSEStream 创建事件流
func NewSSEStream() *SSEStream {
	return &SSEStream{C: make(chan *sse.Event, 5), id: 0}
}

// Nex 获取下一个事件并返回是否关闭
func (s *SSEStream) Nex() (*sse.Event, bool) {
	event, ok := <-s.C
	if !ok {
		return nil, false
	}
	event.ID = strconv.Itoa(s.id)
	s.id++
	return event, true
}

// 实现sse流响应, TODO 实现中断
func makeSSE(c *app.RequestContext, stream *SSEStream) {
	w := sse.NewWriter(c)
	defer func(w *sse.Writer) {
		if err := w.Close(); err != nil {
			logx.Error("close sse writer fail, err=%v", err)
		}
	}(w)

	var ok bool
	var event *sse.Event
	for {
		if event, ok = stream.Nex(); !ok {
			return
		}
		if err := w.Write(event); err != nil {
			logx.Error("write sse-event error, err=%s", err.Error())
			return
		}
	}
}

// EventMeta 对话元数据
type EventMeta struct {
	MessageId        string `json:"messageId"`
	ConversationId   string `json:"conversationId"`
	SectionId        string `json:"sectionId"`
	MessageIndex     int    `json:"messageIndex"`
	ConversationType int    `json:"conversationType"`
}

type EventModel struct {
	Model   string `json:"model"`
	BotId   string `json:"botId"`
	BotName string `json:"botName"`
}

type ChatMessage struct {
	Content     string `json:"content"`
	ContentType int    `json:"contentType"`
}

// EventChat 模型消息
type EventChat struct {
	Message          *ChatMessage `json:"message"`
	ConversationId   string       `json:"conversationId"`
	SectionId        string       `json:"sectionId"`
	ReplyId          string       `json:"replyId"`
	IsDeleted        bool         `json:"isDeleted"`
	Status           int          `json:"status"`
	InputContentType int          `json:"inputContentType"`
	MessageIndex     int          `json:"messageIndex"`
	BotId            string       `json:"botId"`
}

type EventEnd struct{}
