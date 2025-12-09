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
	id     int
	closed bool
	W      *sse.Writer
	Done   chan struct{}
}

// NewSSEStream 创建事件流
func NewSSEStream(c *app.RequestContext) *SSEStream {
	return &SSEStream{id: -1, Done: make(chan struct{}), W: sse.NewWriter(c)}
}

func (s *SSEStream) Close() {
	_ = s.W.Close()
}

func (s *SSEStream) Write(e *sse.Event) (err error) {
	e.ID = s.getID()
	if err = s.W.Write(e); err != nil {
		s.closed = true
		logs.Errorf("[interaction] write see err: %s", errorx.ErrorWithoutStack(err))
	}
	return s.W.Write(e)
}

func (s *SSEStream) getID() string {
	s.id++
	return strconv.Itoa(s.id)
}

// SSE 实现sse流响应
func SSE(ctx context.Context, c *app.RequestContext, req any, stream *SSEStream, err error) {
	b3.New().Inject(ctx, &headerProvider{headers: &c.Response.Header})
	logs.CtxInfof(ctx, "[%s] req=%s, resp=event stream, err=%s, trace=%s", c.Path(), util.JSONF(req), errorx.ErrorWithoutStack(err), trace.SpanContextFromContext(ctx).TraceID().String())

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
