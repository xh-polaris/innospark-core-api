package manage

import (
	"github.com/xh-polaris/innospark-core-api/biz/infra/mapper/feedback"
	"github.com/xh-polaris/innospark-core-api/biz/infra/mapper/message"
	"github.com/xh-polaris/innospark-core-api/biz/infra/mapper/user"
)

func InitManageSVC(user user.MongoMapper, feedback feedback.MongoMapper, message message.MongoMapper) {
	ManageSVC = &ManageService{
		UserMapper:     user,
		FeedbackMapper: feedback,
		MessageMapper:  message,
	}
}
