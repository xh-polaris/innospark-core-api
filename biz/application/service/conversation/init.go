package conversation

import (
	"github.com/xh-polaris/innospark-core-api/biz/infra/mapper/conversation"
	mmsg "github.com/xh-polaris/innospark-core-api/biz/infra/mapper/message"
)

func InitConversationSVC(conversation conversation.MongoMapper, message mmsg.MongoMapper) {
	ConversationSVC = &ConversationService{
		ConversationMapper: conversation,
		MessageMapper:      message,
	}
}
