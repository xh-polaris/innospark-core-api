package graph

// tool.search.bocha 使用博查API的搜索工具

import (
	"context"
	"encoding/json"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/xh-polaris/innospark-core-api/biz/domain/state"
	"github.com/xh-polaris/innospark-core-api/biz/domain/state/info"
	mmsg "github.com/xh-polaris/innospark-core-api/biz/infra/mapper/message"
	"github.com/xh-polaris/innospark-core-api/biz/infra/util/httpx"
)

const webAPIEndPoint = "https://api.bochaai.com/v1/web-search"

type BochaSearchTool struct {
	relay  *state.RelayContext
	apiKey string
}

type webAPIReq struct {
	Query     string `json:"query"`               // 搜错词
	Freshness string `json:"freshness,omitempty"` // 时间范围 noLimit/oneDay/oneWeek/oneMonth/oneYear/YYYY-MM-DD..YYYY-MM-DD
	Summary   bool   `json:"summary,omitempty"`   // 是否显示文本摘要
	Include   string `json:"include,omitempty"`   // 指定搜索的网站范围
	Exclude   string `json:"exclude,omitempty"`   // 排除搜索的网站范围
	Count     int    `json:"count,omitempty"`     // 返回的结果数(1-50, 默认10, 可能小于count)
}
type webAPIResp struct {
	Code  int    `json:"code"`
	LogId string `json:"log_id"`
	Msg   string `json:"message"`
	Data  struct {
		Type         string `json:"_type"`
		QueryContext struct {
			OriginalQuery string `json:"originalQuery"`
		} `json:"queryContext"`
		WebPages struct {
			WebSearchUrl          string `json:"webSearchUrl"` // 匹配到的总数
			TotalEstimatedMatches int    `json:"totalEstimatedMatches"`
			Value                 []*struct {
				ID               string `json:"id"`               // 网页排序ID
				Name             string `json:"name"`             // 网页标题
				URL              string `json:"url"`              // 网页URL
				DisplayUrl       string `json:"displayUrl"`       // 解码后网页URL
				Snippet          string `json:"snippet"`          // 简短描述
				Summary          string `json:"summary"`          // 总结
				SiteName         string `json:"siteName"`         // 站点名称
				SiteIcon         string `json:"siteIcon"`         // 站点图标
				DatePublished    string `json:"datePublished"`    // 发布时间
				DateLastCrawled  string `json:"dateLastCrawled"`  // 发布时间
				CachedPageUrl    string `json:"cachedPageUrl"`    // 缓存的URL
				Language         string `json:"language"`         //网页语言
				IsFamilyFriendly bool   `json:"isFamilyFriendly"` // 是否家庭友好
				IsNavigational   bool   `json:"isNavigational"`   //是否导航性页面
			} `json:"value"`
		} `json:"webPages"`
	} `json:"data"`
}

func NewBochaSearchTool(relay *state.RelayContext, apiKey string) WebSearchTool {
	return &BochaSearchTool{relay: relay, apiKey: apiKey}
}

func (t *BochaSearchTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "博查API",
		Desc: "使用博查WebAPI进行联网搜索",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"query": {Type: "String", Desc: "搜索词", Required: true},
		}),
	}, nil
}

func (t *BochaSearchTool) InvokableRun(ctx context.Context, jsonStr string, _ ...tool.Option) (_ string, err error) {
	var args map[string]any
	if err = json.Unmarshal([]byte(jsonStr), &args); err != nil {
		return "", err
	}

	inter, inf := t.relay.Interaction, t.relay.Info
	if err = inter.WriteEvent(inter.SearchStartEvent()); err != nil { // 开始搜索
		return "", err
	}

	var resp *webAPIResp
	header := http.Header{}
	header.Add("Content-Type", "application/json")
	header.Add("Authorization", "Bearer "+t.apiKey)
	body := &webAPIReq{Query: args["query"].(string), Freshness: "noLimit", Summary: true}
	if resp, err = httpx.Post[*webAPIResp](ctx, webAPIEndPoint, header, body); err != nil {
		return "", err
	}

	// SSE: 查找多少篇?
	find := resp.Data.WebPages.TotalEstimatedMatches%50 + len(resp.Data.WebPages.Value)
	if err = inter.WriteEvent(inter.SearchFindEvent(find)); err != nil {
		return "", err
	}
	// 随机引用一下, 避免每次引用次数都是一样多
	if len(resp.Data.WebPages.Value) > 6 {
		resp.Data.WebPages.Value = resp.Data.WebPages.Value[:6+rand.IntN(len(resp.Data.WebPages.Value)-5)]
	}
	// SSE: 选择多少篇
	choose := len(resp.Data.WebPages.Value)
	if err = inter.WriteEvent(inter.SearchChooseEvent(choose)); err != nil {
		return "", err
	}

	// 处理结果, 大模型只需要知道cite编号和summary内容, summary内容暂时选择不存储
	// index, name, url, snippet, siteName. siteIcon, datePublished都需要给前端
	var sb strings.Builder
	var cite []*mmsg.Cite
	for i, v := range resp.Data.WebPages.Value {
		sb.WriteString("索引:")
		sb.WriteString(strconv.Itoa(i))
		sb.WriteString(strings.Replace(v.Summary, "\n", " ", -1))
		sb.WriteString("\n")
		c := &mmsg.Cite{Index: int32(i), Name: v.Name, URL: v.URL, Snippet: strings.Replace(v.Snippet, "\n", " ", -1),
			SiteName: v.SiteName, SiteIcon: v.SiteIcon, DatePublished: v.DatePublished}
		cite = append(cite, c)
		// SSE: 返回引用
		if err = inter.WriteEvent(inter.SearchCiteEvent(c)); err != nil {
			return "", err
		}
	}
	if err = inter.WriteEvent(inter.SearchEndEvent()); err != nil {
		return "", err
	}
	inf.SearchInfo = &info.SearchInfo{Find: find, Choose: choose, Cite: cite}
	return sb.String(), nil
}
