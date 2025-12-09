package message

import (
	"time"

	"github.com/xh-polaris/innospark-core-api/biz/domain/state"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	mmsg "github.com/xh-polaris/innospark-core-api/biz/infra/mapper/message"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// NewUserMMsg 根据状态构建存储域用户消息
func NewUserMMsg(relay *state.RelayContext, index int) (m *mmsg.Message) {
	now := time.Now()
	m = &mmsg.Message{
		MessageId:      primitive.NewObjectID(),
		ConversationId: relay.Info.ConversationId,
		SectionId:      relay.Info.SectionId,
		UserId:         relay.Info.UserId,
		Index:          int32(index),
		Role:           cst.UserEnum,
		ContentType:    relay.Info.OriginMessage.ContentType,
		Ext:            &mmsg.Ext{Brief: relay.Info.OriginMessage.Content},
		CreateTime:     now,
		UpdateTime:     now,
		Status:         0,
	}
	if len(relay.Info.OriginMessage.Attaches) == 0 { // 有附件
		m.Content = relay.Info.OriginMessage.Content
		m.MessageType = cst.MessageTypeText
	} else {
		m.MessageType = cst.MessageTypeMultiple
		m.UserInputMultiContent = []*mmsg.MessageInputPart{}
		m.UserInputMultiContent = append(m.UserInputMultiContent, &mmsg.MessageInputPart{
			Type: mmsg.ChatMessagePartTypeText,
			Text: relay.Info.OriginMessage.Content,
		})
		for _, attach := range relay.Info.OriginMessage.Attaches { // 添加图片消息
			m.UserInputMultiContent = append(m.UserInputMultiContent, &mmsg.MessageInputPart{
				Type: mmsg.ChatMessagePartTypeImageURL,
				Image: &mmsg.MessageInputImage{
					MessagePartCommon: mmsg.MessagePartCommon{
						URL: &attach,
					},
					Detail: mmsg.ImageURLDetailAuto,
				},
			})
		}
	}
	return
}

// NewModelMMsg 构建存储域模型消息
func NewModelMMsg(relay *state.RelayContext, index int) *mmsg.Message {
	var err error
	var replayId primitive.ObjectID
	if relay.Info.ReplyId != "" {
		if replayId, err = primitive.ObjectIDFromHex(relay.Info.ReplyId); err != nil {
			replayId = primitive.NewObjectID()
			relay.Info.ReplyId = replayId.Hex()
		}
	}
	now := time.Now()
	return &mmsg.Message{
		MessageId:      primitive.NewObjectID(),
		ConversationId: relay.Info.ConversationId,
		SectionId:      relay.Info.SectionId,
		UserId:         relay.Info.UserId,
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
