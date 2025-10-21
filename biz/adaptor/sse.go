package adaptor

import (
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/sse"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util/errorx"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util/logx"
)

// SSEStream SSE事件流
type SSEStream struct {
	C    chan *sse.Event
	id   int
	Done chan struct{}
}

// NewSSEStream 创建事件流
func NewSSEStream() *SSEStream {
	return &SSEStream{C: make(chan *sse.Event, 5), id: 0, Done: make(chan struct{})}
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
	var err error
	w := sse.NewWriter(c)
	defer func(w *sse.Writer) {
		close(stream.Done) // 关闭结束channel
		if closeErr := w.Close(); err != nil && closeErr != nil {
			logx.Error("close sse writer fail, err=%s", errorx.ErrorWithoutStack(err))
		}
	}(w)

	var ok bool
	var event *sse.Event
	for {
		if event, ok = stream.Nex(); !ok {
			return
		}
		if err = w.Write(event); err != nil {
			stream.Done <- struct{}{} // 给这个流写入提前终止信号
			logx.Error("write sse-event error, err=%s", errorx.ErrorWithoutStack(err))
			return
		}
	}
}

// EventMeta 对话元数据
type EventMeta struct {
	ReplyId          string `json:"replyId"`
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
	IsDelta          bool         `json:"isDelta"`
	Status           int          `json:"status"`
	InputContentType int          `json:"inputContentType"`
	MessageIndex     int          `json:"messageIndex"`
	BotId            string       `json:"botId"`
}

// EventSearchCite 引用内容
type EventSearchCite struct {
	Index         int32  `json:"index" bson:"index"`
	Name          string `json:"name" bson:"name"`
	URL           string `json:"url" bson:"url"`
	Snippet       string `json:"snippet" bson:"snippet"`
	SiteName      string `json:"siteName" bson:"siteName"`
	SiteIcon      string `json:"siteIcon" bson:"siteIcon"`
	DatePublished string `json:"datePublished" bson:"datePublished"`
}

type EventEnd struct{}
