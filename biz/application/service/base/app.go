package base

import (
	"github.com/xh-polaris/innospark-core-api/biz/application/service/completions"
	conversationapp "github.com/xh-polaris/innospark-core-api/biz/application/service/conversation"
	feedbackapp "github.com/xh-polaris/innospark-core-api/biz/application/service/feedback"
	"github.com/xh-polaris/innospark-core-api/biz/application/service/intelligence"
	manageapp "github.com/xh-polaris/innospark-core-api/biz/application/service/manage"
	"github.com/xh-polaris/innospark-core-api/biz/application/service/system"
	userapp "github.com/xh-polaris/innospark-core-api/biz/application/service/user"
	"github.com/xh-polaris/innospark-core-api/biz/conf"
	"github.com/xh-polaris/innospark-core-api/biz/domain/memory"
	"github.com/xh-polaris/innospark-core-api/biz/domain/memory/history"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cache"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cache/redis"
	"github.com/xh-polaris/innospark-core-api/biz/infra/mapper/conversation"
	"github.com/xh-polaris/innospark-core-api/biz/infra/mapper/feedback"
	"github.com/xh-polaris/innospark-core-api/biz/infra/mapper/message"
	"github.com/xh-polaris/innospark-core-api/biz/infra/mapper/user"
	"github.com/xh-polaris/innospark-core-api/biz/infra/storage"
	"github.com/xh-polaris/innospark-core-api/pkg/ac"
)

type AppDependency struct {
	Cache              cache.Cmdable
	COS                storage.COS
	MessageMapper      message.MongoMapper
	UserMapper         user.MongoMapper
	ConversationMapper conversation.MongoMapper
	FeedbackMapper     feedback.MongoMapper

	His    *history.HistoryManager
	Memory *memory.MemoryManager
}

func InitInfra(deps *AppDependency) {
	deps.Cache = redis.New(conf.GetConfig())
	deps.COS = storage.NewCOS(conf.GetConfig())
	deps.MessageMapper = message.NewMessageMongoMapper(conf.GetConfig())
	deps.UserMapper = user.NewUserMongoMapper(conf.GetConfig())
	deps.ConversationMapper = conversation.NewConversationMongoMapper(conf.GetConfig())
	deps.FeedbackMapper = feedback.NewFeedbackMongoMapper(conf.GetConfig())
	if err := ac.InitAc(conf.GetConfig().Sensitive.Sensitive); err != nil {
		panic(err)
	}
}

func InitApp() {
	deps := &AppDependency{}
	InitInfra(deps)
	InitComponent(deps)
	InitService(deps)
}
func InitComponent(deps *AppDependency) {
	deps.His = history.New(deps.Cache, deps.MessageMapper)
	deps.Memory = memory.New(deps.His)
}

func InitService(deps *AppDependency) {
	completions.InitCompletionsSVC(deps.Memory)
	conversationapp.InitConversationSVC(deps.ConversationMapper, deps.MessageMapper)
	feedbackapp.InitFeedbackSVC(deps.MessageMapper, deps.FeedbackMapper, deps.His)
	userapp.InitUserSVC(deps.UserMapper)
	intelligence.InitIntelligenceSVC()
	manageapp.InitManageSVC(deps.UserMapper, deps.FeedbackMapper)
	system.InitAttachSVC(deps.COS, deps.UserMapper)
}
