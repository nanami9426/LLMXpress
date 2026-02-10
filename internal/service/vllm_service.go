package service

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nanami9426/imgo/internal/utils"
)

func ProxyToVLLM() gin.HandlerFunc {
	upstream := "http://127.0.0.1:8000"
	target, err := url.Parse(upstream)
	if err != nil || target.Scheme == "" || target.Host == "" {
		// 启动时配置有问题，直接返回handler，所有请求500
		return func(c *gin.Context) {
			utils.Abort(c, http.StatusInternalServerError, utils.StatInternalError, "invalid VLLM_URL: "+upstream, nil)
		}
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.FlushInterval = 50 * time.Millisecond
	proxy.Transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		// 自定义拨号上下文，控制TCP连接的行为
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,

		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		// 禁用HTTP压缩，网关、代理服务器自己解压再压缩会浪费CPU资源
		DisableCompression: true,
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":{"message":"upstream error","type":"bad_gateway"}}`))
	}
	return func(c *gin.Context) {
		if user_id, ok := c.Get("user_id"); ok {
			c.Request.Header.Set("X-User-ID", fmt.Sprintf("%v", user_id))
		}
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}
