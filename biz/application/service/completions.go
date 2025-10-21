package service

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/google/wire"
	"github.com/xh-polaris/innospark-core-api/biz/adaptor"
	"github.com/xh-polaris/innospark-core-api/biz/application/dto/core_api"
	"github.com/xh-polaris/innospark-core-api/biz/domain/graph"
	"github.com/xh-polaris/innospark-core-api/biz/domain/info"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util/errorx"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util/logx"
)

type ICompletionsService interface {
	Completions(c *app.RequestContext, ctx context.Context, req *core_api.CompletionsReq) (any, error)
}

type CompletionsService struct {
	CompletionGraph *graph.CompletionGraph
}

var CompletionsServiceSet = wire.NewSet(
	wire.Struct(new(CompletionsService), "*"),
	wire.Bind(new(ICompletionsService), new(*CompletionsService)),
)

func (s *CompletionsService) Completions(c *app.RequestContext, ctx context.Context, req *core_api.CompletionsReq) (any, error) {
	// 鉴权
	uid, err := adaptor.ExtractUserId(ctx)
	if err != nil {
		logx.Error("extract user id error: %v", err)
		return nil, errorx.WrapByCode(err, cst.UnAuthErrCode)
	}

	// 暂时只支持一个新增对话
	if len(req.Messages) > 1 {
		return nil, errorx.New(cst.UnImplementErrCode)
	}

	// 构建RelayContext
	oids, err := util.ObjectIDsFromHex(uid, req.ConversationId)
	if err != nil {
		return nil, errorx.New(cst.UnImplementErrCode)
	}
	state := &info.RelayContext{
		RequestContext: c,
		CompletionOptions: &info.CompletionOptions{
			ReplyId:         req.ReplyId,
			IsRegen:         req.CompletionsOption.IsRegen,
			IsReplace:       req.CompletionsOption.IsReplace,
			SelectedRegenId: req.CompletionsOption.SelectedRegenId},
		ModelInfo:      &info.ModelInfo{Model: req.Model, BotId: req.BotId, WebSearch: req.CompletionsOption.GetWebSearch()},
		MessageInfo:    &info.MessageInfo{},
		ConversationId: oids[1],
		SectionId:      oids[1],
		UserId:         oids[0],
		OriginMessage: &info.ReqMessage{
			Content: req.Messages[0].Content, ContentType: req.Messages[0].ContentType,
			Attaches: req.Messages[0].Attaches, References: req.Messages[0].References,
		},
	}

	_, err = s.CompletionGraph.CompileAndStream(ctx, state)
	return nil, errorx.WrapByCode(err, cst.CompletionsErrCode)
}
