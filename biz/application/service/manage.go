package service

import (
	"context"
	"time"

	"github.com/google/wire"
	"github.com/xh-polaris/innospark-core-api/biz/adaptor"
	"github.com/xh-polaris/innospark-core-api/biz/application/dto/manage"
	"github.com/xh-polaris/innospark-core-api/biz/infra/config"
	"github.com/xh-polaris/innospark-core-api/biz/infra/mapper/feedback"
	"github.com/xh-polaris/innospark-core-api/biz/infra/mapper/user"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util"
	"github.com/xh-polaris/innospark-core-api/pkg/errorx"
	"github.com/xh-polaris/innospark-core-api/types/errno"
)

type IManageService interface {
	AdminLogin(ctx context.Context, req *manage.AdminLoginReq) (resp *manage.AdminLoginResp, err error)
	ListUser(ctx context.Context, req *manage.ListUserReq) (resp *manage.ListUserResp, err error)
	Forbidden(ctx context.Context, req *manage.ForbiddenUserReq) (resp *manage.ForbiddenUserResp, err error)
	ListFeedback(ctx context.Context, req *manage.ListFeedBackReq) (resp *manage.ListFeedBackResp, err error)
}

type ManageService struct {
	UserMapper     user.MongoMapper
	FeedbackMapper feedback.MongoMapper
}

var ManageServiceSet = wire.NewSet(
	wire.Struct(new(ManageService), "*"),
	wire.Bind(new(IManageService), new(*ManageService)),
)

func (m *ManageService) AdminLogin(ctx context.Context, req *manage.AdminLoginReq) (resp *manage.AdminLoginResp, err error) {
	if req.Account != config.GetConfig().Admin.Account || req.Password != config.GetConfig().Admin.Password {
		return nil, errorx.New(errno.ErrLogin)
	}
	return &manage.AdminLoginResp{Resp: util.Success(), Token: config.GetConfig().Admin.Token}, nil
}

func (m *ManageService) ListUser(ctx context.Context, req *manage.ListUserReq) (resp *manage.ListUserResp, err error) {
	if err = checkAdmin(ctx); err != nil {
		return
	}
	total, us, err := m.UserMapper.ListUser(ctx, req.Page, req.Status, req.SortedBy, req.Reverse)
	if err != nil {
		return
	}
	var users []*manage.User
	for _, u := range us {
		var expire int64
		if !u.Expire.IsZero() {
			expire = u.Expire.Unix()
		}
		users = append(users, &manage.User{
			Id:         u.ID.Hex(),
			Phone:      u.Phone,
			Name:       u.Name,
			Avatar:     u.Avatar,
			Warnings:   u.Warnings,
			Status:     u.Status,
			Expire:     expire,
			LoginTime:  u.LoginTime.Unix(),
			CreateTime: u.CreateTime.Unix(),
			UpdateTime: u.UpdateTime.Unix(),
		})
	}
	return &manage.ListUserResp{
		Resp:  util.Success(),
		Total: total,
		User:  users,
	}, nil
}

func (m *ManageService) Forbidden(ctx context.Context, req *manage.ForbiddenUserReq) (resp *manage.ForbiddenUserResp, err error) {
	if err = checkAdmin(ctx); err != nil {
		return
	}
	if req.Status == user.StatusForbidden && req.Expire != nil {
		err = m.UserMapper.Forbidden(ctx, req.Id, time.Unix(*req.Expire, 0))
	} else if req.Status == user.StatusNormal {
		err = m.UserMapper.UnForbidden(ctx, req.Id)
	}
	if err != nil {
		return
	}
	return &manage.ForbiddenUserResp{Resp: util.Success()}, nil
}

func (m *ManageService) ListFeedback(ctx context.Context, req *manage.ListFeedBackReq) (resp *manage.ListFeedBackResp, err error) {
	if err = checkAdmin(ctx); err != nil {
		return
	}
	total, fbs, err := m.FeedbackMapper.ListFeedback(ctx, req.Page, req.MessageId, req.UserId, req.Action, req.Type)
	if err != nil {
		return
	}
	var feedbacks []*manage.ListFeedBackResp_FeedBack
	for _, fb := range fbs {
		feedbacks = append(feedbacks, &manage.ListFeedBackResp_FeedBack{
			MessageId:  fb.MessageId.Hex(),
			UserId:     fb.UserId.Hex(),
			Action:     fb.Action,
			Type:       fb.Type,
			Content:    fb.Content,
			CreateTime: fb.UpdateTime.Unix(),
		})
	}
	return &manage.ListFeedBackResp{
		Resp:      util.Success(),
		Feedbacks: feedbacks,
		Total:     total,
	}, nil
}

func checkAdmin(ctx context.Context) error {
	if c, err := adaptor.ExtractContext(ctx); err != nil {
		return err
	} else if string(c.GetHeader("Authorization")) != config.GetConfig().Admin.Token {
		return errorx.New(errno.UnAuthErrCode)
	}
	return nil
}
