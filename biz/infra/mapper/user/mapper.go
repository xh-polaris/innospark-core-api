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

var Mapper MongoMapper = (*mongoMapper)(nil)

const (
	collection     = "user"
	cacheKeyPrefix = "cache:user:"
)

type MongoMapper interface {
	FindOrCreateUser(ctx context.Context, id string, login bool) (*User, error) // 查找或创建一个用户
	CheckForbidden(ctx context.Context, id string) (int, bool, time.Time, error)
	Warn(ctx context.Context, id string) error
	Forbidden(ctx context.Context, id string, expire time.Time) error
	UnForbidden(ctx context.Context, id string) error
}

type mongoMapper struct {
	conn *monc.Model
	rs   *redis.Redis
}

func NewUserMongoMapper(config *config.Config) MongoMapper {
	conn := monc.MustNewModel(config.Mongo.URL, config.Mongo.DB, collection, config.Cache)
	rs := redis.MustNewRedis(config.Redis)
	m := &mongoMapper{conn: conn, rs: rs}
	Mapper = m // 这里依赖的provider的初始化来创建一个全局变量, 不是很好
	return m
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

func (m *mongoMapper) CheckForbidden(ctx context.Context, id string) (int, bool, time.Time, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return 0, false, time.Time{}, err
	}
	filter := bson.M{cst.Id: oid}
	key := cacheKeyPrefix + id
	u := &User{}
	if err = m.conn.FindOne(ctx, key, u, filter); err != nil {
		return 0, false, time.Time{}, err
	}
	if u.Status == StatusForbidden {
		if time.Now().Unix() >= u.Expire.Unix() { // 解封
			u.Status = StatusNormal
			if err = m.UnForbidden(ctx, id); err != nil {
				return 0, false, time.Time{}, err
			}
		}
	}
	return int(u.Status), u.Status == StatusForbidden, u.Expire, nil
}

func (m *mongoMapper) Warn(ctx context.Context, id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	key := cacheKeyPrefix + id
	filter := bson.M{cst.Id: oid}
	update := bson.M{"$inc": bson.M{"warnings": 1}}
	_, err = m.conn.UpdateOne(ctx, key, filter, update)
	return err
}

func (m *mongoMapper) Forbidden(ctx context.Context, id string, expire time.Time) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	key := cacheKeyPrefix + id
	filter := bson.M{cst.Id: oid}
	update := bson.M{"$set": bson.M{cst.Status: StatusForbidden, cst.Expire: expire}}
	_, err = m.conn.UpdateOne(ctx, key, filter, update)
	return err
}

func (m *mongoMapper) UnForbidden(ctx context.Context, id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	key := cacheKeyPrefix + id
	filter := bson.M{cst.Id: oid}
	update := bson.M{"$set": bson.M{cst.Status: StatusNormal, cst.Expire: time.Time{}}}
	_, err = m.conn.UpdateOne(ctx, key, filter, update)
	return err
}
