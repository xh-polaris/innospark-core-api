package model

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/xh-polaris/innospark-core-api/biz/infra/config"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util"
)

func init() {
	RegisterModel(Doubao15Pro32K, NewDoubao15Pro32KChatModel)
}

const (
	Doubao15Pro32K = "doubao-1-5-pro-32k-250115"
	ARKBeijing     = "https://ark.cn-beijing.volces.com/api/v3"
)

type ARKChatModel struct {
	cli   *ark.ChatModel
	model string
}

func NewDoubao15Pro32KChatModel(ctx context.Context, uid, _ string) (_ model.ToolCallingChatModel, err error) {
	var cli *ark.ChatModel
	cli, err = ark.NewChatModel(ctx, &ark.ChatModelConfig{
		BaseURL:    ARKBeijing,
		Region:     "cn-beijing",
		APIKey:     config.GetConfig().ARK.APIKey,
		Model:      Doubao15Pro32K,
		HTTPClient: nil,
	})
	return &ARKChatModel{cli: cli, model: Doubao15Pro32K}, nil
}

func (c *ARKChatModel) Generate(ctx context.Context, in []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	// messages翻转顺序, 调用模型时消息应该正序
	var reverse []*schema.Message
	for i := len(in) - 1; i >= 0; i-- {
		if in[i].Content != "" {
			reverse = append(reverse, in[i])
		}
	}
	return c.cli.Generate(ctx, reverse, opts...)
}

func (c *ARKChatModel) Stream(ctx context.Context, in []*schema.Message, opts ...model.Option) (processReader *schema.StreamReader[*schema.Message], err error) {
	var raw *schema.StreamReader[*schema.Message]
	// messages翻转顺序, 调用模型时消息应该正序
	var reverse []*schema.Message
	for i := len(in) - 1; i >= 0; i-- {
		if in[i].Content != "" {
			in[i].Name = ""
			reverse = append(reverse, in[i])
		}
	}
	if raw, err = c.cli.Stream(ctx, reverse, opts...); err != nil {
		return nil, err
	}
	processReader, processWriter := schema.Pipe[*schema.Message](5)
	switch c.model {
	case Doubao15Pro32K:
		go c.process(ctx, raw, processWriter)
	default:
		raw.Close()
	}
	return processReader, nil
}

func (c *ARKChatModel) process(ctx context.Context, reader *schema.StreamReader[*schema.Message], writer *schema.StreamWriter[*schema.Message]) {
	defer reader.Close()
	defer writer.Close()

	var err error
	var msg *schema.Message
	var segment int       // 段落数
	var isCode, pass bool // 是否为代码内容以及记录代码类型

	var status = cst.EventMessageContentTypeText
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if msg, err = reader.Recv(); err != nil {
				writer.Send(nil, err)
				return
			}
			if pass { // 记录代码类型
				status = cst.EventMessageContentTypeCodeType // 代码类型
				pass = !pass
			} else {
				if isCode {
					status = cst.EventMessageContentTypeCode
				}
				// 处理消息
				switch msg.Content {
				case cst.CodeBound: // 代码内容边界, 需要在text中写入一个[code:x]来标注代码出现位置
					if !isCode {
						status, pass = cst.EventMessageContentTypeCode, true
						msg.Content = fmt.Sprintf("[code:%d]", segment)
						util.AddExtra(msg, cst.EventMessageContentType, cst.EventMessageContentTypeText)
						writer.Send(msg, nil)
					} else {
						status = cst.EventMessageContentTypeText
						segment++
					}
					isCode = !isCode
					continue
				}
			}
			util.AddExtra(msg, cst.EventMessageContentType, status)
			writer.Send(msg, nil)
		}
	}
}

func (c *ARKChatModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return c.cli.WithTools(tools)
}
