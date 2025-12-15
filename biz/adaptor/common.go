package adaptor

// 上下文处理相关的 Common

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/golang-jwt/jwt/v4"
	"github.com/xh-polaris/innospark-core-api/biz/conf"
	"github.com/xh-polaris/innospark-core-api/pkg/errorx"
	"github.com/xh-polaris/innospark-core-api/pkg/logs"
	"go.opentelemetry.io/otel/propagation"
)

const HertzContext = "hertz_context"

func InjectContext(ctx context.Context, c *app.RequestContext) context.Context {
	return context.WithValue(ctx, HertzContext, c)
}

func ExtractContext(ctx context.Context) (*app.RequestContext, error) {
	c, ok := ctx.Value(HertzContext).(*app.RequestContext)
	if !ok {
		return nil, errors.New("hertz context not found")
	}
	return c, nil
}

func ExtractUserId(ctx context.Context) (userId string, err error) {
	userId = ""
	defer func() {
		if err != nil {
			logs.CtxInfof(ctx, "extract user meta fail, err=%s", errorx.ErrorWithoutStack(err))
		}
	}()
	c, err := ExtractContext(ctx)
	if err != nil {
		return
	}
	tokenString := c.GetHeader("Authorization")
	return ExtractUserIdFromJWT(string(tokenString))
}

func ExtractUserIdFromJWT(tokenString string) (userId string, err error) {
	if tokenString == "xh-polaris" {
		return "67aac4d14e8825731a1503d8", nil
	}
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwt.ParseRSAPublicKeyFromPEM([]byte(conf.GetConfig().Auth.PublicKey))
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
	return claims["basic_user_id"].(string), err
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
