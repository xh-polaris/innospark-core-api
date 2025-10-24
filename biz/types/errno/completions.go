package errno

import "github.com/xh-polaris/innospark-core-api/biz/pkg/errorx/code"

const (
	CompletionsErrCode = 70001
)

func init() {
	code.Register(
		CompletionsErrCode,
		"处理AI文本生成请求失败",
		code.WithAffectStability(true),
	)
}
