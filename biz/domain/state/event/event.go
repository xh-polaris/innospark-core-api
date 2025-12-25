package event

import (
	"errors"

	"github.com/cloudwego/eino/schema"
	"github.com/cloudwego/hertz/pkg/protocol/sse"
)

// EventStream 事件流
// 图内部事件经由interaction处理后写入

var Closed = errors.New("event stream is closed")

type EventStream struct {
	R *schema.StreamReader[*Event] // 写入
	W *schema.StreamWriter[*Event] // 读出
}

func NewEventStream() *EventStream {
	r, w := schema.Pipe[*Event](100)
	return &EventStream{
		R: r,
		W: w,
	}
}

func (es *EventStream) Close() {
	es.R.Close()
	es.W.Close()
}

func (es *EventStream) Write(e *Event, err error) error {
	if es.W.Send(e, err) {
		return Closed
	}
	return nil
}

func (es *EventStream) Read() (*Event, error) {
	return es.R.Recv()
}

const (
	SSE       = "sse"        // sse事件
	ChatModel = "chat_model" // 模型消息
	Suggest   = "suggest"    // 建议消息
)

type Event struct {
	Type     string
	Message  *schema.Message
	SSEEvent *sse.Event
}
