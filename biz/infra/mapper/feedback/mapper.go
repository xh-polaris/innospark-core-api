package feedback

import (
	"context"
	"time"

	"github.com/xh-polaris/innospark-core-api/biz/application/dto/basic"
	"github.com/xh-polaris/innospark-core-api/biz/conf"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util"
	"github.com/zeromicro/go-zero/core/stores/monc"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var Mapper MongoMapper = (*mongoMapper)(nil)

const (
	collection     = "feedback"
	cacheKeyPrefix = "cache:feedback:"
)

type MongoMapper interface {
	UpdateFeedback(ctx context.Context, feedback *FeedBack) error
	Insert(ctx context.Context, uid string, action, typ int32, content string) error
	ListFeedback(ctx context.Context, p *basic.Page, mid, uid *string, action, typ *int32) (int64, []*FeedBack, error)
}

type mongoMapper struct {
	conn *monc.Model
}

func NewFeedbackMongoMapper(config *conf.Config) MongoMapper {
	conn := monc.MustNewModel(config.Mongo.URL, config.Mongo.DB, collection, config.CacheConf)
	Mapper = &mongoMapper{conn: conn}
	return Mapper
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

func (m *mongoMapper) ListFeedback(ctx context.Context, p *basic.Page, message, user *string, action, typ *int32) (total int64, fbs []*FeedBack, err error) {
	filter := bson.M{}
	if message != nil { // 筛选消息
		mid, err := primitive.ObjectIDFromHex(*message)
		if err != nil {
			return 0, nil, err
		}
		filter[cst.Id] = mid
	}
	if user != nil { // 筛选用户
		uid, err := primitive.ObjectIDFromHex(*user)
		if err != nil {
			return 0, nil, err
		}
		filter[cst.UserId] = uid
	}
	if action != nil { // 删选action
		filter[cst.Action] = *action
	}
	if typ != nil { // 筛选type
		filter[cst.Type] = *typ
	}
	option := util.BuildFindOption(p).SetSort(bson.M{cst.UpdateTime: -1})
	err = m.conn.Find(ctx, &fbs, filter, option)
	total, err = m.conn.CountDocuments(ctx, filter)
	return total, fbs, err
}
