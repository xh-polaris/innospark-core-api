package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/wire"
	"github.com/xh-polaris/innospark-core-api/biz/adaptor"
	"github.com/xh-polaris/innospark-core-api/biz/application/dto/core_api"
	"github.com/xh-polaris/innospark-core-api/biz/infra/mapper/user"
	"github.com/xh-polaris/innospark-core-api/biz/infra/storage"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util"
	"github.com/xh-polaris/innospark-core-api/pkg/errorx"
	"github.com/xh-polaris/innospark-core-api/pkg/logs"
	"github.com/xh-polaris/innospark-core-api/types/errno"
)

type IAttachService interface {
	GenSignedURL(ctx context.Context, req *core_api.GenSignedURLReq) (*core_api.GenSignedURLResp, error)
}

type AttachService struct {
	CosInfra   storage.COSInfra
	UserMapper user.MongoMapper
}

var AttachServiceSet = wire.NewSet(
	wire.Struct(new(AttachService), "*"),
	wire.Bind(new(IAttachService), new(*AttachService)),
)

// GenSignedURL 后端返回预签名url给前端，由前端完成实际上传
func (s *AttachService) GenSignedURL(ctx context.Context, req *core_api.GenSignedURLReq) (*core_api.GenSignedURLResp, error) {
	// 鉴权
	uid, err := adaptor.ExtractUserId(ctx)
	if err != nil {
		logs.Error("extract user id error: %s", errorx.ErrorWithoutStack(err))
		return nil, errorx.WrapByCode(err, errno.UnAuthErrCode)
	}
	if _, forbidden, expire, err := s.UserMapper.CheckForbidden(ctx, uid); err != nil {
		return nil, errorx.WrapByCode(err, errno.CompletionsErrCode)
	} else if forbidden { // 封禁中
		return nil, errorx.New(errno.ErrForbidden, errorx.KV("time", expire.Local().Format(time.RFC3339)))
	}

	// 使用 userID/prefix/时间戳 作为对象键
	key := strings.Join([]string{uid, req.GetPrefix(), time.Now().String()}, "/")

	signedURL, err := s.CosInfra.GenPresignURL(ctx, key, nil)
	if err != nil || signedURL == "" {
		logs.Error("get signedURL error: %s", errorx.ErrorWithoutStack(err))
		return nil, errorx.WrapByCode(err, errno.AttachUploadErrCode)
	}
	// 获取永久accessURL
	accessURL := util.SignedCOS2CDN(signedURL)

	return &core_api.GenSignedURLResp{
		Resp:         util.Success(),
		PresignedURL: signedURL,
		AccessURL:    accessURL,
	}, nil
}
