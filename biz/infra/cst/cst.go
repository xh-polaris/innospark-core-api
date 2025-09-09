package cst

const (
	// Assistant is the role of an assistant, means the message is returned by ChatModel.
	Assistant = "assistant"
	// User is the role of a user, means the message is a user message.
	User = "user"
	// System is the role of a system, means the message is a system message.
	System = "system"
	// Tool is the role of a tool, means the message is a tool call output.
	Tool = "tool"
)

// ctx 存储键
const (
	SSEStream      = "sse_stream"
	CompletionInfo = "completion_info"
)

// sse事件类型
const (
	// EventMeta 消息元数据
	EventMeta = "meta"
	// EventModel 模型基本信息
	EventModel = "model"
	// EventChat 模型返回消息
	EventChat = "chat"
	// EventEnd 流结束
	EventEnd      = "end"
	EventEndValue = "{}"
)

// Event中各种类型枚举值
const (
	MessageContentTypeText    = 0
	MessageContentTypeThink   = 1
	MessageContentTypeSuggest = 2
	MessageStatus             = 0
	InputContentTypeText      = 0
	ConversationTypeText      = 0
)

// schema.Message 中Extra携带信息
const (
	MessageContentType = "message_content_type" // 模型消息
	FinalMessage       = "final_message"        // 最终消息
)

// 流式响应标签
const (
	ThinkStart   = "<think>"
	ThinkEnd     = "</think>"
	SuggestStart = "<suggest>"
	SuggestEnd   = "</suggest>"
)

// mapper层字段枚举
const (
	ConversationId = "conversation_id"
	UserId         = "user_id"
	CreateTime     = "create_time"
	UpdateTime     = "update_time"
	DeleteTime     = "delete_time"
	Brief          = "brief"

	Status        = "status"
	DeletedStatus = -1

	NQ  = "$nq"
	Set = "$set"
)
