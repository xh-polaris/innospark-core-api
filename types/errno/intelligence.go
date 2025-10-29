package errno

import "github.com/xh-polaris/innospark-core-api/pkg/errorx/code"

const (
	ErrListIntelligence = 200_000_001
	ErrGetIntelligence  = 200_000_002
)

func init() {
	code.Register(
		ErrListIntelligence,
		"获取智能体列表失败: {msg}",
		code.WithAffectStability(false),
	)
	code.Register(
		ErrGetIntelligence,
		"获取智能体失败: {msg}",
		code.WithAffectStability(false),
	)
}
