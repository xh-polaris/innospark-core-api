package graph

import (
	"github.com/xh-polaris/innospark-core-api/biz/domain/info"
	"github.com/xh-polaris/innospark-core-api/biz/domain/msg"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	mmsg "github.com/xh-polaris/innospark-core-api/biz/infra/mapper/message"
)

const (
	Default     = "default"
	Regen       = "regen"
	Replace     = "replace"
	SelectRegen = "select_regen"
)

func DoCompletionOption(relay *info.RelayContext, his []*mmsg.Message) ([]*mmsg.Message, error) {
	opt := relay.CompletionOptions
	opt.Typ = Default
	// 据自定义对话选项, 对消息进行处理
	switch {
	case opt.IsRegen: // 重新生成, 覆盖掉最新的模型输出, 生成regen_list, 不需要增添user message
		var regens []*mmsg.Message
		relay.ReplyId = *opt.ReplyId
		for _, msg := range his { // 将此前同一个replyId且不为空的消息置为空 TODO: 暂时没有覆盖模型的多模态输出(因为没有)
			if msg.ReplyId.Hex() == *opt.ReplyId && msg.Content != "" {
				msg.Content = ""
				regens = append(regens, msg)
			} else if msg.Role == cst.UserEnum && msg.Content != "" { // 找到的第一个用户消息
				break
			}
		}
		opt.Typ, opt.RegenList = Regen, regens // 保存regen_list
	case opt.IsReplace: // 替换最新的一条用户消息, 实际是将最近一轮有效对话设为空且不保留, 需要新的user message
		opt.Typ = Replace
		for _, msg := range his {
			if msg.Content != "" {
				msg.Content = ""
				opt.ReplaceList = append(opt.ReplaceList, msg)
			}
			if len(opt.ReplaceList) >= 2 {
				break
			}
		}
	case opt.SelectedRegenId != nil: // 选择一个重新生成的结果, 并开始新的对话, 需要增加用户消息
		opt.Typ = SelectRegen
		reply := his[0].ReplyId
		for _, msg := range his { // 只保留一个regen, 其余清空
			if msg.ReplyId != reply {
				break
			}
			if msg.MessageId.Hex() == *opt.SelectedRegenId {
				msg.Content = msg.Ext.Brief
			} else {
				msg.Content = ""
			}
			opt.SelectRegenList = append(opt.SelectRegenList, msg)
		}
	}
	if !opt.IsRegen { // 加上用户消息
		um := msg.UserMMsg(relay, len(his))
		his = append([]*mmsg.Message{um}, his...)
		relay.UserMessage = um
		relay.ReplyId = um.MessageId.Hex()
	}
	relay.MessageInfo.AssistantMessage = msg.NewModelMsg(relay, len(his))
	if err := relay.SSEEvent(relay.MetaEvent()); err != nil { // 元数据事件
		return nil, err
	}
	if err := relay.SSEEvent(relay.ModelEvent()); err != nil { // 模型信息事件
		return nil, err
	}
	return his, nil
}
