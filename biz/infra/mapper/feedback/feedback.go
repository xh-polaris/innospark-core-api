package feedback

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type FeedBack struct {
	MessageId  primitive.ObjectID `json:"message_id" bson:"_id"`                              // 用message_id作为主键, 一个消息只会有一条feedback, 类型为4时不绑定消息
	UserId     primitive.ObjectID `json:"user_id" bson:"user_id"`                             // 用户id
	Action     int32              `json:"action,omitempty" bson:"action,omitempty"`           // 触发反馈的原因, none/like/dislike/delete/侧边栏反馈 0/1/2/3/4
	Type       int32              `json:"type,omitempty" bson:"type,omitempty"`               // 反馈类型
	Content    string             `json:"content,omitempty" bson:"content,omitempty"`         // 反馈的额外内容, 比如选择其他时填的内容
	UpdateTime time.Time          `json:"update_time,omitempty" bson:"update_time,omitempty"` // 修改时间
}
