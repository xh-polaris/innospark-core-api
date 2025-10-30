package util

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/xh-polaris/innospark-core-api/biz/application/dto/basic"
	"github.com/xh-polaris/innospark-core-api/biz/infra/config"
)

func DPrintf(format string, a ...interface{}) {
	if config.GetConfig().State == "debug" {
		fmt.Printf(format, a...)
	}
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
	if config.GetConfig().State == "debug" {
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
