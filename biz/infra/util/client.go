package util

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/cloudwego/hertz/pkg/common/json"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util/logx"
)

// httpx/client 是一个简单的http客户端
// 支持流式与非流式请求, 通过StreamReader包装流式请求的响应

var (
	client *HttpClient
	once   sync.Once
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
func (c *HttpClient) do(method, url string, headers http.Header, body any) (resp *http.Response, err error) {
	// 序列化 body 为 JSON
	var bodyBytes []byte
	var req *http.Request
	if bodyBytes, err = json.Marshal(body); err != nil {
		return nil, fmt.Errorf("[httpx]请求体序列化失败: %w", err)
	}
	// 创建新的请求
	if req, err = http.NewRequest(method, url, bytes.NewBuffer(bodyBytes)); err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	// 设置请求头
	for k, vv := range headers {
		req.Header[k] = vv
	}
	// 发送请求
	return c.Client.Do(req)
}

func (c *HttpClient) ReqWithHeader(method, url string, headers http.Header, body any) (header http.Header, resp map[string]any, err error) {
	var response *http.Response
	if response, err = c.do(method, url, headers, body); err != nil {
		return nil, nil, fmt.Errorf("[httpx] 发送请求失败: %w", err)
	}
	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil {
			logx.Error("[httpx] 关闭请求失败: ", closeErr)
		}
	}()
	// 检查响应状态码
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_resp, _ := io.ReadAll(response.Body)
		errMsg := fmt.Sprintf("unexpected status code: %d, response body: %s", response.StatusCode, _resp)
		return response.Header, nil, fmt.Errorf(errMsg)
	}
	// 读取响应体
	var _resp []byte
	if _resp, err = io.ReadAll(response.Body); err != nil {
		return response.Header, nil, fmt.Errorf("读取响应失败: %w", err)
	}
	// 反序列化响应体
	if err = json.Unmarshal(_resp, &resp); err != nil {
		return response.Header, nil, fmt.Errorf("反序列化响应失败: %w", err)
	}
	return response.Header, resp, nil
}

// Req 非流式HTTP请求
func (c *HttpClient) Req(method, url string, headers http.Header, body any) (resp map[string]any, err error) {
	_, resp, err = c.ReqWithHeader(method, url, headers, body)
	return resp, err
}

// GetWithHeader 非流式Get, 返回请求头
func (c *HttpClient) GetWithHeader(url string, headers http.Header, body any) (header http.Header, resp map[string]any, err error) {
	return c.ReqWithHeader("GET", url, headers, body)
}

// Get 非流式Get
func (c *HttpClient) Get(url string, headers http.Header, body any) (resp map[string]any, err error) {
	return c.Req("GET", url, headers, body)
}

// PostWithHeader 非流式Post, 返回请求头
func (c *HttpClient) PostWithHeader(url string, headers http.Header, body any) (header http.Header, resp map[string]any, err error) {
	return c.ReqWithHeader("POST", url, headers, body)
}

// Post 非流式Post
func (c *HttpClient) Post(url string, headers http.Header, body any) (resp map[string]any, err error) {
	return c.Req("POST", url, headers, body)
}

// StreamWithHeader 流式HTTP请求. 返回请求头
func (c *HttpClient) StreamWithHeader(method, url string, headers http.Header, body interface{}) (http.Header, *StreamReader, error) {
	resp, err := c.do(method, url, headers, body)
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
func (c *HttpClient) Stream(method, url string, headers http.Header, body interface{}) (*StreamReader, error) {
	_, reader, err := c.StreamWithHeader(method, url, headers, body)
	return reader, err
}

// StreamGetWithHeader 流式Get请求, 返回请求头
func (c *HttpClient) StreamGetWithHeader(url string, headers http.Header, body any) (http.Header, *StreamReader, error) {
	return c.StreamWithHeader("GET", url, headers, body)
}

// StreamGet 流式Get请求
func (c *HttpClient) StreamGet(url string, headers http.Header, body any) (*StreamReader, error) {
	return c.Stream("GET", url, headers, body)
}

// StreamPostWithHeader 流式Post请求, 返回请求头
func (c *HttpClient) StreamPostWithHeader(url string, headers http.Header, body any) (http.Header, *StreamReader, error) {
	return c.StreamWithHeader("POST", url, headers, body)
}

// StreamPost 流式Post请求
func (c *HttpClient) StreamPost(url string, headers http.Header, body any) (*StreamReader, error) {
	return c.Stream("POST", url, headers, body)
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
