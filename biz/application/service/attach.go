package service

import (
	"context"
	"io"

	"github.com/google/wire"
	"github.com/tencentyun/cos-go-sdk-v5"
	ccos "github.com/xh-polaris/innospark-core-api/biz/infra/cos"
)

type IAttachService interface {
	// TODO 修改idl 生成attach相关req和resp 获得idl脚本，注意路径，手动拷google目录
	Upload(ctx context.Context, key string, r io.Reader) (string, error) // 需改为req resp
	Download(ctx context.Context, key string) (io.ReadCloser, error)
}

type AttachService struct {
	CosInfra ccos.ICOSClient
}

var AttachServiceSet = wire.NewSet(
	wire.Struct(new(AttachService), "*"),
	wire.Bind(new(IAttachService), new(*AttachService)),
)

// Upload 后端返回预签名url给前端，由前端完成实际上传
func (s *AttachService) Upload(ctx context.Context, key string) (string, error) {
	// 将文件流存入ctx
	opt := &cos.PresignedURLOptions{}
	u, err := s.CosInfra.GetPresignURL(ctx, key, opt)
	if err != nil {
		return "", err
	}
	//http: //cos.sdsadaxsalkjdlka/dsaxxsax/dsa.png
	// 存储：使用cos域名
	// 后端dto处理：改为CDN域名 强制用https协议
	return u.String(), nil
}
