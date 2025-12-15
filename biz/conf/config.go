package conf

import (
	"io"
	"os"
	"strings"
	"sync"

	confx "github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/core/stores/cache"
)

var (
	config *Config
	once   sync.Once
)

type Config struct {
	service.ServiceConf
	ListenOn   string
	State      string
	SynapseURL string
	Auth       *Auth
	InnoSpark  *InnoSpark
	CacheConf  cache.CacheConf
	Cache      *Cache
	Mongo      *Mongo
	Bocha      *Bocha
	ARK        *ARK
	Claude     *Claude
	Coze       *Coze
	ASR        *ASR
	Sensitive  *Sensitive
	Admin      *Admin
	TitleGen   string
	COS        *COS
	CoTea      *CoTea
}

func NewConfig() (*Config, error) {
	once.Do(func() {
		paths := []string{"etc/config.yaml", "etc/sensitive.yaml", "etc/cotea.yaml"}
		var err error
		var data []byte
		var yamlDocs []string
		for _, path := range paths {
			var f *os.File
			if f, err = os.Open(path); err != nil {
				panic(err)
			}
			if data, err = io.ReadAll(f); err != nil {
				panic(err)
			}
			yamlDocs = append(yamlDocs, string(data))
		}
		c, yaml := new(Config), []byte(strings.Join(yamlDocs, "\r\n"))
		// 用 "---\n" 拼接多个 YAML 文档
		if err = confx.LoadFromYamlBytes(yaml, c); err != nil {
			panic(err)
		}
		if err = c.SetUp(); err != nil {
			panic(err)
		}
		config = c
	})
	return config, nil
}

func GetConfig() *Config {
	_, _ = NewConfig()
	return config
}
