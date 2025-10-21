package service

import (
	"context"
	"net/http"

	"github.com/google/wire"
	"github.com/xh-polaris/innospark-core-api/biz/adaptor"
	"github.com/xh-polaris/innospark-core-api/biz/application/dto/basic"
	"github.com/xh-polaris/innospark-core-api/biz/application/dto/core_api"
	"github.com/xh-polaris/innospark-core-api/biz/infra/config"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util/errorx"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util/httpx"
)

type IUserService interface {
	SendVerifyCode(ctx context.Context, req *core_api.SendVerifyCodeReq) (*core_api.SendVerifyCodeResp, error)
	Register(ctx context.Context, req *core_api.BasicUserRegisterReq) (*core_api.BasicUserRegisterResp, error)
	Login(ctx context.Context, req *core_api.BasicUserLoginReq) (*core_api.BasicUserLoginResp, error)
	ResetPassword(ctx context.Context, req *core_api.BasicUserResetPasswordReq) (*core_api.BasicUserResetPasswordResp, error)
	ThirdPartyLogin(ctx context.Context, req *core_api.ThirdPartyLoginReq) (*core_api.ThirdPartyLoginResp, error)
}

type UserService struct {
}

var UserServiceSet = wire.NewSet(
	wire.Struct(new(UserService), "*"),
	wire.Bind(new(IUserService), new(*UserService)),
)

func (u *UserService) SendVerifyCode(ctx context.Context, req *core_api.SendVerifyCodeReq) (*core_api.SendVerifyCodeResp, error) {
	c := config.GetConfig()
	header := http.Header{}
	header.Set("content-type", "application/json")
	if c.State != "test" {
		header.Set("X-Xh-Env", "test")
	}
	body := map[string]any{
		"authType": req.AuthType,
		"authId":   req.AuthId,
		"expire":   300,
		"cause":    "passport",
		"app":      map[string]any{"name": "InnoSpark"},
	}

	url := config.GetConfig().SynapseURL + "/system/send_verify_code"
	resp, err := httpx.GetHttpClient().Post(url, header, body)
	if err != nil {
		return nil, errorx.WrapByCode(err, cst.SynapseErrCode, errorx.KV("url", url))
	}
	if resp["code"].(float64) != 0 {
		return &core_api.SendVerifyCodeResp{
			Resp: &basic.Response{
				Code: int32(resp["code"].(float64)),
				Msg:  resp["msg"].(string),
			},
		}, nil
	}
	return &core_api.SendVerifyCodeResp{
		Resp: util.Success(),
	}, nil
}

func (u *UserService) Register(ctx context.Context, req *core_api.BasicUserRegisterReq) (*core_api.BasicUserRegisterResp, error) {
	c := config.GetConfig()
	header := http.Header{}
	header.Set("content-type", "application/json")
	if c.State != "test" {
		header.Set("X-Xh-Env", "test")
	}
	body := map[string]any{
		"authType": req.AuthType,
		"authId":   req.AuthId,
		"verify":   req.Verify,
		"password": req.Password,
		"app":      map[string]any{"name": "InnoSpark"},
	}

	url := config.GetConfig().SynapseURL + "/basic_user/register"
	resp, err := httpx.GetHttpClient().Post(url, header, body)
	if err != nil {
		return nil, errorx.WrapByCode(err, cst.SynapseErrCode, errorx.KV("url", url))
	}
	if resp["code"].(float64) != 0 {
		return &core_api.BasicUserRegisterResp{
			Resp: &basic.Response{
				Code: int32(resp["code"].(float64)),
				Msg:  resp["msg"].(string),
			},
		}, nil
	}
	return &core_api.BasicUserRegisterResp{
		Resp:  util.Success(),
		Token: resp["token"].(string),
	}, nil
}

func (u *UserService) Login(ctx context.Context, req *core_api.BasicUserLoginReq) (*core_api.BasicUserLoginResp, error) {
	c := config.GetConfig()
	header := http.Header{}
	header.Set("content-type", "application/json")
	if c.State != "test" {
		header.Set("X-Xh-Env", "test")
	}
	body := map[string]any{
		"authType": req.AuthType,
		"authId":   req.AuthId,
		"verify":   req.Verify,
		"app":      map[string]any{"name": "InnoSpark"},
	}

	url := config.GetConfig().SynapseURL + "/basic_user/login"
	resp, err := httpx.GetHttpClient().Post(url, header, body)
	if err != nil {
		return nil, errorx.WrapByCode(err, cst.SynapseErrCode, errorx.KV("url", url))
	}
	if resp["code"].(float64) != 0 {
		return &core_api.BasicUserLoginResp{
			Resp: &basic.Response{
				Code: int32(resp["code"].(float64)),
				Msg:  resp["msg"].(string),
			},
		}, nil
	}
	return &core_api.BasicUserLoginResp{
		Resp:  util.Success(),
		Token: resp["token"].(string),
		New:   resp["new"].(bool),
	}, nil
}

func (u *UserService) ResetPassword(ctx context.Context, req *core_api.BasicUserResetPasswordReq) (*core_api.BasicUserResetPasswordResp, error) {
	c := config.GetConfig()
	rc, err := adaptor.ExtractContext(ctx)
	if err != nil {
		return nil, errorx.WrapByCode(err, cst.UnAuthErrCode)
	}
	header := http.Header{}
	header.Set("content-type", "application/json")
	header.Set("Authorization", string(rc.GetHeader("Authorization")))
	if c.State != "test" {
		header.Set("X-Xh-Env", "test")
	}
	body := map[string]any{
		"newPassword": req.NewPassword,
		"app":         map[string]any{"name": "InnoSpark"},
	}

	url := config.GetConfig().SynapseURL + "/basic_user/reset_password"
	resp, err := httpx.GetHttpClient().Post(url, header, body)
	if err != nil {
		return nil, errorx.WrapByCode(err, cst.SynapseErrCode, errorx.KV("url", url))
	}
	if resp["code"].(float64) != 0 {
		return &core_api.BasicUserResetPasswordResp{
			Resp: &basic.Response{
				Code: int32(resp["code"].(float64)),
				Msg:  resp["msg"].(string),
			},
		}, nil
	}
	return &core_api.BasicUserResetPasswordResp{
		Resp: util.Success(),
	}, nil
}

func (u *UserService) ThirdPartyLogin(ctx context.Context, req *core_api.ThirdPartyLoginReq) (*core_api.ThirdPartyLoginResp, error) {
	c := config.GetConfig()
	header := http.Header{}
	header.Set("content-type", "application/json")
	if c.State != "test" {
		header.Set("X-Xh-Env", "test")
	}
	body := map[string]any{
		"thirdparty": req.Thirdparty,
		"ticket":     req.Ticket,
	}

	url := config.GetConfig().SynapseURL + "/thirdparty/login"
	resp, err := httpx.GetHttpClient().Post(url, header, body)
	if err != nil {
		return nil, errorx.WrapByCode(err, cst.SynapseErrCode, errorx.KV("url", url))
	}
	if resp["code"].(float64) != 0 {
		return &core_api.ThirdPartyLoginResp{
			Resp: &basic.Response{
				Code: int32(resp["code"].(float64)),
				Msg:  resp["msg"].(string),
			},
		}, nil
	}
	return &core_api.ThirdPartyLoginResp{
		Resp:  util.Success(),
		Token: resp["token"].(string),
		New:   false,
	}, nil
}
