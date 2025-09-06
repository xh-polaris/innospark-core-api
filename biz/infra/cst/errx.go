package cst

import (
	"fmt"
)

var (
	UnAuthErr = New(1000, "身份认证失败")
	HisErr    = New(20000, "获取对话记录失败")
)

const unknowCode = 999

// Errorx 是HTTP服务的业务异常
// 若返回Errorx给前端, 则HTTP状态码应该是200, 且响应体为Errorx内容
// 最佳实践:
// - 业务处理链路的末端使用Errorx, PostProcess处理后给出用户友好的响应
// - 预定义一些Errorx作为常量
// - 除却末端的Errorx外, 其余的error照常处理

type IErrorx interface {
	GetCode() int
	GetMsg() string
}

type Errorx struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func New(code int, msg string) *Errorx {
	return &Errorx{
		Code: code,
		Msg:  msg,
	}
}

// Error 实现了error接口, 返回错误字符串
func (e Errorx) Error() string {
	return fmt.Sprintf("code=%d, msg=%s", e.Code, e.Msg)
}

// GetCode 获取Code
func (e Errorx) GetCode() int {
	return e.Code
}

// GetMsg 获取Msg
func (e Errorx) GetMsg() string {
	return e.Msg
}
