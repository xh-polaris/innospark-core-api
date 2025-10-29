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
const ()

// sse事件类型
const (
	// EventMeta 消息元数据
	EventMeta = "meta"
	// EventModel 模型基本信息
	EventModel = "model"
	// EventChat 模型返回消息
	EventChat = "chat"
	// EventEnd 流结束
	EventEnd = "end"
	// EventNotifyValue 仅作标记, 无实际值
	EventNotifyValue  = "{}"
	EventSearchStart  = "searchStart"
	EventSearchEnd    = "searchEnd"
	EventSearchFind   = "searchFind"
	EventSearchChoose = "searchChoice"
	EventSearchCite   = "searchCite"
)

// Event中各种类型枚举值
const (
	EventMessageContentTypeText     = 0
	EventMessageContentTypeThink    = 1
	EventMessageContentTypeSuggest  = 2
	EventMessageContentTypeCode     = 3 // 代码
	EventMessageContentTypeCodeType = 4 // 代码
	MessageStatus                   = 0
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
	RawMessage              = "raw_message"                // 模型原始消息
	ModelCite               = "model_cite"
)

// 流式响应标签
const (
	ThinkStart   = "<think>"
	ThinkEnd     = "</think>"
	SuggestStart = "<suggest>"
	SuggestEnd   = "</suggest>"
	CodeBound    = "```"
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
	LoginTime      = "login_time"
	Brief          = "brief"
	Feedback       = "feedback"

	Status        = "status"
	DeletedStatus = -1
	Meta          = "$meta"
	TextScore     = "textScore"
	Score         = "score"
	NE            = "$ne"
	LT            = "$lt"
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
