package util

import (
	"context"
	"errors"
	"net/http"

	"github.com/xh-polaris/innospark-core-api/biz/infra/util/httpx"
	"github.com/xh-polaris/innospark-core-api/pkg/logs"
)

const (
	URLTransfer = iota
	Base64Transfer
)

type transferType int

// OCR 识别图片返回Tex公式
func OCR(ctx context.Context, baseURL, key, imgURL string, imgType transferType) (string, error) {
	h := http.Header{"content-type": []string{"application/json"}, "X-API-Key": []string{key}}
	b := make(map[string]interface{})

	switch imgType {
	case Base64Transfer: // base64
		b["image_base64"] = imgURL
	default:
		b["image_url"] = imgURL
	}

	// 默认prompt："请帮我把图片中的公式转为LaTex"
	resp, err := httpx.GetHttpClient().Post(ctx, baseURL, h, b)
	if err != nil || resp["status"] != "success" {
		logs.Error("ocr failed: ", err)
		return "", errors.New("call ocr model failed")
	}
	return resp["result"].(string), nil
}
