package conversation

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Conversation struct {
	ConversationId primitive.ObjectID `json:"conversation_id" bson:"conversation_id"`
	UserId         primitive.ObjectID `json:"user_id" bson:"user_id"`
	Brief          string             `json:"brief" bson:"brief"`
	CreateTime     time.Time          `json:"create_time" bson:"create_time"`
	UpdateTime     time.Time          `json:"update_time" bson:"update_time"`
	DeleteTime     time.Time          `json:"delete_time,omitempty" bson:"delete_time,omitempty"`
	Status         int                `json:"status" bson:"status"`
}
