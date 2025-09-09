package provider

import (
	"github.com/google/wire"
	"github.com/xh-polaris/innospark-core-api/biz/application/service"
	"github.com/xh-polaris/innospark-core-api/biz/infra/config"
	"github.com/xh-polaris/innospark-core-api/biz/infra/mapper/conversation"
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
}

func Get() *Provider {
	return provider
}

var RPCSet = wire.NewSet()

var ApplicationSet = wire.NewSet(
	service.CompletionsServiceSet,
	service.ConversationServiceSet,
)

var DomainSet = wire.NewSet()

var InfraSet = wire.NewSet(
	config.NewConfig,
	RPCSet,
	conversation.NewConversationMongoMapper,
)

var AllProvider = wire.NewSet(
	ApplicationSet,
	DomainSet,
	InfraSet,
)
