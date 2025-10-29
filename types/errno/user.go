package errno

import "github.com/xh-polaris/innospark-core-api/pkg/errorx/code"

const (
	ErrLogin           = 100_000_001
	ErrRegister        = 100_000_002
	ErrForbidden       = 100_000_003
	ErrUpdateUserField = 100_000_004
)

func init() {
	code.Register(
		ErrLogin,
		"登录失败",
		code.WithAffectStability(false),
	)
	code.Register(
		ErrRegister,
		"注册失败",
		code.WithAffectStability(false),
	)
	code.Register(
		ErrForbidden,
		"用户被封禁至 {time}",
		code.WithAffectStability(false))

	code.Register(
		ErrUpdateUserField,
		"更新用户字段失败",
		code.WithAffectStability(false),
	)
}
