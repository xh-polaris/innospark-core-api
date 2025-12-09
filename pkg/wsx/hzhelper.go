package wsx

import (
	"context"
	"fmt"
	"github.com/hertz-contrib/websocket"
	"github.com/xh-polaris/psych-pkg/util/logx"
	"net/http"
	"sync"
	"time"
)

// classifyErr 将错误归类
func (ws *HZWSClient) classifyErr(err error) error {
	switch {
	case err == nil:
		return nil
	case websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseNoStatusReceived):
		ws.closed = true
		return NormalCloseErr
	case websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseNoStatusReceived):
		// 为了避免内部错误被隐藏, 此处日志记录错误原因
		logx.Error("[HZWSClient] close error", err)
		ws.closed = true
		return AbnormalCloseErr
	default:
		return err
	}
}

// HZWSClient 是基于hertz-contribute/websocket的工具类, 封装了常见读写操作, 简化了异常处理
// 最佳实践是单线程读, 所以此处不设读锁, 若并发读, 需自行维护读锁
// 一个client和一个conn此处设计为一一对应, 不支持更改client的conn
type HZWSClient struct {
	// 写锁
	mu   sync.Mutex
	conn *websocket.Conn
	// 连接是否关闭
	closed bool
}

// NewHZWSClient 生成管理传入参数的client
func NewHZWSClient(conn *websocket.Conn) *HZWSClient {
	return &HZWSClient{
		mu:   sync.Mutex{},
		conn: conn,
	}
}

// NewHZWSClientWithDial 根据指定的参数创建新的连接
// 由于只能将hertz http响应升级ws, 暂时没有这个需求所以先搁置
func NewHZWSClientWithDial(ctx context.Context, url string, header http.Header) (*HZWSClient, error) {
	return nil, fmt.Errorf("no implementation")
}

// Read 读取一条消息, 同时返回错误
func (ws *HZWSClient) Read() (mt int, data []byte, err error) {
	mt, data, err = ws.conn.ReadMessage()
	return mt, data, ws.classifyErr(err)
}

// ReadBytes 读取一条二进制消息
func (ws *HZWSClient) ReadBytes() (data []byte, err error) {
	_, data, err = ws.Read()
	return data, err
}

// ReadString 读取一条文本消息
func (ws *HZWSClient) ReadString() (string, error) {
	_, data, err := ws.Read()
	return string(data), err
}

// ReadJSON 读取一个JSON对象, 并写入指定位置
func (ws *HZWSClient) ReadJSON(obj any) (err error) {
	return ws.classifyErr(ws.conn.ReadJSON(obj))
}

// Write 写入指定类型消息
func (ws *HZWSClient) Write(mt int, data []byte) (err error) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	err = ws.conn.WriteMessage(mt, data)
	return ws.classifyErr(err)
}

// WriteBytes 写入二进制消息
func (ws *HZWSClient) WriteBytes(data []byte) (err error) {
	return ws.Write(websocket.BinaryMessage, data)
}

// WriteString 写入字符串消息
func (ws *HZWSClient) WriteString(data string) (err error) {
	return ws.Write(websocket.TextMessage, []byte(data))
}

// WriteJSON 写入序列化为JSON的对象
func (ws *HZWSClient) WriteJSON(obj any) (err error) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	return ws.classifyErr(ws.conn.WriteJSON(obj))
}

// Ping 写入心跳消息
func (ws *HZWSClient) Ping(data []byte) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	return ws.conn.WriteControl(websocket.PingMessage, data, time.Now().Add(DefaultTimeout))
}

// Pong 写入Pong消息
func (ws *HZWSClient) Pong(data []byte) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	return ws.conn.WriteControl(websocket.PongMessage, data, time.Now().Add(DefaultTimeout))
}

func (ws *HZWSClient) SetPingHandler(h func(appData string) error) {
	ws.conn.SetPingHandler(h)
}

func (ws *HZWSClient) ControlClose(data []byte) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	return ws.conn.WriteControl(websocket.CloseMessage, data, time.Now().Add(DefaultTimeout))
}

func (ws *HZWSClient) SetCloseHandler(h func(code int, text string) error) {
	ws.conn.SetCloseHandler(h)
}

// Close 关闭连接
func (ws *HZWSClient) Close() error {
	if !ws.closed {
		if err := ws.conn.WriteControl(websocket.CloseMessage, NormalCLoseMsg, time.Now().Add(DefaultTimeout)); err != nil {
			logx.Error("[HZWSClient] send close message error", err)
		}
		ws.closed = true
		return ws.conn.Close()
	}
	return nil
}

func (ws *HZWSClient) IsClosed() bool {
	return ws.closed
}
