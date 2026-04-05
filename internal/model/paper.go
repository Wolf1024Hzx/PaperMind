package model

import (
	"time"

	"github.com/google/uuid"
)

type Paper struct {
	ID         uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	UserID     uuid.UUID `gorm:"column:user_id;not null" json:"userID"`
	Filename   string    `gorm:"column:filename;not null" json:"filename"`
	FileSize   int64     `gorm:"column:file_size;not null" json:"fileSize"`
	FileHash   string    `gorm:"column:file_hash;not null" json:"fileHash"`
	Title      string    `gorm:"column:title" json:"title"`
	Authors    string    `gorm:"column:authors" json:"authors"`
	Year       *int      `gorm:"column:year" json:"year"` // 可空，用指针
	Venue      string    `gorm:"column:venue" json:"venue"`
	Abstract   string    `gorm:"column:abstract" json:"abstract"`
	ChunkCount int       `gorm:"column:chunk_count;not null;default:0" json:"chunkCount"`
	Status     string    `gorm:"column:status;not null;default:pending" json:"status"`
	CreatedAt  time.Time `gorm:"column:created_at;not null" json:"createdAt"`
	UpdatedAt  time.Time `gorm:"column:updated_at;not null" json:"updatedAt"`
}

func (Paper) TableName() string {
	return "papers"
}
