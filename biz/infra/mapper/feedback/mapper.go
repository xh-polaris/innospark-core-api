package feedback

import (
	"context"

	"github.com/xh-polaris/innospark-core-api/biz/infra/config"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	"github.com/zeromicro/go-zero/core/stores/monc"
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
