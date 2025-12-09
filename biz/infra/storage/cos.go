package storage

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/tencentyun/cos-go-sdk-v5"
	"github.com/xh-polaris/innospark-core-api/biz/conf"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util/httpx"
)

var _ COS = (*cosClient)(nil)

type COS interface {
	Upload(ctx context.Context, key string, r io.Reader, opt *cos.ObjectPutOptions) (*cos.Response, error)
	GenPresignURL(ctx context.Context, key string, opt *cos.PresignedURLOptions) (string, error)
	GetPermanentAccessURL(key string) string
}

type cosClient struct {
	Conf   *conf.COS
	Client *cos.Client
}

func NewCOS(c *conf.Config) COS {
	return newcosClient(c)
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
		time.Minute, // 1分钟内过期
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

func newcosClient(c *conf.Config) *cosClient {
	b := &cos.BaseURL{
		BucketURL: util.Str2URL(c.COS.BucketURL), // 访问 bucket, object 相关 API 的基础 URL（不包含 path 部分）
	}
	client := cos.NewClient(b, mustNewCOSHTTPClient(c))
	return &cosClient{
		Conf:   c.COS,
		Client: client,
	}
}

func mustNewCOSHTTPClient(c *conf.Config) *http.Client {
	// 与全局单例http客户端采用不同transport，单独为cos服务创建新http客户端实例
	// 其余配置复用单例cli
	gCli := httpx.GetHttpClient()

	authTransport := &cos.AuthorizationTransport{
		SecretID:  c.COS.SecretID,
		SecretKey: c.COS.SecretKey,
		Transport: gCli.Client.Transport,
	}

	return &http.Client{
		Transport: authTransport,
	}
}
