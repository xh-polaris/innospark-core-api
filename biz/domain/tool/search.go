package graph

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/xh-polaris/innospark-core-api/biz/domain/state"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util"
)

type WebSearchTool interface {
	Info(_ context.Context) (*schema.ToolInfo, error)
	InvokableRun(ctx context.Context, jsonStr string, _ ...tool.Option) (_ string, err error)
}

func NewSearchTool(provider string, relay *state.RelayContext, apiKey string) WebSearchTool {
	switch provider {
	case "bocha":
		return NewBochaSearchTool(relay, apiKey)
	}
	return nil
}

func Search(ctx context.Context, provider, apiKey, template string, input []*schema.Message) (_ []*schema.Message, err error) {
	var relay *state.RelayContext
	if relay, err = util.GetState[*state.RelayContext](ctx); err != nil {
		return
	}
	// 搜索, 过程中会给前端对应反馈
	search := NewSearchTool(provider, relay, apiKey)
	var result string
	// Mock了模型的function call
	if result, err = search.InvokableRun(ctx, fmt.Sprintf("{\"query\":\"%s\"}", relay.Info.OriginMessage.Content)); err != nil {
		return nil, err
	}

	// 填充模板
	format, err := prompt.FromMessages(schema.FString, &schema.Message{Role: "user", Content: template}).Format(ctx,
		map[string]any{"searchContent": result, "userQuery": relay.Info.OriginMessage.Content})
	if err != nil {
		return nil, err
	}

	cite := format[0].Content
	// 找到最近一条有效的用户消息, 替换为带引用内容
	for _, m := range input {
		if m.Role == schema.User {
			if m.Content != "" {
				m.Content = cite
			} else if len(m.UserInputMultiContent) != 0 {
				for i := range m.UserInputMultiContent {
					if m.UserInputMultiContent[i].Type == schema.ChatMessagePartTypeText {
						m.UserInputMultiContent[i].Text = cite
					}
				}
			}
		}
	}
	return input, nil
}
