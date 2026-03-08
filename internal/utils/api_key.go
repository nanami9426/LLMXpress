package utils

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
)

const (
	apiKeyPrefix      = "sk_"
	apiKeyPublicLen   = 12
	apiKeySecretBytes = 32
)

func APIKeyPepper() []byte {
	return []byte(DefaultAPIKeyPepper)
}

func GenerateAPIKeyToken() (string, string, string, error) {
	if len(APIKeyPepper()) == 0 {
		return "", "", "", errors.New("empty api key pepper")
	}

	publicPart, err := randomAlphaNumLower(apiKeyPublicLen)
	if err != nil {
		return "", "", "", err
	}
	secretBytes := make([]byte, apiKeySecretBytes)
	if _, err := rand.Read(secretBytes); err != nil {
		return "", "", "", err
	}

	prefix := apiKeyPrefix + publicPart
	secret := base64.RawURLEncoding.EncodeToString(secretBytes)
	fullKey := prefix + "." + secret
	return prefix, fullKey, HashAPIKeySecret(secret), nil
}

func HashAPIKeySecret(secret string) string {
	mac := hmac.New(sha256.New, APIKeyPepper())
	_, _ = mac.Write([]byte(secret))
	return hex.EncodeToString(mac.Sum(nil))
}

func VerifyAPIKeySecret(secret string, expectedHash string) bool {
	calculated := HashAPIKeySecret(secret)
	return hmac.Equal([]byte(calculated), []byte(strings.TrimSpace(expectedHash)))
}

func SplitAPIKeyToken(token string) (string, string, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return "", "", errors.New("empty api key")
	}
	if !strings.HasPrefix(token, apiKeyPrefix) {
		return "", "", errors.New("invalid api key prefix")
	}

	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return "", "", errors.New("invalid api key format")
	}
	prefix := strings.TrimSpace(parts[0])
	secret := strings.TrimSpace(parts[1])
	if len(prefix) != len(apiKeyPrefix)+apiKeyPublicLen {
		return "", "", errors.New("invalid api key prefix length")
	}
	if !isLowerAlphaNum(prefix[len(apiKeyPrefix):]) {
		return "", "", errors.New("invalid api key prefix")
	}
	if secret == "" {
		return "", "", errors.New("missing api key secret")
	}
	decoded, err := base64.RawURLEncoding.DecodeString(secret)
	if err != nil {
		return "", "", errors.New("invalid api key secret")
	}
	if len(decoded) != apiKeySecretBytes {
		return "", "", errors.New("invalid api key secret length")
	}
	return prefix, secret, nil
}

func randomAlphaNumLower(n int) (string, error) {
	if n <= 0 {
		return "", errors.New("invalid random length")
	}
	const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	buf := make([]byte, n)
	raw := make([]byte, n)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	for i := range buf {
		buf[i] = alphabet[int(raw[i])%len(alphabet)]
	}
	return string(buf), nil
}

func isLowerAlphaNum(s string) bool {
	if s == "" {
		return false
	}
	for _, ch := range s {
		switch {
		case ch >= 'a' && ch <= 'z':
		case ch >= '0' && ch <= '9':
		default:
			return false
		}
	}
	return true
}
