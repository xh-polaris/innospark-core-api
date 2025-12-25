package flow

import (
	"context"
	"strings"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
	"github.com/xh-polaris/innospark-core-api/biz/domain/state"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util"
)

func DoOCR(ctx context.Context, st *state.RelayContext, baseURL, prompts string, input []*schema.Message) (_ []*schema.Message, err error) {
	var (
		tmp string
		ocr strings.Builder
	)
	for _, at := range st.Info.OriginMessage.Attaches {
		if util.IsImg(at) {
			if tmp, err = util.OCR(ctx, baseURL, at, util.URLTransfer); err != nil {
				return nil, err
			}
			ocr.WriteString(tmp)
		}
	}
	st.Info.UserMessage.Ext.Ocr = ocr.String()
	// 将用户提问注入提示词模板中
	format, err := prompt.FromMessages(schema.FString, &schema.Message{Role: schema.User, Content: prompts}).Format(ctx,
		map[string]any{"ocr": st.Info.UserMessage.Ext.Ocr, "query": util.GetInputText(input[0])})
	if err != nil {
		return nil, err
	}
	input[0] = format[0]
	// 删除输入中的图片
	return input, nil
}
