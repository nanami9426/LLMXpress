package models

import (
	"errors"
	"time"

	"github.com/nanami9426/imgo/internal/utils"
	"gorm.io/gorm"
)

const (
	APIKeyStatusActive  = "active"
	APIKeyStatusRevoked = "revoked"
)

type APIKey struct {
	APIKeyID   int64      `gorm:"primarykey"`
	UserID     int64      `gorm:"index"`
	Name       string     `gorm:"type:varchar(191)"`
	Prefix     string     `gorm:"type:char(15);uniqueIndex"`
	SecretHash string     `gorm:"type:char(64)"`
	Status     string     `gorm:"type:varchar(16);index"`
	ExpiresAt  *time.Time `gorm:"index"`
	LastUsedAt *time.Time
	LastUsedIP string `gorm:"type:varchar(45)"`
	Basic
}

func (a *APIKey) TableName() string {
	return "api_key"
}

func CreateAPIKey(apiKey *APIKey) error {
	return utils.DB.Create(apiKey).Error
}

func ListAPIKeysByUser(userID int64) ([]*APIKey, error) {
	var list []*APIKey
	err := utils.DB.
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Order("api_key_id DESC").
		Find(&list).Error
	return list, err
}

func GetAPIKeyByPrefix(prefix string) (*APIKey, error) {
	var apiKey APIKey
	err := utils.DB.Where("prefix = ?", prefix).First(&apiKey).Error
	if err != nil {
		return nil, err
	}
	return &apiKey, nil
}

func RevokeAPIKeyByIDAndUser(apiKeyID int64, userID int64) (bool, error) {
	var apiKey APIKey
	err := utils.DB.
		Where("api_key_id = ? AND user_id = ?", apiKeyID, userID).
		First(&apiKey).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	if apiKey.Status == APIKeyStatusRevoked {
		return true, nil
	}
	err = utils.DB.Model(&APIKey{}).
		Where("api_key_id = ? AND user_id = ?", apiKeyID, userID).
		Updates(map[string]interface{}{
			"status":     APIKeyStatusRevoked,
			"updated_at": time.Now().UTC(),
		}).Error
	if err != nil {
		return false, err
	}
	return true, nil
}

func TouchAPIKeyUsage(apiKeyID int64, lastUsedAt time.Time, lastUsedIP string) error {
	return utils.DB.Model(&APIKey{}).
		Where("api_key_id = ?", apiKeyID).
		Updates(map[string]interface{}{
			"last_used_at": lastUsedAt.UTC(),
			"last_used_ip": lastUsedIP,
		}).Error
}
