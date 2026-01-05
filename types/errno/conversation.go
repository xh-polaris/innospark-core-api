package errno

import (
	"github.com/xh-polaris/innospark-core-api/pkg/errorx/code"
)

const (
	ConversationCreateErrCode        = 30001
	ConversationRenameErrCode        = 30002
	ConversationListErrCode          = 30003
	ConversationGetErrCode           = 30004
	ConversationDeleteErrCode        = 30005
	ConversationSearchErrCode        = 30006
	ConversationGenerateBriefErrCode = 30007
	ConversationExtUpdateErrCode     = 30008
)

func init() {
	code.Register(
		ConversationCreateErrCode,
		"创建对话失败",
		code.WithAffectStability(false),
	)
	code.Register(
		ConversationRenameErrCode,
		"对话标题重命名失败",
		code.WithAffectStability(false),
	)
	code.Register(
		ConversationListErrCode,
		"分页获取历史对话失败",
		code.WithAffectStability(false),
	)
	code.Register(
		ConversationGetErrCode,
		"获取对话历史记录失败",
		code.WithAffectStability(false),
	)
	code.Register(
		ConversationDeleteErrCode,
		"删除历史记录失败",
		code.WithAffectStability(false),
	)
	code.Register(
		ConversationSearchErrCode,
		"搜索历史记录失败",
		code.WithAffectStability(false),
	)
	code.Register(
		ConversationGenerateBriefErrCode,
		"生成对话摘要失败",
		code.WithAffectStability(false),
	)
	code.Register(
		ConversationExtUpdateErrCode,
		"更新对话扩展信息失败",
		code.WithAffectStability(false),
	)
}
