package user

import "go.mongodb.org/mongo-driver/bson/primitive"

type User struct {
	UserId   primitive.ObjectID `json:"user_id" bson:"_id"`
	Username string             `json:"username" bson:"username"`
	Avatar   string             `json:"avatar" bson:"avatar"`
}
