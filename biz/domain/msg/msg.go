package msg

import (
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/xh-polaris/innospark-core-api/biz/application/dto/core_api"
	"github.com/xh-polaris/innospark-core-api/biz/domain/info"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	mmsg "github.com/xh-polaris/innospark-core-api/biz/infra/mapper/message"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func UserMMsg(relay *info.RelayContext, index int) *mmsg.Message {
	now := time.Now()
	return &mmsg.Message{
		MessageId:      primitive.NewObjectID(),
		ConversationId: relay.ConversationId,
		SectionId:      relay.SectionId,
		UserId:         relay.UserId,
		Index:          int32(index),
		Content:        relay.OriginMessage.Content,
		ContentType:    relay.OriginMessage.ContentType,
		MessageType:    cst.MessageTypeText,
		Ext:            &mmsg.Ext{Brief: relay.OriginMessage.Content},
		Role:           cst.UserEnum,
		CreateTime:     now,
		UpdateTime:     now,
		Status:         0,
	}
}

func NewModelMsg(relay *info.RelayContext, index int) *mmsg.Message {
	var err error
	var replayId primitive.ObjectID
	if relay.ReplyId != "" {
		if replayId, err = primitive.ObjectIDFromHex(relay.ReplyId); err != nil {
			replayId = primitive.NewObjectID()
			relay.ReplyId = replayId.Hex()
		}
	}
	now := time.Now()
	return &mmsg.Message{
		MessageId:      primitive.NewObjectID(),
		ConversationId: relay.ConversationId,
		SectionId:      relay.SectionId,
		UserId:         relay.UserId,
		Index:          int32(index),
		ReplyId:        replayId,
		ContentType:    cst.ContentTypeText,
		MessageType:    cst.ContentTypeText,
		Ext:            nil,
		Role:           cst.AssistantEnum,
		CreateTime:     now,
		UpdateTime:     now,
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
	m := &schema.Message{
		Role:    schema.RoleType(mmsg.RoleItoS[msg.Role]),
		Content: msg.Content,
		Name:    msg.MessageId.Hex(),
	}
	if msg.Ext.ContentWithCite != nil {
		m.Content = *msg.Ext.ContentWithCite
	}
	return m
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
