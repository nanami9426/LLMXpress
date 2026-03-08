package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/nanami9426/imgo/internal/models"
)

func TestAPILoggingMiddlewareRecordsAPIKeyContext(t *testing.T) {
	origCreateAPIUsageFn := createAPIUsageFn
	defer func() {
		createAPIUsageFn = origCreateAPIUsageFn
	}()

	var captured *models.APIUsage
	createAPIUsageFn = func(usage *models.APIUsage) error {
		copied := *usage
		captured = &copied
		return nil
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(contextKeyUserID, int64(11))
		c.Set(contextKeyAPIKeyID, int64(22))
		c.Set(contextKeyAuthType, authTypeAPIKey)
		c.Next()
	})
	r.Use(APILoggingMiddleware())
	r.POST("/v1/chat/completions", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"id":    "cmpl-1",
			"model": "test-model",
			"usage": gin.H{
				"prompt_tokens":     1,
				"completion_tokens": 2,
				"total_tokens":      3,
			},
		})
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d body=%s", rec.Code, rec.Body.String())
	}
	if captured == nil {
		t.Fatalf("expected usage record to be created")
	}
	if captured.UserID != 11 {
		t.Fatalf("unexpected user_id: %+v", captured)
	}
	if captured.APIKeyID == nil || *captured.APIKeyID != 22 {
		t.Fatalf("unexpected api_key_id: %+v", captured)
	}
	if captured.AuthType != authTypeAPIKey {
		t.Fatalf("unexpected auth_type: %+v", captured)
	}
}
