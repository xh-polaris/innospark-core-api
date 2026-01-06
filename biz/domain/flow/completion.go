package flow

import (
	"context"
	"errors"
	"sync"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/xh-polaris/innospark-core-api/biz/conf"
	"github.com/xh-polaris/innospark-core-api/biz/domain/interaction"
	"github.com/xh-polaris/innospark-core-api/biz/domain/memory"
	"github.com/xh-polaris/innospark-core-api/biz/domain/message"
	dmodel "github.com/xh-polaris/innospark-core-api/biz/domain/model"
	"github.com/xh-polaris/innospark-core-api/biz/domain/state"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	mmsg "github.com/xh-polaris/innospark-core-api/biz/infra/mapper/message"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util"
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
	if messages, err = BuildChatModel(ctx, st, messages); err != nil {
		return err
	}
	subCtx, cancel := context.WithCancel(ctx)
	st.CancelFunc = cancel

	// 收集事件处理后响应给前端
	inter := interaction.NewInteraction(st)
	defer func() {
		if ice := inter.Close(); ice != nil {
			logs.CtxErrorf(ctx, "close interaction error: %s", ice)
		}
	}()
	// 特殊agent第一次对话, 表单提取
	if needExtract(st, messages) {
		if err = extractInfo(ctx, inter, subCtx, st, messages); err != nil {
			return err
		}
		// 结束消息
		if err = inter.EndEvent(); err != nil {
			logs.CtxErrorf(subCtx, "end event error: %s", err)
		}
		return nil
	}

	var wg sync.WaitGroup
	var err1, err2 error
	wg.Add(2)
	// 收集图中事件
	go func() {
		defer wg.Done()
		err1 = inter.HandleEvent(subCtx)
	}()
	// 构建图并执行, 有效数据会通过st.EventStream传递给interaction
	go func() {
		defer wg.Done()
		_, err2 = StreamExecuteFlow(BuildFlow(st), subCtx, messages)
	}()
	wg.Wait()

	if (err1 != nil && !errors.Is(err1, interaction.Interrupt)) || (err2 != nil && !errors.Is(err2, interaction.Interrupt)) {
		return err
	}

	if needSuggest(st) {
		wg.Add(2)
		// 收集图中事件
		go func() {
			defer wg.Done()
			err1 = inter.HandleEvent(subCtx)
		}()
		// 构建图并执行, 有效数据会通过st.EventStream传递给interaction
		go func() {
			defer wg.Done()
			_, err2 = StreamExecuteSuggest(BuildSuggest(st), subCtx, st)
		}()
		wg.Wait()
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

func needExtract(st *state.RelayContext, messages []*schema.Message) bool {
	if len(st.Info.ModelInfo.BotId) > 6 && conf.GetConfig().CoTea.AgentPrompts[st.Info.ModelInfo.BotId[6:]].ExtractInfo && len(messages) <= 2 {
		if v, ok := st.Info.Ext["info.completed"]; ok && v == "false" {
			return true
		}
	}
	return false
}

func extractInfo(ctx context.Context, inter *interaction.Interaction, subCtx context.Context, st *state.RelayContext, messages []*schema.Message) (err error) {
	// 提取信息事件
	eie, err := interaction.ExtractInfoEvent()
	if err != nil {
		logs.CtxErrorf(ctx, "extract info event error: %s", err)
		return err
	}
	if err = inter.SSE.Write(eie.SSEEvent); err != nil {
		logs.CtxErrorf(subCtx, "send extract info event error: %s", err)
		return err
	}

	// 提取信息
	info := st.Info.Ext
	if info, err = extract(ctx, st, messages, info); err != nil {
		logs.CtxErrorf(ctx, "extract info error: %s", err)
		return err
	}
	// 校验信息是否完全
	var completed = true
	for _, i := range conf.GetConfig().CoTea.AgentPrompts[st.Info.ModelInfo.BotId[6:]].Key {
		if _, ok := info[i]; !ok {
			completed = false
		}
	}
	resp := make(map[string]any)
	resp["completed"] = completed
	resp["botId"] = st.Info.ModelInfo.BotId
	resp["keys"] = conf.GetConfig().CoTea.AgentPrompts[st.Info.ModelInfo.BotId[6:]].Key
	resp["info"] = info

	// 提取结束
	eiee, err := interaction.ExtractInfoEndEvent(resp)
	if err != nil {
		return err
	}
	if err = inter.SSE.Write(eiee.SSEEvent); err != nil {
		logs.CtxErrorf(subCtx, "send extract info event end error: %s", err)
		return err
	}
	return nil
}

func extract(ctx context.Context, st *state.RelayContext, messages []*schema.Message, info map[string]string) (_ map[string]string, err error) {
	var (
		m       model.ToolCallingChatModel
		msg     *schema.Message
		newInfo map[string]string
	)
	// 提取表单
	if m, err = dmodel.NewDoubao15Pro32KChatModel(ctx, st.Info.UserId.Hex(), ""); err != nil {
		logs.CtxErrorf(ctx, "new model error: %s", err)
		return nil, err
	}
	sys := schema.SystemMessage(conf.GetConfig().CoTea.AgentPrompts[st.Info.ModelInfo.BotId[6:]].ExtractPrompt)
	messages[len(messages)-1] = sys
	if msg, err = m.Generate(ctx, messages); err != nil {
		return nil, err
	}
	// 解析信息
	jsonStr := util.PurifyJson(msg.Content)
	if err = sonic.Unmarshal([]byte(jsonStr), &newInfo); err != nil {
		return info, nil
	}
	// 将newInfo中不为空的值赋给info
	for k, v := range newInfo {
		if v != "" {
			info[k] = v
		}
	}
	return info, nil
}
