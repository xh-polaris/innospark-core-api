package cos

import (
	"net/http"

	"github.com/tencentyun/cos-go-sdk-v5"
	"github.com/xh-polaris/innospark-core-api/biz/infra/config"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util/httpx"
)

type Client struct {
	Conf   *config.COS
	Client *cos.Client
}

func NewClient() *Client {
	conf := config.GetConfig().COS

	b := &cos.BaseURL{
		BucketURL: util.Str2URL(conf.BucketURL), // 访问 bucket, object 相关 API 的基础 URL（不包含 path 部分）
	}

	client := cos.NewClient(b, mustNewCOSHTTPClient())

	return &Client{
		Conf:   conf,
		Client: client,
	}
}

func mustNewCOSHTTPClient() *http.Client {
	// 与全局单例http客户端采用不同transport，单独为cos服务创建新http客户端实例
	// 其余配置复用单例cli
	conf := config.GetConfig().COS
	gCli := httpx.GetHttpClient()

	authTransport := &cos.AuthorizationTransport{
		SecretID:  conf.SecretID,
		SecretKey: conf.SecretKey,
		Transport: gCli.Client.Transport,
	}

	return &http.Client{
		Transport: authTransport,
	}
}
