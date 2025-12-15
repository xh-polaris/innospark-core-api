package message

// 不同邻域消息的转换

import (
	"github.com/cloudwego/eino/schema"
	"github.com/xh-polaris/innospark-core-api/biz/application/dto/core_api"
	mmsg "github.com/xh-polaris/innospark-core-api/biz/infra/mapper/message"
)

// MMsgToEMsgList 将 core_api.Message 切片转换为 eino/schema.Message 切片
func MMsgToEMsgList(messages []*mmsg.Message) (msgs []*schema.Message) {
	for _, msg := range messages {
		msgs = append(msgs, MMsgToEMsg(msg))
	}
	return
}

// MMsgToEMsg 将单个 core_api.Message 转换为 eino/schema.Message
func MMsgToEMsg(msg *mmsg.Message) *schema.Message {
	m := &schema.Message{
		Role:    schema.RoleType(mmsg.RoleItoS[msg.Role]),
		Content: msg.Content,
		Name:    msg.MessageId.Hex(),
	}
	if msg.Ext.ContentWithCite != nil { // 联网搜索到的内容
		if len(m.UserInputMultiContent) != 0 {
			for i := range m.UserInputMultiContent {
				if mmsg.ChatMessagePartType(m.UserInputMultiContent[i].Type) == mmsg.ChatMessagePartTypeText {
					m.UserInputMultiContent[i].Text = *msg.Ext.ContentWithCite
				}
			}
		} else {
			m.Content = *msg.Ext.ContentWithCite
		}
	}
	if len(msg.UserInputMultiContent) != 0 {
		for _, uimc := range msg.UserInputMultiContent {
			part := schema.MessageInputPart{Type: schema.ChatMessagePartType(uimc.Type)}
			switch uimc.Type {
			case mmsg.ChatMessagePartTypeText:
				part.Text = uimc.Text
			case mmsg.ChatMessagePartTypeImageURL:
				part.Image = &schema.MessageInputImage{
					MessagePartCommon: schema.MessagePartCommon{
						URL:        uimc.Image.URL,
						Base64Data: uimc.Image.Base64Data,
						MIMEType:   uimc.Image.MIMEType,
						Extra:      uimc.Image.Extra,
					},
					Detail: schema.ImageURLDetail(uimc.Image.Detail),
				}
			case mmsg.ChatMessagePartTypeAudioURL:
				part.Audio = &schema.MessageInputAudio{
					MessagePartCommon: schema.MessagePartCommon{
						URL:        uimc.Audio.URL,
						Base64Data: uimc.Audio.Base64Data,
						MIMEType:   uimc.Audio.MIMEType,
						Extra:      uimc.Audio.Extra,
					},
				}
			case mmsg.ChatMessagePartTypeVideoURL:
				part.Video = &schema.MessageInputVideo{
					MessagePartCommon: schema.MessagePartCommon{
						URL:        uimc.Video.URL,
						Base64Data: uimc.Video.Base64Data,
						MIMEType:   uimc.Video.MIMEType,
						Extra:      uimc.Video.Extra,
					},
				}
			case mmsg.ChatMessagePartTypeFileURL:
				part.File = &schema.MessageInputFile{
					MessagePartCommon: schema.MessagePartCommon{
						URL:        uimc.File.URL,
						Base64Data: uimc.File.Base64Data,
						MIMEType:   uimc.File.MIMEType,
						Extra:      uimc.File.Extra,
					},
				}
			}
			m.UserInputMultiContent = append(m.UserInputMultiContent, part)
		}
	}
	if len(msg.AssistantGenMultiContent) != 0 {
		for _, agmc := range msg.AssistantGenMultiContent {
			part := schema.MessageOutputPart{Type: schema.ChatMessagePartType(agmc.Type)}
			switch agmc.Type {
			case mmsg.ChatMessagePartTypeText:
				part.Text = agmc.Text

			case mmsg.ChatMessagePartTypeImageURL:
				part.Image = &schema.MessageOutputImage{
					MessagePartCommon: schema.MessagePartCommon{
						URL:        agmc.Image.URL,
						Base64Data: agmc.Image.Base64Data,
						MIMEType:   agmc.Image.MIMEType,
						Extra:      agmc.Image.Extra,
					},
				}

			case mmsg.ChatMessagePartTypeAudioURL:
				part.Audio = &schema.MessageOutputAudio{
					MessagePartCommon: schema.MessagePartCommon{
						URL:        agmc.Audio.URL,
						Base64Data: agmc.Audio.Base64Data,
						MIMEType:   agmc.Audio.MIMEType,
						Extra:      agmc.Audio.Extra,
					},
				}

			case mmsg.ChatMessagePartTypeVideoURL:
				part.Video = &schema.MessageOutputVideo{
					MessagePartCommon: schema.MessagePartCommon{
						URL:        agmc.Video.URL,
						Base64Data: agmc.Video.Base64Data,
						MIMEType:   agmc.Video.MIMEType,
						Extra:      agmc.Video.Extra,
					},
				}
			}
			m.AssistantGenMultiContent = append(m.AssistantGenMultiContent, part)
		}
	}
	return m
}

func MMsgToFMsgList(messages []*mmsg.Message) (msgs []*core_api.FullMessage) {
	for _, msg := range messages {
		msgs = append(msgs, MMsgToFMsg(msg))
	}
	return
}

func MMsgToFMsg(msg *mmsg.Message) *core_api.FullMessage {
	fm := &core_api.FullMessage{
		ConversationId: msg.ConversationId.Hex(),
		SectionId:      msg.SectionId.Hex(),
		MessageId:      msg.MessageId.Hex(),
		Index:          msg.Index,
		Status:         msg.Status,
		CreateTime:     msg.CreateTime.Unix(),
		MessageType:    msg.MessageType,
		ContentType:    msg.ContentType,
		Content:        msg.Content,
		Ext: &core_api.Ext{
			BotState:  msg.Ext.BotState,
			Brief:     msg.Ext.Brief,
			Think:     msg.Ext.Think,
			Suggest:   msg.Ext.Suggest,
			Cite:      MCiteToFCiteList(msg.Ext.Cite),
			Code:      MCodeToFCodeList(msg.Ext.Code),
			Sensitive: msg.Ext.Sensitive,
			Usage:     MUsageToFUsage(msg.Ext.Usage),
		},
		Feedback: msg.Feedback,
		UserType: msg.Role,
	}
	if !msg.ReplyId.IsZero() {
		reply := msg.ReplyId.Hex()
		fm.ReplyId = &reply
	}
	if msg.UserInputMultiContent != nil {
		for _, c := range msg.UserInputMultiContent {
			part := &core_api.MessageInputPart{
				Type: string(c.Type),
			}
			switch c.Type {
			case mmsg.ChatMessagePartTypeText:
				part.Text = &c.Text
			case mmsg.ChatMessagePartTypeImageURL:
				if c.Image != nil {
					part.Image = &core_api.MessageInputImage{
						Url:        c.Image.URL,
						Base64Data: c.Image.Base64Data,
						MimeType:   &c.Image.MIMEType,
						Detail:     string(c.Image.Detail),
					}
				}
			case mmsg.ChatMessagePartTypeAudioURL:
				if c.Audio != nil {
					part.Audio = &core_api.MessageInputAudio{
						Url:        c.Audio.URL,
						Base64Data: c.Audio.Base64Data,
						MimeType:   &c.Audio.MIMEType,
					}
				}
			case mmsg.ChatMessagePartTypeVideoURL:
				if c.Video != nil {
					part.Video = &core_api.MessageInputVideo{
						Url:        c.Video.URL,
						Base64Data: c.Video.Base64Data,
						MimeType:   &c.Video.MIMEType,
					}
				}
			case mmsg.ChatMessagePartTypeFileURL:
				if c.File != nil {
					part.File = &core_api.MessageInputFile{
						Url:        c.File.URL,
						Base64Data: c.File.Base64Data,
						MimeType:   &c.File.MIMEType,
					}
				}
			}
			fm.UserInputMultiContent = append(fm.UserInputMultiContent, part)
		}
	}
	if msg.AssistantGenMultiContent != nil {
		for _, c := range msg.AssistantGenMultiContent {
			part := &core_api.MessageOutputPart{
				Type: string(c.Type),
			}
			switch c.Type {
			case mmsg.ChatMessagePartTypeText:
				part.Text = &c.Text
			case mmsg.ChatMessagePartTypeImageURL:
				if c.Image != nil {
					part.Image = &core_api.MessageOutputImage{
						Url:        c.Image.URL,
						Base64Data: c.Image.Base64Data,
						MimeType:   &c.Image.MIMEType,
					}
				}
			case mmsg.ChatMessagePartTypeAudioURL:
				if c.Audio != nil {
					part.Audio = &core_api.MessageOutputAudio{
						Url:        c.Audio.URL,
						Base64Data: c.Audio.Base64Data,
						MimeType:   &c.Audio.MIMEType,
					}
				}
			case mmsg.ChatMessagePartTypeVideoURL:
				if c.Video != nil {
					part.Video = &core_api.MessageOutputVideo{
						Url:        c.Video.URL,
						Base64Data: c.Video.Base64Data,
						MimeType:   &c.Video.MIMEType,
					}
				}
			}
			fm.AssistantGenMultiContent = append(fm.AssistantGenMultiContent, part)
		}
	}
	return fm
}

func MCiteToFCiteList(cites []*mmsg.Cite) (cs []*core_api.Cite) {
	for _, c := range cites {
		cs = append(cs, MCiteToFCite(c))
	}
	return
}

func MCiteToFCite(cite *mmsg.Cite) *core_api.Cite {
	return &core_api.Cite{
		Index:         cite.Index,
		Name:          cite.Name,
		Url:           cite.URL,
		Snippet:       cite.Snippet,
		SiteName:      cite.SiteName,
		SiteIcon:      cite.SiteIcon,
		DatePublished: cite.DatePublished,
	}
}

func MCodeToFCodeList(codes []*mmsg.Code) (cs []*core_api.Code) {
	for _, c := range codes {
		cs = append(cs, MCodeToFCode(c))
	}
	return
}
func MCodeToFCode(code *mmsg.Code) *core_api.Code {
	return &core_api.Code{
		Index:    code.Index,
		CodeType: code.CodeType,
		Code:     code.Code,
	}
}

func MUsageToFUsage(usage *mmsg.Usage) *core_api.Usage {
	if usage == nil {
		return nil
	}
	return &core_api.Usage{
		PromptTokens:       int64(usage.PromptTokens),
		PromptTokenDetails: &core_api.Usage_PromptTokenDetails{CachedTokens: int64(usage.PromptTokenDetails.CachedTokens)},
		CompletionTokens:   int64(usage.CompletionTokens),
		TotalTokens:        int64(usage.TotalTokens),
	}
}
