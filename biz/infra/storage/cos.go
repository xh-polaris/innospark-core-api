package storage

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/tencentyun/cos-go-sdk-v5"
	"github.com/xh-polaris/innospark-core-api/biz/infra/config"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util/httpx"
)

var _ COSInfra = (*cosClient)(nil)

type COSInfra interface {
	Upload(ctx context.Context, key string, r io.Reader, opt *cos.ObjectPutOptions) (*cos.Response, error)
	GenPresignURL(ctx context.Context, key string, opt *cos.PresignedURLOptions) (string, error)
	GetPermanentAccessURL(key string) string
}

type cosClient struct {
	Conf   *config.COS
	Client *cos.Client
}

func NewCOSInfra() COSInfra {
	return newcosClient()
}

// Upload 管理员上传对象
// key 对象键 应为/{user_id}/{conversation_id}/时间戳
// opt 上传配置，包括缓存策略等 允许为空
// 返回accessURL
func (c *cosClient) Upload(ctx context.Context, key string, r io.Reader, opt *cos.ObjectPutOptions) (*cos.Response, error) {
	if opt == nil {
		opt = &cos.ObjectPutOptions{}
	}

	resp, err := c.Client.Object.Put(ctx, key, r, opt)
	if err != nil {
		// TODO err处理 & log
		return nil, err
	}
	resp.Header.Get("x-cos-hash-crc64ecma") // CRC64校验值
	return resp, nil
}

func (c *cosClient) GenPresignURL(ctx context.Context, key string, opt *cos.PresignedURLOptions) (string, error) {
	if opt == nil {
		opt = &cos.PresignedURLOptions{}
	}
	u, err := c.Client.Object.GetPresignedURL2(ctx, http.MethodPut, key,
		time.Hour, // 1分钟内过期
		opt,
	)
	if err != nil || u == nil {
		return "", err
	}
	return u.String(), nil
}

func (c *cosClient) GetPermanentAccessURL(key string) string {
	return c.Client.Object.GetObjectURL(key).String()
}

func newcosClient() *cosClient {
	conf := config.GetConfig().COS
	b := &cos.BaseURL{
		BucketURL: util.Str2URL(conf.BucketURL), // 访问 bucket, object 相关 API 的基础 URL（不包含 path 部分）
	}
	client := cos.NewClient(b, mustNewCOSHTTPClient())
	return &cosClient{
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
