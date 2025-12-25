package adaptor

// HTTP 响应相关

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	hertz "github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/xh-polaris/gopkg/util"
	"github.com/xh-polaris/innospark-core-api/pkg/errorx"
	"github.com/xh-polaris/innospark-core-api/pkg/logs"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel/trace"
)

type data struct {
	Code int32  `json:"code"`
	Msg  string `json:"msg"`
}

// PostProcess 处理http响应, resp要求指针或接口类型
// 在日志中记录本次调用详情, 同时向响应头中注入符合b3规范的链路信息, 主要是trace_id
// 最佳实践:
// - 在controller中调用业务处理, 处理结束后调用PostProcess
func PostProcess(ctx context.Context, c *app.RequestContext, req, resp any, err error) {
	b3.New().Inject(ctx, &headerProvider{headers: &c.Response.Header})
	logs.CtxInfof(ctx, "[%s] req=%s, resp=%s, err=%s, trace=%s", c.Path(), util.JSONF(req), util.JSONF(resp), errorx.ErrorWithoutStack(err), trace.SpanContextFromContext(ctx).TraceID().String())

	// 无错, 正常响应
	if err == nil {
		c.JSON(hertz.StatusOK, makeResponse(resp))
		return
	}
	PostError(ctx, c, err)
}

// PostError 处理错误
func PostError(ctx context.Context, c *app.RequestContext, err error) {
	var customErr errorx.StatusError
	if errors.As(err, &customErr) && customErr.Code() != 0 {
		logs.CtxWarnf(ctx, "[ErrorX] error:  %v %v \n", customErr.Code(), err)
		c.AbortWithStatusJSON(http.StatusOK, data{Code: customErr.Code(), Msg: customErr.Msg()})
		return
	} else { // 常规错误, 状态码500
		logs.CtxErrorf(ctx, "internal error, err=%s", errorx.ErrorWithoutStack(err))
		code := hertz.StatusInternalServerError
		c.String(code, err.Error())
		return
	}
}

// makeResponse 通过反射构造嵌套格式的响应体
func makeResponse(resp any) map[string]any {
	if resp == nil {
		return nil
	}
	v := reflect.ValueOf(resp)
	if v.IsZero() || v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return nil
	}
	// 构建返回数据
	v = v.Elem()
	r := v.FieldByName("Resp").Elem()
	response := map[string]any{"code": r.FieldByName("Code").Int(), "msg": r.FieldByName("Msg").String()}

	data := make(map[string]any)
	for i := 0; i < v.NumField(); i++ {
		field := v.Type().Field(i)
		if jsonTag := field.Tag.Get("json"); jsonTag != "" && field.Name != "Resp" {
			if fieldValue := v.Field(i).Interface(); !reflect.ValueOf(fieldValue).IsZero() || !strings.Contains(jsonTag, "omitempty") {
				data[jsonTag] = fieldValue
			}
		}
	}
	if len(data) > 0 {
		response["data"] = data
	}
	return response
}
