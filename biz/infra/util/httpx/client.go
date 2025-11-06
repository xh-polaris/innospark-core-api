package httpx

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/cloudwego/hertz/pkg/common/json"
	"github.com/xh-polaris/innospark-core-api/pkg/errorx"
	"github.com/xh-polaris/innospark-core-api/pkg/logs"
)

// httpx/client 是一个简单的http客户端
// 支持流式与非流式请求, 通过StreamReader包装流式请求的响应

var (
	client *HttpClient
	once   sync.Once
)

const (
	GET  = "GET"
	POST = "POST"
)

// HttpClient 是一个简单的 HTTP 客户端
type HttpClient struct {
	Client *http.Client
}

// NewHttpClient 单例模式维护一个client
func NewHttpClient() *HttpClient {
	once.Do(func() {
		client = &HttpClient{
			Client: http.DefaultClient,
		}
	})
	return client
}

func GetHttpClient() *HttpClient {
	return NewHttpClient()
}

// do 发送请求
func (c *HttpClient) do(ctx context.Context, method, url string, headers http.Header, body any) (resp *http.Response, err error) {
	// 序列化 body 为 JSON
	var bodyBytes []byte
	var req *http.Request
	if bodyBytes, err = json.Marshal(body); err != nil {
		return nil, fmt.Errorf("[httpx]请求体序列化失败: %w", err)
	}
	// 创建新的请求
	if req, err = http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(bodyBytes)); err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	// 设置请求头
	for k, vv := range headers {
		req.Header[k] = vv
	}
	// 发送请求
	return c.Client.Do(req)
}

func (c *HttpClient) ReqWithHeader(ctx context.Context, method, url string, headers http.Header, body any) (header http.Header, resp map[string]any, err error) {
	// 读取响应体
	var _resp []byte
	if header, _resp, err = c.getResp(ctx, method, url, headers, body); err != nil {
		return
	}
	// 反序列化响应体
	if err = json.Unmarshal(_resp, &resp); err != nil {
		return header, nil, fmt.Errorf("反序列化响应失败: %w", err)
	}
	return header, resp, nil
}

func checkStatusCode(resp *http.Response) error {
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_resp, _ := io.ReadAll(resp.Body)
		errMsg := fmt.Sprintf("unexpected status code: %d, response body: %s", resp.StatusCode, _resp)
		return fmt.Errorf(errMsg)
	}
	return nil
}

func (c *HttpClient) getResp(ctx context.Context, method, url string, headers http.Header, body any) (header http.Header, resp []byte, err error) {
	var response *http.Response
	if response, err = c.do(ctx, method, url, headers, body); err != nil {
		return nil, nil, fmt.Errorf("[httpx] 发送请求失败: %w", err)
	}
	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil {
			logs.Errorf("[httpx] 关闭请求失败: %s", errorx.ErrorWithoutStack(closeErr))
		}
	}()
	// 检查响应状态码
	if err = checkStatusCode(response); err != nil {
		return response.Header, nil, err
	}
	// 读取响应体
	var _resp []byte
	if _resp, err = io.ReadAll(response.Body); err != nil {
		return response.Header, nil, fmt.Errorf("读取响应失败: %w", err)
	}
	return response.Header, _resp, nil
}

// Req 非流式HTTP请求
func (c *HttpClient) Req(ctx context.Context, method, url string, headers http.Header, body any) (resp map[string]any, err error) {
	_, resp, err = c.ReqWithHeader(ctx, method, url, headers, body)
	return resp, err
}

// GetWithHeader 非流式Get, 返回请求头
func (c *HttpClient) GetWithHeader(ctx context.Context, url string, headers http.Header, body any) (header http.Header, resp map[string]any, err error) {
	return c.ReqWithHeader(ctx, GET, url, headers, body)
}

// Get 非流式Get
func (c *HttpClient) Get(ctx context.Context, url string, headers http.Header, body any) (resp map[string]any, err error) {
	return c.Req(ctx, GET, url, headers, body)
}

// PostWithHeader 非流式Post, 返回请求头
func (c *HttpClient) PostWithHeader(ctx context.Context, url string, headers http.Header, body any) (header http.Header, resp map[string]any, err error) {
	return c.ReqWithHeader(ctx, POST, url, headers, body)
}

// Post 非流式Post
func (c *HttpClient) Post(ctx context.Context, url string, headers http.Header, body any) (resp map[string]any, err error) {
	return c.Req(ctx, POST, url, headers, body)
}

// StreamWithHeader 流式HTTP请求. 返回请求头
func (c *HttpClient) StreamWithHeader(ctx context.Context, method, url string, headers http.Header, body interface{}) (http.Header, *StreamReader, error) {
	resp, err := c.do(ctx, method, url, headers, body)
	if err != nil {
		return nil, nil, fmt.Errorf("发送请求失败: %w", err)
	}
	reader := &StreamReader{
		resp:   resp,
		reader: resp.Body,
	}
	// 检查响应状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer func() { _ = reader.Close() }()
		_resp, _ := reader.ReadAll()
		errMsg := fmt.Sprintf("unexpected status code: %d, response body: %s", resp.StatusCode, _resp)
		return resp.Header, nil, fmt.Errorf(errMsg)
	}
	return resp.Header, reader, nil
}

// Stream 流式HTTP请求
func (c *HttpClient) Stream(ctx context.Context, method, url string, headers http.Header, body interface{}) (*StreamReader, error) {
	_, reader, err := c.StreamWithHeader(ctx, method, url, headers, body)
	return reader, err
}

// StreamGetWithHeader 流式Get请求, 返回请求头
func (c *HttpClient) StreamGetWithHeader(ctx context.Context, url string, headers http.Header, body any) (http.Header, *StreamReader, error) {
	return c.StreamWithHeader(ctx, GET, url, headers, body)
}

// StreamGet 流式Get请求
func (c *HttpClient) StreamGet(ctx context.Context, url string, headers http.Header, body any) (*StreamReader, error) {
	return c.Stream(ctx, GET, url, headers, body)
}

// StreamPostWithHeader 流式Post请求, 返回请求头
func (c *HttpClient) StreamPostWithHeader(ctx context.Context, url string, headers http.Header, body any) (http.Header, *StreamReader, error) {
	return c.StreamWithHeader(ctx, POST, url, headers, body)
}

// StreamPost 流式Post请求
func (c *HttpClient) StreamPost(ctx context.Context, url string, headers http.Header, body any) (*StreamReader, error) {
	return c.Stream(ctx, POST, url, headers, body)
}

func ReqWithHeader[T any](ctx context.Context, method, url string, headers http.Header, body any) (header http.Header, resp T, err error) {
	// 读取响应体
	var _resp []byte
	if header, _resp, err = GetHttpClient().getResp(ctx, method, url, headers, body); err != nil {
		return
	}
	// 反序列化响应体
	if err = json.Unmarshal(_resp, &resp); err != nil {
		return header, resp, fmt.Errorf("反序列化响应失败: %w", err)
	}
	return header, resp, nil
}

func Req[T any](ctx context.Context, method, url string, headers http.Header, body any) (resp T, err error) {
	_, resp, err = ReqWithHeader[T](ctx, method, url, headers, body)
	return resp, err
}

func Get[T any](ctx context.Context, url string, headers http.Header, body any) (resp T, err error) {
	_, resp, err = ReqWithHeader[T](ctx, GET, url, headers, body)
	return resp, err
}

func Post[T any](ctx context.Context, url string, headers http.Header, body any) (resp T, err error) {
	_, resp, err = ReqWithHeader[T](ctx, POST, url, headers, body)
	return resp, err
}

// StreamReader 流式请求Reader, 封装是为了避免只返回reader时无法关闭resp.Body
// 调用方需要负责将流关闭
type StreamReader struct {
	resp   *http.Response
	reader io.ReadCloser
}

// Read 从Reader中读取
func (r *StreamReader) Read(p []byte) (n int, err error) {
	return r.reader.Read(p)
}

// ReadAll 读取所有的
func (r *StreamReader) ReadAll() ([]byte, error) {
	return io.ReadAll(r.reader)
}

// Close 关闭resp.Body
func (r *StreamReader) Close() error {
	return r.resp.Body.Close()
}
