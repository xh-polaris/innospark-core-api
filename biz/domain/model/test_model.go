package model

import (
	"context"
	"io"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/xh-polaris/innospark-core-api/biz/domain/state"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util"
)

func init() {
	RegisterModel("Test-Model", NewTestModel)
}

func NewTestModel(ctx context.Context, uid, botId string) (_ model.ToolCallingChatModel, err error) {
	return &TestModelFactory{}, err
}

type TestModelFactory struct{}

func (m *TestModelFactory) Generate(ctx context.Context, in []*schema.Message, opts ...model.Option) (_ *schema.Message, err error) {
	return nil, nil
}

func (m *TestModelFactory) Stream(ctx context.Context, in []*schema.Message, opts ...model.Option) (_ *schema.StreamReader[*schema.Message], err error) {
	var r *state.RelayContext
	if r, err = util.GetState[*state.RelayContext](ctx); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(ctx)
	r.Info.ModelCancel = cancel

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
	schema.AssistantMessage(",", nil),
	schema.AssistantMessage(",", nil),
	schema.AssistantMessage(",", nil),
	schema.AssistantMessage(",", nil),
	schema.AssistantMessage(",", nil),
	schema.AssistantMessage(",", nil),
	schema.AssistantMessage(",", nil),
	schema.AssistantMessage(",", nil),
	schema.AssistantMessage(",", nil),
	schema.AssistantMessage(",", nil),
	schema.AssistantMessage(",", nil),
	schema.AssistantMessage(",", nil),
	schema.AssistantMessage(",", nil),
	schema.AssistantMessage(",", nil),
	schema.AssistantMessage(",", nil),
	schema.AssistantMessage(",", nil),
	schema.AssistantMessage(",", nil),
	schema.AssistantMessage(",", nil),
	schema.AssistantMessage(",", nil),
	schema.AssistantMessage(",", nil),
	schema.AssistantMessage(",", nil),
	schema.AssistantMessage(",", nil),
	schema.AssistantMessage(",", nil),
	schema.AssistantMessage(",", nil),
	schema.AssistantMessage(",", nil),
	schema.AssistantMessage(",", nil),
	schema.AssistantMessage(",", nil),
	schema.AssistantMessage(",", nil),
	schema.AssistantMessage(",", nil),
	schema.AssistantMessage(",", nil),
	schema.AssistantMessage(",", nil),
	schema.AssistantMessage("維尼", nil),
	schema.AssistantMessage("下台", nil),
	schema.AssistantMessage(".", nil),
}

func (m *TestModelFactory) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return nil, nil
}
