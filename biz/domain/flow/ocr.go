package flow

import (
	"context"
	"github.com/cloudwego/eino/schema"
	"github.com/xh-polaris/innospark-core-api/biz/domain/state"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util"
	"strings"
)

func DoOCR(ctx context.Context, baseURL string, input []*schema.Message) (_ []*schema.Message, err error) {
	var st *state.RelayContext
	if st, err = util.GetState[*state.RelayContext](ctx); err != nil {
		return
	}

	var tex strings.Builder
	var tmp string

	for _, at := range st.Info.OriginMessage.Attaches {
		if util.IsImg(at) {
			if tmp, err = util.OCR(ctx, baseURL, at, util.URLTransfer); err != nil {
				return input, err
			}
			tex.Write([]byte(tmp))
		}
	}

	st.Info.UserMessage.Ext.Tex = tex.String()
	return input, nil
}
