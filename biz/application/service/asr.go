package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/bytedance/sonic"
	"github.com/hertz-contrib/websocket"
	"github.com/xh-polaris/innospark-core-api/biz/adaptor"
	"github.com/xh-polaris/innospark-core-api/biz/infra/config"
	"github.com/xh-polaris/innospark-core-api/biz/pkg/errorx"
	"github.com/xh-polaris/innospark-core-api/biz/types/errno"
	"github.com/xh-polaris/innospark-core-api/pkg/logs"
	"github.com/xh-polaris/innospark-core-api/pkg/wsx"
)

const (
	funASR = "funasr"
	meta   = "{\"mode\": \"2pass\", \"wav_name\": \"%s\", \"is_speaking\": true, \"wav_format\":\"pcm\", \"chunk_size\":[5,10,5], \"itn\":true,\"audio_fs\":16000}"
)

func ASR(ctx context.Context, conn *websocket.Conn) {
	switch config.GetConfig().ASR.Typ {
	case funASR:
		funasr(ctx, conn)
	default:
		if err := conn.Close(); err != nil {
			logs.Error(err)
		}
		return
	}
}

func funasr(ctx context.Context, conn *websocket.Conn) {
	manager := newFunASRManager(ctx, conn)
	if err := manager.run(); err != nil && !wsx.IsNormal(wsx.Classify(err)) {
		logs.Errorf("ASR processing failed: %v", err)
	}
}

type funASRManager struct {
	ctx       context.Context
	cancel    context.CancelFunc
	conn      *websocket.Conn
	cli       *wsx.WSClient
	wg        sync.WaitGroup
	closeOnce sync.Once
	meta      string
	errChan   chan error
	ini       bool
}

func newFunASRManager(ctx context.Context, conn *websocket.Conn) *funASRManager {
	ctx, cancel := context.WithCancel(ctx)
	return &funASRManager{
		ctx:     ctx,
		cancel:  cancel,
		conn:    conn,
		errChan: make(chan error, 2), // 缓冲两个错误
	}
}

func (m *funASRManager) run() error {
	defer m.cleanup()
	// 启动错误监控
	m.wg.Add(1)
	go m.monitorErrors()
	// 主处理循环
	for {
		select {
		case <-m.ctx.Done():
			return m.ctx.Err()
		default:
			mt, data, err := m.conn.ReadMessage()
			if err != nil {
				m.sendError(err)
				return err
			}
			if err = m.handleMessage(mt, data); err != nil {
				m.sendError(err)
				return err
			}
		}
	}
}

func (m *funASRManager) handleMessage(mt int, data []byte) error {
	switch mt {
	case websocket.TextMessage:
		return m.handleTextMessage(data)
	case websocket.BinaryMessage:
		return m.handleBinaryMessage(data)
	case websocket.CloseMessage:
		m.Close()
		return nil
	default:
		return fmt.Errorf("unsupported message type: %d", mt)
	}
}

func (m *funASRManager) handleTextMessage(data []byte) error {
	if m.ini {
		return nil // 已经初始化，忽略后续文本消息
	}
	info := map[string]any{}
	if err := json.Unmarshal(data, &info); err != nil {
		return err
	}
	jwt, ok := info["Authorization"]
	if !ok {
		return errorx.New(errno.UnAuthErrCode)
	}
	token, ok := jwt.(string)
	if !ok {
		return errorx.New(errno.UnAuthErrCode)
	}
	uid, err := adaptor.ExtractUserIdFromJWT(token)
	if err != nil {
		return fmt.Errorf("extract user id failed: %s", err)
	}
	m.meta = fmt.Sprintf(meta, uid)
	return nil
}

func (m *funASRManager) handleBinaryMessage(data []byte) error {
	if IsFirstASR(data) {
		return m.iniCli()
	} else if IsLastASR(data) {
		return m.endCli()
	}
	if !m.ini {
		return fmt.Errorf("ASR client not initialized")
	}
	// 安全发送音频数据
	if m.cli != nil {
		if err := m.cli.WriteBytes(data); err != nil {
			logs.Error("send audio data failed: %s", err)
			return err
		}
	}
	return nil
}

func (m *funASRManager) iniCli() (err error) {
	// 创建 ASR 客户端
	headers := http.Header{}
	m.cli, err = wsx.NewWSClientWithDial(m.ctx, config.GetConfig().ASR.URL, headers)
	if err != nil {
		return fmt.Errorf("create ASR client failed: %s", err)
	}
	// 发送初始化元数据
	if err = m.cli.WriteString(m.meta); err != nil {
		logs.Errorf("send meta data failed: %s", err)
		return err
	}
	// 启动接收协程
	m.wg.Add(1)
	go m.receiveASRResults()
	m.ini = true
	return
}

func (m *funASRManager) endCli() error {
	return m.cli.WriteString("{\"is_speaking\": false}")
}

func (m *funASRManager) receiveASRResults() {
	defer m.wg.Done()
	defer m.Close() // 接收协程退出时触发关闭

	for {
		select {
		case <-m.ctx.Done():
			return
		default:
			resp, err := m.cli.ReadString()
			if err != nil {
				m.sendError(err)
				return
			}
			fmt.Println(resp)
			if m.conn != nil {
				if err := m.conn.WriteMessage(websocket.TextMessage, []byte(resp)); err != nil {
					logs.Errorf("send result to user failed: %s", err)
					m.sendError(err)
					return
				}
			}
			info := map[string]any{}
			if err = sonic.Unmarshal([]byte(resp), &info); err != nil {
				logs.Errorf("unmarshal ASR result failed: %s", err)
				m.sendError(err)
			}
			if info["is_final"].(bool) {
				return
			}
		}
	}
}

func (m *funASRManager) sendError(err error) {
	select {
	case m.errChan <- err:
	case <-m.ctx.Done():
	default:
		// 错误通道已满，直接记录日志
		logs.Errorf("Error channel full, dropped error: %v", err)
	}
}

func (m *funASRManager) monitorErrors() {
	defer m.wg.Done()
	select {
	case <-m.ctx.Done():
		return
	case err := <-m.errChan:
		if err != nil && !wsx.IsNormal(wsx.Classify(err)) {
			logs.Errorf("ASR manager error: %v", err)
		}
		m.Close()
	}
}

func (m *funASRManager) cleanup() {
	m.Close()
	m.wg.Wait() // 等待所有协程退出
	close(m.errChan)
}

func (m *funASRManager) Close() {
	m.closeOnce.Do(func() {
		m.cancel() // 取消上下文，通知所有协程退出
		if m.conn != nil {
			if err := m.conn.Close(); err != nil && !wsx.IsNormal(wsx.Classify(err)) {
				logs.Errorf("Close user connection failed: %v", err)
			}
		}
		if m.cli != nil {
			if err := m.cli.Close(); err != nil && !wsx.IsNormal(wsx.Classify(err)) {
				logs.Errorf("Close ASR client failed: %v", err)
			}
		}
	})
}

var (
	FirstASR byte = 0   // 标识开始
	LastASR  byte = 255 // 标识结束
)

// IsFirstASR 判断是否是开始包
func IsFirstASR(data []byte) bool {
	return len(data) == 1 && data[0] == FirstASR
}

// IsLastASR 判断是否是结束包
func IsLastASR(data []byte) bool {
	return len(data) == 1 && data[0] == LastASR
}
