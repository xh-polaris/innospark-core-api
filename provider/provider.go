package provider

import (
	"github.com/google/wire"
	"github.com/xh-polaris/innospark-core-api/biz/application/service"
	"github.com/xh-polaris/innospark-core-api/biz/domain/graph"
	"github.com/xh-polaris/innospark-core-api/biz/infra/config"
	"github.com/xh-polaris/innospark-core-api/biz/infra/mapper/conversation"
	"github.com/xh-polaris/innospark-core-api/biz/infra/mapper/feedback"
	"github.com/xh-polaris/innospark-core-api/biz/infra/mapper/message"
	"github.com/xh-polaris/innospark-core-api/biz/infra/mapper/user"
)

var provider *Provider

func Init() {
	var err error
	provider, err = NewProvider()
	if err != nil {
		panic(err)
	}
}

// Provider 提供controller依赖的对象
type Provider struct {
	Config              *config.Config
	CompletionsService  service.ICompletionsService
	ConversationService service.IConversationService
	FeedbackService     service.IFeedbackService
	UserService         service.IUserService
	IntelligenceService service.IIntelligenceService
	ManageService       service.IManageService
	CompletionGraph     *graph.CompletionGraph
}

func Get() *Provider {
	return provider
}

var RPCSet = wire.NewSet()

var ApplicationSet = wire.NewSet(
	service.CompletionsServiceSet,
	service.ConversationServiceSet,
	service.FeedbackServiceSet,
	service.UserServiceSet,
	service.IntelligenceServiceSet,
	service.ManageServiceSet,
)

var DomainSet = wire.NewSet(
	graph.HistoryDomainSet,
	graph.DrawCompletionGraph,
)

var InfraSet = wire.NewSet(
	config.NewConfig,
	RPCSet,
	conversation.NewConversationMongoMapper,
	message.NewMessageMongoMapper,
	feedback.NewFeedbackMongoMapper,
	user.NewUserMongoMapper,
)

var AllProvider = wire.NewSet(
	ApplicationSet,
	DomainSet,
	InfraSet,
)
