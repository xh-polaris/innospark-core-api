package graph

import (
	"context"
	"fmt"

	"github.com/google/wire"
	mmsg "github.com/xh-polaris/innospark-core-api/biz/infra/mapper/message"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util/logx"
)

// HistoryDomain 是历史记录域, 负责维护对话的上下文
type HistoryDomain struct {
	MsgMapper mmsg.MongoMapper
}

var HistoryDomainSet = wire.NewSet(wire.Struct(new(HistoryDomain), "*"))

func (d *HistoryDomain) RetrieveHistory(ctx context.Context, relay *RelayContext) (mmsgs []*mmsg.Message, err error) {
	// 获取历史记录
	mmsgs, err = d.MsgMapper.AllMessage(ctx, relay.ConversationId.Hex())
	return
}

func (d *HistoryDomain) StoreHistory(ctx context.Context, relay *RelayContext) (err error) {
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
		logx.Error("[domain message] process history option err: %v", err)
		return err
	}

	// 用户消息
	if err = d.MsgMapper.CreateNewMessage(context.WithoutCancel(ctx), relay.UserMessage); err != nil {
		logx.Error("[domain message] store user message err: %v", err)
	}
	completeAssistantMsg(relay)
	// 模型消息
	if err = d.MsgMapper.CreateNewMessage(context.WithoutCancel(ctx), relay.MessageInfo.AssistantMessage); err != nil {
		logx.Error("[domain message] store assistant message err: %v", err)
	}
	return
}

func completeAssistantMsg(relay *RelayContext) {
	am := relay.MessageInfo.AssistantMessage
	am.Content, am.Ext = relay.MessageInfo.Text, &mmsg.Ext{
		BotState: fmt.Sprintf("{\"model\":\"%s\",\"bot_id\":\"%s\",\"bot_name\":\"%s\"}", relay.ModelInfo.Model, relay.ModelInfo.BotId, relay.ModelInfo.BotName),
		Brief:    relay.MessageInfo.Text,
		Think:    relay.MessageInfo.Think,
		Suggest:  relay.MessageInfo.Suggest,
	}
}
