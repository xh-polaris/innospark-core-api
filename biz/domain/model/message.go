package model

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/google/wire"
	"github.com/xh-polaris/innospark-core-api/biz/application/dto/core_api"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	mmsg "github.com/xh-polaris/innospark-core-api/biz/infra/mapper/message"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util/logx"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	Text        = 1
	Default     = "default"
	Regen       = "regen"
	Replace     = "replace"
	SelectRegen = "select_regen"
)

type OptionInfo struct {
	Typ         string
	ReplyId     string
	Regen       []*mmsg.Message
	Replace     []*mmsg.Message
	SelectRegen []*mmsg.Message
	UserMessage *mmsg.Message
}

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
		Name:    msg.MessageId.Hex(),
	}
}

func MMsgToFMsgList(messages []*mmsg.Message) (msgs []*core_api.FullMessage) {
	for _, msg := range messages {
		msgs = append(msgs, MMsgToFMsg(msg))
	}
	return
}

func MMsgToFMsg(msg *mmsg.Message) *core_api.FullMessage {
	fm := &core_api.FullMessage{
		ConversationId: msg.ConversationId.Hex(),
		SectionId:      msg.SectionId.Hex(),
		MessageId:      msg.MessageId.Hex(),
		Index:          msg.Index,
		Status:         msg.Status,
		CreateTime:     msg.CreateTime.Unix(),
		MessageType:    msg.MessageType,
		ContentType:    msg.ContentType,
		Content:        msg.Content,
		Ext: &core_api.Ext{
			BotState: msg.Ext.BotState,
			Brief:    msg.Ext.Brief,
			Think:    msg.Ext.Think,
			Suggest:  msg.Ext.Suggest,
		},
		Feedback: msg.Feedback,
		UserType: msg.Role,
	}
	if !msg.ReplyId.IsZero() {
		reply := msg.ReplyId.Hex()
		fm.ReplyId = &reply
	}
	return fm
}

// GetMessagesAndInjectContext
// 消息顺序: 按时间倒序
// 处理得到合适的对话记录以供模型生成使用, 同时根据不同的配置项构造消息记录和OptionInfo
func (m *MessageDomain) GetMessagesAndInjectContext(ctx context.Context, user string, req *core_api.CompletionsReq) (_ context.Context, messages []*schema.Message, info *OptionInfo, err error) {
	info = &OptionInfo{Typ: Default}
	// 获取历史记录
	cid, err := primitive.ObjectIDFromHex(req.ConversationId)
	if err != nil {
		return nil, nil, info, err
	}
	uid, err := primitive.ObjectIDFromHex(user)
	if err != nil {
		return nil, nil, info, err
	}
	his, err := m.getHistory(ctx, req.ConversationId)
	if err != nil {
		return nil, nil, info, err
	}
	mmsgs := his

	// 据自定义配置, 对消息进行处理
	nc, option := ctx, req.CompletionsOption
	switch {
	case option.IsRegen: // 重新生成, 覆盖掉最新的模型输出, 生成regen_list, 不需要增添user message
		var regens []*mmsg.Message
		for _, msg := range mmsgs { // 将此前同一个replyId且不为空的消息置为空
			if msg.ReplyId.Hex() == *req.ReplyId && msg.Content != "" {
				msg.Content = ""
				regens = append(regens, msg)
			} else if msg.Role == cst.UserEnum && msg.Content != "" { // 找到的第一个用户消息
				info.ReplyId = msg.MessageId.Hex()
				break
			}
		}
		info.Typ, info.Regen = Regen, regens // 保存regen_list
	case option.IsReplace: // 替换最新的一条用户消息, 实际是将最近一轮有效对话设为空且不保留, 需要新的user message
		info.Typ = Replace
		for _, msg := range mmsgs {
			if msg.Content != "" {
				switch msg.Role {
				case cst.UserEnum:
					msg.Content = ""
					info.Replace = append(info.Replace, msg)
				case cst.AssistantEnum:
					msg.Content = ""
					info.Replace = append(info.Replace, msg)
				}
			} else if len(info.Replace) == 2 {
				break
			}
		}
	case option.SelectedRegenId != nil: // 选择一个重新生成的结果, 并开始新的对话, 需要增加用户消息
		var sr []*mmsg.Message
		reply := mmsgs[0].ReplyId
		for _, msg := range mmsgs { // 只保留一个regen, 其余清空
			if msg.ReplyId == reply {
				if msg.MessageId.Hex() == *option.SelectedRegenId {
					msg.Content = msg.Ext.Brief
				} else {
					msg.Content = ""
				}
				sr = append(sr, msg)
			} else {
				break
			}
		}
		info.Typ, info.SelectRegen = SelectRegen, sr
	}
	if !option.IsRegen {
		um := userMessage(cid, uid, len(his), req)
		mmsgs = append([]*mmsg.Message{um}, mmsgs...)
		info.UserMessage = um
		info.ReplyId = um.MessageId.Hex()
	}
	return nc, MMsgToEMsgList(mmsgs), info, err
}

// 构造用户消息
func userMessage(cid, uid primitive.ObjectID, index int, req *core_api.CompletionsReq) *mmsg.Message {
	now := time.Now()
	return &mmsg.Message{
		MessageId:      primitive.NewObjectID(),
		ConversationId: cid,
		SectionId:      cid,
		UserId:         uid,
		Index:          int32(index),
		Content:        req.Messages[0].Content,
		ContentType:    cst.ContentTypeText,
		MessageType:    cst.MessageTypeText,
		Ext:            &mmsg.Ext{Brief: req.Messages[0].Content},
		Feedback:       0,
		Role:           cst.UserEnum,
		CreateTime:     now,
		UpdateTime:     now,
		Status:         0,
	}
}

type MessageDomain struct {
	MsgMapper mmsg.MongoMapper
}

var MessageDomainSet = wire.NewSet(wire.Struct(new(MessageDomain), "*"))

// getHistory
func (m *MessageDomain) getHistory(ctx context.Context, conversation string) (messages []*mmsg.Message, err error) {
	msgs, err := m.MsgMapper.AllMessage(ctx, conversation)
	if err != nil {
		return nil, err
	}
	return msgs, nil
}

func (m *MessageDomain) ProcessHistory(ctx context.Context, info *CompletionInfo) {
	option := info.OptionInfo
	if option.Typ != Default {
		var msg []*mmsg.Message
		// 处理选项
		switch option.Typ {
		case Regen: // 重新生成, 更新regen列表中的消息
			msg = append(msg, option.Regen...)
		case Replace: // 替换, 更新replace列表中的消息
			msg = append(msg, option.Replace...)
		case SelectRegen: // 选择重新生成, 更新SelectRegen中的消息
			msg = append(msg, option.SelectRegen...)
		}
		if err := m.MsgMapper.UpdateMany(ctx, msg); err != nil {
			logx.Error("[domain message] process history option err: %v", err)
			return
		}
	}

	oids, err := util.ObjectIDsFromHex(info.MessageId, info.ConversationId, info.SectionId, info.UserId, info.ReplyId)
	if err != nil {
		logx.Error("[domain message] process history new message err: %v", err)
		return
	}
	// 增加模型消息
	now := time.Now()
	mm := &mmsg.Message{
		MessageId: oids[0], ConversationId: oids[1], SectionId: oids[2], UserId: oids[3],
		Index: int32(info.MessageIndex), ReplyId: oids[4],
		Content: info.Text, ContentType: int32(info.ContentType), MessageType: int32(info.MessageType),
		Ext: &mmsg.Ext{
			BotState: fmt.Sprintf("{\"model\":\"%s\",\"bot_id\":\"%s\",\"bot_name\":\"%s\"}", info.Model, info.BotId, info.BotName),
			Brief:    info.Text,
			Think:    info.Think,
			Suggest:  info.Suggest,
		},
		Feedback: 0, Role: cst.AssistantEnum,
		CreateTime: now, UpdateTime: now, Status: 0,
	}
	if err = m.MsgMapper.CreateNewMessage(context.WithoutCancel(ctx), option.UserMessage); err != nil {
		logx.Error("[domain message] process history new message err: %v", err)
	}
	if err = m.MsgMapper.CreateNewMessage(context.WithoutCancel(ctx), mm); err != nil {
		logx.Error("[domain message] process history create new err: %v]", err)
	}
	return
}
