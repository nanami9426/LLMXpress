package service

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nanami9426/imgo/internal/models"
	"github.com/nanami9426/imgo/internal/utils"
)

func TestCreateAPIKeyReturnsFullKeyOnce(t *testing.T) {
	restore := saveUserServiceGlobals()
	defer restore()

	var captured *models.APIKey
	createAPIKeyFn = func(apiKey *models.APIKey) error {
		apiKey.CreatedAt = time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)
		captured = apiKey
		return nil
	}
	generateAPIKeyTokenFn = func() (string, string, string, error) {
		return "sk_abcdefghijkl", "sk_abcdefghijkl.secret", "secret-hash", nil
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", int64(88))
		c.Next()
	})
	r.POST("/user/create_api_key", CreateAPIKey)

	body := strings.NewReader("name=my-key&expires_at=2026-03-10T00:00:00Z")
	req := httptest.NewRequest(http.MethodPost, "/user/create_api_key", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d body=%s", rec.Code, rec.Body.String())
	}
	if captured == nil {
		t.Fatalf("expected api key to be captured")
	}
	if captured.UserID != 88 || captured.SecretHash != "secret-hash" || captured.Prefix != "sk_abcdefghijkl" {
		t.Fatalf("unexpected stored api key: %+v", captured)
	}

	var resp utils.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success response: %+v", resp)
	}
	data := resp.Data.(map[string]interface{})
	if data["key"] != "sk_abcdefghijkl.secret" {
		t.Fatalf("expected full key in response, got %+v", data)
	}
	if _, ok := data["secret_hash"]; ok {
		t.Fatalf("response should not expose secret_hash: %+v", data)
	}
}

func TestListAPIKeysDoesNotLeakSecrets(t *testing.T) {
	restore := saveUserServiceGlobals()
	defer restore()

	lastUsedAt := time.Date(2026, 3, 8, 13, 0, 0, 0, time.UTC)
	listAPIKeysByUserFn = func(userID int64) ([]*models.APIKey, error) {
		if userID != 5 {
			t.Fatalf("unexpected user_id: %d", userID)
		}
		return []*models.APIKey{
			{
				APIKeyID:   1,
				UserID:     5,
				Name:       "key-a",
				Prefix:     "sk_abcdefghijkl",
				SecretHash: "hidden",
				Status:     models.APIKeyStatusActive,
				LastUsedAt: &lastUsedAt,
				LastUsedIP: "127.0.0.1",
			},
		}, nil
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", int64(5))
		c.Next()
	})
	r.POST("/user/api_key_list", ListAPIKeys)

	req := httptest.NewRequest(http.MethodPost, "/user/api_key_list", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d body=%s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "hidden") || strings.Contains(rec.Body.String(), "\"key\":") {
		t.Fatalf("response leaked sensitive fields: %s", rec.Body.String())
	}
}

func TestRevokeAPIKeyHandlesFoundAndNotFound(t *testing.T) {
	restore := saveUserServiceGlobals()
	defer restore()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", int64(9))
		c.Next()
	})
	r.POST("/user/revoke_api_key", RevokeAPIKey)

	revokeAPIKeyByIDFn = func(apiKeyID int64, userID int64) (bool, error) {
		if userID != 9 {
			t.Fatalf("unexpected user_id: %d", userID)
		}
		return apiKeyID == 1, nil
	}

	successReq := httptest.NewRequest(http.MethodPost, "/user/revoke_api_key", strings.NewReader("api_key_id=1"))
	successReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	successRec := httptest.NewRecorder()
	r.ServeHTTP(successRec, successReq)
	if successRec.Code != http.StatusOK {
		t.Fatalf("unexpected success status: got %d body=%s", successRec.Code, successRec.Body.String())
	}

	failReq := httptest.NewRequest(http.MethodPost, "/user/revoke_api_key", strings.NewReader("api_key_id=2"))
	failReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	failRec := httptest.NewRecorder()
	r.ServeHTTP(failRec, failReq)
	if failRec.Code != http.StatusOK {
		t.Fatalf("unexpected failure status: got %d body=%s", failRec.Code, failRec.Body.String())
	}

	var resp utils.Response
	if err := json.Unmarshal(failRec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}
	if resp.Success {
		t.Fatalf("expected failure response for missing key")
	}
}

func saveUserServiceGlobals() func() {
	origCreateAPIKeyFn := createAPIKeyFn
	origListAPIKeysByUserFn := listAPIKeysByUserFn
	origRevokeAPIKeyByIDFn := revokeAPIKeyByIDFn
	origGenerateAPIKeyTokenFn := generateAPIKeyTokenFn
	return func() {
		createAPIKeyFn = origCreateAPIKeyFn
		listAPIKeysByUserFn = origListAPIKeysByUserFn
		revokeAPIKeyByIDFn = origRevokeAPIKeyByIDFn
		generateAPIKeyTokenFn = origGenerateAPIKeyTokenFn
	}
}
