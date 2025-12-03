package info

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/cloudwego/eino/schema"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/sse"
	"github.com/xh-polaris/innospark-core-api/biz/adaptor"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	mmsg "github.com/xh-polaris/innospark-core-api/biz/infra/mapper/message"
	"github.com/xh-polaris/innospark-core-api/biz/infra/mapper/user"
	"github.com/xh-polaris/innospark-core-api/pkg/errorx"
	"github.com/xh-polaris/innospark-core-api/pkg/logs"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RelayContext 存储Completion接口过程中的上下文信息
type RelayContext struct {
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
	SSEWriter         *sse.Writer          // SSE输出
	SSEIndex          int                  // SSE事件索引
	ModelCancel       context.CancelFunc   // 中断模型输出
	SearchInfo        *SearchInfo          // 搜素信息
	Sensitive         *Sensitive
	Attach            []string // 附件信息
}

func (r *RelayContext) id() string {
	i := strconv.Itoa(r.SSEIndex)
	r.SSEIndex++
	return i
}

func (r *RelayContext) ChatEvent(content string, typ int) *sse.Event {
	chat := &adaptor.EventChat{
		Message:          &adaptor.ChatMessage{Content: content, ContentType: typ},
		ConversationId:   r.ConversationId.Hex(),
		SectionId:        r.SectionId.Hex(),
		ReplyId:          r.ReplyId,
		IsDelta:          true,
		Status:           cst.MessageStatus,
		InputContentType: cst.InputContentTypeText,
		MessageIndex:     int(r.MessageInfo.AssistantMessage.Index),
		BotId:            r.ModelInfo.BotId,
	}
	return Event(r.id(), cst.EventChat, chat)
}

func (r *RelayContext) MetaEvent() *sse.Event {
	meta := &adaptor.EventMeta{
		MessageId:        r.MessageInfo.AssistantMessage.MessageId.Hex(),
		ConversationId:   r.ConversationId.Hex(),
		SectionId:        r.SectionId.Hex(),
		MessageIndex:     int(r.MessageInfo.AssistantMessage.Index),
		ConversationType: cst.ConversationTypeText,
		ReplyId:          r.ReplyId,
	}
	return Event(r.id(), cst.EventMeta, meta)
}

func (r *RelayContext) ModelEvent() *sse.Event {
	m := &adaptor.EventModel{Model: r.ModelInfo.Model, BotId: r.ModelInfo.BotId, BotName: r.ModelInfo.BotName}
	return Event(r.id(), cst.EventModel, m)
}

func (r *RelayContext) EndEvent() *sse.Event {
	return EventWithoutMarshal(r.id(), cst.EventEnd, []byte(cst.EventNotifyValue))
}

func (r *RelayContext) ErrorEvent(code int, msg string) *sse.Event {
	return EventWithoutMarshal(r.id(), cst.EventError, []byte(fmt.Sprintf("{\"code\":%d,\"msg\":\"%s\"}", code, msg)))
}

func (r *RelayContext) SearchStartEvent() *sse.Event {
	return EventWithoutMarshal(r.id(), cst.EventSearchStart, []byte(cst.EventNotifyValue))
}

func (r *RelayContext) SearchEndEvent() *sse.Event {
	return EventWithoutMarshal(r.id(), cst.EventSearchEnd, []byte(cst.EventNotifyValue))
}

func (r *RelayContext) SearchFindEvent(n int) *sse.Event {
	return EventWithoutMarshal(r.id(), cst.EventSearchFind, []byte(strconv.Itoa(n)))
}

func (r *RelayContext) SearchChooseEvent(n int) *sse.Event {
	return EventWithoutMarshal(r.id(), cst.EventSearchChoose, []byte(strconv.Itoa(n)))
}

func (r *RelayContext) SearchCiteEvent(cite *mmsg.Cite) *sse.Event {
	c := &adaptor.EventSearchCite{
		Index: cite.Index, Name: cite.Name, URL: cite.URL, Snippet: cite.Snippet,
		SiteName: cite.SiteName, SiteIcon: cite.SiteIcon, DatePublished: cite.DatePublished}
	return Event(r.id(), cst.EventSearchCite, c)
}

func Event(index string, typ string, obj any) *sse.Event {
	var err error
	var data []byte
	if data, err = json.Marshal(obj); err != nil {
		logs.Errorf("[graph sse] event marshal error: %s", errorx.ErrorWithoutStack(err))
	}
	return &sse.Event{ID: index, Type: typ, Data: data}
}

func EventWithoutMarshal(index string, typ string, data []byte) *sse.Event {
	return &sse.Event{ID: index, Type: typ, Data: data}
}

// SSEEvent 写入一个sse事件
func (r *RelayContext) SSEEvent(e *sse.Event) error {
	return r.SSEWriter.Write(e)
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
