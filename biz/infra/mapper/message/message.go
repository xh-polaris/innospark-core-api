package message

import (
	"time"

	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	RoleStoI = map[string]int32{cst.System: 0, cst.Assistant: 1, cst.User: 2, cst.Tool: 3}
	RoleItoS = map[int32]string{0: cst.System, 1: cst.Assistant, 2: cst.User, 3: cst.Tool}
)

// Message 一条消息, 可能归属于用户或模型
type Message struct {
	MessageId                primitive.ObjectID   `json:"message_id" bson:"_id"`                                                                 // 主键
	ConversationId           primitive.ObjectID   `json:"conversation_id" bson:"conversation_id"`                                                // 归属的对话id
	SectionId                primitive.ObjectID   `json:"section_id" bson:"section_id"`                                                          // 归属的段落id
	UserId                   primitive.ObjectID   `json:"user_id" bson:"user_id"`                                                                // 用户id
	Index                    int32                `json:"index" bson:"index"`                                                                    // 消息索引
	ReplyId                  primitive.ObjectID   `json:"reply_id,omitempty" bson:"reply_id,omitempty"`                                          // 回复id, 只有模型消息有
	Content                  string               `json:"content" bson:"content"`                                                                // 消息内容, json字符串
	ContentType              int32                `json:"content_type" bson:"content_type"`                                                      // 内容类型, text/think/suggest, 依次为0,1,2
	MessageType              int32                `json:"message_type" bson:"message_type"`                                                      // 消息类型, 默认为text, 0
	UserInputMultiContent    []*MessageInputPart  `json:"user_input_multi_content,omitempty" bson:"user_input_multi_content,omitempty"`          // 只有多模态时存在, 这个存在时就不使用Content
	AssistantGenMultiContent []*MessageOutputPart `json:"assistant_output_multi_content,omitempty" bson:"assistant_gen_multi_content,omitempty"` // 只有多模态时存在, 这个存在时就不使用Content
	Ext                      *Ext                 `json:"ext" bson:"ext"`                                                                        // 额外信息
	Feedback                 int32                `json:"feedback,omitempty" bson:"feedback,omitempty"`                                          // 反馈, 无/喜欢/踩/删除, 依次为0,1,2,3
	Role                     int32                `json:"role" bson:"role"`                                                                      // 角色, system/assistant/user/tool, 依次为0,1,2,3,4
	CreateTime               time.Time            `json:"create_time" bson:"create_time"`                                                        // 创建时间
	UpdateTime               time.Time            `json:"update_time" bson:"update_time"`                                                        // 更新时间
	DeleteTime               time.Time            `json:"delete_time,omitempty" bson:"delete_time,omitempty"`                                    // 删除时间
	Status                   int32                `json:"status" bson:"status"`                                                                  // 状态, 默认/regen未选择/regen被选择/替换过/中断, 依次是0,1,2,3
}

type Ext struct {
	BotState        string        `json:"bot_state" bson:"bot_state"`                         // json字符串, 模型信息
	Brief           string        `json:"brief,omitempty" bson:"brief,omitempty"`             // 内容备份
	Think           string        `json:"think,omitempty" bson:"think,omitempty"`             // 深度思考内容
	Suggest         string        `json:"suggest,omitempty" bson:"suggest,omitempty"`         // 建议内容
	Cite            []*Cite       `json:"cite,omitempty" bson:"cite,omitempty"`               // 引用
	Code            []*Code       `json:"code,omitempty" bson:"code,omitempty"`               // 代码
	ContentWithCite *string       `json:"-" bson:"-"`                                         // 模型用到的引用, 会替换模型域的消息
	Sensitive       bool          `json:"sensitive,omitempty" bson:"sensitive,omitempty"`     // 是否触发违禁词
	AttachInfo      []*AttachInfo `json:"attach_info,omitempty" bson:"attach_info,omitempty"` // 附件信息
	Usage           *Usage        `json:"usage,omitempty" bson:"usage,omitempty"`             // 用量信息
}

type Cite struct {
	Index         int32  `json:"index" bson:"index"`
	Name          string `json:"name" bson:"name"`
	URL           string `json:"url" bson:"url"`
	Snippet       string `json:"snippet" bson:"snippet"`
	SiteName      string `json:"siteName" bson:"site_name"`
	SiteIcon      string `json:"siteIcon" bson:"site_icon"`
	DatePublished string `json:"datePublished" bson:"date_published"`
}

type Usage struct {
	// PromptTokens is the number of prompt tokens, including all the input tokens of this request.
	PromptTokens int `json:"prompt_tokens,omitempty" bson:"prompt_tokens,omitempty"`
	// PromptTokenDetails is a breakdown of the prompt tokens.
	PromptTokenDetails *PromptTokenDetails `json:"prompt_token_details,omitempty" bson:"prompt_token_details,omitempty"`
	// CompletionTokens is the number of completion tokens.
	CompletionTokens int `json:"completion_tokens,omitempty" bson:"completion_tokens,omitempty"`
	// TotalTokens is the total number of tokens.
	TotalTokens int `json:"total_tokens,omitempty" bson:"total_tokens,omitempty"`
}

type PromptTokenDetails struct {
	// Cached tokens present in the prompt.
	CachedTokens int `json:"cached_tokens,omitempty" bson:"cached_tokens,omitempty"`
}

type Code struct {
	Index    int32  `json:"index" bson:"index"`
	CodeType string `json:"codeType" bson:"code_type"`
	Code     string `json:"code" bson:"code"`
}

type AttachInfo struct {
	AccessURL string `json:"access_url" bson:"access_url"`
	Key       string `json:"key" bson:"key"`
}

type ChatMessagePartType string

const (
	// ChatMessagePartTypeText means the part is a text.
	ChatMessagePartTypeText ChatMessagePartType = "text"
	// ChatMessagePartTypeImageURL means the part is an image url.
	ChatMessagePartTypeImageURL ChatMessagePartType = "image_url"
	// ChatMessagePartTypeAudioURL means the part is an audio url.
	ChatMessagePartTypeAudioURL ChatMessagePartType = "audio_url"
	// ChatMessagePartTypeVideoURL means the part is a video url.
	ChatMessagePartTypeVideoURL ChatMessagePartType = "video_url"
	// ChatMessagePartTypeFileURL means the part is a file url.
	ChatMessagePartTypeFileURL ChatMessagePartType = "file_url"
)

// MessageInputPart represents the input part of message.
type MessageInputPart struct {
	Type ChatMessagePartType `json:"type" bson:"type"`

	Text string `json:"text,omitempty" bson:"text,omitempty"`

	// Image is the image input of the part, it's used when Type is "image_url".
	Image *MessageInputImage `json:"image,omitempty" bson:"image,omitempty"`

	// Audio  is the audio input of the part, it's used when Type is "audio_url".
	Audio *MessageInputAudio `json:"audio,omitempty" bson:"audio,omitempty"`

	// Video is the video input of the part, it's used when Type is "video_url".
	Video *MessageInputVideo `json:"video,omitempty" bson:"video,omitempty"`

	// File is the file input of the part, it's used when Type is "file_url".
	File *MessageInputFile `json:"file,omitempty" bson:"file,omitempty"`
}

// MessageInputImage is used to represent an image part in message.
// Choose either URL or Base64Data.
type MessageInputImage struct {
	MessagePartCommon

	// Detail is the quality of the image url.
	Detail ImageURLDetail `json:"detail,omitempty" bson:"detail,omitempty"`
}

// MessageOutputPart represents a part of an assistant-generated message.
// It can contain text, or multimedia content like images, audio, or video.
type MessageOutputPart struct {
	// Type is the type of the part, eg. "text", "image_url", "audio_url", "video_url".
	Type ChatMessagePartType `json:"type" bson:"type"`

	// Text is the text of the part, it's used when Type is "text".
	Text string `json:"text,omitempty" bson:"text,omitempty"`

	// Image is the image output of the part, used when Type is ChatMessagePartTypeImageURL.
	Image *MessageOutputImage `json:"image,omitempty" bson:"image,omitempty"`

	// Audio is the audio output of the part, used when Type is ChatMessagePartTypeAudioURL.
	Audio *MessageOutputAudio `json:"audio,omitempty" bson:"audio,omitempty"`

	// Video is the video output of the part, used when Type is ChatMessagePartTypeVideoURL.
	Video *MessageOutputVideo `json:"video,omitempty" bson:"video,omitempty"`
}

// MessageInputAudio is used to represent an audio part in message.
// Choose either URL or Base64Data.
type MessageInputAudio struct {
	MessagePartCommon
}

// MessageInputVideo is used to represent a video part in message.
// Choose either URL or Base64Data.
type MessageInputVideo struct {
	MessagePartCommon
}

// MessageInputFile is used to represent a file part in message.
// Choose either URL or Base64Data.
type MessageInputFile struct {
	MessagePartCommon
}

// MessageOutputImage is used to represent an image part in message.
type MessageOutputImage struct {
	MessagePartCommon
}

// MessageOutputAudio is used to represent an audio part in message.
type MessageOutputAudio struct {
	MessagePartCommon
}

// MessageOutputVideo is used to represent a video part in message.
type MessageOutputVideo struct {
	MessagePartCommon
}

// MessagePartCommon represents the common abstract components for input and output of multi-modal types.
type MessagePartCommon struct {
	// URL can either be a traditional URL or a special URL conforming to RFC-2397 (https://www.rfc-editor.org/rfc/rfc2397).
	// double check with model implementations for detailed instructions on how to use this.
	URL *string `json:"url,omitempty" bson:"url,omitempty"`

	// Base64Data represents the binary data in Base64 encoded string format.
	Base64Data *string `json:"base64data,omitempty" bson:"base64data,omitempty"`

	// MIMEType is the mime type , eg."image/png",""audio/wav" etc.
	MIMEType string `json:"mime_type,omitempty" bson:"mime_type,omitempty"`

	// Extra is used to store extra information.
	Extra map[string]any `json:"extra,omitempty" bson:"extra,omitempty"`
}

// ImageURLDetail is the detail of the image url.
type ImageURLDetail string

const (
	// ImageURLDetailHigh means the high quality image url.
	ImageURLDetailHigh ImageURLDetail = "high"
	// ImageURLDetailLow means the low quality image url.
	ImageURLDetailLow ImageURLDetail = "low"
	// ImageURLDetailAuto means the auto quality image url.
	ImageURLDetailAuto ImageURLDetail = "auto"
)
