package service

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nanami9426/imgo/internal/utils"
)

func rewriteUpstreamHeaders(header http.Header) {
	header.Del("Host")
	header.Del("Accept-Encoding")

	apiKey := strings.TrimSpace(utils.UpstreamAPIKey)
	if apiKey == "" {
		// 不把网关自身 JWT / API Key 透传给第三方上游。
		header.Del("Authorization")
		return
	}

	header.Set("Authorization", "Bearer "+apiKey)
}

func buildUpstreamURL(target *url.URL, requestURL *url.URL) string {
	upstreamURL := *target
	basePath := strings.TrimSuffix(target.Path, "/")
	requestPath := strings.TrimPrefix(requestURL.Path, "/")
	switch {
	case basePath == "":
		upstreamURL.Path = "/" + requestPath
	case requestPath == "":
		upstreamURL.Path = basePath
	default:
		upstreamURL.Path = path.Join(basePath, requestPath)
		if !strings.HasPrefix(upstreamURL.Path, "/") {
			upstreamURL.Path = "/" + upstreamURL.Path
		}
	}
	upstreamURL.RawQuery = requestURL.RawQuery
	return upstreamURL.String()
}

func ProxyToVLLM() gin.HandlerFunc {
	upstream := utils.UpstreamBaseURL
	target, err := url.Parse(upstream)
	if err != nil || target.Scheme == "" || target.Host == "" {
		// 启动时配置有问题，直接返回handler，所有请求500
		return func(c *gin.Context) {
			utils.Abort(c, http.StatusInternalServerError, utils.StatInternalError, "invalid VLLM_URL: "+upstream, nil)
		}
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		rewriteUpstreamHeaders(req.Header)
		req.Host = target.Host
	}
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

func ChatCompletionsHandler() gin.HandlerFunc {
	upstream := utils.UpstreamBaseURL
	target, err := url.Parse(upstream)
	if err != nil || target.Scheme == "" || target.Host == "" {
		// 启动时配置有问题，直接返回handler，所有请求500
		return func(c *gin.Context) {
			utils.Abort(c, http.StatusInternalServerError, utils.StatInternalError, "invalid VLLM_URL: "+upstream, nil)
		}
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableCompression:    true,
	}
	client := &http.Client{
		// http.Client.Timeout 是一个总超时
		// 包含建立连接、重定向、读取响应 body（包括 stream 期间一直读）等整个请求生命周期。
		// 设置为0表示不启用这个总超时，请求可以一直持续下去。
		Timeout:   0,
		Transport: transport,
	}
	return func(c *gin.Context) {
		req, err := http.NewRequestWithContext(
			c.Request.Context(),
			c.Request.Method,
			buildUpstreamURL(target, c.Request.URL),
			c.Request.Body,
		)
		if err != nil {
			utils.Abort(c, http.StatusInternalServerError, utils.StatInternalError, "build request failed", err)
			return
		}

		req.Header = c.Request.Header.Clone()
		rewriteUpstreamHeaders(req.Header)
		req.ContentLength = c.Request.ContentLength
		if userID, ok := c.Get("user_id"); ok {
			req.Header.Set("X-User-ID", fmt.Sprintf("%v", userID))
		}
		req.Host = target.Host

		resp, err := client.Do(req)
		if err != nil {
			c.Writer.Header().Set("Content-Type", "application/json")
			c.Writer.WriteHeader(http.StatusBadGateway)
			_, _ = c.Writer.Write([]byte(`{"error":{"message":"upstream error","type":"bad_gateway"}}`))
			return
		}
		defer resp.Body.Close()

		for k, vv := range resp.Header {
			if isHopByHopOrCORSHeader(k) {
				continue
			}
			for _, v := range vv {
				c.Writer.Header().Add(k, v)
			}
		}

		contentType := strings.ToLower(resp.Header.Get("Content-Type"))
		isStream := strings.HasPrefix(contentType, "text/event-stream")
		if isStream {
			if c.Writer.Header().Get("Cache-Control") == "" {
				c.Writer.Header().Set("Cache-Control", "no-cache")
			}
			if c.Writer.Header().Get("Connection") == "" {
				c.Writer.Header().Set("Connection", "keep-alive")
			}
			if c.Writer.Header().Get("X-Accel-Buffering") == "" {
				c.Writer.Header().Set("X-Accel-Buffering", "no")
			}
		}

		c.Writer.WriteHeader(resp.StatusCode)

		if isStream {
			flusher, ok := c.Writer.(http.Flusher)
			if !ok {
				_, _ = io.Copy(c.Writer, resp.Body)
				return
			}
			buf := make([]byte, 8*1024)
			for {
				select {
				case <-c.Request.Context().Done():
					return
				default:
				}
				n, readErr := resp.Body.Read(buf)
				if n > 0 {
					if _, writeErr := c.Writer.Write(buf[:n]); writeErr != nil {
						return
					}
					flusher.Flush()
				}
				if readErr != nil {
					if readErr == io.EOF {
						return
					}
					return
				}
			}
		}

		_, _ = io.Copy(c.Writer, resp.Body)
	}
}

func isHopByHopOrCORSHeader(name string) bool {
	switch strings.ToLower(name) {
	case "connection",
		"keep-alive",
		"proxy-authenticate",
		"proxy-authorization",
		"te",
		"trailer",
		"transfer-encoding",
		"upgrade",
		"vary",
		"access-control-allow-origin",
		"access-control-allow-credentials",
		"access-control-allow-headers",
		"access-control-allow-methods",
		"access-control-expose-headers",
		"access-control-max-age":
		return true
	default:
		return false
	}
}
