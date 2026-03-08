package middlewares

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nanami9426/imgo/internal/models"
	"github.com/nanami9426/imgo/internal/utils"
)

const (
	contextKeyUserID        = "user_id"
	contextKeyRole          = "role"
	contextKeyAuthType      = "auth_type"
	contextKeyAPIKeyID      = "api_key_id"
	contextKeyPrincipalID   = "principal_id"
	contextKeyPrincipalType = "principal_type"

	authTypeJWT    = "jwt"
	authTypeAPIKey = "api_key"

	principalTypeAPIKey = "api_key"
)

var (
	checkTokenFn        = utils.CheckToken
	getTokenVersionFn   = utils.GetTokenVersion
	getAPIKeyByPrefixFn = models.GetAPIKeyByPrefix
	touchAPIKeyUsageFn  = models.TouchAPIKeyUsage
	authNowFn           = func() time.Time { return time.Now().UTC() }
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := tokenFromAuthorizationHeader(c)
		if token == "" {
			token = strings.TrimSpace(c.Query("token"))
		}
		if token == "" {
			utils.Abort(c, http.StatusUnauthorized, utils.StatUnauthorized, "token不能为空", nil)
			return
		}
		if err := authenticateJWT(c, token); err != nil {
			utils.Abort(c, http.StatusUnauthorized, utils.StatUnauthorized, "token无效或已过期", err)
			return
		}
		c.Next()
	}
}

func GatewayAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		headerToken := tokenFromAuthorizationHeader(c)
		if strings.HasPrefix(headerToken, "sk_") {
			if err := authenticateAPIKey(c, headerToken); err != nil {
				utils.Abort(c, http.StatusUnauthorized, utils.StatUnauthorized, "API Key无效或已过期", err)
				return
			}
			c.Next()
			return
		}

		token := headerToken
		if token == "" {
			token = strings.TrimSpace(c.Query("token"))
		}
		if token == "" {
			utils.Abort(c, http.StatusUnauthorized, utils.StatUnauthorized, "token不能为空", nil)
			return
		}
		if err := authenticateJWT(c, token); err != nil {
			utils.Abort(c, http.StatusUnauthorized, utils.StatUnauthorized, "token无效或已过期", err)
			return
		}
		c.Next()
	}
}

func tokenFromAuthorizationHeader(c *gin.Context) string {
	token := strings.TrimSpace(c.GetHeader("Authorization"))
	if strings.HasPrefix(strings.ToLower(token), "bearer ") {
		token = strings.TrimSpace(token[7:])
	}
	return token
}

func authenticateJWT(c *gin.Context, token string) error {
	claims, err := checkTokenFn(token, utils.JWTSecret())
	if err != nil {
		return err
	}

	latestVersion, err := getTokenVersionFn(c, claims.UserID)
	if err != nil {
		utils.Log.Errorf("failed to load token version: user_id=%d err=%v", claims.UserID, err)
		latestVersion = claims.Version
	}
	diff := uintDiff(latestVersion, claims.Version)
	if diff >= utils.LoginDeviceMax {
		return errors.New("登录设备达到上限")
	}

	c.Set(contextKeyUserID, claims.UserID)
	c.Set(contextKeyRole, claims.Role)
	c.Set(contextKeyAuthType, authTypeJWT)
	return nil
}

func authenticateAPIKey(c *gin.Context, token string) error {
	prefix, secret, err := utils.SplitAPIKeyToken(token)
	if err != nil {
		return err
	}

	apiKey, err := getAPIKeyByPrefixFn(prefix)
	if err != nil {
		return err
	}
	if apiKey.Status != models.APIKeyStatusActive {
		return errors.New("api key revoked")
	}
	now := authNowFn()
	if apiKey.ExpiresAt != nil && apiKey.ExpiresAt.Before(now) {
		return errors.New("api key expired")
	}
	if !utils.VerifyAPIKeySecret(secret, apiKey.SecretHash) {
		return errors.New("api key secret mismatch")
	}

	c.Set(contextKeyUserID, apiKey.UserID)
	c.Set(contextKeyAuthType, authTypeAPIKey)
	c.Set(contextKeyAPIKeyID, apiKey.APIKeyID)
	c.Set(contextKeyPrincipalID, apiKey.APIKeyID)
	c.Set(contextKeyPrincipalType, principalTypeAPIKey)

	if err := touchAPIKeyUsageFn(apiKey.APIKeyID, now, c.ClientIP()); err != nil {
		utils.Log.Errorf("failed to update api key last used: api_key_id=%d err=%v", apiKey.APIKeyID, err)
	}
	return nil
}

func uintDiff(a uint, b uint) uint {
	if a >= b {
		return a - b
	}
	return b - a
}

func parseInt64ContextKey(c *gin.Context, key string) (int64, bool) {
	v, ok := c.Get(key)
	if !ok {
		return 0, false
	}
	return parseInt64ContextValue(v)
}

func parseInt64ContextValue(v interface{}) (int64, bool) {
	switch val := v.(type) {
	case int64:
		return val, true
	case int:
		return int64(val), true
	case uint:
		return int64(val), true
	case uint64:
		return int64(val), true
	case float64:
		return int64(val), true
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(val), 10, 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}
