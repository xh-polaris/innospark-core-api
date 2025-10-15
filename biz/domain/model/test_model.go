package model

import (
	"context"
	"io"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/xh-polaris/innospark-core-api/biz/domain/info"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util"
)

type TestModelFactory struct{}

func (m *TestModelFactory) Generate(ctx context.Context, in []*schema.Message, opts ...model.Option) (_ *schema.Message, err error) {
	return nil, nil
}

func (m *TestModelFactory) Stream(ctx context.Context, in []*schema.Message, opts ...model.Option) (_ *schema.StreamReader[*schema.Message], err error) {
	var r *info.RelayContext
	if r, err = util.GetState[*info.RelayContext](ctx); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(ctx)
	r.ModelCancel = cancel

	sr, sw := schema.Pipe[*schema.Message](5)
	go func() {
		for _, s := range ss {
			s.Extra = map[string]any{cst.EventMessageContentType: cst.EventMessageContentTypeText}
			sw.Send(s, nil)
		}
		sw.Send(nil, io.EOF)
	}()
	return sr, nil
}

var ss = []*schema.Message{
	schema.AssistantMessage("你好", nil),
	schema.AssistantMessage(",", nil),
	schema.AssistantMessage("我是", nil),
	schema.AssistantMessage("启创", nil),
	schema.AssistantMessage(".", nil),
}
