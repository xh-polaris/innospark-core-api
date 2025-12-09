package model

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/xh-polaris/innospark-core-api/biz/conf"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util"
)

func init() {
	RegisterModel(Claude4Sonnet, NewClaudeChatModel)
}

const (
	Claude4Sonnet = "claude-4-sonnet"
)

var MaxToken = 65535

type ClaudeChatModel struct {
	cli   *openai.ChatModel
	model string
}

func NewClaudeChatModel(ctx context.Context, uid, _ string) (_ model.ToolCallingChatModel, err error) {
	var cli *openai.ChatModel
	cli, err = openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:     conf.GetConfig().Claude.APIKey,
		BaseURL:    conf.GetConfig().Claude.BaseURL,
		Model:      Claude4Sonnet,
		MaxTokens:  &MaxToken,
		User:       &uid,
		HTTPClient: util.NewDebugClient(),
	})
	if err != nil {
		return nil, err
	}
	return &ClaudeChatModel{cli: cli, model: Claude4Sonnet}, nil
}

func (c *ClaudeChatModel) Generate(ctx context.Context, in []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	// messages翻转顺序, 调用模型时消息应该正序
	return c.cli.Generate(ctx, in, opts...)
}

func (c *ClaudeChatModel) Stream(ctx context.Context, in []*schema.Message, opts ...model.Option) (processReader *schema.StreamReader[*schema.Message], err error) {
	var raw *schema.StreamReader[*schema.Message]
	if raw, err = c.cli.Stream(ctx, in, opts...); err != nil {
		return nil, err
	}
	processReader, processWriter := schema.Pipe[*schema.Message](5)
	switch c.model {
	case Claude4Sonnet:
		go c.process(ctx, raw, processWriter)
	default:
		raw.Close()
	}
	return processReader, nil
}

func (c *ClaudeChatModel) process(ctx context.Context, reader *schema.StreamReader[*schema.Message], writer *schema.StreamWriter[*schema.Message]) {
	defer reader.Close()
	defer writer.Close()

	var err error
	var msg *schema.Message
	var segment int             // 段落数
	var isCode, noCodeType bool // 是否为代码内容以及记录代码类型

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
			if noCodeType { // 记录代码类型
				//var find bool
				//for i := 0; i < len(message.Content)-1; i++ {
				//	if message.Content[i] == '<' { // 有代码内容
				//		find = true
				//		codeType := schema.AssistantMessage(strings.Trim(message.Content[:i], "\n"), nil)
				//		util.AddExtra(codeType, cst.EventMessageContentType, cst.EventMessageContentTypeCodeType)
				//		writer.Send(codeType, nil)
				//		message.Content = message.Content[i:]
				//		break
				//	}
				//}
				//if !find { // 没有代码内容, 整个都是代码类型
				//	message.Content = strings.Trim(message.Content, "\n")
				//	util.AddExtra(message, cst.EventMessageContentType, cst.EventMessageContentTypeCodeType)
				//	writer.Send(message, nil)
				//	continue
				//}
				right := strings.Split(msg.Content, "html")
				codeType := schema.AssistantMessage("html", nil)
				util.AddExtra(codeType, cst.EventMessageContentType, cst.EventMessageContentTypeCodeType)
				writer.Send(codeType, nil)
				if len(right) > 1 {
					code := schema.AssistantMessage(strings.TrimLeft(right[1], "\n"), nil)
					util.AddExtra(code, cst.EventMessageContentType, cst.EventMessageContentTypeCode)
					writer.Send(code, nil)
				}
				noCodeType = !noCodeType
			} else {
				// 处理消息
				switch {
				case strings.Contains(msg.Content, cst.CodeBound): // 包含代码内容边界, 若是开始需要在text中写入一个[code:x]来标注代码出现位置
					if !isCode {
						ss := strings.Split(msg.Content, cst.CodeBound)
						// 左侧文本内容
						textMsg := schema.AssistantMessage(ss[0], nil)
						util.AddExtra(textMsg, cst.EventMessageContentType, cst.EventMessageContentTypeText)
						writer.Send(textMsg, nil)
						msg.Content = fmt.Sprintf("[code:%d]", segment)
						util.AddExtra(msg, cst.EventMessageContentType, cst.EventMessageContentTypeText)
						writer.Send(msg, nil)
						// 右侧可能是代码类型, 也可能有代码, 也可能什么都没有
						noCodeType = true
						if len(ss) > 1 && len(strings.Trim(ss[1], "\n")) > 0 {
							right := strings.Split(ss[1], "html")
							codeType := schema.AssistantMessage("html", nil)
							util.AddExtra(codeType, cst.EventMessageContentType, cst.EventMessageContentTypeCodeType)
							writer.Send(codeType, nil)
							if len(right) > 1 {
								code := schema.AssistantMessage(strings.TrimLeft(right[1], "\n"), nil)
								util.AddExtra(code, cst.EventMessageContentType, cst.EventMessageContentTypeCode)
								writer.Send(code, nil)
							}
							noCodeType = false // 有了代码类型
							//for i := 0; i < len(ss[1])-1; i++ {
							//	if ss[1][i] == '<' || ss[1][i] == '\n' { // 有代码内容
							//		codeType := schema.AssistantMessage(strings.Trim(ss[1][:i], "\n"), nil)
							//		util.AddExtra(codeType, cst.EventMessageContentType, cst.EventMessageContentTypeCodeType)
							//		writer.Send(codeType, nil)
							//		message.Content = strings.TrimLeft(ss[1][i:], "\n")
							//		util.AddExtra(message, cst.EventMessageContentType, cst.EventMessageContentTypeCode)
							//		writer.Send(message, nil)
							//		noCodeType = false // 有了代码类型
							//		break
							//	}
							//}
						}
						status = cst.EventMessageContentTypeCode
					} else {
						segment++
						ss := strings.Split(msg.Content, cst.CodeBound)
						// 左侧代码内容
						codeMsg := schema.AssistantMessage(ss[0], nil)
						util.AddExtra(codeMsg, cst.EventMessageContentType, cst.EventMessageContentTypeCode)
						writer.Send(codeMsg, nil)
						// 右侧可能是文本内容也可能没有了
						if len(ss) > 1 && len(strings.Trim(ss[1], "\n")) > 0 {
							textMsg := schema.AssistantMessage(ss[1], nil)
							util.AddExtra(textMsg, cst.EventMessageContentType, cst.EventMessageContentTypeText)
							writer.Send(textMsg, nil)
						}
						status = cst.EventMessageContentTypeText

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

func (c *ClaudeChatModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return c.cli.WithTools(tools)
}
