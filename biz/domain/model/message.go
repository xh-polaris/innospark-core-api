package model

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/google/wire"
	"github.com/jinzhu/copier"
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
	Regen       []*mmsg.Message
	Replace     []*mmsg.Message
	SelectRegen []*mmsg.Message
	CreateTime  time.Time
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

// GetMessagesAndCallBacks
// 消息顺序: 按时间倒序
// 处理得到合适的对话记录以供模型生成使用, 同时根据不同的配置项注入不同的切面
func (m *MessageDomain) GetMessagesAndCallBacks(ctx context.Context, user string, req *core_api.CompletionsReq) (_ context.Context, messages []*schema.Message, err error) {
	info := &OptionInfo{CreateTime: time.Now(), Typ: Default}
	// 获取历史记录
	cid, err := primitive.ObjectIDFromHex(req.ConversationId)
	if err != nil {
		return nil, nil, err
	}
	uid, err := primitive.ObjectIDFromHex(user)
	if err != nil {
		return nil, nil, err
	}
	his, err := m.getHistory(ctx, req.ConversationId)
	if err != nil {
		return nil, nil, err
	}
	// 构造用户消息
	// 增加用户消息
	um := &mmsg.Message{
		MessageId:      primitive.NewObjectID(),
		ConversationId: cid,
		SectionId:      cid,
		UserId:         uid,
		Index:          int32(len(his)),
		Content:        req.Messages[0].Content,
		ContentType:    cst.ContentTypeText,
		MessageType:    cst.MessageTypeText,
		Ext:            &mmsg.Ext{Brief: req.Messages[0].Content},
		Feedback:       0,
		Role:           cst.UserEnum,
		CreateTime:     info.CreateTime,
		UpdateTime:     info.CreateTime,
		Status:         0,
	}
	info.UserMessage = um
	mmsgs := append([]*mmsg.Message{um}, his...)

	// 据自定义配置, 对消息进行处理
	nc, option := ctx, req.CompletionsOption
	switch {
	case option.IsRegen: // 重新生成, 覆盖掉最新的模型输出, 生成regen_list
		mmsgs = mmsgs[1:] // 因为是重新生成, 所以新的message没用
		var regens []*mmsg.Message
		for _, msg := range mmsgs { // 将此前同一个replyId且不为空的消息置为空
			if msg.ReplyId.Hex() == *req.ReplyId && msg.Content != "" {
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
		info.Typ, info.Regen = Regen, regens // 保存regen_list
	case option.IsReplace: // 替换, 替换最新的一条用户消息, 实际是将最近一轮有效对话设为空且不保留
		info.Typ = Replace
		for _, msg := range mmsgs[1:] {
			if msg.Content != "" {
				switch msg.Role {
				case cst.UserEnum:
					msg.Ext.Brief, msg.Content = msg.Content, ""
					info.Replace = append(info.Replace, msg)
				case cst.AssistantEnum:
					msg.Ext.Brief, msg.Content = msg.Content, ""
					info.Replace = append(info.Replace, msg)
				}
			} else if len(info.Replace) == 2 {
				break
			}
		}
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

				if msg.MessageId.Hex() == *option.SelectedRegenId {
					msg.Content = msg.Ext.Brief
				} else {
					msg.Ext.Brief, msg.Content = msg.Content, ""
				}
			} else {
				break
			}
		}
		info.Typ, info.SelectRegen = SelectRegen, sr
	}
	nc = context.WithValue(nc, cst.OptionInfo, info)
	return nc, MMsgToEMsgList(mmsgs), err
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
	option := ctx.Value(cst.OptionInfo).(*OptionInfo)
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
	if err = m.MsgMapper.CreateNewMessage(ctx, option.UserMessage); err != nil {
		logx.Error("[domain message] process history new message err: %v", err)
	}
	if err = m.MsgMapper.CreateNewMessage(ctx, mm); err != nil {
		logx.Error("[domain message] process history create new err: %v]", err)
	}
	return
}
