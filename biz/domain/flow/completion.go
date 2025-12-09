package flow

import (
	"context"
	"errors"

	"github.com/xh-polaris/innospark-core-api/biz/domain/interaction"
	"github.com/xh-polaris/innospark-core-api/biz/domain/memory"
	"github.com/xh-polaris/innospark-core-api/biz/domain/message"
	"github.com/xh-polaris/innospark-core-api/biz/domain/state"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	mmsg "github.com/xh-polaris/innospark-core-api/biz/infra/mapper/message"
	"github.com/xh-polaris/innospark-core-api/pkg/ctxcache"
	"github.com/xh-polaris/innospark-core-api/pkg/logs"
)

func DoCompletions(ctx context.Context, st *state.RelayContext, memory *memory.MemoryManager) (err error) {
	var history []*mmsg.Message

	ctx = ctxcache.Init(ctx)
	ctxcache.Store(ctx, cst.CtxState, st)

	defer st.Close() // 释放状态中资源
	// 获取记忆
	if history, err = memory.RetrieveMemory(ctx, st); err != nil {
		return err
	}
	// 处理配置项
	if history, err = DoCompletionOption(st, history); err != nil {
		return err
	}
	// 转换存储域消息为模型域消息
	messages := message.MMsgToEMsgList(history)
	// 构建模型
	if err = BuildChatModel(ctx, st, messages); err != nil {
		return err
	}
	subCtx, cancel := context.WithCancel(ctx)
	st.CancelFunc = cancel
	// 构建图并执行, 有效数据会通过st.EventStream传递给interaction
	if _, err = StreamExecute(BuildFlow(st), subCtx, messages); err != nil {
		return err
	}

	// 收集事件处理后响应给前端
	inter := interaction.NewInteraction(st)
	defer func() {
		if ice := inter.Close(); ice != nil {
			logs.CtxErrorf(ctx, "close interaction error: %s", ice)
		}
	}()
	if err = inter.HandleEvent(subCtx); err != nil && !errors.Is(err, interaction.Interrupt) {
		return err
	}
	// 除中断外错误均不存储历史记录
	if err = memory.StoreHistory(ctx, st); err != nil {
		return err
	}
	// 结束消息
	if err = inter.EndEvent(); err != nil {
		logs.CtxErrorf(ctx, "end event error: %s", err)
	}
	return nil
}
