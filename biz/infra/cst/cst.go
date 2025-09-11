package cst

const (
	// System is the role of a system, means the message is a system message.
	System     = "system"
	SystemEnum = 0
	// Assistant is the role of an assistant, means the message is returned by ChatModel.
	Assistant     = "assistant"
	AssistantEnum = 1
	// User is the role of a user, means the message is a user message.
	User     = "user"
	UserEnum = 2
	// Tool is the role of a tool, means the message is a tool call output.
	Tool     = "tool"
	ToolEnum = 3
)

// ctx 存储键
const (
	SSEStream      = "sse_stream"
	CompletionInfo = "completion_info"
	OptionInfo     = "option_info"
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
	EventMessageContentTypeText    = 0
	EventMessageContentTypeThink   = 1
	EventMessageContentTypeSuggest = 2
	MessageStatus                  = 0
)

// 消息相关枚举值
const (
	ContentTypeText      = 0
	MessageTypeText      = 0
	InputContentTypeText = 0
	ConversationTypeText = 0
)

// schema.Message 中Extra携带信息
const (
	EventMessageContentType = "event_message_content_type" // 模型消息
	FinalMessage            = "final_message"              // 最终消息
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
	Id             = "_id"
	ConversationId = "conversation_id"
	MessageId      = "message_id"
	UserId         = "user_id"
	CreateTime     = "create_time"
	UpdateTime     = "update_time"
	DeleteTime     = "delete_time"
	Brief          = "brief"
	Feedback       = "feedback"

	Status        = "status"
	DeletedStatus = -1
	Meta          = "$meta"
	TextScore     = "textScore"
	Score         = "score"
	NE            = "$ne"
	Set           = "$set"
	Text          = "$text"
	Search        = "$search"
	Regex         = "$regex"
	Options       = "$options"
)

const (
	FeedbackNone = iota
	FeedbackLike
	FeedbackDislike
	FeedbackDelete
)
