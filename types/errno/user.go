package errno

import "github.com/xh-polaris/innospark-core-api/pkg/errorx/code"

const (
	UpdateUsernameErrCode  = 10001
	UsernameExistedErrCode = 10002
	UpdateAvatarErrCode    = 10003
)

func init() {
	code.Register(
		UpdateUsernameErrCode,
		"更新用户名失败",
		code.WithAffectStability(true),
	)
	code.Register(
		UpdateAvatarErrCode,
		"更新头像失败",
		code.WithAffectStability(true),
	)
	code.Register(
		UsernameExistedErrCode,
		"用户名 %s 已存在",
		code.WithAffectStability(false),
	)
}
