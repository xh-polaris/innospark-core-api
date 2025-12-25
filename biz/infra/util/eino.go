package util

import (
	"context"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

func GetState[I any](ctx context.Context) (relay I, err error) {
	err = compose.ProcessState(ctx, func(ctx context.Context, s I) (err error) {
		relay = s
		return
	})
	return
}

func MustAddLambdaNode[I any, O any](g *compose.Graph[I, O], key string, node *compose.Lambda, opts ...compose.GraphAddNodeOpt) {
	if err := g.AddLambdaNode(key, node, opts...); err != nil {
		panic(err)
	}
}

func MustAddToolsNode[I any, O any](g *compose.Graph[I, O], key string, node *compose.ToolsNode, opts ...compose.GraphAddNodeOpt) {
	if err := g.AddToolsNode(key, node, opts...); err != nil {
		panic(err)
	}
}

func MustAddGraphBranch[I any, O any](g *compose.Graph[I, O], key string, node *compose.GraphBranch) {
	if err := g.AddBranch(key, node); err != nil {
		panic(err)
	}
}

func MustAddChatModelNode[I any, O any](g *compose.Graph[I, O], key string, node model.BaseChatModel, opts ...compose.GraphAddNodeOpt) {
	if err := g.AddChatModelNode(key, node, opts...); err != nil {
		panic(err)
	}
}

func MustChain[I any, O any](g *compose.Graph[I, O], nodes ...string) {
	if len(nodes) < 2 {
		return
	}
	for i := 1; i < len(nodes); i++ {
		MustAddEdge(g, nodes[i-1], nodes[i])
	}
}

func MustAddEdge[I any, O any](g *compose.Graph[I, O], start, end string) {
	if err := g.AddEdge(start, end); err != nil {
		panic(err)
	}
}

func AddExtra(m *schema.Message, key string, value any) {
	if m.Extra == nil {
		m.Extra = make(map[string]any)
	}
	m.Extra[key] = value
}

// GetInputText 获取消息中的文本
func GetInputText(m *schema.Message) string {
	if len(m.UserInputMultiContent) > 0 {
		for _, c := range m.UserInputMultiContent {
			if c.Type == schema.ChatMessagePartTypeText {
				return c.Text
			}
		}
	}
	return m.Content
}

// SetInputText 设置消息的文本
func SetInputText(m *schema.Message, text string) {
	if len(m.UserInputMultiContent) > 0 {
		for _, c := range m.UserInputMultiContent {
			if c.Type == schema.ChatMessagePartTypeText {
				c.Text = text
				return
			}
		}
	}
	m.Content = text
}

func RemoveImg(m *schema.Message) {
	if len(m.UserInputMultiContent) > 0 {
		ipt := make([]schema.MessageInputPart, 0)
		for _, c := range m.UserInputMultiContent {
			if c.Type != schema.ChatMessagePartTypeImageURL {
				ipt = append(ipt, c)
			}
		}
		m.UserInputMultiContent = ipt
	}
	return
}
