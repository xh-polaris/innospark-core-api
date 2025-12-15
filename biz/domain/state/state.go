package state

// 状态域, 承担跨域数据和各类基础域

import (
	"github.com/xh-polaris/innospark-core-api/biz/domain/state/info"
	"github.com/xh-polaris/innospark-core-api/biz/domain/state/interaction"
)

// RelayContext 存储Completion接口过程中的上下文信息
type RelayContext struct {
	Info        *info.Info               // 信息
	Interaction *interaction.Interaction // 交互
}
