package errno

import (
	"github.com/xh-polaris/innospark-core-api/pkg/errorx/code"
)

const (
	CompletionsErrCode = 70001
	ErrSensitive       = 700_000_002
)

func init() {
	code.Register(
		CompletionsErrCode,
		"对话生成失败",
		code.WithAffectStability(false),
	)
	code.Register(
		ErrSensitive,
		"输入 {text} 为违禁词, 请不要谈论敏感话题, 否则账号将遭到封禁",
		code.WithAffectStability(false))
}
