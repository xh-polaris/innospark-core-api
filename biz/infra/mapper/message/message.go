package message

import (
	"time"

	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	RoleStoI = map[string]int32{cst.System: 0, cst.Assistant: 1, cst.User: 2, cst.Tool: 3}
	RoleItoS = map[int32]string{0: cst.System, 1: cst.Assistant, 2: cst.User, 3: cst.Tool}
)

// Message 一条消息, 可能归属于用户或模型
type Message struct {
	MessageId      primitive.ObjectID `json:"message_id" bson:"_id"`                              // 主键
	ConversationId primitive.ObjectID `json:"conversation_id" bson:"conversation_id"`             // 归属的对话id
	SectionId      primitive.ObjectID `json:"section_id" bson:"section_id"`                       // 归属的段落id
	UserId         primitive.ObjectID `json:"user_id" bson:"user_id"`                             // 用户id
	Index          int32              `json:"index" bson:"index"`                                 // 消息索引
	ReplyId        primitive.ObjectID `json:"reply_id,omitempty" bson:"reply_id,omitempty"`       // 回复id, 只有模型消息有
	Content        string             `json:"content" bson:"content"`                             // 消息内容, json字符串
	ContentType    int32              `json:"content_type" bson:"content_type"`                   // 内容类型, text/think/suggest, 依次为0,1,2
	MessageType    int32              `json:"message_type" bson:"message_type"`                   // 消息类型, 默认为text, 0
	Ext            *Ext               `json:"ext" bson:"ext"`                                     // 额外信息
	Feedback       int32              `json:"feedback,omitempty" bson:"feedback,omitempty"`       // 反馈, 无/喜欢/踩/删除, 依次为0,1,2,3
	Role           int32              `json:"role" bson:"role"`                                   // 角色, system/assistant/user/tool, 依次为0,1,2,3,4
	CreateTime     time.Time          `json:"create_time" bson:"create_time"`                     // 创建时间
	UpdateTime     time.Time          `json:"update_time" bson:"update_time"`                     // 更新时间
	DeleteTime     time.Time          `json:"delete_time,omitempty" bson:"delete_time,omitempty"` // 删除时间
	Status         int32              `json:"status" bson:"status"`                               // 状态, 默认/regen未选择/regen被选择/替换过/中断, 依次是0,1,2,3
}

type Ext struct {
	BotState string `json:"bot_state" bson:"bot_state"`                 // json字符串, 模型信息
	Brief    string `json:"brief,omitempty" bson:"brief,omitempty"`     // 内容备份
	Think    string `json:"think,omitempty" bson:"think,omitempty"`     // 深度思考内容
	Suggest  string `json:"suggest,omitempty" bson:"suggest,omitempty"` // 建议内容
}
