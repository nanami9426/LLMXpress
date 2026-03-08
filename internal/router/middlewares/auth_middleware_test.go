package middlewares

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nanami9426/imgo/internal/models"
	"github.com/nanami9426/imgo/internal/utils"
)

func TestGatewayAuthMiddlewareAcceptsAPIKey(t *testing.T) {
	restore := saveAuthMiddlewareGlobals()
	defer restore()

	originalPepper := utils.DefaultAPIKeyPepper
	utils.DefaultAPIKeyPepper = "test-pepper"
	defer func() {
		utils.DefaultAPIKeyPepper = originalPepper
	}()

	prefix, fullKey, secretHash, err := utils.GenerateAPIKeyToken()
	if err != nil {
		t.Fatalf("GenerateAPIKeyToken returned error: %v", err)
	}

	var touchedAPIKeyID int64
	getAPIKeyByPrefixFn = func(gotPrefix string) (*models.APIKey, error) {
		if gotPrefix != prefix {
			t.Fatalf("unexpected prefix: got %q want %q", gotPrefix, prefix)
		}
		return &models.APIKey{
			APIKeyID:   9,
			UserID:     42,
			Prefix:     prefix,
			SecretHash: secretHash,
			Status:     models.APIKeyStatusActive,
		}, nil
	}
	touchAPIKeyUsageFn = func(apiKeyID int64, lastUsedAt time.Time, lastUsedIP string) error {
		touchedAPIKeyID = apiKeyID
		return nil
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(GatewayAuthMiddleware())
	r.GET("/v1/chat/completions", func(c *gin.Context) {
		userID, _ := parseInt64ContextKey(c, contextKeyUserID)
		apiKeyID, _ := parseInt64ContextKey(c, contextKeyAPIKeyID)
		principalID, _ := parseInt64ContextKey(c, contextKeyPrincipalID)
		authType, _ := c.Get(contextKeyAuthType)
		principalType, _ := c.Get(contextKeyPrincipalType)
		c.JSON(http.StatusOK, gin.H{
			"user_id":        userID,
			"api_key_id":     apiKeyID,
			"principal_id":   principalID,
			"auth_type":      authType,
			"principal_type": principalType,
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/chat/completions", nil)
	req.Header.Set("Authorization", "Bearer "+fullKey)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d body=%s", rec.Code, rec.Body.String())
	}

	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}
	if int64(body["user_id"].(float64)) != 42 {
		t.Fatalf("unexpected user_id: %+v", body)
	}
	if int64(body["api_key_id"].(float64)) != 9 {
		t.Fatalf("unexpected api_key_id: %+v", body)
	}
	if body["auth_type"] != authTypeAPIKey {
		t.Fatalf("unexpected auth_type: %+v", body)
	}
	if body["principal_type"] != principalTypeAPIKey {
		t.Fatalf("unexpected principal_type: %+v", body)
	}
	if touchedAPIKeyID != 9 {
		t.Fatalf("expected API key touch, got %d", touchedAPIKeyID)
	}
}

func TestGatewayAuthMiddlewareAcceptsJWT(t *testing.T) {
	restore := saveAuthMiddlewareGlobals()
	defer restore()

	origLoginDeviceMax := utils.LoginDeviceMax
	utils.LoginDeviceMax = 1
	defer func() {
		utils.LoginDeviceMax = origLoginDeviceMax
	}()

	checkTokenFn = func(tokenString string, secret []byte) (*utils.Claims, error) {
		if tokenString != "jwt-token" {
			t.Fatalf("unexpected token: %q", tokenString)
		}
		return &utils.Claims{UserID: 7, Role: "user", Version: 3}, nil
	}
	getTokenVersionFn = func(ctx context.Context, userID uint) (uint, error) {
		return 3, nil
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(GatewayAuthMiddleware())
	r.GET("/v1/chat/completions", func(c *gin.Context) {
		_, hasAPIKeyID := c.Get(contextKeyAPIKeyID)
		authType, _ := c.Get(contextKeyAuthType)
		c.JSON(http.StatusOK, gin.H{
			"has_api_key_id": hasAPIKeyID,
			"auth_type":      authType,
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/chat/completions", nil)
	req.Header.Set("Authorization", "Bearer jwt-token")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d body=%s", rec.Code, rec.Body.String())
	}

	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}
	if body["auth_type"] != authTypeJWT {
		t.Fatalf("unexpected auth_type: %+v", body)
	}
	if body["has_api_key_id"].(bool) {
		t.Fatalf("jwt request should not set api_key_id")
	}
}

func TestGatewayAuthMiddlewareRejectsInvalidAPIKeys(t *testing.T) {
	restore := saveAuthMiddlewareGlobals()
	defer restore()

	originalPepper := utils.DefaultAPIKeyPepper
	utils.DefaultAPIKeyPepper = "test-pepper"
	defer func() {
		utils.DefaultAPIKeyPepper = originalPepper
	}()

	prefix, validKey, secretHash, err := utils.GenerateAPIKeyToken()
	if err != nil {
		t.Fatalf("GenerateAPIKeyToken returned error: %v", err)
	}
	_, wrongKey, wrongHash, err := utils.GenerateAPIKeyToken()
	if err != nil {
		t.Fatalf("GenerateAPIKeyToken returned error: %v", err)
	}

	now := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)
	authNowFn = func() time.Time { return now }
	checkTokenFn = func(tokenString string, secret []byte) (*utils.Claims, error) {
		return nil, errors.New("not a jwt")
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(GatewayAuthMiddleware())
	r.GET("/v1/chat/completions", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	cases := []struct {
		name       string
		header     string
		target     string
		buildModel func() (*models.APIKey, error)
	}{
		{
			name:   "malformed",
			header: "Bearer sk_bad",
			target: "/v1/chat/completions",
			buildModel: func() (*models.APIKey, error) {
				return nil, errors.New("should not query model")
			},
		},
		{
			name:   "secret_mismatch",
			header: "Bearer " + validKey,
			target: "/v1/chat/completions",
			buildModel: func() (*models.APIKey, error) {
				return &models.APIKey{
					APIKeyID:   1,
					UserID:     2,
					Prefix:     prefix,
					SecretHash: wrongHash,
					Status:     models.APIKeyStatusActive,
				}, nil
			},
		},
		{
			name:   "revoked",
			header: "Bearer " + validKey,
			target: "/v1/chat/completions",
			buildModel: func() (*models.APIKey, error) {
				return &models.APIKey{
					APIKeyID:   1,
					UserID:     2,
					Prefix:     prefix,
					SecretHash: secretHash,
					Status:     models.APIKeyStatusRevoked,
				}, nil
			},
		},
		{
			name:   "expired",
			header: "Bearer " + validKey,
			target: "/v1/chat/completions",
			buildModel: func() (*models.APIKey, error) {
				expiresAt := now.Add(-time.Minute)
				return &models.APIKey{
					APIKeyID:   1,
					UserID:     2,
					Prefix:     prefix,
					SecretHash: secretHash,
					Status:     models.APIKeyStatusActive,
					ExpiresAt:  &expiresAt,
				}, nil
			},
		},
		{
			name:   "query_token_not_allowed",
			target: "/v1/chat/completions?token=" + validKey,
			buildModel: func() (*models.APIKey, error) {
				return nil, errors.New("should not query model")
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			getAPIKeyByPrefixFn = func(gotPrefix string) (*models.APIKey, error) {
				if tc.name == "query_token_not_allowed" || tc.name == "malformed" {
					t.Fatalf("getAPIKeyByPrefixFn should not be called for %s", tc.name)
				}
				if gotPrefix != prefix {
					t.Fatalf("unexpected prefix: got %q want %q", gotPrefix, prefix)
				}
				return tc.buildModel()
			}

			req := httptest.NewRequest(http.MethodGet, tc.target, nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("unexpected status: got %d body=%s", rec.Code, rec.Body.String())
			}
		})
	}

	_ = wrongKey
}

func saveAuthMiddlewareGlobals() func() {
	origCheckTokenFn := checkTokenFn
	origGetTokenVersionFn := getTokenVersionFn
	origGetAPIKeyByPrefixFn := getAPIKeyByPrefixFn
	origTouchAPIKeyUsageFn := touchAPIKeyUsageFn
	origAuthNowFn := authNowFn
	return func() {
		checkTokenFn = origCheckTokenFn
		getTokenVersionFn = origGetTokenVersionFn
		getAPIKeyByPrefixFn = origGetAPIKeyByPrefixFn
		touchAPIKeyUsageFn = origTouchAPIKeyUsageFn
		authNowFn = origAuthNowFn
	}
}
