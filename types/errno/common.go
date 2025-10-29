package errno

import (
	"github.com/xh-polaris/innospark-core-api/pkg/errorx/code"
)

const (
	UnAuthErrCode      = 1000
	UnImplementErrCode = 888
	OIDErrCode         = 777
	InterruptCode      = 666
	unknowCode         = 999
)

func init() {
	code.Register(
		UnAuthErrCode,
		"身份认证失败",
		code.WithAffectStability(false),
	)
	code.Register(
		UnImplementErrCode,
		"功能暂未实现",
		code.WithAffectStability(true),
	)
}
