package model

import (
	"context"
	"testing"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/xh-polaris/innospark-core-api/biz/adaptor"
	"github.com/xh-polaris/innospark-core-api/biz/application/dto/core_api"
)

type TestHistoryDomain struct{}

type TestCompletionInfo struct {
	HistoryDomain *TestHistoryDomain
	OriginMessage []*core_api.Message
}

func TestGraph(t *testing.T) {
	// 全局状态
	gls := func(ctx context.Context) *TestCompletionInfo {
		return &TestCompletionInfo{}
	}
	// Completion Graph
	cg := compose.NewGraph[*core_api.CompletionsReq, adaptor.SSEStream](compose.WithGenLocalState[*TestCompletionInfo](gls))
	// 历史记录节点
	history := compose.InvokableLambda(func(ctx context.Context, input *core_api.CompletionsReq) (output []*schema.Message, err error) {
		history := []*schema.Message{schema.UserMessage("你好"), schema.AssistantMessage("我是测试1", nil)}
		if err != nil {
			return nil, err
		}
		return history, err
	})
	// 处理对话配置
	completionOption := compose.WithStatePostHandler(func(ctx context.Context, out []*schema.Message, state *TestCompletionInfo) ([]*schema.Message, error) {
		return out, nil
	})
	if err := cg.AddLambdaNode("build-history", history, completionOption); err != nil {
		return
	}
	// 获取模型, 工厂方法实现
	model, err := openai.NewChatModel(context.Background(), &openai.ChatModelConfig{})
	if err != nil {
		return
	}
	err = cg.AddChatModelNode("model-factory", model)
	if err != nil {
		return
	}
	r, err := cg.Compile(context.Background())
}
