package graph

import (
	"context"
	"fmt"

	"github.com/google/wire"
	"github.com/xh-polaris/innospark-core-api/biz/domain/info"
	mmsg "github.com/xh-polaris/innospark-core-api/biz/infra/mapper/message"
	"github.com/xh-polaris/innospark-core-api/pkg/errorx"
	"github.com/xh-polaris/innospark-core-api/pkg/logs"
)

// HistoryDomain 是历史记录域, 负责维护对话的上下文
type HistoryDomain struct {
	MsgMapper mmsg.MongoMapper
}

var HistoryDomainSet = wire.NewSet(wire.Struct(new(HistoryDomain), "*"))

func (d *HistoryDomain) RetrieveHistory(ctx context.Context, relay *info.RelayContext) (mmsgs []*mmsg.Message, err error) {
	// 获取历史记录
	mmsgs, err = d.MsgMapper.AllMessage(ctx, relay.ConversationId.Hex())
	return
}

func (d *HistoryDomain) StoreHistory(ctx context.Context, relay *info.RelayContext) (err error) {
	var update []*mmsg.Message
	switch relay.CompletionOptions.Typ {
	case Regen:
		update = append(update, relay.CompletionOptions.RegenList...)
	case Replace:
		update = append(update, relay.CompletionOptions.ReplaceList...)
	case SelectRegen:
		update = append(update, relay.CompletionOptions.SelectRegenList...)
	}
	if err = d.MsgMapper.UpdateMany(ctx, update); err != nil {
		logs.Errorf("[domain message] process history option err: %s", errorx.ErrorWithoutStack(err))
		return err
	}

	// 用户消息
	if err = d.MsgMapper.CreateNewMessage(context.WithoutCancel(ctx), relay.UserMessage); err != nil {
		logs.Errorf("[domain message] store user message err: %s", errorx.ErrorWithoutStack(err))
	}
	completeAssistantMsg(relay)
	// 模型消息
	if err = d.MsgMapper.CreateNewMessage(context.WithoutCancel(ctx), relay.MessageInfo.AssistantMessage); err != nil {
		logs.Errorf("[domain message] store assistant message err: %s", errorx.ErrorWithoutStack(err))
	}
	return
}

func completeAssistantMsg(relay *info.RelayContext) {
	am := relay.MessageInfo.AssistantMessage
	am.Content, am.Ext = relay.MessageInfo.Text, &mmsg.Ext{ // 模型信息和基本内容
		BotState: fmt.Sprintf("{\"model\":\"%s\",\"bot_id\":\"%s\",\"bot_name\":\"%s\"}", relay.ModelInfo.Model, relay.ModelInfo.BotId, relay.ModelInfo.BotName),
		Brief:    relay.MessageInfo.Text,
		Think:    relay.MessageInfo.Think,
		Suggest:  relay.MessageInfo.Suggest,
	}
	if relay.SearchInfo != nil { // 搜索信息
		am.Ext.Cite = relay.SearchInfo.Cite
	}
	if relay.Sensitive.Hits != nil && len(relay.Sensitive.Hits) > 0 { // 敏感词信息
		am.Content = ""
		am.Ext.Sensitive = true
	}
	if relay.ResponseMeta != nil { // 用量信息
		am.Ext.Usage = &mmsg.Usage{
			PromptTokens: relay.ResponseMeta.Usage.PromptTokens,
			PromptTokenDetails: &mmsg.PromptTokenDetails{
				CachedTokens: relay.ResponseMeta.Usage.PromptTokenDetails.CachedTokens,
			},
			CompletionTokens: relay.ResponseMeta.Usage.CompletionTokens,
			TotalTokens:      relay.ResponseMeta.Usage.TotalTokens,
		}
	}
	am.Ext.Code = relay.MessageInfo.Code
}
