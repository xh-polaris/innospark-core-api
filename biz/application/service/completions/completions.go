package completions

import (
	"context"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/xh-polaris/innospark-core-api/biz/adaptor"
	"github.com/xh-polaris/innospark-core-api/biz/application/dto/core_api"
	"github.com/xh-polaris/innospark-core-api/biz/domain/graph"
	"github.com/xh-polaris/innospark-core-api/biz/domain/state"
	"github.com/xh-polaris/innospark-core-api/biz/domain/state/info"
	"github.com/xh-polaris/innospark-core-api/biz/domain/state/interaction"
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
	CompletionGraph *graph.CompletionGraph
	UserMapper      user.MongoMapper
}

func (s *CompletionsService) Completions(c *app.RequestContext, ctx context.Context, req *core_api.CompletionsReq) (*adaptor.SSEStream, error) {
	// 鉴权
	uid, err := adaptor.ExtractUserId(ctx)
	if err != nil {
		logs.Error("extract user id error: %s", errorx.ErrorWithoutStack(err))
		return nil, errorx.WrapByCode(err, errno.UnAuthErrCode)
	}

	var (
		u         *user.User
		expire    time.Time
		forbidden bool
	)

	// 封禁判断
	if u, _, forbidden, expire, err = s.UserMapper.CheckForbidden(ctx, uid); err != nil {
		return nil, errorx.WrapByCode(err, errno.CompletionsErrCode)
	} else if forbidden { // 封禁中
		return nil, errorx.New(errno.ErrForbidden, errorx.KV("time", expire.Local().Format(time.RFC3339)))
	}

	// 暂时只支持一个新增对话
	if len(req.Messages) > 1 {
		return nil, errorx.New(errno.UnImplementErrCode)
	}

	// 检查用户输入是否有违禁词
	sensitive, hits := ac.AcSearch(req.Messages[0].Content, true, cst.SensitivePre)
	if sensitive {
		if err = s.UserMapper.Warn(ctx, uid); err != nil {
			logs.Errorf("warn err: %v", err)
		}
		return nil, errorx.New(errno.ErrSensitive, errorx.KV("text", strings.Join(hits, ",")))
	}

	oids, err := util.ObjectIDsFromHex(uid, req.ConversationId)
	if err != nil {
		return nil, err
	}

	inf := &info.Info{
		RequestContext: c,                       // 请求上下文
		SSE:            adaptor.NewSSEStream(c), // SSE流
		CompletionOptions: &info.CompletionOptions{ // 对话配置
			ReplyId:         req.ReplyId,                            // 回复ID
			IsRegen:         req.CompletionsOption.IsRegen,          // 重新生成用
			IsReplace:       req.CompletionsOption.IsReplace,        // 替换消息用户
			SelectedRegenId: req.CompletionsOption.SelectedRegenId}, // 确定重新生成用
		Profile: util.NilDefault(u.Profile, &user.Profile{Role: "未知"}),           // 个性化信息
		Ext:     util.NilDefault(req.CompletionsOption.Ext, map[string]string{}), // 额外信息(用于cotea模式)
		ModelInfo: &info.ModelInfo{
			Model:     req.Model,                            // 模型名称
			BotId:     req.BotId,                            // agent名称
			WebSearch: req.CompletionsOption.GetWebSearch(), // 是否搜索
			Thinking:  req.CompletionsOption.UseDeepThink},  // 是否深度思考
		MessageInfo:    &info.MessageInfo{}, // 消息信息
		ConversationId: oids[1],             // 对话id
		SectionId:      oids[1],             // 段id
		UserId:         oids[0],             // 用户id
		OriginMessage: &info.ReqMessage{ // 原始消息
			Content:     req.Messages[0].Content,                                          // 原始内容
			ContentType: req.Messages[0].ContentType,                                      // 原始消息类型
			Attaches:    req.Messages[0].Attaches, References: req.Messages[0].References, // 附件
		},
		Sensitive: &info.Sensitive{}, // 命中的敏感词
	}
	inf.Ext["query"] = req.Messages[0].Content // 将用户原始提问存入query中, 简化可能存在的提示词注入
	inf.Ext["role"] = inf.Profile.Role         // 用户角色

	st := &state.RelayContext{Info: inf, Interaction: interaction.NewInteraction(inf)} // 构建状态
	_, err = s.CompletionGraph.CompileAndStream(ctx, st)                               // 编译图
	return inf.SSE, errorx.WrapByCode(err, errno.CompletionsErrCode)
}
