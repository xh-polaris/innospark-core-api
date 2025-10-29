package user

import (
	"context"

	"github.com/xh-polaris/innospark-core-api/biz/infra/config"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	"github.com/xh-polaris/innospark-core-api/pkg/errorx"
	"github.com/xh-polaris/innospark-core-api/pkg/logs"
	"github.com/zeromicro/go-zero/core/stores/monc"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/v2/bson"
)

var _ MongoMapper = (*mongoMapper)(nil)

const (
	collection     = "user"
	cacheKeyPrefix = "cache:user:"
)

type MongoMapper interface {
	updateField(ctx context.Context, uid primitive.ObjectID, update bson.M) error
	UpdateUsername(ctx context.Context, uid primitive.ObjectID, username string) error
	UpdateAvatar(ctx context.Context, uid primitive.ObjectID, avatar string) error

	existField(ctx context.Context, field string, value interface{}) (bool, error)
	ExistUsername(ctx context.Context, username string) (bool, error)
}

type mongoMapper struct {
	conn *monc.Model
}

func NewUserMongoMapper(config *config.Config) MongoMapper {
	conn := monc.MustNewModel(config.Mongo.URL, config.Mongo.DB, collection, config.Cache)
	return &mongoMapper{conn: conn}
}

// updateField 更新字段
func (m *mongoMapper) updateField(ctx context.Context, uid primitive.ObjectID, update bson.M) error {
	if _, err := m.conn.UpdateByIDNoCache(ctx, uid, bson.M{"$set": update}); err != nil {
		logs.CtxErrorf(ctx, "failed to update user %s: %s", uid.Hex(), errorx.ErrorWithoutStack(err))
		return err
	}

	return nil
}

// UpdateUsername 更新用户名
func (m *mongoMapper) UpdateUsername(ctx context.Context, uid primitive.ObjectID, username string) error {
	return m.updateField(ctx, uid, bson.M{cst.Username: username})
}

// UpdateAvatar 更新用户头像
func (m *mongoMapper) UpdateAvatar(ctx context.Context, uid primitive.ObjectID, avatar string) error {
	return m.updateField(ctx, uid, bson.M{cst.Avatar: avatar})
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
	return m.existField(ctx, cst.Username, username)
}
