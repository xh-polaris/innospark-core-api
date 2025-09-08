package config

import (
	"os"
	"sync"

	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/core/stores/cache"

	"github.com/zeromicro/go-zero/core/conf"
)

var (
	config *Config
	once   sync.Once
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

type Deyu struct {
	APIKey  string
	BaseURL string
}

type InnoSpark struct {
	DefaultAPIKey    string
	DefaultBaseURL   string
	DeepThinkAPIKey  string
	DeepThinkBaseURL string
}

type Config struct {
	service.ServiceConf
	ListenOn  string
	Auth      Auth
	Deyu      Deyu
	InnoSpark InnoSpark
	Cache     cache.CacheConf
	Mongo     Mongo
}

func NewConfig() (*Config, error) {
	c := new(Config)
	path := os.Getenv("CONFIG_PATH")
	if path == "" {
		path = "etc/config.yaml"
	}
	err := conf.Load(path, c)
	if err != nil {
		return nil, err
	}
	err = c.SetUp()
	if err != nil {
		return nil, err
	}
	config = c
	return config, nil
}

func GetConfig() *Config {
	return config
}
