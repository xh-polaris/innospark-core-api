package msg

import (
	"context"
	"time"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/schema"
	"github.com/jinzhu/copier"
	"github.com/xh-polaris/innospark-core-api/biz/application/dto/core_api"
	mmsg "github.com/xh-polaris/innospark-core-api/biz/infra/mapper/message"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	Text        = 1
	Id          = "Id"
	Brief       = "brief"
	Regen       = "regen"
	Replace     = "replace"
	SelectRegen = "select_regen"
)

var info = &callbacks.RunInfo{Name: "Model-Completion", Type: "History", Component: "History"}
var DefaultHandler = callbacks.NewHandlerBuilder().OnStartFn(initMessage).OnEndFn(updateHistory).Build()
var RegenHandler = callbacks.NewHandlerBuilder().OnStartFn(initMessage).OnEndFn(updateHistory).Build()
var SelectRegenHandler = callbacks.NewHandlerBuilder().OnStartFn(initMessage).OnEndFn(updateHistory).Build()
var ReplaceHandler = callbacks.NewHandlerBuilder().OnStartFn(initMessage).OnEndFn(updateHistory).Build()

// CMsgToMMsgList 将 core_api.Message 切片转换为 message.Message
func CMsgToMMsgList(cid, sid primitive.ObjectID, messages []*core_api.Message) (msgs []*mmsg.Message) {
	for _, msg := range messages {
		msgs = append(msgs, CMsgToMMsg(cid, sid, msg))
	}
	return
}

// CMsgToMMsg 将 core_api.Message 转换为 message.Message
func CMsgToMMsg(cid, sid primitive.ObjectID, messages *core_api.Message) (msgs *mmsg.Message) {
	return &mmsg.Message{
		MessageId:      primitive.NewObjectID(),
		ConversationId: cid,
		SectionId:      sid,
		Content:        messages.Content,
		ContentType:    messages.ContentType,
		Role:           mmsg.RoleStoI[messages.Role],
		CreateTime:     time.Now(),
		UpdateTime:     time.Now(),
		Status:         0,
	}
}

// MMsgToEMsgList 将 core_api.Message 切片转换为 eino/schema.Message 切片
func MMsgToEMsgList(messages []*mmsg.Message) (msgs []*schema.Message) {
	for _, msg := range messages {
		msgs = append(msgs, MMsgToEMsg(msg))
	}
	return
}

// MMsgToEMsg 将单个 core_api.Message 转换为 eino/schema.Message
func MMsgToEMsg(msg *mmsg.Message) *schema.Message {
	return &schema.Message{
		Role:    schema.RoleType(mmsg.RoleItoS[msg.Role]),
		Content: msg.Content,
		Name:    msg.MessageId.String(),
	}
}

func ConvFromEinoList(messages []*schema.Message) (msgs []*core_api.Message) {
	for _, msg := range messages {
		msgs = append(msgs, ConvFromEino(msg))
	}
	return
}

func ConvFromEino(msg *schema.Message) *core_api.Message {
	return &core_api.Message{
		Content:     msg.Content,
		ContentType: Text, // 目前都是Content, 所以忽略不管
		Role:        string(msg.Role),
	}
}

// GetMessagesAndCallBacks
// 消息顺序: 按时间倒序
// 处理得到合适的对话记录以供模型生成使用, 同时根据不同的配置项注入不同的切面
func GetMessagesAndCallBacks(ctx context.Context, req *core_api.CompletionsReq) (nc context.Context, messages []*schema.Message, err error) {
	// 获取历史记录
	cid, err := primitive.ObjectIDFromHex(req.ConversationId)
	if err != nil {
		return nil, nil, err
	}
	mmsgs := append(CMsgToMMsgList(cid, cid, req.Messages), getHistory(req.ConversationId)...)

	// 据自定义配置, 对消息进行处理
	nc = ctx
	option := req.CompletionsOption
	switch {
	case option.IsRegen: // 重新生成, 覆盖掉最新的模型输出, 生成regen_list
		mmsgs = mmsgs[1:] // 因为是重新生成, 所以新的message没用
		var regens []*mmsg.Message
		for _, msg := range mmsgs { // 将此前同一个replyId的消息置为空
			if msg.ReplyId.Hex() == *req.ReplyId {
				rmsg := &mmsg.Message{Ext: &mmsg.Ext{}}
				if err = copier.Copy(msg, rmsg); err != nil {
					return nil, nil, err
				}
				regens = append(regens, rmsg)
				msg.Ext.Brief, msg.Content = msg.Content, ""
			} else {
				break
			}
		}
		nc = context.WithValue(nc, Regen, regens) // 保存regen_list
		nc = callbacks.InitCallbacks(nc, info, RegenHandler)
	case option.IsReplace: // 替换, 替换最新的一条用户消息, 实际是将最近一轮对话设为空且不保留
		mmsgs[1].Ext.Brief, mmsgs[1].Content = mmsgs[1].Content, ""
		mmsgs[2].Ext.Brief, mmsgs[2].Content = mmsgs[2].Content, ""
		nc = context.WithValue(nc, Replace, []primitive.ObjectID{mmsgs[1].MessageId, mmsgs[2].MessageId})
		nc = callbacks.InitCallbacks(nc, info, ReplaceHandler)
	case option.SelectedRegenId != nil: // 选择一个重新生成的结果, 并开始新的对话
		var sr []*mmsg.Message
		reply := mmsgs[1].ReplyId
		for _, msg := range mmsgs[1:] { // 只保留一个regen, 其余清空
			if msg.ReplyId == reply {
				rmsg := &mmsg.Message{Ext: &mmsg.Ext{}}
				if err = copier.Copy(msg, rmsg); err != nil {
					return nil, nil, err
				}
				sr = append(sr, rmsg)

				if msg.MessageId.Hex() != *option.SelectedRegenId {
					msg.Content = msg.Ext.Brief
				} else {
					msg.Ext.Brief, msg.Content = msg.Content, ""
				}
			} else {
				break
			}
		}
		nc = context.WithValue(nc, SelectRegen, sr)
		nc = callbacks.InitCallbacks(nc, info, SelectRegenHandler)
	default: // 默认情况, 生成对话, 更新历史记录
		nc = callbacks.InitCallbacks(nc, info, DefaultHandler)
	}
	return nc, MMsgToEMsgList(mmsgs), err
}

// getHistory TODO 获取历史记录
// 获取逻辑: 优先从redis中获取, 若不存在再尝试从数据库中装配
func getHistory(conversation string) (messages []*mmsg.Message) {
	// 从redis中获取
	// 从数据库中获取
	// 转换为 schema.Message
	return messages
}

// updateHistory TODO 更新历史记录
//
// 返回一个channel, 等待获取新历史记录并更新
// 更新逻辑: 先更新redis, 然后同步到数据库中
func updateHistory(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
	util.DPrintf("[updateHistory callbacks] %+v\n%+v\n", info, output)
	return ctx
}

// initMessage
func initMessage(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
	util.DPrintf("OnStart initMessage: %+v\n%+v\n", info, input)
	return ctx
}
