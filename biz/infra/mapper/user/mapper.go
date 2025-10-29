package user

import (
	"context"
	"time"

	"github.com/xh-polaris/innospark-core-api/biz/infra/config"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	"github.com/zeromicro/go-zero/core/stores/monc"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var _ MongoMapper = (*mongoMapper)(nil)

const (
	collection     = "user"
	cacheKeyPrefix = "cache:user:"
)

type MongoMapper interface {
	FindOrCreateUser(ctx context.Context, id string, login bool) (*User, error) // 查找或创建一个用户
}

type mongoMapper struct {
	conn *monc.Model
	rs   *redis.Redis
}

func NewUserMongoMapper(config *config.Config) MongoMapper {
	conn := monc.MustNewModel(config.Mongo.URL, config.Mongo.DB, collection, config.Cache)
	rs := redis.MustNewRedis(config.Redis)
	return &mongoMapper{conn: conn, rs: rs}
}

func (m *mongoMapper) FindOrCreateUser(ctx context.Context, id string, login bool) (*User, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	key := cacheKeyPrefix + id
	filter := bson.M{cst.Id: oid}
	update := bson.M{"$setOnInsert": bson.M{
		cst.Id:         oid,
		cst.CreateTime: time.Now(),
		cst.UpdateTime: time.Now(),
	}}
	if login {
		update["$set"] = bson.M{cst.LoginTime: time.Now()}
	}
	var u User
	err = m.conn.FindOneAndUpdate(ctx, key, &u, filter, update, options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After))
	return &u, err
}
