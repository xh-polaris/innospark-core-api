package prompt_inject

import (
	"context"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
	"github.com/xh-polaris/innospark-core-api/biz/domain/info"
	"github.com/xh-polaris/innospark-core-api/biz/infra/config"
)

// CoTeaSysInject 注入CoTea所需系统提示词
// 消息倒序, 加入最早的系统提示词也就是放在最后面
func CoTeaSysInject(ctx context.Context, in []*schema.Message, info *info.RelayContext) ([]*schema.Message, error) {
	bot := info.ModelInfo.BotId[6:]
	template := config.GetConfig().CoTea.AgentPrompts[bot]

	injectInfo := make(map[string]any)
	for _, k := range template.Key {
		if v, ok := info.Ext[k]; ok {
			injectInfo[k] = v
		}
	}
	sys, err := prompt.FromMessages(schema.FString, &schema.Message{Role: schema.System, Content: template.Template}).Format(ctx, injectInfo)
	if err != nil {
		return nil, err
	}
	in = append(in, sys...)
	return in, nil
}
