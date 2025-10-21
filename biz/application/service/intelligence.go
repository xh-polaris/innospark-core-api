package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/wire"
	"github.com/xh-polaris/innospark-core-api/biz/application/dto/basic"
	"github.com/xh-polaris/innospark-core-api/biz/application/dto/core_api"
	"github.com/xh-polaris/innospark-core-api/biz/infra/config"
	"github.com/xh-polaris/innospark-core-api/biz/infra/cst"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util/errorx"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util/httpx"
)

type IIntelligenceService interface {
	ListIntelligence(ctx context.Context, req *core_api.ListIntelligenceReq) (*core_api.ListIntelligenceResp, error)
	GetIntelligenceInfo(ctx context.Context, req *core_api.GetIntelligenceReq) (*core_api.GetIntelligenceResp, error)
}

type IntelligenceService struct {
}

var IntelligenceServiceSet = wire.NewSet(
	wire.Struct(new(IntelligenceService), "*"),
	wire.Bind(new(IIntelligenceService), new(*IntelligenceService)),
)

func (i *IntelligenceService) ListIntelligence(ctx context.Context, req *core_api.ListIntelligenceReq) (*core_api.ListIntelligenceResp, error) {
	header := http.Header{}
	header.Set("Cookie", config.GetConfig().Coze.GetCookie())
	header.Set("Content-Type", "application/json")
	var listBody = map[string]interface{}{
		"space_id":      "7558114583873847296",
		"name":          "",
		"has_published": true,
		"recently_open": false,
		"status":        []int{1, 3, 4},
		"types":         []int{1, 2},
		"search_scope":  0,
		"order_by":      2,
	}
	if req.Page != nil && req.Page.Size != nil {
		listBody["size"] = req.Page.Size
	} else {
		listBody["size"] = 100
	}
	if req.Page != nil && req.Page.Cursor != nil {
		listBody["cursor_id"] = req.Page.Cursor
	}

	url := "https://coze.aiecnu.net/api/intelligence_api/search/get_draft_intelligence_list"
	resp, err := httpx.GetHttpClient().Post(url, header, listBody)
	if err != nil {
		return nil, errorx.WrapByCode(err, cst.SynapseErrCode, errorx.KV("url", url))
	}

	if resp["code"].(float64) != 0 && resp["code"].(float64) != 700012006 {
		return nil, cst.New(999, resp["msg"].(string))
	}
	if resp["code"].(float64) == 700012006 {
		header.Set("Cookie", config.GetConfig().Coze.RefreshCookie())
		resp, err = httpx.GetHttpClient().Post(url, header, listBody)
		if err != nil {
			return nil, errorx.WrapByCode(err, cst.SynapseErrCode, errorx.KV("url", url))
		}
		if resp["code"].(float64) != 0 {
			return nil, cst.New(999, resp["msg"].(string))
		}
	}
	data := resp["data"].(map[string]interface{})
	var intelligences []*core_api.Intelligence
	ins := data["intelligences"].([]interface{})
	for _, in := range ins {
		intelligence := in.(map[string]interface{})["basic_info"].(map[string]interface{})
		fullname := intelligence["name"].(string)
		s := strings.Split(fullname, "|")
		intelligences = append(intelligences, &core_api.Intelligence{
			Id:          intelligence["id"].(string),
			Name:        s[1],
			Type:        s[0],
			Description: intelligence["description"].(string),
			IconUrl:     util.ToCDN(intelligence["icon_url"].(string), "coze-studio-statics.aiecnu.net"),
			CreateTime:  intelligence["create_time"].(string),
			UpdateTime:  intelligence["update_time"].(string),
			PublishTime: intelligence["publish_time"].(string),
		})
	}
	liResp := &core_api.ListIntelligenceResp{
		Resp: &basic.Response{
			Code: int32(int(resp["code"].(float64))),
			Msg:  resp["msg"].(string),
		},
		Intelligences: intelligences,
		HasMore:       data["has_more"].(bool),
	}
	if ncid, ok := data["next_cursor_id"]; ok {
		liResp.NextCursor = ncid.(string)
	}
	return liResp, nil
}

func (i *IntelligenceService) GetIntelligenceInfo(ctx context.Context, req *core_api.GetIntelligenceReq) (*core_api.GetIntelligenceResp, error) {
	header := http.Header{}
	header.Set("Authorization", "Bearer"+config.GetConfig().Coze.PAT)
	header.Set("Content-Type", "application/json")
	url := fmt.Sprintf("https://coze.aiecnu.net/api/intelligence_api/intelligence/%s", req.GetId())
	resp, err := httpx.GetHttpClient().Get(url, header, nil)
	if err != nil {
		return nil, errorx.WrapByCode(err, cst.SynapseErrCode, errorx.KV("url", url))
	}
	if resp["code"].(float64) != 0 {
		return nil, cst.New(999, resp["msg"].(string))
	}
	data := resp["data"].(map[string]interface{})
	modelInfo := data["model_info"].(map[string]interface{})
	s := strings.Split(data["name"].(string), "|")
	var name, typ string
	if len(s) == 1 {
		name = s[0]
	} else {
		typ, name = s[0], s[1]
	}
	var questions []string
	for _, q := range data["onboarding_info"].(map[string]interface{})["suggested_questions"].([]interface{}) {
		questions = append(questions, q.(string))
	}
	info := &core_api.IntelligenceInfo{
		BotId:       data["bot_id"].(string),
		Name:        name,
		Description: data["description"].(string),
		IconUrl:     util.ToCDN(data["icon_url"].(string), "coze-studio-statics.aiecnu.net"),
		CreateTime:  0,
		UpdateTime:  0,
		Version:     data["version"].(string),
		PromptInfo:  &core_api.PromptInfo{Prompt: data["prompt_info"].(map[string]interface{})["prompt"].(string)},
		OnboardingInfo: &core_api.OnboardingInfo{
			Prologue:                   data["onboarding_info"].(map[string]interface{})["prologue"].(string),
			SuggestedQuestionsShowMode: int32(data["onboarding_info"].(map[string]interface{})["suggested_questions_show_mode"].(float64)),
			SuggestedQuestions:         questions,
		},
		BotMode: int32(data["bot_mode"].(float64)),
		ModelInfo: &core_api.ModelInfo{
			ModelId:           modelInfo["model_id"].(string),
			Temperature:       float32(modelInfo["temperature"].(float64)),
			MaxTokens:         int32(modelInfo["max_tokens"].(float64)),
			TopP:              float32(modelInfo["top_p"].(float64)),
			ShortMemoryPolicy: &core_api.ShortMemoryPolicy{HistoryRound: int32(modelInfo["short_memory_policy"].(map[string]interface{})["history_round"].(float64))},
			ModelStyle:        int32(modelInfo["model_style"].(float64)),
		},
		PluginInfoList:       []*core_api.PluginInfo{},
		WorkflowInfoList:     []*core_api.WorkflowInfo{},
		ShortcutCommands:     "",
		DefaultUserInputType: "",
		SuggestReplyInfo:     "",
		BackgroundImageInfo:  "",
		Variables:            "",
		Type:                 typ,
	}
	return &core_api.GetIntelligenceResp{
		Resp: util.Success(),
		Info: info,
	}, nil
}
