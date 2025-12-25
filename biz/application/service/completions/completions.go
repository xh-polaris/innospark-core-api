package completions

import (
	"context"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/xh-polaris/innospark-core-api/biz/adaptor"
	"github.com/xh-polaris/innospark-core-api/biz/application/dto/core_api"
	"github.com/xh-polaris/innospark-core-api/biz/domain/flow"
	"github.com/xh-polaris/innospark-core-api/biz/domain/memory"
	"github.com/xh-polaris/innospark-core-api/biz/domain/state"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	"github.com/xh-polaris/innospark-core-api/biz/infra/mapper/user"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util"
	"github.com/xh-polaris/innospark-core-api/pkg/ac"
	"github.com/xh-polaris/innospark-core-api/pkg/errorx"
	"github.com/xh-polaris/innospark-core-api/pkg/logs"
	"github.com/xh-polaris/innospark-core-api/types/errno"
)

var CompletionsSVC *CompletionsService

type CompletionsService struct {
	Memory     *memory.MemoryManager
	UserMapper user.MongoMapper
}

func (s *CompletionsService) Completions(c *app.RequestContext, ctx context.Context, req *core_api.CompletionsReq) error {
	// 鉴权
	uid, err := adaptor.ExtractUserId(ctx)
	if err != nil {
		logs.Error("extract user id error: %s", errorx.ErrorWithoutStack(err))
		return errorx.WrapByCode(err, errno.UnAuthErrCode)
	}

	var (
		u         *user.User
		expire    time.Time
		forbidden bool
	)

	// 封禁判断
	if u, _, forbidden, expire, err = s.UserMapper.CheckForbidden(ctx, uid); err != nil {
		return errorx.WrapByCode(err, errno.CompletionsErrCode)
	} else if forbidden { // 封禁中
		return errorx.New(errno.ErrForbidden, errorx.KV("time", expire.Local().Format(time.RFC3339)))
	}

	// 暂时只支持一个新增对话
	if len(req.Messages) > 1 {
		return errorx.New(errno.UnImplementErrCode)
	}

	// 检查用户输入是否有违禁词
	sensitive, hits := ac.AcSearch(req.Messages[0].Content, true, cst.SensitivePre)
	if sensitive {
		if err = s.UserMapper.Warn(ctx, uid); err != nil {
			logs.Errorf("warn err: %v", err)
		}
		return errorx.New(errno.ErrSensitive, errorx.KV("text", strings.Join(hits, ",")))
	}

	// 构建对话状态
	oids, err := util.ObjectIDsFromHex(req.ConversationId)
	if err != nil {
		return err
	}
	st := state.NewState(c, req, u, oids[0], oids[0])

	return flow.DoCompletions(ctx, st, s.Memory)
}
