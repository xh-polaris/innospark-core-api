package sse

import (
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/sse"
	"github.com/xh-polaris/innospark-core-api/pkg/errorx"
	"github.com/xh-polaris/innospark-core-api/pkg/logs"
)

// SSEStream 事件流
type SSEStream struct {
	id   int
	w    *sse.Writer
	Done chan struct{}
}

// NewSSEStream 创建事件流
func NewSSEStream(c *app.RequestContext) *SSEStream {
	return &SSEStream{id: -1, Done: make(chan struct{}), w: sse.NewWriter(c)}
}

func (s *SSEStream) Write(e *sse.Event) (err error) {
	e.ID = s.getID()
	if err = s.w.Write(e); err != nil {
		logs.Errorf("write see err: %s", errorx.ErrorWithoutStack(err))
	}
	return err
}

func (s *SSEStream) Close() error {
	return s.w.Close()
}

func (s *SSEStream) getID() string {
	s.id++
	return strconv.Itoa(s.id)
}
