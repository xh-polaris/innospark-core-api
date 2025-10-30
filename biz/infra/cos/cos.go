package cos

import (
	"context"
	"io"
	"net/url"
	"time"

	"github.com/tencentyun/cos-go-sdk-v5"
)

type ICOSClient interface {
	Upload(ctx context.Context, key string, r io.Reader, opt *cos.ObjectPutOptions) (*cos.Response, error)
	GetPresignURL(ctx context.Context, key string, opt *cos.PresignedURLOptions) (*url.URL, error)
	GetObjMeta(ctx context.Context, key string, opt *cos.ObjectHeadOptions) (*cos.Response, error)
}

// Upload 管理员上传对象
// key 对象键 应为/{user_id}/{conversation_id}/时间戳
// opt 上传配置，包括缓存策略等 允许为空
// 返回accessURL
func (c *Client) Upload(ctx context.Context, key string, r io.Reader, opt *cos.ObjectPutOptions) (*cos.Response, error) {
	//
	if opt == nil {
		opt = &cos.ObjectPutOptions{
			ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{
				ContentType: "text/html",
			},
		}
	}

	resp, err := c.Client.Object.Put(ctx, key, r, opt)
	if err != nil {
		// TODO err处理 & log
		return nil, err
	}
	resp.Header.Get("x-cos-hash-crc64ecma") // CRC64校验值
	return resp, nil
}

func (c *Client) GetPresignURL(ctx context.Context, key string, opt *cos.PresignedURLOptions) (*url.URL, error) {
	u, err := c.Client.Object.GetPresignedURL(ctx, "PUT", key,
		c.Conf.SecretID,
		c.Conf.SecretKey,
		time.Second,
		opt,
	)
	if err != nil || u == nil {
		return nil, err
	}
	return u, nil
}

func (c *Client) GetObjMeta(ctx context.Context, key string, opt *cos.ObjectHeadOptions) (*cos.Response, error) {
	resp, err := c.Client.Object.Head(ctx, key, opt)
	if err != nil {
		// TODO err处理 & log
		return nil, err
	}
	return resp, nil
}
