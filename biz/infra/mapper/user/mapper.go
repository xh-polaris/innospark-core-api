package user

import (
	"context"
	"time"

	"github.com/xh-polaris/innospark-core-api/biz/application/dto/basic"
	"github.com/xh-polaris/innospark-core-api/biz/infra/config"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util"
	"github.com/xh-polaris/innospark-core-api/pkg/errorx"
	"github.com/xh-polaris/innospark-core-api/pkg/logs"
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
	FindOrCreateUser(ctx context.Context, id string, phone string, login bool) (*User, error) // 查找或创建一个用户
	FindById(ctx context.Context, id string) (*User, error)

	CheckForbidden(ctx context.Context, id string) (*User, int, bool, time.Time, error)
	Warn(ctx context.Context, id string) error
	Forbidden(ctx context.Context, id string, expire time.Time) error
	UnForbidden(ctx context.Context, id string) error
	ListUser(ctx context.Context, page *basic.Page, status, sortedBy, reverse int32) (int64, []*User, error)
	CountUserByCreateTime(ctx context.Context, time time.Time, after bool) (int64, error)

	UpdateField(ctx context.Context, uid primitive.ObjectID, update bson.M) error
	existField(ctx context.Context, field string, value interface{}) (bool, error)
	ExistUsername(ctx context.Context, username string) (bool, error)
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

func (m *mongoMapper) FindOrCreateUser(ctx context.Context, id string, phone string, login bool) (*User, error) {
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
		cst.Phone:      phone,
		cst.Status:     0,
	}}
	if login {
		update["$set"] = bson.M{cst.LoginTime: time.Now()}
	}
	var u User
	err = m.conn.FindOneAndUpdate(ctx, key, &u, filter, update, options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After))
	return &u, err
}

func (m *mongoMapper) FindById(ctx context.Context, id string) (*User, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	key := cacheKeyPrefix + id
	filter := bson.M{cst.Id: oid}
	var u User
	err = m.conn.FindOne(ctx, key, &u, filter)
	return &u, err
}

func (m *mongoMapper) CheckForbidden(ctx context.Context, id string) (*User, int, bool, time.Time, error) {
	u, err := m.FindById(ctx, id)
	if err != nil {
		return nil, 0, false, time.Time{}, err
	}
	if u.Status == StatusForbidden {
		if time.Now().Unix() >= u.Expire.Unix() { // 解封
			u.Status = StatusNormal
			if err = m.UnForbidden(ctx, id); err != nil {
				return nil, 0, false, time.Time{}, err
			}
		}
	}
	return u, int(u.Status), u.Status == StatusForbidden, u.Expire, nil
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

const (
	SortedByCreateTime = 0
	SortedByUpdateTime = 1
	SortedByLoginTime  = 2
)

func (m *mongoMapper) ListUser(ctx context.Context, page *basic.Page, status, sortedBy, reverse int32) (int64, []*User, error) {
	var users []*User
	filter := bson.M{cst.Status: status}
	var sort string
	switch sortedBy {
	case SortedByUpdateTime:
		sort = cst.UpdateTime
	case SortedByLoginTime:
		sort = cst.LoginTime
	case SortedByCreateTime:
		sort = cst.CreateTime
	default:
		sort = cst.CreateTime
	}
	opts := util.BuildFindOption(page).SetSort(bson.M{sort: reverse})
	if err := m.conn.Find(ctx, &users, filter, opts); err != nil {
		return 0, nil, err
	}
	total, err := m.conn.CountDocuments(ctx, filter)
	return total, users, err
}

// UpdateField 更新字段
func (m *mongoMapper) UpdateField(ctx context.Context, uid primitive.ObjectID, update bson.M) error {
	key := cacheKeyPrefix + uid.Hex()
	if _, err := m.conn.UpdateByID(ctx, key, uid, bson.M{"$set": update}); err != nil {
		logs.CtxErrorf(ctx, "failed to update user %s: %s", uid.Hex(), errorx.ErrorWithoutStack(err))
		return err
	}

	return nil
}

// existField 检查字段是否存在
func (m *mongoMapper) existField(ctx context.Context, field string, value interface{}) (bool, error) {
	var err error
	var count int64
	if count, err = m.conn.CountDocuments(ctx, bson.M{field: value}); err != nil {
		logs.CtxErrorf(ctx, "failed to check existing %s: %s", field, errorx.ErrorWithoutStack(err))
		return false, err
	}

	return count > 0, nil
}

// ExistUsername 检查用户名是否存在
func (m *mongoMapper) ExistUsername(ctx context.Context, username string) (bool, error) {
	return m.existField(ctx, cst.Name, username)
}

func (m *mongoMapper) CountUserByCreateTime(ctx context.Context, t time.Time, after bool) (int64, error) {
	var filter bson.M
	if after {
		filter = bson.M{cst.CreateTime: bson.M{cst.GTE: t}} // 统计 t 之后（含 t）
	} else {
		filter = bson.M{cst.CreateTime: bson.M{cst.LT: t}} // 统计 t 之前
	}
	total, err := m.conn.CountDocuments(ctx, filter)
	return total, err
}
