package info

// 信息域, 负责跨节点消息传递, 只持有, 不操作

import (
	"context"

	"github.com/cloudwego/eino/schema"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/xh-polaris/innospark-core-api/biz/adaptor"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	mmsg "github.com/xh-polaris/innospark-core-api/biz/infra/mapper/message"
	"github.com/xh-polaris/innospark-core-api/biz/infra/mapper/user"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Info 存储Completion接口过程中的上下文信息
type Info struct {
	RequestContext    *app.RequestContext
	CompletionOptions *CompletionOptions   // 对话配置
	ModelInfo         *ModelInfo           // 模型信息
	MessageInfo       *MessageInfo         // 消息信息
	ConversationId    primitive.ObjectID   // 对话id
	SectionId         primitive.ObjectID   // 段落id
	UserId            primitive.ObjectID   // 用户id
	ReplyId           string               // 响应ID
	OriginMessage     *ReqMessage          // 用户原始消息
	UserMessage       *mmsg.Message        // 用户消息
	Profile           *user.Profile        // 用户个性化配置
	Ext               map[string]string    // 额外配置
	ResponseMeta      *schema.ResponseMeta // 用量
	SSE               *adaptor.SSEStream   // SSE流
	SSEIndex          int                  // SSE事件索引
	ModelCancel       context.CancelFunc   // 中断模型输出
	SearchInfo        *SearchInfo          // 搜素信息
	Sensitive         *Sensitive
	Attach            []string // 附件信息
}

// CompletionOptions 是对话相关配置
type CompletionOptions struct {
	Typ             string
	ReplyId         *string
	IsRegen         bool
	IsReplace       bool
	SelectedRegenId *string
	RegenList       []*mmsg.Message
	ReplaceList     []*mmsg.Message
	SelectRegenList []*mmsg.Message
}

// ModelInfo 是模型相关配置
type ModelInfo struct {
	WebSearch bool   // 是否联网搜索
	Thinking  bool   // 是否深度思考
	Model     string // 模型名称
	BotId     string // 智能体id
	BotName   string // 智能体名称
}

type MessageInfo struct {
	AssistantMessage *mmsg.Message
	Text             string       // 对话内容
	Think            string       // 思考内容
	Suggest          string       // 建议内容
	Code             []*mmsg.Code // 代码内容
}

type ReqMessage struct {
	Content     string
	ContentType int32
	Attaches    []string
	References  []string
}

type SearchInfo struct {
	Find   int          // 找到的数量
	Choose int          // 选择的数量
	Cite   []*mmsg.Cite // 引用
}

type RefineContent struct {
	Typ      int    `json:"-"`
	Think    string `json:"think,omitempty"`
	Text     string `json:"text,omitempty"`
	Suggest  string `json:"suggest,omitempty"`
	Code     string `json:"code,omitempty"`
	CodeType string `json:"codeType,omitempty"`
}

func (r *RefineContent) GetContent() string {
	switch r.Typ {
	case cst.EventMessageContentTypeThink:
		return r.Think
	case cst.EventMessageContentTypeSuggest:
		return r.Suggest
	case cst.EventMessageContentTypeCode:
		return r.Code
	case cst.EventMessageContentTypeCodeType:
		return r.CodeType
	case cst.EventMessageContentTypeText:
		fallthrough
	default:
		return r.Text
	}
}

func (r *RefineContent) SetContent(s string) string {
	switch r.Typ {
	case cst.EventMessageContentTypeThink:
		r.Think = s
	case cst.EventMessageContentTypeSuggest:
		r.Suggest = s
	case cst.EventMessageContentTypeCode:
		r.Code = s
	case cst.EventMessageContentTypeCodeType:
		r.CodeType = s
	case cst.EventMessageContentTypeText:
		fallthrough
	default:
		r.Text = s
	}
	return s
}
func (r *RefineContent) SetContentWithTyp(s string, t int) (string, int) {
	r.Typ = t
	r.SetContent(s)
	return s, t
}

type Sensitive struct {
	Hits []string
}
