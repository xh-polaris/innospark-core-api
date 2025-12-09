package memory

import (
	"context"
	"fmt"

	"github.com/xh-polaris/innospark-core-api/biz/domain/memory/history"
	"github.com/xh-polaris/innospark-core-api/biz/domain/state"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	mmsg "github.com/xh-polaris/innospark-core-api/biz/infra/mapper/message"
	"github.com/xh-polaris/innospark-core-api/pkg/errorx"
	"github.com/xh-polaris/innospark-core-api/pkg/logs"
)

var Memory *MemoryManager

// MemoryManager 管理大模型记忆
type MemoryManager struct {
	his *history.HistoryManager
}

func New(his *history.HistoryManager) *MemoryManager {
	Memory = &MemoryManager{his: his}
	return Memory
}

func (m *MemoryManager) RetrieveMemory(ctx context.Context, st *state.RelayContext) (mmsgs []*mmsg.Message, err error) {
	// 获取历史记录
	mmsgs, err = m.his.RetrieveMessage(ctx, st.Info.ConversationId.Hex(), 0)
	return
}

func (m *MemoryManager) StoreHistory(ctx context.Context, relay *state.RelayContext) (err error) {
	var update []*mmsg.Message
	info := relay.Info
	switch info.CompletionOptions.Typ {
	case cst.Regen:
		update = append(update, info.CompletionOptions.RegenList...)
	case cst.Replace:
		update = append(update, info.CompletionOptions.ReplaceList...)
	case cst.SelectRegen:
		update = append(update, info.CompletionOptions.SelectRegenList...)
	}
	if err = m.his.UpdateMessages(ctx, update); err != nil {
		logs.Errorf("[domain message] process history option err: %s", errorx.ErrorWithoutStack(err))
		return err
	}

	// 用户消息
	if err = m.his.AddMessage(context.WithoutCancel(ctx), info.UserMessage.ConversationId.Hex(), info.UserMessage); err != nil {
		logs.Errorf("[domain message] store user message err: %s", errorx.ErrorWithoutStack(err))
	}
	completeAssistantMMsg(relay)
	// 模型消息
	if err = m.his.AddMessage(context.WithoutCancel(ctx), info.MessageInfo.AssistantMessage.ConversationId.Hex(), info.MessageInfo.AssistantMessage); err != nil {
		logs.Errorf("[domain message] store assistant message err: %s", errorx.ErrorWithoutStack(err))
	}
	return
}

// 补完模型消息
func completeAssistantMMsg(st *state.RelayContext) {
	info := st.Info
	am := info.MessageInfo.AssistantMessage
	am.Content, am.Ext = info.MessageInfo.Text, &mmsg.Ext{ // 模型信息和基本内容
		BotState: fmt.Sprintf("{\"model\":\"%s\",\"bot_id\":\"%s\",\"bot_name\":\"%s\"}", info.ModelInfo.Model, info.ModelInfo.BotId, info.ModelInfo.BotName),
		Brief:    info.MessageInfo.Text,
		Think:    info.MessageInfo.Think,
		Suggest:  info.MessageInfo.Suggest,
	}
	if info.SearchInfo != nil { // 搜索信息
		am.Ext.Cite = info.SearchInfo.Cite
	}
	if info.Sensitive.Hits != nil && len(info.Sensitive.Hits) > 0 { // 敏感词信息
		am.Content = ""
		am.Ext.Sensitive = true
	}
	if info.ResponseMeta != nil { // 用量信息
		am.Ext.Usage = &mmsg.Usage{
			PromptTokens: info.ResponseMeta.Usage.PromptTokens,
			PromptTokenDetails: &mmsg.PromptTokenDetails{
				CachedTokens: info.ResponseMeta.Usage.PromptTokenDetails.CachedTokens,
			},
			CompletionTokens: info.ResponseMeta.Usage.CompletionTokens,
			TotalTokens:      info.ResponseMeta.Usage.TotalTokens,
		}
	}
	am.Ext.Code = info.MessageInfo.Code
}
