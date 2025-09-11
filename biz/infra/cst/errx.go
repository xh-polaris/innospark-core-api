package cst

import (
	"fmt"
)

var (
	UnAuthErr      = New(1000, "身份认证失败")
	UnImplementErr = New(888, "尚未实现的功能")
	OIDErr         = New(777, "id错误")
	Interrupt      = New(666, "中断")
)

// conversation 相关
var (
	ConversationCreationErr = New(30001, "创建对话失败")
	ConversationRenameErr   = New(30002, "对话标题重命名失败")
	ConversationListErr     = New(30003, "分页获取历史对话失败")
	ConversationGetErr      = New(30004, "获取对话历史记录失败")
	ConversationDeleteErr   = New(30005, "删除历史记录失败")
	ConversationSearchErr   = New(30006, "搜索历史记录失败")
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
