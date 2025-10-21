package conversation

import (
	"context"
	"time"

	"github.com/xh-polaris/innospark-core-api/biz/application/dto/basic"
	"github.com/xh-polaris/innospark-core-api/biz/infra/config"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util/errorx"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util/logx"
	"github.com/zeromicro/go-zero/core/stores/monc"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var _ MongoMapper = (*mongoMapper)(nil)

const (
	collection     = "conversation"
	cacheKeyPrefix = "cache:conversation:"
)

type MongoMapper interface {
	CreateNewConversation(ctx context.Context, uid, botId string) (c *Conversation, err error)
	ListConversations(ctx context.Context, uid string, page *basic.Page) (cs []*Conversation, hasMore bool, err error)
	UpdateConversationBrief(ctx context.Context, uid, cid, brief string) (err error)
	DeleteConversation(ctx context.Context, uid, cid string) (err error)
	SearchConversations(ctx context.Context, uid, key string, page *basic.Page) (cs []*Conversation, hasMore bool, err error)
}

type mongoMapper struct {
	conn *monc.Model
}

func NewConversationMongoMapper(config *config.Config) MongoMapper {
	conn := monc.MustNewModel(config.Mongo.URL, config.Mongo.DB, collection, config.Cache)
	return &mongoMapper{conn: conn}
}

// CreateNewConversation 创建并缓存一个新的对话
func (m *mongoMapper) CreateNewConversation(ctx context.Context, uid, botId string) (c *Conversation, err error) {
	// 转换成ObjectID
	oid, err := primitive.ObjectIDFromHex(uid)
	if err != nil {
		logx.Error("[mapper] [conversation] [CreateNewConversation] from hex err:%s", errorx.ErrorWithoutStack(err))
		return nil, err
	}

	// 创建新Conversation
	now := time.Now()
	c = &Conversation{
		ConversationId: primitive.NewObjectID(),
		UserId:         oid,
		BotId:          botId,
		CreateTime:     now,
		UpdateTime:     now,
		Brief:          "未命名对话",
	}

	// 插入
	_, err = m.conn.InsertOne(ctx, cacheKeyPrefix+c.ConversationId.Hex(), c)
	return c, err
}

// ListConversations 分页查询用户对话列表
func (m *mongoMapper) ListConversations(ctx context.Context, uid string, page *basic.Page) (cs []*Conversation, hasMore bool, err error) {
	// 转换为ObjectID
	oid, err := primitive.ObjectIDFromHex(uid)
	if err != nil {
		logx.Error("[mapper] [conversation] [ListConversation] from hex err:%s", errorx.ErrorWithoutStack(err))
		return nil, false, err
	}

	// 创建ID倒序
	opts := options.Find().SetSort(bson.M{cst.Id: -1}).SetLimit(page.GetSize() + 1)
	filter := bson.M{cst.UserId: oid, cst.Status: bson.M{cst.NE: cst.DeletedStatus}}
	if page != nil && page.Cursor != nil { // 存在cursor时, 查询创建时间小于Cursor的
		cursor, err := primitive.ObjectIDFromHex(*page.Cursor)
		if err != nil {
			return nil, false, err
		}
		filter[cst.Id] = bson.M{cst.LT: cursor}
	}
	if err = m.conn.Find(ctx, &cs, filter, opts); err != nil {
		return nil, false, err
	}
	cs, hasMore = util.SplitAndHasMore(cs, page)
	return cs, hasMore, err
}

func (m *mongoMapper) DeleteConversation(ctx context.Context, uid, cid string) (err error) {
	ouid, err := primitive.ObjectIDFromHex(uid)
	if err != nil {
		logx.Error("[mapper] [conversation] [DeleteConversation] from hex err:%s", errorx.ErrorWithoutStack(err))
		return err
	}
	ocid, err := primitive.ObjectIDFromHex(cid)
	if err != nil {
		logx.Error("[mapper] [conversation] [DeleteConversation] from hex err:%s", errorx.ErrorWithoutStack(err))
		return err
	}

	// 更新对应uid,cid且未删除的对话
	filter := bson.M{cst.Id: ocid, cst.UserId: ouid, cst.Status: bson.M{cst.NE: cst.DeletedStatus}}
	_, err = m.conn.UpdateOne(ctx, cacheKeyPrefix+cid, filter,
		bson.M{cst.Set: bson.M{cst.UpdateTime: time.Now(), cst.DeleteTime: time.Now(), cst.Status: cst.DeletedStatus}})
	return err
}

// UpdateConversationBrief 更新对话简要概述
func (m *mongoMapper) UpdateConversationBrief(ctx context.Context, uid, cid, brief string) (err error) {
	oids, err := util.ObjectIDsFromHex(uid, cid)
	if err != nil {
		logx.Error("[mapper] [conversation] [UpdateConversation] from hex err:%s", errorx.ErrorWithoutStack(err))
		return err
	}
	ouid, ocid := oids[0], oids[1]
	filter := bson.M{cst.Id: ocid, cst.UserId: ouid, cst.Status: bson.M{cst.NE: cst.DeletedStatus}}
	_, err = m.conn.UpdateOne(ctx, cacheKeyPrefix+cid, filter,
		bson.M{cst.Set: bson.M{cst.UpdateTime: time.Now(), cst.Brief: brief}})
	return err
}

func (m *mongoMapper) SearchConversations(ctx context.Context, uid, key string, page *basic.Page) (cs []*Conversation, hasMore bool, err error) {
	// 转换为ObjectID
	oid, err := primitive.ObjectIDFromHex(uid)
	if err != nil {
		logx.Error("[mapper] [conversation] [ListConversation] from hex err:%s", errorx.ErrorWithoutStack(err))
		return nil, false, err
	}

	// 分词搜索key
	filter := bson.M{cst.UserId: oid, cst.Status: bson.M{cst.NE: cst.DeletedStatus}, cst.Brief: bson.M{cst.Regex: key, cst.Options: "i"}}
	// 分页, 创建时间倒序
	opts := options.Find().SetSort(bson.M{cst.Id: -1}).SetLimit(page.GetSize() + 1)
	if page != nil && page.Cursor != nil { // 存在cursor时, 查询创建时间小于Cursor的
		cursor, err := primitive.ObjectIDFromHex(*page.Cursor)
		if err != nil {
			return nil, false, err
		}
		filter[cst.Id] = bson.M{cst.LT: cursor}
	}
	if err = m.conn.Find(ctx, &cs, filter, opts); err != nil {
		return nil, false, err
	}
	cs, hasMore = util.SplitAndHasMore(cs, page)
	return cs, hasMore, err
}
