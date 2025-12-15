package feedback

import (
	"context"
	"time"

	"github.com/xh-polaris/innospark-core-api/biz/adaptor"
	"github.com/xh-polaris/innospark-core-api/biz/application/dto/core_api"
	"github.com/xh-polaris/innospark-core-api/biz/domain/memory/history"
	mf "github.com/xh-polaris/innospark-core-api/biz/infra/mapper/feedback"
	mmsg "github.com/xh-polaris/innospark-core-api/biz/infra/mapper/message"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util"
	"github.com/xh-polaris/innospark-core-api/pkg/errorx"
	"github.com/xh-polaris/innospark-core-api/types/errno"
)

var FeedbackSVC *FeedbackService

type FeedbackService struct {
	MessageMapper  mmsg.MongoMapper
	FeedbackMapper mf.MongoMapper
	His            *history.HistoryManager
}

func (f *FeedbackService) Feedback(ctx context.Context, req *core_api.FeedbackReq) (*core_api.FeedbackResp, error) {
	// 鉴权
	uid, err := adaptor.ExtractUserId(ctx)
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.UnAuthErrCode)
	}
	ids, err := util.ObjectIDsFromHex(uid, req.MessageId)
	if err != nil {
		return nil, err
	}

	feedback := &mf.FeedBack{MessageId: ids[1], UserId: ids[0], Action: req.Action, UpdateTime: time.Now()}
	if req.Feedback != nil {
		feedback.Type, feedback.Content = req.Feedback.Type, req.Feedback.Content
	}
	// 更新反馈状态
	if err = f.FeedbackMapper.UpdateFeedback(ctx, feedback); err != nil {
		return nil, errorx.WrapByCode(err, errno.FeedbackErrCode)
	}
	// 更新消息状态
	if err = f.His.Feedback(ctx, ids[1], req.Action); err != nil {
		return nil, errorx.WrapByCode(err, errno.FeedbackErrCode)
	}
	return &core_api.FeedbackResp{Resp: util.Success()}, nil
}

func (f *FeedbackService) Content(ctx context.Context, req *core_api.FeedbackReq) (*core_api.FeedbackResp, error) {
	// 鉴权
	uid, err := adaptor.ExtractUserId(ctx)
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.UnAuthErrCode)
	}
	err = f.FeedbackMapper.Insert(ctx, uid, req.Action, req.GetFeedback().GetType(), req.GetFeedback().GetContent())
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.FeedbackErrCode)
	}
	return &core_api.FeedbackResp{Resp: util.Success()}, nil
}
