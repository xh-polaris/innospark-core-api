package service

import (
	"context"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/google/wire"
	"github.com/xh-polaris/innospark-core-api/biz/adaptor"
	"github.com/xh-polaris/innospark-core-api/biz/application/dto/core_api"
	"github.com/xh-polaris/innospark-core-api/biz/domain/graph"
	"github.com/xh-polaris/innospark-core-api/biz/domain/info"
	"github.com/xh-polaris/innospark-core-api/biz/infra/mapper/user"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util"
	"github.com/xh-polaris/innospark-core-api/pkg/ac"
	"github.com/xh-polaris/innospark-core-api/pkg/errorx"
	"github.com/xh-polaris/innospark-core-api/pkg/logs"
	"github.com/xh-polaris/innospark-core-api/types/errno"
)

type ICompletionsService interface {
	Completions(c *app.RequestContext, ctx context.Context, req *core_api.CompletionsReq) (any, error)
}

type CompletionsService struct {
	CompletionGraph *graph.CompletionGraph
	UserMapper      user.MongoMapper
}

var CompletionsServiceSet = wire.NewSet(
	wire.Struct(new(CompletionsService), "*"),
	wire.Bind(new(ICompletionsService), new(*CompletionsService)),
)

func (s *CompletionsService) Completions(c *app.RequestContext, ctx context.Context, req *core_api.CompletionsReq) (any, error) {
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

	// 暂时只支持一个新增对话
	if len(req.Messages) > 1 {
		return nil, errorx.New(errno.UnImplementErrCode)
	}

	// 检查用户输入是否有违禁词
	sensitive, hits := ac.AcSearch(req.Messages[0].Content, true)
	if sensitive {
		if err = s.UserMapper.Warn(ctx, uid); err != nil {
			logs.Errorf("warn err: %v", err)
		}
		return nil, errorx.New(errno.ErrSensitive, errorx.KV("text", strings.Join(hits, ",")))
	}

	// 构建RelayContext
	oids, err := util.ObjectIDsFromHex(uid, req.ConversationId)
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.UnAuthErrCode)
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
		Sensitive: &info.Sensitive{},
	}

	_, err = s.CompletionGraph.CompileAndStream(ctx, state)
	return nil, errorx.WrapByCode(err, errno.CompletionsErrCode)
}
