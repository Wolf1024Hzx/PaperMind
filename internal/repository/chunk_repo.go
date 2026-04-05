package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"wolfden.website/papermind/internal/model"
)

type ChunkRepository struct {
	db *gorm.DB
}

func NewChunkRepository(db *gorm.DB) *ChunkRepository {
	return &ChunkRepository{
		db: db,
	}
}

func (r *ChunkRepository) BatchCreate(ctx context.Context, chunks []model.Chunk) error {
	return r.db.WithContext(ctx).Create(&chunks).Error
}

func (r *ChunkRepository) FindByPaperID(ctx context.Context, paperID uuid.UUID) ([]model.Chunk, error) {
	var chunks []model.Chunk
	err := r.db.WithContext(ctx).Where("paper_id = ?", paperID).Order("chunk_index ASC").Find(&chunks).Error
	if err != nil {
		return nil, err
	}

	return chunks, nil
}

func (r *ChunkRepository) DeleteChunksByPaperID(ctx context.Context, paperID uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.Chunk{}, "paper_id = ?", paperID).Error
}
