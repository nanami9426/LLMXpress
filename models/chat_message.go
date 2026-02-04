package models

type ChatMessage struct {
	MessageID    int64  `gorm:"primarykey"`
	FromID       int64  // 发送者
	ToID         int64  // 接收者
	MessageType  int    // 消息类型（群聊私聊广播）
	MessageMedia int    // 消息载体（图片文字音频）
	Content      string // 消息内容
	Basic
}

func (c *ChatMessage) TableName() string {
	return "chat_message"
}
