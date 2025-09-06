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
