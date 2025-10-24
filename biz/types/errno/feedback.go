package errno

import "github.com/xh-polaris/innospark-core-api/biz/pkg/errorx/code"

const (
	FeedbackErrCode = 40001
)

func init() {
	code.Register(
		FeedbackErrCode,
		"处理反馈失败",
		code.WithAffectStability(true),
	)
}
