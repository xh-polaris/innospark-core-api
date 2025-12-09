package feedback

import (
	"github.com/xh-polaris/innospark-core-api/biz/domain/memory/history"
	"github.com/xh-polaris/innospark-core-api/biz/infra/mapper/feedback"
	"github.com/xh-polaris/innospark-core-api/biz/infra/mapper/message"
)

func InitFeedbackSVC(message message.MongoMapper, feedback feedback.MongoMapper, his *history.HistoryManager) {
	FeedbackSVC = &FeedbackService{
		MessageMapper:  message,
		FeedbackMapper: feedback,
		His:            his,
	}
}
