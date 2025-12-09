package message

import (
	"context"
	"errors"

	"github.com/xh-polaris/innospark-core-api/biz/application/dto/basic"
	"github.com/xh-polaris/innospark-core-api/biz/conf"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util"
	"github.com/xh-polaris/innospark-core-api/pkg/errorx"
	"github.com/xh-polaris/innospark-core-api/pkg/logs"
	"github.com/zeromicro/go-zero/core/stores/monc"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var _ MongoMapper = (*mongoMapper)(nil)

const (
	collection     = "message"
	cacheKeyPrefix = "cache:message:"
)

type MongoMapper interface {
	UpdateMany(ctx context.Context, msg []*Message) (err error)
	ListMessage(ctx context.Context, conversation string, page *basic.Page) (msgs []*Message, hasMore bool, err error)
	Feedback(ctx context.Context, mid primitive.ObjectID, feedback int32) (_ *Message, err error)
	RetrieveMessages(ctx context.Context, conversation string, size int) (msgs []*Message, err error)
	InsertOne(ctx context.Context, msg *Message) error
}

type mongoMapper struct {
	conn *monc.Model
}

func NewMessageMongoMapper(config *conf.Config) MongoMapper {
	conn := monc.MustNewModel(config.Mongo.URL, config.Mongo.DB, collection, config.CacheConf)
	return &mongoMapper{conn: conn}
}

// RetrieveMessages 取出按时间顺序取出size条msg记录, 为0则取出所有的
func (m *mongoMapper) RetrieveMessages(ctx context.Context, conversation string, size int) (msgs []*Message, err error) {
	oid, err := primitive.ObjectIDFromHex(conversation)
	if err != nil {
		return nil, err
	}

	opts := options.Find().SetSort(bson.M{cst.CreateTime: -1})
	if size > 0 {
		opts.SetLimit(int64(size))
	}
	if err = m.conn.Find(ctx, &msgs, bson.M{cst.ConversationId: oid, cst.Status: bson.M{cst.NE: cst.DeletedStatus}},
		opts); err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		logs.Errorf("[message mapper] find err:%s", errorx.ErrorWithoutStack(err))
		return nil, err
	}
	return msgs, nil
}

// InsertOne 插入一条msg
func (m *mongoMapper) InsertOne(ctx context.Context, msg *Message) error {
	_, err := m.conn.InsertOneNoCache(ctx, msg)
	return err
}

// UpdateMany 批量更新信息, 一个事务
func (m *mongoMapper) UpdateMany(ctx context.Context, msgs []*Message) (err error) {
	if msgs == nil || len(msgs) == 0 {
		return nil
	}
	// 开启会话
	session, err := m.conn.StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)
	// 在会话中执行一个事务
	if _, err = session.WithTransaction(ctx, func(sessCtx context.Context) (any, error) {
		var operations []mongo.WriteModel
		for _, msg := range msgs { // 设置批量更新行为
			filter := bson.M{cst.Id: msg.MessageId}
			update := bson.M{cst.Set: msg}
			operations = append(operations, mongo.NewUpdateOneModel().SetFilter(filter).SetUpdate(update))
		}
		_, err = m.conn.BulkWrite(ctx, operations)
		return nil, err
	}); err != nil {
		logs.Errorf("[message mapper] update many: bulk write err:%s", errorx.ErrorWithoutStack(err))
	}
	return err
}

// ListMessage 分页获取Message
func (m *mongoMapper) ListMessage(ctx context.Context, conversation string, page *basic.Page) (msgs []*Message, hasMore bool, err error) {
	ocid, err := primitive.ObjectIDFromHex(conversation)
	if err != nil {
		return nil, false, err
	}
	opts := options.Find().SetSort(bson.M{cst.Id: -1}).SetLimit(page.GetSize() + 1)
	filter := bson.M{cst.ConversationId: ocid, cst.Status: bson.M{cst.NE: cst.DeletedStatus}}
	if page != nil && page.Cursor != nil { // 创建时间更小的
		cursor, err := primitive.ObjectIDFromHex(*page.Cursor)
		if err != nil {
			return nil, false, err
		}
		filter[cst.Id] = bson.M{cst.LT: cursor}
	}
	if err = m.conn.Find(ctx, &msgs, filter, opts); err != nil {
		logs.Errorf("[message mapper] find err:%s", errorx.ErrorWithoutStack(err))
		return nil, false, err
	}
	msgs, hasMore = util.SplitAndHasMore(msgs, page)
	return msgs, hasMore, err
}

// Feedback 修改消息反馈状态
func (m *mongoMapper) Feedback(ctx context.Context, mid primitive.ObjectID, feedback int32) (_ *Message, err error) {
	var ori Message
	if err = m.conn.FindOneAndUpdateNoCache(ctx, &ori, bson.M{cst.Id: mid}, bson.M{cst.Set: bson.M{cst.Feedback: feedback}}); err != nil {
		return nil, err
	}
	return &ori, err
}
