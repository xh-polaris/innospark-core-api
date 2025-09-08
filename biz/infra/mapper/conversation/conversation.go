package conversation

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Conversation 记录一次对话, 包含用户和模型的所有消息以及对话的基本信息
type Conversation struct {
	ConversationId primitive.ObjectID `json:"conversation_id" bson:"conversation_id"`             // 主键
	UserId         primitive.ObjectID `json:"user_id" bson:"user_id"`                             // 索引
	Brief          string             `json:"brief" bson:"brief"`                                 // 对话标题
	CreateTime     time.Time          `json:"create_time" bson:"create_time"`                     // 创建时间
	UpdateTime     time.Time          `json:"update_time" bson:"update_time"`                     // 更新时间
	DeleteTime     time.Time          `json:"delete_time,omitempty" bson:"delete_time,omitempty"` // 删除时间
	Status         int32              `json:"status" bson:"status"`                               // 状态
}
