package deyu

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
)

func TestOpenAIFormat(t *testing.T) {
	model := getModel()
	messages := getMessages()
	stream, err := model.Stream(context.Background(), messages)
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	var sb strings.Builder
	for {
		chunk, err := stream.Recv()
		if err != nil {
			break
		}
		sb.WriteString(chunk.Content)
		t.Logf("%+v\n", chunk)
	}
	t.Log(sb.String())
}

func TestConcurrentCallWithMultiModel(t *testing.T) {
	wg := sync.WaitGroup{}
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int, t *testing.T) {
			model := getModel()
			stream, err := model.Stream(context.Background(), getMessages())
			if err != nil {
				t.Errorf("%d|%s\n", i, err)
				return
			}
			defer stream.Close()
			var sb strings.Builder
			for {
				chunk, err := stream.Recv()
				if err != nil {
					break
				}
				sb.WriteString(chunk.Content)
				//t.Logf("%+v\n", chunk)
			}
			t.Logf("%d|%s\n", i, sb.String())
			wg.Done()
		}(i, t)
	}
	wg.Wait()
}

func getModel() *openai.ChatModel {
	model, err := openai.NewChatModel(context.Background(), &openai.ChatModelConfig{
		BaseURL: "https://edusys1.sii.edu.cn/deyu/14b/bzr_only/v1", // Azure API 基础 URL
		// 基础配置
		APIKey:  "test-key",       // API 密钥
		Timeout: 30 * time.Second, // 超时时间
		// 模型参数
		Model: "deyu", // 模型名称
	})
	if err != nil {
		panic(err)
	}
	return model
}

func getMessages() []*schema.Message {
	messages := []*schema.Message{
		// 系统消息
		schema.SystemMessage("介绍你自己"),
	}
	return messages
}
