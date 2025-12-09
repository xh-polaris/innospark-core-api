package interaction

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/cloudwego/eino/schema"
	"github.com/cloudwego/hertz/pkg/protocol/sse"
	"github.com/xh-polaris/innospark-core-api/biz/conf"
	"github.com/xh-polaris/innospark-core-api/biz/domain/state/info"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	mmsg "github.com/xh-polaris/innospark-core-api/biz/infra/mapper/message"
	"github.com/xh-polaris/innospark-core-api/biz/infra/mapper/user"
	"github.com/xh-polaris/innospark-core-api/pkg/ac"
	"github.com/xh-polaris/innospark-core-api/pkg/errorx"
	"github.com/xh-polaris/innospark-core-api/pkg/logs"
)

// Interaction 交互域, 负责下游消息组装与转换, 并响应给前端
type Interaction struct {
	info       *info.Info               // 状态上下文
	containers map[int]*strings.Builder // 记录不同类型内容
	code       []*strings.Builder       // 记录代码内容
	codeTyp    []string                 // 记录代码类型
}

// NewInteraction 创建交互
func NewInteraction(info *info.Info) (i *Interaction) {
	i = &Interaction{info: info, containers: map[int]*strings.Builder{
		cst.EventMessageContentTypeText:    {}, // 文本消息
		cst.EventMessageContentTypeThink:   {}, // 思考消息
		cst.EventMessageContentTypeSuggest: {}, // 建议消息
	}}
	return
}

// AssembleModelEvents 组装模型事件, 将模型消息转换为ChatEvent
// 同时兼顾消息内容收集和敏感词检测
func (i *Interaction) AssembleModelEvents(mr *schema.StreamReader[*schema.Message], sw *schema.StreamWriter[*sse.Event]) {
	defer mr.Close() // 关闭模型读
	defer sw.Close() // 关闭sse写 [暂时模型消息结束就关闭sse, 后续有其他过程再修改]

	var (
		cnt int
		err error
		msg *schema.Message
		sb  strings.Builder
	)

	defer i.collect()                           // 收集各类型消息
	defer func() { sw.Send(i.EndEvent(err)) }() // 发送结束消息
	defer func(sb *strings.Builder) {           // 最终校验是否有违禁词
		if i.sensitive(sb) {
			sw.Send(i.ErrorEvent(999, fmt.Sprintf("这个话题暂时还不能聊哦, 也请不要引导我聊敏感话题否则会被封禁哦"))) // 写入敏感错误
		}
	}(&sb)

	for {
		select {
		case <-i.info.SSE.Done: // sse中断
			i.info.ModelCancel() // 提前关闭模型
			return
		default:
			// 获取模型消息
			msg, err = mr.Recv()
			if err != nil {
				logs.CondErrorf(!errors.Is(err, io.EOF), "[graph transformer] recv err:%s", errorx.ErrorWithoutStack(err))
				return
			}
			if msg.ResponseMeta != nil { // 收集用量信息
				i.info.ResponseMeta = msg.ResponseMeta
			}

			// 精化消息
			refine := &info.RefineContent{}
			content, typ := refine.SetContentWithTyp(msg.Content, msg.Extra[cst.EventMessageContentType].(int))

			// 收集信息
			sb.WriteString(content)
			if typ == cst.EventMessageContentTypeCodeType {
				i.codeTyp = append(i.codeTyp, content)
				i.code = append(i.code, &strings.Builder{})
			} else if typ == cst.EventMessageContentTypeCode {
				i.code[len(i.code)-1].WriteString(content)
			} else {
				i.containers[typ].WriteString(content)
			}

			// 检查违禁词
			cnt++
			if needSensitiveCheck(cnt) {
				if i.sensitive(&sb) {
					sw.Send(i.ErrorEvent(999, fmt.Sprintf("这个话题暂时还不能聊哦, 也请不要引导我聊敏感话题否则会被封禁哦")))
					return
				}
			}
			sw.Send(i.ChatEvent(refine, typ))
		}
	}
}

// collect 收集各类消息内容
func (i *Interaction) collect() {
	// 存储各类消息内容
	i.info.MessageInfo.Text = i.containers[cst.EventMessageContentTypeText].String()       // 文本
	i.info.MessageInfo.Think = i.containers[cst.EventMessageContentTypeThink].String()     // 思考
	i.info.MessageInfo.Suggest = i.containers[cst.EventMessageContentTypeSuggest].String() // 建议

	// 构造代码段信息
	codes := make([]*mmsg.Code, len(i.code))
	for index, code := range i.code {
		codes[index] = &mmsg.Code{
			Index:    int32(index),
			CodeType: i.codeTyp[index],
			Code:     code.String(),
		}
	}
	i.info.MessageInfo.Code = codes
}

// sensitive 检查是否有违禁词
func (i *Interaction) sensitive(sb *strings.Builder) bool {
	sensitive, hits := ac.AcSearch(sb.String(), true, cst.SensitivePost)
	if sensitive {
		i.info.Sensitive.Hits = hits
		i.info.ModelCancel()
		if err := user.Mapper.Warn(context.Background(), i.info.UserId.Hex()); err != nil {
			logs.Errorf("[interaction] warn err: %v", err)
		}
	}
	return sensitive
}

// 判断是否需要检查违禁词
func needSensitiveCheck(cnt int) bool {
	return cnt%conf.GetConfig().Sensitive.SensitiveStreamGap == 0
}
