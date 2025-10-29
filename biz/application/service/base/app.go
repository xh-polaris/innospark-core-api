package base

import (
	"github.com/xh-polaris/innospark-core-api/biz/infra/config"
	"github.com/xh-polaris/innospark-core-api/pkg/ac"
)

func InitInfra() {
	if err := ac.InitAc(config.GetConfig().Sensitive); err != nil {
		panic(err)
	}
}

func InitApp() {
	InitInfra()
}
