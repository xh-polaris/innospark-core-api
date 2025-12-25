package history

import (
	"context"
	"sort"
	"strconv"
	"time"

	"github.com/bytedance/sonic"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cache"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	"github.com/xh-polaris/innospark-core-api/biz/infra/mapper/message"
	"github.com/xh-polaris/innospark-core-api/pkg/errorx"
	"github.com/xh-polaris/innospark-core-api/pkg/logs"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/* 对话历史记录 */

var Mgr *HistoryManager

const cachePrefix = "inno:message:"

// HistoryManager 历史记录管理, 所有的历史记录都按照从旧到新排序
type HistoryManager struct {
	cache  cache.Cmdable
	mapper message.MongoMapper
}

// New 创建一个新的历史记录管理器
func New(cache cache.Cmdable, mapper message.MongoMapper) *HistoryManager {
	Mgr = &HistoryManager{cache: cache, mapper: mapper}
	return Mgr
}

// RetrieveMessage 获取消息, size 小于等于0时取出所有
// 首先从缓存中获取, 获取失败时从数据库中获取, 后重新构建缓存
func (h *HistoryManager) RetrieveMessage(ctx context.Context, id string, size int) (msgs []*message.Message, err error) {
	// retrieve cache
	if msgs, err = h.RetrieveMessagesFromCache(ctx, id, size); err == nil {
		return msgs, nil
	}
	// retrieve storage
	if msgs, err = h.mapper.RetrieveMessages(ctx, id, size); err != nil {
		return nil, err
	}
	// rebuild cache
	if len(msgs) > 0 {
		if err = h.CacheMessages(ctx, id, msgs, true); err != nil {
			logs.Errorf("cache msgs err: %s", err)
		}
	}
	return msgs, nil
}

// RetrieveMessagesFromCache 从缓存中获取一批消息
// 缓存中找不到时返回cache.Nil, 否则返回size指定的数量
func (h *HistoryManager) RetrieveMessagesFromCache(ctx context.Context, id string, size int) ([]*message.Message, error) {
	result, err := h.cache.HGetAll(ctx, key(id)).Result()
	if err != nil {
		return nil, err
	} else if len(result) == 0 {
		return nil, cache.Nil
	}

	msgs := make([]*message.Message, len(result), len(result))
	i := 0
	for _, data := range result {
		var msg message.Message
		if err = sonic.Unmarshal([]byte(data), &msg); err != nil {
			logs.Errorf("[message mapper] listAllMsg: json.Unmarshal err:%s", errorx.ErrorWithoutStack(err))
			return nil, err
		}
		msgs[i] = &msg
		i++
	}

	if len(msgs) > 0 {
		sort.Slice(msgs, func(i, j int) bool { return msgs[i].Index > msgs[j].Index }) // 倒序
	}
	if size > 0 && len(msgs) > size {
		return msgs[:size], nil
	}
	return msgs, nil
}

// CacheMessages 构建msgs的缓存, re为true时删除已有缓存
func (h *HistoryManager) CacheMessages(ctx context.Context, id string, msgs []*message.Message, re bool) (err error) {
	fields := make(map[string]string, len(msgs))
	for _, msg := range msgs {
		var data []byte
		if data, err = sonic.Marshal(msg); err != nil {
			return err
		}
		fields[strconv.Itoa(int(msg.Index))] = string(data)
	}
	p, k := h.cache.Pipeline(), key(id)
	if re {
		p.Del(ctx, k)
	}
	p.HSet(ctx, k, fields)
	p.Expire(ctx, k, time.Hour*6)
	_, err = p.Exec(ctx)
	return
}

// AddMessage 新增消息
// 首先插入数据库, 然后缓存消息
func (h *HistoryManager) AddMessage(ctx context.Context, id string, msg *message.Message) (err error) {
	// add to storage
	if err = h.mapper.InsertOne(ctx, msg); err != nil {
		logs.Errorf("add message err: %s", err)
		return
	}
	// add to cache
	if err = h.CacheMessages(ctx, key(id), []*message.Message{msg}, false); err != nil {
		logs.Errorf("cache msgs err: %s", err)
	}
	return
}

// UpdateMessages 批量更新消息
// 批量更新数据库, 然后删除缓存
func (h *HistoryManager) UpdateMessages(ctx context.Context, msgs []*message.Message) (err error) {
	if msgs == nil || len(msgs) == 0 {
		return
	}

	if err = h.mapper.UpdateMany(ctx, msgs); err != nil {
		logs.Errorf("update message err: %s", err)
		return
	}
	if err = h.cache.Del(ctx, keyFromMsg(msgs[0])).Err(); err != nil {
		logs.Errorf("delete cache err: %s", err)
	}
	return
}

// Feedback 对消息进行反馈, 由于涉及到删除消息所以此处处理
func (h *HistoryManager) Feedback(ctx context.Context, mid primitive.ObjectID, feedback int32) (err error) {
	var msg *message.Message
	if msg, err = h.mapper.Feedback(ctx, mid, feedback); err != nil {
		logs.Errorf("feedback err: %s", err)
		return
	}
	if feedback == cst.FeedbackDelete {
		if err = h.cache.HDel(ctx, keyFromMsg(msg)).Err(); err != nil {
			logs.Errorf("delete cache err: %s", err)
			return
		}
	}
	return
}

func key(id string) string {
	return cachePrefix + id
}

func keyFromMsg(msg *message.Message) string {
	return key(msg.ConversationId.Hex())
}
