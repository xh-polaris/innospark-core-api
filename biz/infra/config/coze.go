package config

import (
	"net/http"
	"sync"

	"github.com/xh-polaris/innospark-core-api/biz/infra/util/httpx"
	"github.com/xh-polaris/innospark-core-api/pkg/errorx"
	"github.com/xh-polaris/innospark-core-api/pkg/logs"
)

type Coze struct {
	Account  string
	Password string
	PAT      string
	cookie   string
	mu       sync.Mutex
}

func (c *Coze) GetCookie() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cookie != "" {
		return c.cookie
	}
	c.cookie = loginCoze(c.Account, c.Password)
	return c.cookie
}

func (c *Coze) RefreshCookie() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cookie = loginCoze(c.Account, c.Password)
	return c.cookie
}

func loginCoze(account, password string) string {
	header := http.Header{}
	header.Set("Content-Type", "application/json")
	body := map[string]string{"email": account, "password": password}
	header, _, err := httpx.GetHttpClient().PostWithHeader("https://coze.aiecnu.net/api/passport/web/email/login/", header, body)
	if err != nil {
		logs.Errorf("loginCoze err: %s", errorx.ErrorWithoutStack(err))
		return ""
	}
	return header.Get("Set-Cookie")
}
