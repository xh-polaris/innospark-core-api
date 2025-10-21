package service

import (
	"context"
	"time"

	"github.com/google/wire"
	"github.com/xh-polaris/innospark-core-api/biz/adaptor"
	"github.com/xh-polaris/innospark-core-api/biz/application/dto/core_api"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	mf "github.com/xh-polaris/innospark-core-api/biz/infra/mapper/feedback"
	mmsg "github.com/xh-polaris/innospark-core-api/biz/infra/mapper/message"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util/errorx"
)

type IFeedbackService interface {
	Feedback(ctx context.Context, req *core_api.FeedbackReq) (*core_api.FeedbackResp, error)
}

type FeedbackService struct {
	MessageMapper  mmsg.MongoMapper
	FeedbackMapper mf.MongoMapper
}

var FeedbackServiceSet = wire.NewSet(
	wire.Struct(new(FeedbackService), "*"),
	wire.Bind(new(IFeedbackService), new(*FeedbackService)),
)

func (f *FeedbackService) Feedback(ctx context.Context, req *core_api.FeedbackReq) (*core_api.FeedbackResp, error) {
	// 鉴权
	uid, err := adaptor.ExtractUserId(ctx)
	if err != nil {
		return nil, errorx.WrapByCode(err, cst.UnAuthErrCode)
	}
	ids, err := util.ObjectIDsFromHex(uid, req.MessageId)
	if err != nil {
		return nil, errorx.WrapByCode(err, cst.OIDErrCode)
	}

	feedback := &mf.FeedBack{MessageId: ids[1], UserId: ids[0], Action: req.Action, UpdateTime: time.Now()}
	if req.Feedback != nil {
		feedback.Type, feedback.Content = req.Feedback.Type, req.Feedback.Content
	}
	// 更新反馈状态
	if err = f.FeedbackMapper.UpdateFeedback(ctx, feedback); err != nil {
		return nil, errorx.WrapByCode(err, cst.FeedbackErrCode)
	}
	// 更新消息状态
	if err = f.MessageMapper.Feedback(ctx, ids[1], req.Action); err != nil {
		return nil, errorx.WrapByCode(err, cst.FeedbackErrCode)
	}
	return &core_api.FeedbackResp{Resp: util.Success()}, nil
}
