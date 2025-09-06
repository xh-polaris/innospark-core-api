package message

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	RoleStoI = map[string]int32{"System": 0, "Assistant": 1, "User": 2, "Tool": 3}
	RoleItoS = map[int32]string{0: "System", 1: "Assistant", 2: "User", 3: "Tool"}
)

type Message struct {
	MessageId      primitive.ObjectID `json:"message_id" bson:"message_id"`
	ConversationId primitive.ObjectID `json:"conversation_id" bson:"conversation_id"`
	SectionId      primitive.ObjectID `json:"section_id" bson:"section_id"`
	Index          int                `json:"index" bson:"index"`
	ReplyId        primitive.ObjectID `json:"reply_id,omitempty" bson:"reply_id,omitempty"`
	Content        string             `json:"content" bson:"content"`
	ContentType    int32              `json:"content_type" bson:"content_type"`
	MessageType    int32              `json:"message_type" bson:"message_type"`
	Ext            *Ext               `json:"ext" bson:"ext"`
	Feedback       int32              `json:"feedback" bson:"feedback"`
	Role           int32              `json:"role" bson:"role"`
	CreateTime     time.Time          `json:"create_time" bson:"create_time"`
	UpdateTime     time.Time          `json:"update_time" bson:"update_time"`
	DeleteTime     time.Time          `json:"delete_time,omitempty" bson:"delete_time,omitempty"`
	Status         int32              `json:"status" bson:"status"`
}

type Ext struct {
	BotState string `json:"bot_state" bson:"bot_state"`
	Brief    string `json:"brief,omitempty" bson:"brief,omitempty"`
}
