package state

// 状态域, 承担跨域数据和各类基础域

import (
	"context"
	"sync"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/xh-polaris/innospark-core-api/biz/application/dto/core_api"
	"github.com/xh-polaris/innospark-core-api/biz/domain/state/event"
	"github.com/xh-polaris/innospark-core-api/biz/domain/state/info"
	"github.com/xh-polaris/innospark-core-api/biz/infra/mapper/user"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RelayContext 存储Completion接口过程中的上下文信息
type RelayContext struct {
	cancelOnce  sync.Once
	Info        *info.Info         // 信息
	EventStream *event.EventStream // 事件流
	CancelFunc  context.CancelFunc // 中断
}

func (st *RelayContext) Close() {
	st.EventStream.Close()
}

// Cancel 取消
func (st *RelayContext) Cancel() {
	st.cancelOnce.Do(func() {
		st.CancelFunc()
	})

}

func NewState(c *app.RequestContext, req *core_api.CompletionsReq, u *user.User, conversationId, sectionId primitive.ObjectID) *RelayContext {
	inf := info.NewInfo(c, req, u, conversationId, sectionId)
	st := &RelayContext{
		Info:        inf, // 信息
		EventStream: event.NewEventStream(),
	}
	return st
}
