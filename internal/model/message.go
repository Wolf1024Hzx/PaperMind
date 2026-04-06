package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Message struct {
	ID             uuid.UUID        `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	ConversationID uuid.UUID        `gorm:"type:uuid;not null;index" json:"conversationId"`
	Role           string           `gorm:"column:role;size:16;not null" json:"role"`
	Content        string           `gorm:"column:content;type:text;not null" json:"content"`
	ReferencesData *json.RawMessage `gorm:"column:references_data;type:jsonb" json:"references,omitempty"`
	TokenUsage     *json.RawMessage `gorm:"column:token_usage;type:jsonb" json:"tokenUsage,omitempty"`
	CreatedAt      time.Time        `gorm:"autoCreateTime" json:"createdAt"`
}

func (Message) TableName() string {
	return "messages"
}
