package errno

import "github.com/xh-polaris/innospark-core-api/pkg/errorx/code"

const (
	AttachUploadErrCode = 80001
)

func init() {
	code.Register(
		AttachUploadErrCode,
		"附件上传失败",
		code.WithAffectStability(false),
	)
}
