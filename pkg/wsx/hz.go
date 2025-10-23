package wsx

import (
	"context"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/hertz-contrib/websocket"
)

// hertz框架中websocket协议服务端

// HzWSHandler 是ws协议处理函数
type HzWSHandler func(ctx context.Context, conn *websocket.Conn)

// HzUpgrader 默认配置的协议升级器, 用于将HTTP请求升级为WebSocket请求
// 没有跨域限制
var HzUpgrader = websocket.HertzUpgrader{
	CheckOrigin: func(ctx *app.RequestContext) bool {
		return true
	},
}

// UpgradeWs 将Http协议升级为WebSocket协议
func UpgradeWs(ctx context.Context, c *app.RequestContext, handler HzWSHandler) error {
	// 尝试升级协议, 处理请求
	if err := HzUpgrader.Upgrade(c, func(conn *websocket.Conn) { handler(ctx, conn) }); err != nil {
		return err
	}
	return nil
}
