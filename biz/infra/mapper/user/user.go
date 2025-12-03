package user

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	StatusNormal    = 0 // 正常状态
	StatusForbidden = 1 // 封禁状态
)

// User 用户
type User struct {
	ID         primitive.ObjectID `json:"id" bson:"_id,omitempty"`                  // ID
	Phone      string             `json:"phone" bson:"phone,omitempty"`             // 手机号
	Avatar     string             `json:"avatar" bson:"avatar,omitempty"`           // 头像
	Name       string             `json:"name" bson:"name,omitempty"`               // 用户名
	Profile    *Profile           `json:"profile" bson:"profile,omitempty"`         // 个性化内容
	Warnings   int32              `json:"warnings" bson:"warnings"`                 // 违规次数
	Status     int32              `json:"status" bson:"status"`                     // 状态
	Expire     time.Time          `json:"expire,omitempty" bson:"expire,omitempty"` // 封禁到期时间
	LoginTime  time.Time          `json:"login_time" bson:"login_time"`             // 最近登录时间
	CreateTime time.Time          `json:"create_time" bson:"create_time"`
	UpdateTime time.Time          `json:"update_time" bson:"update_time"`
}

// Profile 个性化内容
type Profile struct {
	Role string `json:"role,omitempty" bson:"role,omitempty"` // 角色设定
}
