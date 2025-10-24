package message

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
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
	CreateNewMessage(ctx context.Context, nm *Message) (err error)
	AllMessage(ctx context.Context, conversation string) ([]*Message, error)
	ListMessage(ctx context.Context, conversation string, page *basic.Page) (msgs []*Message, hasMore bool, err error)
	Feedback(ctx context.Context, mid primitive.ObjectID, feedback int32) (err error)
}

type mongoMapper struct {
	conn *monc.Model
	rs   *redis.Redis
}

func NewMessageMongoMapper(config *config.Config) MongoMapper {
	conn := monc.MustNewModel(config.Mongo.URL, config.Mongo.DB, collection, config.Cache)
	rs := redis.MustNewRedis(config.Redis)
	return &mongoMapper{conn: conn, rs: rs}
}

// CreateNewMessage 创建一条新的消息, 需要填充所有的字段
func (m *mongoMapper) CreateNewMessage(ctx context.Context, message *Message) (err error) {
	if message == nil {
		return
	}
	if _, err = m.conn.InsertOneNoCache(ctx, message); err != nil {
		logs.Errorf("[message mapper] insert one err:%s", errorx.ErrorWithoutStack(err))
		return
	}
	_ = m.addOneMsg(ctx, message)
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
		return err
	}

	if err = m.buildCache(ctx, msgs); err != nil {
		logs.Errorf("[message mapper] update many: build cache err:%s", errorx.ErrorWithoutStack(err))
		return nil
	}
	return err
}

// AllMessage 获取对话中所有的message
func (m *mongoMapper) AllMessage(ctx context.Context, conversation string) (msgs []*Message, err error) {
	if msgs, err = m.listAllMsg(ctx, cacheKeyPrefix+conversation); err == nil { // 缓存中找到了 optimize ?如果出现缓存更新失败?
		return
	}
	ocid, err := primitive.ObjectIDFromHex(conversation)
	if err != nil {
		return nil, err
	}
	if err = m.conn.Find(ctx, &msgs, bson.M{cst.ConversationId: ocid, cst.Status: bson.M{cst.NE: cst.DeletedStatus}},
		options.Find().SetSort(bson.M{cst.CreateTime: -1})); err != nil {
		if !errors.Is(err, mongo.ErrNoDocuments) {
			logs.Errorf("[message mapper] find err:%s", errorx.ErrorWithoutStack(err))
			return nil, err
		}
		return msgs, nil
	}
	if len(msgs) == 0 {
		return msgs, nil
	}
	// 删除原缓存
	if _, err = m.rs.DelCtx(ctx, genCacheKey(msgs[0])); err != nil {
		logs.Errorf("[message mapper] delete cache err:%s", errorx.ErrorWithoutStack(err))
		return msgs, nil // 缓存不应该影响正常使用
	}
	// 构建新缓存
	if err = m.buildCache(ctx, msgs); err != nil {
		logs.Errorf("[message mapper] build cache err:%s", errorx.ErrorWithoutStack(err))
		return msgs, nil
	}
	return msgs, nil
}

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

// 向redis中加入一个msg
func (m *mongoMapper) addOneMsg(ctx context.Context, msg *Message) error {
	key := genCacheKey(msg)
	field := key + strconv.Itoa(int(msg.Index))
	// 序列化消息
	data, err := json.Marshal(msg)
	if err != nil {
		logs.Errorf("[message mapper] addOneMsg: json.Marshal err:%s", errorx.ErrorWithoutStack(err))
		return err
	}
	err = m.rs.PipelinedCtx(ctx, func(pipeliner redis.Pipeliner) error {
		if err := pipeliner.HSet(ctx, key, field, string(data)).Err(); err != nil {
			logs.Errorf("[message mapper] addOneMsg: HSet err:%s", errorx.ErrorWithoutStack(err))
			return err
		}
		if err := pipeliner.Expire(ctx, key, 6*time.Hour).Err(); err != nil {
			logs.Errorf("[message mapper] addOneMsg: Expire err:%s", errorx.ErrorWithoutStack(err))
			return err
		}
		return nil
	})
	return err
}

// 获取redis中所有的msg
func (m *mongoMapper) listAllMsg(ctx context.Context, key string) ([]*Message, error) {
	result, err := m.rs.HgetallCtx(ctx, key)
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, errors.New("the key is not found")
	}
	msgs := make([]*Message, len(result), len(result))
	for _, data := range result {
		var msg Message
		if err = json.Unmarshal([]byte(data), &msg); err != nil {
			logs.Errorf("[message mapper] listAllMsg: json.Unmarshal err:%s", errorx.ErrorWithoutStack(err))
			return nil, err
		}
		msgs[len(result)-1-int(msg.Index)] = &msg // 倒序
	}
	return msgs, nil
}

func (m *mongoMapper) buildCache(ctx context.Context, msgs []*Message) (err error) {
	if len(msgs) == 0 {
		return nil
	}

	key := genCacheKey(msgs[0])
	fields := make(map[string]string, len(msgs))
	for _, msg := range msgs {
		var data []byte
		field := key + strconv.Itoa(int(msg.Index))
		if data, err = json.Marshal(msg); err != nil {
			logs.Errorf("[message mapper] buildCache: json.Marshal err:%s", errorx.ErrorWithoutStack(err))
			return err
		}
		fields[field] = string(data)
	}
	// 构建缓存并设置过期时间
	err = m.rs.PipelinedCtx(ctx, func(pipeliner redis.Pipeliner) error {
		if err := pipeliner.HMSet(ctx, key, fields).Err(); err != nil {
			logs.Errorf("[message mapper] buildCache: HMSET err:%s", errorx.ErrorWithoutStack(err))
			return err
		}
		if err := pipeliner.Expire(ctx, key, time.Hour*6).Err(); err != nil {
			logs.Errorf("[message mapper] buildCache: Expire err:%s", errorx.ErrorWithoutStack(err))
			return err
		}
		return nil
	})
	return err
}

func genCacheKey(msg *Message) string {
	return cacheKeyPrefix + msg.ConversationId.Hex()
}

// Feedback 修改消息反馈状态, 这里没有修改redis中消息缓存状态, 因为redis中的状态只用于模型对话, 与反馈状态无关
func (m *mongoMapper) Feedback(ctx context.Context, mid primitive.ObjectID, feedback int32) (err error) {
	var ori Message
	if err = m.conn.FindOneAndUpdateNoCache(ctx, &ori, bson.M{cst.Id: mid}, bson.M{cst.Set: bson.M{cst.Feedback: feedback}}); err != nil {
		return err
	}
	if feedback == cst.FeedbackDelete {
		key := genCacheKey(&ori)
		if _, err = m.rs.HdelCtx(ctx, key, key+strconv.Itoa(int(ori.Index))); err != nil {
			return err
		}
	}
	return err
}
