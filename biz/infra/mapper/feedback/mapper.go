package feedback

import (
	"context"
	"time"

	"github.com/xh-polaris/innospark-core-api/biz/infra/config"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	"github.com/zeromicro/go-zero/core/stores/monc"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var _ MongoMapper = (*mongoMapper)(nil)

const (
	collection     = "feedback"
	cacheKeyPrefix = "cache:feedback:"
)

type MongoMapper interface {
	UpdateFeedback(ctx context.Context, feedback *FeedBack) error
	Insert(ctx context.Context, uid string, action, typ int32, content string) error
}

type mongoMapper struct {
	conn *monc.Model
}

func NewFeedbackMongoMapper(config *config.Config) MongoMapper {
	conn := monc.MustNewModel(config.Mongo.URL, config.Mongo.DB, collection, config.Cache)
	return &mongoMapper{conn: conn}
}

// UpdateFeedback 更新(不存在则插入)反馈
func (m *mongoMapper) UpdateFeedback(ctx context.Context, feedback *FeedBack) (err error) {
	_, err = m.conn.UpdateOneNoCache(ctx, bson.M{cst.Id: feedback.MessageId, cst.UserId: feedback.UserId}, bson.M{cst.Set: feedback},
		options.UpdateOne().SetUpsert(true))
	return
}
func (m *mongoMapper) Insert(ctx context.Context, uid string, action, typ int32, content string) (err error) {
	oid, err := primitive.ObjectIDFromHex(uid)
	if err != nil {
		return
	}
	f := &FeedBack{
		MessageId:  primitive.NewObjectID(),
		UserId:     oid,
		Action:     action,
		Type:       typ,
		Content:    content,
		UpdateTime: time.Now(),
	}
	_, err = m.conn.InsertOneNoCache(ctx, f)
	return
}
