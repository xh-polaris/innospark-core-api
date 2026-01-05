package util

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"reflect"
	"regexp"
	"strings"

	"github.com/xh-polaris/innospark-core-api/biz/application/dto/basic"
	"github.com/xh-polaris/innospark-core-api/biz/conf"
)

func DPrintf(format string, a ...interface{}) {
	if conf.GetConfig().State == "debug" {
		fmt.Printf(format, a...)
	}
}

func NilDefault[T any](v T, def T) T {
	if isNil(v) {
		return def
	}
	return v
}

func isNil(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Ptr, reflect.Map, reflect.Slice,
		reflect.Func, reflect.Chan, reflect.Interface:
		return rv.IsNil()
	default:
		return false
	}
}

func ZeroDefault[T comparable](v, def T) T {
	var zero T
	if v == zero {
		return def
	}
	return v
}

func Of[T any](v T) *T {
	return &v
}

func Deref[T any](v *T) T {
	var zero T
	if v == nil {
		return zero
	}
	return *v
}

// Success 返回成功的basic.Response指针
func Success() *basic.Response {
	return &basic.Response{
		Code: 0,
		Msg:  "success",
	}
}

func ToCDN(u, host string) string {
	// 解析 URL
	parsedURL, err := url.Parse(u)
	if err != nil {
		return ""
	}
	parsedURL.Host = host
	parsedURL.RawQuery = ""
	return parsedURL.String()
}

func NewDebugClient() *http.Client {
	if conf.GetConfig().State == "debug" {
		return &http.Client{
			Transport: NewLoggingTransport(),
		}
	}
	return http.DefaultClient
}

// LoggingTransport 是一个自定义 Transport，用于打印 HTTP 请求和响应
type LoggingTransport struct {
	Transport http.RoundTripper // 底层 Transport（默认使用 http.DefaultTransport）
}

func NewLoggingTransport() *LoggingTransport {
	return &LoggingTransport{
		Transport: http.DefaultTransport,
	}
}

// RoundTrip 实现 http.RoundTripper 接口，拦截请求和响应
func (t *LoggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// 打印请求
	dumpReq, err := httputil.DumpRequestOut(req, true) // true 表示包含 Body
	if err != nil {
		return nil, err
	}
	fmt.Println("===== HTTP Request =====")
	fmt.Println(string(dumpReq))
	fmt.Println("=======================")

	// 使用底层 Transport 发送请求
	resp, err := t.Transport.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	// 打印响应
	dumpResp, err := httputil.DumpResponse(resp, true) // true 表示包含 Body
	if err != nil {
		return nil, err
	}
	fmt.Println("===== HTTP Response =====")
	fmt.Println(string(dumpResp))
	fmt.Println("========================")

	return resp, nil
}

func Str2URL(raw string) *url.URL {
	if u, err := url.Parse(raw); err == nil {
		return u
	}
	return nil
}

// cOS2CDN COS源站URL转CDN URL 需在COS Console中配置自定义CDN域名和鉴权
// CDN上的文件为公有读 采用回源鉴权
func cOS2CDN(raw string) string {
	conf := conf.GetConfig().COS
	return strings.Replace(raw, conf.BucketURL, conf.CDN, 1)
}

func SignedCOS2CDN(raw string) string {
	// 预签名url去掉参数后即为
	return cOS2CDN(strings.Split(raw, "?")[0])
}

var imgExt = []string{".jpg", ".jpeg", ".png", ".webp"}

func IsImg(s string) bool {
	s = strings.ToLower(s)
	for _, ext := range imgExt {
		if strings.HasSuffix(s, ext) {
			return true
		}
	}
	return false
}

func PurifyJson(raw string) string {
	s := strings.TrimSpace(raw)

	// 去掉开头的 ``` 和可选语言标识
	reStart := regexp.MustCompile("^```\\w*")
	s = reStart.ReplaceAllString(s, "")
	s = strings.TrimSpace(s)

	// 去掉结尾的 ```
	reEnd := regexp.MustCompile("```$")
	s = reEnd.ReplaceAllString(s, "")
	s = strings.TrimSpace(s)

	return s
}
