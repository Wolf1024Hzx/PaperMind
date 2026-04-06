package repository

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"wolfden.website/papermind/internal/dto"
)

type VectorRepository struct {
	db *gorm.DB
}

func NewVectorRepository(db *gorm.DB) *VectorRepository {
	return &VectorRepository{db: db}
}

func (vr *VectorRepository) Search(ctx context.Context, retrievalRequest *dto.RetrievalRequest) ([]dto.RetrievalResult, error) {
	if retrievalRequest.TopK <= 0 {
		retrievalRequest.TopK = 5
	}

	sql := `
		SELECT
			c.id AS chunk_id,
			c.paper_id,
			COALESCE(p.title, '') AS paper_title,
			COALESCE(p.authors, '') AS authors,
			p.year,
			COALESCE(c.section_type, 'other') AS section_type,
			COALESCE(c.section_title, '') AS section_title,
			c.page_number,
			c.content,
			1 - (c.embedding <=> $1::vector) AS similarity
		FROM chunks c
		JOIN papers p ON c.paper_id = p.id
		WHERE p.user_id = $2
	`

	args := []interface{}{retrievalRequest.QueryVector.String(), retrievalRequest.UserID}
	argIndex := 3 // $1 是向量，$2 是 user_id，下一个是 $3

	// 如果有 PaperIDs 过滤
	if len(retrievalRequest.PaperIDs) > 0 {
		placeholders := make([]string, len(retrievalRequest.PaperIDs))
		for i, id := range retrievalRequest.PaperIDs {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, id)
			argIndex++
		}
		sql += fmt.Sprintf(" AND c.paper_id IN (%s)", strings.Join(placeholders, ","))
	}

	// 如果有 SectionTypes 过滤
	if len(retrievalRequest.SectionTypes) > 0 {
		placeholders := make([]string, len(retrievalRequest.SectionTypes))
		for i, st := range retrievalRequest.SectionTypes {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, st)
			argIndex++
		}
		sql += fmt.Sprintf(" AND c.section_type IN (%s)", strings.Join(placeholders, ","))
	}

	// 如果有 YearFrom 过滤
	if retrievalRequest.YearFrom != nil {
		sql += fmt.Sprintf(" AND p.year >= $%d", argIndex)
		args = append(args, *retrievalRequest.YearFrom)
		argIndex++
	}

	// 排序
	sql += fmt.Sprintf(" ORDER BY c.embedding <=> $1::vector LIMIT $%d", argIndex)
	args = append(args, retrievalRequest.TopK)

	var retrievalResults []dto.RetrievalResult
	err := vr.db.WithContext(ctx).Raw(sql, args...).Scan(&retrievalResults).Error
	return retrievalResults, err
}
