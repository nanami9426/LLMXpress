package models

import (
	"errors"
	"strings"
	"time"

	"github.com/nanami9426/imgo/internal/utils"
	"gorm.io/gorm"
)

type LLMConversation struct {
	ConversationID     int64 `gorm:"primarykey"`
	UserID             int64 `gorm:"index"`
	Title              string
	Model              string
	MessageCount       int
	LastMessagePreview string
	LastMessageAt      time.Time `gorm:"index"`
	Basic
}

func (c *LLMConversation) TableName() string {
	return "llm_conversation"
}

type LLMConversationMessage struct {
	MessageID      int64 `gorm:"primarykey"`
	ConversationID int64 `gorm:"index"`
	UserID         int64 `gorm:"index"`
	Role           string
	Content        string `gorm:"type:longtext"`
	MessageJSON    string `gorm:"type:longtext"`
	Model          string
	Basic
}

func (m *LLMConversationMessage) TableName() string {
	return "llm_conversation_message"
}

func CreateLLMConversation(conversation *LLMConversation) error {
	return utils.DB.Create(conversation).Error
}

func GetLLMConversationByIDAndUser(conversationID int64, userID int64) (*LLMConversation, error) {
	var conversation LLMConversation
	err := utils.DB.
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		First(&conversation).Error
	if err != nil {
		return nil, err
	}
	return &conversation, nil
}

func ConversationBelongsToUser(conversationID int64, userID int64) (bool, error) {
	var count int64
	err := utils.DB.
		Model(&LLMConversation{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Count(&count).Error
	return count > 0, err
}

func CountLLMConversationsByUser(userID int64) (int64, error) {
	var count int64
	err := utils.DB.
		Model(&LLMConversation{}).
		Where("user_id = ?", userID).
		Count(&count).Error
	return count, err
}

func ListLLMConversationsByUser(userID int64, offset int, limit int) ([]*LLMConversation, error) {
	var list []*LLMConversation
	err := utils.DB.
		Where("user_id = ?", userID).
		Order("last_message_at DESC").
		Order("conversation_id DESC").
		Offset(offset).
		Limit(limit).
		Find(&list).Error
	return list, err
}

func CreateLLMConversationMessages(messages []*LLMConversationMessage) error {
	if len(messages) == 0 {
		return nil
	}
	return utils.DB.Create(messages).Error
}

func CountLLMConversationMessages(conversationID int64) (int64, error) {
	var count int64
	err := utils.DB.
		Model(&LLMConversationMessage{}).
		Where("conversation_id = ?", conversationID).
		Count(&count).Error
	return count, err
}

func ListLLMConversationMessages(conversationID int64, offset int, limit int) ([]*LLMConversationMessage, error) {
	var list []*LLMConversationMessage
	err := utils.DB.
		Where("conversation_id = ?", conversationID).
		Order("created_at ASC").
		Order("message_id ASC").
		Offset(offset).
		Limit(limit).
		Find(&list).Error
	return list, err
}

func GetRecentLLMConversationMessages(conversationID int64, limit int) ([]*LLMConversationMessage, error) {
	var list []*LLMConversationMessage
	db := utils.DB.
		Where("conversation_id = ?", conversationID).
		Order("created_at DESC").
		Order("message_id DESC")
	if limit > 0 {
		db = db.Limit(limit)
	}
	if err := db.Find(&list).Error; err != nil {
		return nil, err
	}
	reverseMessages(list)
	return list, nil
}

func RefreshLLMConversationStats(conversationID int64, model string) error {
	count, err := CountLLMConversationMessages(conversationID)
	if err != nil {
		return err
	}

	var last LLMConversationMessage
	err = utils.DB.
		Where("conversation_id = ?", conversationID).
		Order("created_at DESC").
		Order("message_id DESC").
		First(&last).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	updates := map[string]interface{}{
		"message_count": int(count),
	}
	if strings.TrimSpace(model) != "" {
		updates["model"] = strings.TrimSpace(model)
	}
	if err == nil {
		updates["last_message_preview"] = truncateRunes(strings.TrimSpace(last.Content), 120)
		updates["last_message_at"] = last.CreatedAt
	}
	return utils.DB.
		Model(&LLMConversation{}).
		Where("conversation_id = ?", conversationID).
		Updates(updates).Error
}

func reverseMessages(messages []*LLMConversationMessage) {
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
}

func truncateRunes(s string, limit int) string {
	if limit <= 0 {
		return ""
	}
	rs := []rune(s)
	if len(rs) <= limit {
		return s
	}
	return string(rs[:limit])
}
