package models

import "time"

// ChatBotSettings stores the per-group opt-in switch for conversational AI.
type ChatBotSettings struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"-"`
	ChatID    int64     `gorm:"column:chat_id;uniqueIndex;not null" json:"chat_id,omitempty"`
	Enabled   bool      `gorm:"column:enabled;default:false" json:"enabled,omitempty"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at,omitempty"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at,omitempty"`
}

func (ChatBotSettings) TableName() string { return "chatbot_settings" }
