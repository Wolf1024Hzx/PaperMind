package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
)

type Chunk struct {
	ID           uuid.UUID       `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	PaperID      uuid.UUID       `gorm:"column:paper_id;not null" json:"paperID"`
	ChunkIndex   int             `gorm:"column:chunk_index;not null" json:"chunkIndex"`
	Content      string          `gorm:"column:content;not null" json:"content"`
	TokenCount   int             `gorm:"column:token_count;not null" json:"tokenCount"`
	Embedding    pgvector.Vector `gorm:"column:embedding;type:vector(1024);not null" json:"-"`
	SectionType  *string         `gorm:"column:section_type" json:"sectionType"`
	SectionTitle *string         `gorm:"column:section_title" json:"sectionTitle"`
	PageNumber   *int            `gorm:"column:page_number" json:"pageNumber"`
	CreatedAt    time.Time       `gorm:"column:created_at" json:"createdAt"`
}

func (Chunk) TableName() string {
	return "chunks"
}
