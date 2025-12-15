package adaptor

// SSE流处理

import (
	"context"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/sse"
	"github.com/xh-polaris/gopkg/util"
	"github.com/xh-polaris/innospark-core-api/pkg/errorx"
	"github.com/xh-polaris/innospark-core-api/pkg/logs"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel/trace"
)

// SSEStream SSE事件流
type SSEStream struct {
	C    chan *sse.Event
	W    *sse.Writer
	id   int
	Done chan struct{}
}

// NewSSEStream 创建事件流
func NewSSEStream(c *app.RequestContext) *SSEStream {
	return &SSEStream{C: make(chan *sse.Event, 100), id: 0, Done: make(chan struct{}), W: sse.NewWriter(c)}
}

func (s *SSEStream) Close() {
	close(s.C)
	_ = s.W.Close()
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

// SSE 实现sse流响应
func SSE(ctx context.Context, c *app.RequestContext, req any, stream *SSEStream, err error) {
	b3.New().Inject(ctx, &headerProvider{headers: &c.Response.Header})
	logs.CtxInfof(ctx, "[%s] req=%s, resp=sse stream, err=%s, trace=%s", c.Path(), util.JSONF(req), errorx.ErrorWithoutStack(err), trace.SpanContextFromContext(ctx).TraceID().String())

	if err != nil { // 有错误
		PostError(ctx, c, err)
		return
	}
}

// EventMeta 元数据事件
type EventMeta struct {
	ReplyId          string `json:"replyId"`
	MessageId        string `json:"messageId"`
	ConversationId   string `json:"conversationId"`
	SectionId        string `json:"sectionId"`
	MessageIndex     int    `json:"messageIndex"`
	ConversationType int    `json:"conversationType"`
}

// EventModel 模型信息事件
type EventModel struct {
	Model   string `json:"model"`
	BotId   string `json:"botId"`
	BotName string `json:"botName"`
}

type ChatMessage struct {
	Content     string `json:"content"`
	ContentType int    `json:"contentType"`
}

// EventChat 消息事件
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

// EventSearchCite 引用内容事件
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
