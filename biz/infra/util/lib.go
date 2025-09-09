package util

import (
	"fmt"

	"github.com/xh-polaris/innospark-core-api/biz/application/dto/basic"
)

var debug = true

func DPrintf(format string, a ...interface{}) {
	if debug {
		fmt.Printf(format, a...)
	}
}

// Success 返回成功的basic.Response指针
func Success() *basic.Response {
	return &basic.Response{
		Code: 200,
		Msg:  "success",
	}
}
