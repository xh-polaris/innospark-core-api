package message

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/xh-polaris/innospark-core-api/biz/application/dto/basic"
	"github.com/xh-polaris/innospark-core-api/biz/infra/config"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util/logx"
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
	if _, err = m.conn.InsertOneNoCache(ctx, message); err != nil {
		logx.Error("[message mapper] insert one err:%+v", err)
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
		logx.Error("[message mapper] update many: bulk write err:%v", err)
		return err
	}

	if err = m.buildCache(ctx, msgs); err != nil {
		logx.Error("[message mapper] update many: build cache err:%v", err)
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
	if err = m.conn.Find(ctx, msgs, bson.M{cst.ConversationId: ocid, cst.Status: bson.M{cst.NE: cst.DeletedStatus}},
		options.Find().SetSort(bson.M{cst.CreateTime: -1})); err != nil {
		logx.Error("[message mapper] find err:%v", err)
		return nil, err
	}
	// 删除原缓存
	if _, err = m.rs.DelCtx(ctx, genCacheKey(msgs[0])); err != nil {
		logx.Error("[message mapper] delete cache err:%v", err)
		return msgs, nil // 缓存不应该影响正常使用
	}
	// 构建新缓存
	if err = m.buildCache(ctx, msgs); err != nil {
		logx.Error("[message mapper] build cache err:%v", err)
		return msgs, nil
	}
	return msgs, nil
}

func (m *mongoMapper) ListMessage(ctx context.Context, conversation string, page *basic.Page) (msgs []*Message, hasMore bool, err error) {
	ocid, err := primitive.ObjectIDFromHex(conversation)
	if err != nil {
		return nil, false, err
	}
	var total int64
	if total, err = m.conn.CountDocuments(ctx, bson.M{cst.ConversationId: ocid, cst.Status: bson.M{cst.NE: cst.DeletedStatus}}); err != nil {
		logx.Error("[message mapper] count documents err:%v", err)
	}
	if err = m.conn.Find(ctx, &msgs, bson.M{cst.ConversationId: ocid, cst.Status: bson.M{cst.NE: cst.DeletedStatus}},
		util.BuildFindOption(page).SetSort(bson.M{cst.CreateTime: -1})); err != nil {
		logx.Error("[message mapper] find err:%v", err)
		return nil, false, err
	}
	return msgs, util.HasMore(total, page), nil
}

// 向redis中加入一个msg
func (m *mongoMapper) addOneMsg(ctx context.Context, msg *Message) error {
	key := genCacheKey(msg)
	field := key + strconv.Itoa(int(msg.Index))
	// 序列化消息
	data, err := json.Marshal(msg)
	if err != nil {
		logx.Error("[message mapper] addOneMsg: json.Marshal err:%v", err)
		return err
	}
	err = m.rs.HsetCtx(ctx, key, field, string(data))
	return err
}

// 获取redis中所有的msg
func (m *mongoMapper) listAllMsg(ctx context.Context, key string) ([]*Message, error) {
	result, err := m.rs.HgetallCtx(ctx, key)
	if err != nil {
		return nil, err
	}
	msgs := make([]*Message, len(result), len(result))
	for _, data := range result {
		var msg Message
		if err = json.Unmarshal([]byte(data), &msg); err != nil {
			logx.Error("[message mapper] listAllMsg: json.Unmarshal err:%v", err)
			return nil, err
		}
		msgs[len(result)-1-int(msg.Index)] = &msg // 倒序
	}
	return msgs, nil
}

func (m *mongoMapper) buildCache(ctx context.Context, msgs []*Message) (err error) {
	var data []byte
	for _, msg := range msgs {
		key := genCacheKey(msg)
		field := key + strconv.Itoa(int(msg.Index))
		if data, err = json.Marshal(msg); err != nil {
			logx.Error("[message mapper] buildCache: json.Marshal err:%v", err)
			return err
		}
		err = m.rs.HsetCtx(ctx, key, field, string(data))
	}
	return err
}

func genCacheKey(msg *Message) string {
	return cacheKeyPrefix + msg.ConversationId.Hex()
}

// Feedback 修改消息反馈状态, 这里没有修改redis中消息缓存状态, 因为redis中的状态只用于模型对话, 与反馈状态无关
func (m *mongoMapper) Feedback(ctx context.Context, mid primitive.ObjectID, feedback int32) (err error) {
	_, err = m.conn.UpdateOneNoCache(ctx, bson.M{cst.Id: mid}, bson.M{cst.Set: bson.M{cst.Feedback: feedback}})
	return err
}
