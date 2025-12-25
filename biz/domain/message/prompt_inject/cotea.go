package prompt_inject

// Cotea 模式下的提示词注入

import (
	"context"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
	"github.com/xh-polaris/innospark-core-api/biz/conf"
	"github.com/xh-polaris/innospark-core-api/biz/domain/state"
)

// CoTeaSysInject 注入CoTea所需系统提示词
// 消息倒序, 加入最早的系统提示词也就是放在最后面
func CoTeaSysInject(ctx context.Context, in []*schema.Message, st *state.RelayContext) ([]*schema.Message, error) {
	bot := st.Info.ModelInfo.BotId[6:]
	template := conf.GetConfig().CoTea.AgentPrompts[bot]

	injectInfo := make(map[string]any)
	for _, k := range template.Key {
		injectInfo[k] = st.Info.Ext[k]
	}
	sys, err := prompt.FromMessages(schema.FString, &schema.Message{Role: schema.System, Content: template.Template}).Format(ctx, injectInfo)
	if err != nil {
		return nil, err
	}
	in = append(in, sys...)
	return in, nil
}
