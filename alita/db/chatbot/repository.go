package chatbot

import (
	"github.com/divkix/Alita_Robot/alita/db"
	"github.com/divkix/Alita_Robot/alita/db/models"
	"gorm.io/gorm/clause"
)

// IsEnabled reports whether conversational AI is enabled in a group.
func IsEnabled(chatID int64) bool {
	var settings models.ChatBotSettings
	if err := db.DB.Select("enabled").Where("chat_id = ?", chatID).First(&settings).Error; err != nil {
		return false
	}
	return settings.Enabled
}

// SetEnabled persists the group chatbot setting across restarts.
func SetEnabled(chatID int64, enabled bool) error {
	return db.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "chat_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"enabled", "updated_at"}),
	}).Create(&models.ChatBotSettings{ChatID: chatID, Enabled: enabled}).Error
}
