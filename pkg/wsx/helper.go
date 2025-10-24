package wsx

import (
	"context"
	"errors"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/xh-polaris/innospark-core-api/pkg/errorx"
	"github.com/xh-polaris/innospark-core-api/pkg/logs"
)

/*
为了更优雅的处理连接关闭导致的各种报错
将所有的异常分为两个级别
1 - Normal   可以容忍的异常, 正常结束连接
2 - Abnormal 不可容忍的异常, 正常结束连接, 需要记录异常情况

gorilla中的Close异常定义
Normal:
	CloseNormalClosure           = 1000
	CloseGoingAway               = 1001
	CloseNoStatusReceived        = 1005

Abnormal:
	CloseProtocolError           = 1002
	CloseUnsupportedData         = 1003
	CloseAbnormalClosure         = 1006 // TODO 需告知前端, 所有结束场景都需发送结束消息
	CloseInvalidFramePayloadData = 1007
	ClosePolicyViolation         = 1008
	CloseMessageTooBig           = 1009
	CloseMandatoryExtension      = 1010
	CloseInternalServerErr       = 1011
	CloseServiceRestart          = 1012
	CloseTryAgainLater           = 1013
	CloseTLSHandshake            = 1015
*/

var (
	NormalCLoseMsg = websocket.FormatCloseMessage(1000, "normal close")
	DefaultTimeout = time.Second * 3
)

var NormalCloseErr = errors.New("[WSClient] normal close error")
var AbnormalCloseErr = errors.New("[WSClient] abnormal close error")

// classifyErr 将错误归类
func (ws *WSClient) classifyErr(err error) error {
	switch {
	case err == nil:
		return nil
	case websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseNoStatusReceived):
		ws.closed = true
		return NormalCloseErr
	case websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseNoStatusReceived):
		// 为了避免内部错误被隐藏, 此处日志记录错误原因
		logs.Errorf("[WSClient] close error: %s", errorx.ErrorWithoutStack(err))
		ws.closed = true
		return AbnormalCloseErr
	default:
		return err
	}
}

func Classify(err error) error {
	switch {
	case err == nil:
		return nil
	case websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseNoStatusReceived):
		return NormalCloseErr
	case websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseNoStatusReceived):
		// 为了避免内部错误被隐藏, 此处日志记录错误原因
		logs.Errorf("[WSClient] close error: %s", errorx.ErrorWithoutStack(err))
		return AbnormalCloseErr
	default:
		return err
	}
}

// WSClient 是基于gorilla/websocket的工具类, 封装了常见读写操作, 简化了异常处理
// 最佳实践是单线程读, 所以此处不设读锁, 若并发读, 需自行维护读锁
// 一个client和一个conn此处设计为一一对应, 不支持更改client的conn
type WSClient struct {
	// 写锁
	mu   sync.Mutex
	conn *websocket.Conn
	// 连接是否关闭
	closed bool
}

// NewWSClient 生成管理传入参数的client
func NewWSClient(conn *websocket.Conn) *WSClient {
	return &WSClient{
		mu:   sync.Mutex{},
		conn: conn,
	}
}

// NewWSClientWithDial 根据指定的参数创建新的连接
func NewWSClientWithDial(ctx context.Context, url string, header http.Header) (*WSClient, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	// 尝试连接
	conn, r, err := websocket.DefaultDialer.DialContext(ctx, url, header)
	if err == nil {
		return NewWSClient(conn), err
	}
	// 连接失败若有响应, 打印错误日志
	if r != nil {
		if body, parseErr := io.ReadAll(r.Body); parseErr == nil {
			logs.Errorf("[WSClient] parse conn resp body: %s", string(body))
		}
	}
	return nil, err
}

// Read 读取一条消息, 同时返回错误
func (ws *WSClient) Read() (mt int, data []byte, err error) {
	mt, data, err = ws.conn.ReadMessage()
	return mt, data, ws.classifyErr(err)
}

// ReadBytes 读取一条二进制消息
func (ws *WSClient) ReadBytes() (data []byte, err error) {
	_, data, err = ws.Read()
	return data, err
}

// ReadString 读取一条文本消息
func (ws *WSClient) ReadString() (string, error) {
	_, data, err := ws.Read()
	return string(data), err
}

// ReadJSON 读取一个JSON对象, 并写入指定位置
func (ws *WSClient) ReadJSON(obj any) (err error) {
	return ws.classifyErr(ws.conn.ReadJSON(obj))
}

// Write 写入指定类型消息
func (ws *WSClient) Write(mt int, data []byte) (err error) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	err = ws.conn.WriteMessage(mt, data)
	return ws.classifyErr(err)
}

// WriteBytes 写入二进制消息
func (ws *WSClient) WriteBytes(data []byte) (err error) {
	return ws.Write(websocket.BinaryMessage, data)
}

// WriteString 写入字符串消息
func (ws *WSClient) WriteString(data string) (err error) {
	return ws.Write(websocket.TextMessage, []byte(data))
}

// WriteJSON 写入序列化为JSON的对象
func (ws *WSClient) WriteJSON(obj any) (err error) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	return ws.classifyErr(ws.conn.WriteJSON(obj))
}

// WritePing 写入心跳消息
func (ws *WSClient) WritePing() error {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	return ws.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(DefaultTimeout))
}

// Close 关闭连接
func (ws *WSClient) Close() error {
	if !ws.closed {
		if err := ws.conn.WriteControl(websocket.CloseMessage, NormalCLoseMsg, time.Now().Add(DefaultTimeout)); err != nil {
			logs.Errorf("[WSClient] send close msg error: %s", errorx.ErrorWithoutStack(err))
		}
		ws.closed = true
		return ws.conn.Close()
	}
	return nil
}

func (ws *WSClient) IsClosed() bool {
	return ws.closed
}
