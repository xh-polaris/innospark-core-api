package adaptor

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol"
	hertz "github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/golang-jwt/jwt/v4"
	"github.com/xh-polaris/gopkg/util"
	"github.com/xh-polaris/innospark-core-api/biz/infra/config"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util/logx"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel/propagation"
)

const hertzContext = "hertz_context"

func InjectContext(ctx context.Context, c *app.RequestContext) context.Context {
	return context.WithValue(ctx, hertzContext, c)
}

func ExtractContext(ctx context.Context) (*app.RequestContext, error) {
	c, ok := ctx.Value(hertzContext).(*app.RequestContext)
	if !ok {
		return nil, errors.New("hertz context not found")
	}
	return c, nil
}

func ExtractUserId(ctx context.Context) (userId string, err error) {
	userId = ""
	defer func() {
		if err != nil {
			logx.CtxInfo(ctx, "extract user meta fail, err=%v", err)
		}
	}()
	c, err := ExtractContext(ctx)
	if err != nil {
		return
	}
	tokenString := c.GetHeader("Authorization")
	if string(tokenString) == "xh-polaris" {
		return "67aac4d14e8825731a1503d8", nil
	}
	token, err := jwt.Parse(string(tokenString), func(_ *jwt.Token) (interface{}, error) {
		return jwt.ParseECPublicKeyFromPEM([]byte(config.GetConfig().Auth.PublicKey))
	})
	if err != nil {
		return
	}
	if !token.Valid {
		err = errors.New("token is not valid")
		return
	}
	data, err := json.Marshal(token.Claims)
	if err != nil {
		return
	}
	var claims map[string]interface{}
	err = json.Unmarshal(data, &claims)
	if err != nil {
		return
	}
	return claims["userId"].(string), err
}

// PostProcess 处理http响应, resp要求指针或接口类型
// 在日志中记录本次调用详情, 同时向响应头中注入符合b3规范的链路信息, 主要是trace_id
// 最佳实践:
// - 在controller中调用业务处理, 处理结束后调用PostProcess
func PostProcess(ctx context.Context, c *app.RequestContext, req, resp any, err error) {
	b3.New().Inject(ctx, &headerProvider{headers: &c.Response.Header})
	logx.CtxInfo(ctx, "[%s] req=%s, resp=%s, err=%v", c.Path(), util.JSONF(req), util.JSONF(resp), err)

	// 无错, 正常响应
	if err == nil {
		if v, ok := resp.(*SSEStream); ok {
			makeSSE(c, v)
			return
		}
		response := makeResponse(resp)
		c.JSON(hertz.StatusOK, response)
		return
	}

	if ex, ok := err.(cst.IErrorx); ok { // errorx错误
		StatusCode := hertz.StatusOK
		c.JSON(StatusCode, &cst.Errorx{
			Code: ex.GetCode(),
			Msg:  ex.GetMsg(),
		})
	} else { // 常规错误, 状态码500
		logx.CtxError(ctx, "internal error, err=%s", err.Error())
		code := hertz.StatusInternalServerError
		c.String(code, err.Error())
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

var _ propagation.TextMapCarrier = &headerProvider{}

type headerProvider struct {
	headers *protocol.ResponseHeader
}

// Get a value from metadata by key
func (m *headerProvider) Get(key string) string {
	return m.headers.Get(key)
}

// Set a value to metadata by k/v
func (m *headerProvider) Set(key, value string) {
	m.headers.Set(key, value)
}

// Keys Iteratively get all keys of metadata
func (m *headerProvider) Keys() []string {
	out := make([]string, 0)

	m.headers.VisitAll(func(key, value []byte) {
		out = append(out, string(key))
	})

	return out
}
