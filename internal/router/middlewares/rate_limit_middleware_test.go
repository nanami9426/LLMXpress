package middlewares

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/nanami9426/imgo/internal/utils"
)

func TestRateLimitMiddlewareUsesPrincipalIDForAPIKeyRequests(t *testing.T) {
	origGetCfg := getRateLimitConfigFn
	origConsume := consumeChatCompletionQuotaFn
	defer func() {
		getRateLimitConfigFn = origGetCfg
		consumeChatCompletionQuotaFn = origConsume
	}()

	getRateLimitConfigFn = func() utils.RateLimitConfig {
		return utils.RateLimitConfig{RequestPerMin: 10, WindowSeconds: 60, RedisPrefix: "rl:test"}
	}

	var gotPrincipalID int64
	consumeChatCompletionQuotaFn = func(_ context.Context, principalID int64, reqCost int64, tokenCost int64) error {
		gotPrincipalID = principalID
		return nil
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(contextKeyUserID, int64(1))
		c.Set(contextKeyPrincipalID, int64(99))
		c.Next()
	})
	r.Use(RateLimitMiddleware())
	r.POST("/v1/chat/completions", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"messages":[{"role":"user","content":"hi"}]}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("unexpected status: got %d body=%s", rec.Code, rec.Body.String())
	}
	if gotPrincipalID != 99 {
		t.Fatalf("expected principal_id to be used, got %d", gotPrincipalID)
	}
}

func TestRateLimitMiddlewareFallsBackToUserIDForJWTRequests(t *testing.T) {
	origGetCfg := getRateLimitConfigFn
	origConsume := consumeChatCompletionQuotaFn
	defer func() {
		getRateLimitConfigFn = origGetCfg
		consumeChatCompletionQuotaFn = origConsume
	}()

	getRateLimitConfigFn = func() utils.RateLimitConfig {
		return utils.RateLimitConfig{RequestPerMin: 10, WindowSeconds: 60, RedisPrefix: "rl:test"}
	}

	var gotPrincipalID int64
	consumeChatCompletionQuotaFn = func(_ context.Context, principalID int64, reqCost int64, tokenCost int64) error {
		gotPrincipalID = principalID
		return nil
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(contextKeyUserID, int64(7))
		c.Next()
	})
	r.Use(RateLimitMiddleware())
	r.POST("/v1/chat/completions", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"messages":[{"role":"user","content":"hi"}]}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("unexpected status: got %d body=%s", rec.Code, rec.Body.String())
	}
	if gotPrincipalID != 7 {
		t.Fatalf("expected user_id fallback, got %d", gotPrincipalID)
	}
}
