package cst

import (
	"fmt"

	"github.com/xh-polaris/innospark-core-api/biz/infra/util/errorx/code"
)

const (
	// 通用错误码
	UnAuthErrCode      = 1000
	UnImplementErrCode = 888
	OIDErrCode         = 777
	InterruptCode      = 666
	unknowCode         = 999

	// 会话相关错误码
	ConversationCreateErrCode        = 30001
	ConversationRenameErrCode        = 30002
	ConversationListErrCode          = 30003
	ConversationGetErrCode           = 30004
	ConversationDeleteErrCode        = 30005
	ConversationSearchErrCode        = 30006
	ConversationGenerateBriefErrCode = 30007

	// Feedback 相关错误码
	FeedbackErrCode = 40001

	// 配置相关错误码
	ConfigLoadErrCode  = 50001
	ConfigSetupErrCode = 50002

	// 依赖服务相关错误码
	SynapseErrCode = 60001
	CozeErrCode    = 60002

	// Completions 相关错误码
	CompletionsErrCode = 70001
)

func init() {
	// 通用错误码
	code.Register(UnAuthErrCode, "身份认证失败", code.WithAffectStability(false))
	code.Register(OIDErrCode, "id错误", code.WithAffectStability(false))

	// 配置相关错误码
	code.Register(ConfigLoadErrCode, "配置加载路径 {path} 失败", code.WithAffectStability(true))
	code.Register(ConfigSetupErrCode, "配置初始化失败", code.WithAffectStability(true))

	// 依赖服务相关错误码
	code.Register(SynapseErrCode, "使用 Synapse 访问 {url} 错误", code.WithAffectStability(true))
	code.Register(CozeErrCode, "使用 Coze 访问 {url} 错误", code.WithAffectStability(true))

	// Feedback 相关错误码
	code.Register(FeedbackErrCode, "处理反馈失败", code.WithAffectStability(true))

	// Conversation 相关错误码
	code.Register(ConversationCreateErrCode, "创建对话失败", code.WithAffectStability(true))
	code.Register(ConversationRenameErrCode, "对话标题重命名失败", code.WithAffectStability(true))
	code.Register(ConversationListErrCode, "分页获取历史对话失败", code.WithAffectStability(true))
	code.Register(ConversationGetErrCode, "获取对话历史记录失败", code.WithAffectStability(true))
	code.Register(ConversationDeleteErrCode, "删除历史记录失败", code.WithAffectStability(true))
	code.Register(ConversationSearchErrCode, "搜索历史记录失败", code.WithAffectStability(true))
	code.Register(ConversationGenerateBriefErrCode, "生成对话摘要失败", code.WithAffectStability(true))

	// Completions 相关错误码
	code.Register(CompletionsErrCode, "处理AI文本生成请求失败", code.WithAffectStability(true))
}

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
