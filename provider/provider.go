package provider

import (
	"github.com/google/wire"
	"github.com/xh-polaris/innospark-core-api/biz/application/service"
	"github.com/xh-polaris/innospark-core-api/biz/infra/config"
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
	Config             *config.Config
	CompletionsService service.ICompletionsService
}

func Get() *Provider {
	return provider
}

var RPCSet = wire.NewSet()

var ApplicationSet = wire.NewSet(
	service.CompletionsServiceSet,
)

var DomainSet = wire.NewSet()

var InfraSet = wire.NewSet(
	config.NewConfig,
	RPCSet,
)

var AllProvider = wire.NewSet(
	ApplicationSet,
	DomainSet,
	InfraSet,
)
