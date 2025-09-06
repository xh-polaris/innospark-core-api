package util

import "fmt"

var debug = true

func DPrintf(format string, a ...interface{}) {
	if debug {
		fmt.Printf(format, a...)
	}
}
