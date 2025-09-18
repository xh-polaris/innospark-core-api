package graph

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/xh-polaris/innospark-core-api/biz/adaptor"
	mmsg "github.com/xh-polaris/innospark-core-api/biz/infra/mapper/message"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RelayContext 存储Completion接口过程中的上下文信息
type RelayContext struct {
	RequestContext    *app.RequestContext
	CompletionOptions *CompletionOptions // 对话配置
	ModelInfo         *ModelInfo         // 模型信息
	MessageInfo       *MessageInfo       // 消息信息
	ConversationId    primitive.ObjectID // 对话id
	SectionId         primitive.ObjectID // 段落id
	UserId            primitive.ObjectID // 用户id
	ReplyId           string
	OriginMessage     *ReqMessage
	UserMessage       *mmsg.Message      // 用户消息
	SSE               *adaptor.SSEStream // SSE流
	ModelCancel       context.CancelFunc // 中断模型输出
}

// CompletionOptions 是对话相关配置
type CompletionOptions struct {
	Typ             string
	ReplyId         *string
	IsRegen         bool
	IsReplace       bool
	SelectedRegenId *string
	RegenList       []*mmsg.Message
	ReplaceList     []*mmsg.Message
	SelectRegenList []*mmsg.Message
}

// ModelInfo 是模型相关配置
type ModelInfo struct {
	Model   string // 模型名称
	BotId   string // 智能体id
	BotName string // 智能体名称
}

type MessageInfo struct {
	AssistantMessage *mmsg.Message
	Text             string // 对话内容
	Think            string // 思考内容
	Suggest          string // 建议内容
}

type ReqMessage struct {
	Content     string
	ContentType int32
	Attaches    []string
	References  []string
}
