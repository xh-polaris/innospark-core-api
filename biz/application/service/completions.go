package service

import (
	"context"

	"github.com/google/wire"
	"github.com/xh-polaris/innospark-core-api/biz/adaptor"
	"github.com/xh-polaris/innospark-core-api/biz/application/dto/core_api"
	_ "github.com/xh-polaris/innospark-core-api/biz/domain/deyu"
	_ "github.com/xh-polaris/innospark-core-api/biz/domain/innospark"
	"github.com/xh-polaris/innospark-core-api/biz/domain/model"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util/logx"
)

type ICompletionsService interface {
	Completions(ctx context.Context, req *core_api.CompletionsReq) (any, error)
}

type CompletionsService struct {
	MsgMaMsgDomain   *model.MessageDomain
	CompletionDomain *model.CompletionDomain
}

var CompletionsServiceSet = wire.NewSet(
	wire.Struct(new(CompletionsService), "*"),
	wire.Bind(new(ICompletionsService), new(*CompletionsService)),
)

func (s *CompletionsService) Completions(ctx context.Context, req *core_api.CompletionsReq) (any, error) {
	// 鉴权
	uid, err := adaptor.ExtractUserId(ctx)
	if err != nil {
		logx.Error("extract user id error: %v", err)
		return nil, cst.UnAuthErr
	}

	// 暂时只支持一个新增对话
	if len(req.Messages) > 1 {
		return nil, cst.UnImplementErr
	}

	// 构建聊天记录和info
	ctx, messages, info, err := s.MsgMaMsgDomain.GetMessagesAndInjectContext(ctx, uid, req)
	if err != nil {
		return nil, err
	}

	// 进行对话, 在最后更新历史记录
	return s.CompletionDomain.Completion(ctx, uid, req, messages, info)
}
