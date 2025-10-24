package errno

import "github.com/xh-polaris/innospark-core-api/biz/pkg/errorx/code"

const (
	SynapseErrCode = 60001
	CozeErrCode    = 60002
)

func init() {
	code.Register(
		SynapseErrCode,
		"使用 Synapse 访问 {url} 错误",
		code.WithAffectStability(true),
	)
	code.Register(
		CozeErrCode,
		"使用 Coze 访问 {url} 错误",
		code.WithAffectStability(true),
	)
}
