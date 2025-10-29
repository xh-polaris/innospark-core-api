package errno

import (
	"github.com/xh-polaris/innospark-core-api/pkg/errorx/code"
)

const (
	CompletionsErrCode = 70001
)

func init() {
	code.Register(
		CompletionsErrCode,
		"对话生成失败",
		code.WithAffectStability(false),
	)
}
