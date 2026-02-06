package router

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/nanami9426/imgo/internal/response"
	"github.com/nanami9426/imgo/internal/utils"
)

func RigisterVLLMRoutes(r *gin.Engine) {
	v1 := r.Group("/v1")
	v1.Use(AuthMiddleware())
	v1.Any("/*any", ProxyToVLLM())
}

func ProxyToVLLM() gin.HandlerFunc {

	upstream := "http://127.0.0.1:8000"

	target, err := url.Parse(upstream)
	if err != nil || target.Scheme == "" || target.Host == "" {
		// 启动时配置有问题，直接返回handler，所有请求500
		return func(c *gin.Context) {
			response.Abort(c, http.StatusInternalServerError, utils.StatInternalError, "invalid VLLM_URL: "+upstream, nil)
		}
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	return func(c *gin.Context) {
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}
