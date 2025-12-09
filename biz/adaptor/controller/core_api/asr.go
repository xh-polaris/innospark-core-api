package core_api

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/xh-polaris/innospark-core-api/biz/application/service/system"
	"github.com/xh-polaris/innospark-core-api/pkg/errorx"
	"github.com/xh-polaris/innospark-core-api/pkg/logs"
	"github.com/xh-polaris/innospark-core-api/pkg/wsx"
)

// ASR asr识别
// @router /asr [GET]
func ASR(ctx context.Context, c *app.RequestContext) {
	if err := wsx.UpgradeWs(ctx, c, system.ASR); err != nil {
		logs.Errorf("[controller] [Chat] websocket upgrade error: %s", errorx.ErrorWithoutStack(err))
	}
}
