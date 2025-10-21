package config

import (
	"os"
	"sync"

	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util/errorx"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

var (
	config *Config
	once   sync.Once
	loadConfigError error
)

type Auth struct {
	SecretKey    string
	PublicKey    string
	AccessExpire int64
}

type Mongo struct {
	URL string
	DB  string
}

type InnoSpark struct {
	DefaultBaseURL   string
	DefaultAPIKey    string
	DeepThinkBaseURL string
	DeepThinkAPIKey  string
}

type Bocha struct {
	APIKey   string
	Template string
}

type ARK struct {
	APIKey          string
	CodeGenTemplate string
}

type Claude struct {
	BaseURL string
	APIKey  string
}

type Config struct {
	service.ServiceConf
	ListenOn   string
	State      string
	SynapseURL string
	Auth       *Auth
	InnoSpark  *InnoSpark
	Cache      cache.CacheConf
	Redis      redis.RedisConf
	Mongo      *Mongo
	Bocha      *Bocha
	ARK        *ARK
	Claude     *Claude
	Coze       *Coze
	ASR        *ASR
}

func NewConfig() (*Config, error) {
	once.Do(func() {
		c := new(Config)
		path := os.Getenv("CONFIG_PATH")
		if path == "" {
			path = "etc/config.yaml"
		}
		err := conf.Load(path, c)
		if err != nil {
			loadConfigError = errorx.WrapByCode(err, cst.ConfigLoadErrCode, errorx.KV("path", path))
			return
		}
		err = c.SetUp()
		if err != nil {
			loadConfigError = errorx.WrapByCode(err, cst.ConfigSetupErrCode)
			return
		}
		config = c
	})

	if loadConfigError != nil {
		return nil, loadConfigError
	}
	
	return config, nil
}

func GetConfig() *Config {
	_, _ = NewConfig()
	return config
}
