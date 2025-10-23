package wsx

import "errors"

func IsNormal(err error) bool {
	return err == nil || errors.Is(err, NormalCloseErr)
}
