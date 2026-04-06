package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"

	"wolfden.website/papermind/internal/dto"
	"wolfden.website/papermind/internal/model"
	"wolfden.website/papermind/internal/pkg/chunker"
	"wolfden.website/papermind/internal/pkg/common"
	"wolfden.website/papermind/internal/pkg/extractor"
	"wolfden.website/papermind/internal/pkg/parser"
	"wolfden.website/papermind/internal/repository"
)

type PaperService struct {
	paperRepo               *repository.PaperRepository
	chunkRepo               *repository.ChunkRepository
	embeddingClient         EmbeddingClient
	uploadDir               string
	embeddingBatchSize      int
	embeddingMaxConcurrency int
}

func NewPaperService(
	paperRepo *repository.PaperRepository,
	chunkRepo *repository.ChunkRepository,
	embeddingClient EmbeddingClient,
	uploadDir string,
	embeddingBatchSize int,
	embeddingMaxConcurrency int,
) *PaperService {
	return &PaperService{
		paperRepo:               paperRepo,
		chunkRepo:               chunkRepo,
		embeddingClient:         embeddingClient,
		uploadDir:               uploadDir,
		embeddingBatchSize:      embeddingBatchSize,
		embeddingMaxConcurrency: embeddingMaxConcurrency,
	}
}

func (s *PaperService) UploadPaper(ctx context.Context, userID string, input dto.UploadPaperInput) (*dto.PaperDTO, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidInput
	}

	// 检查文件是否已存在（去重）
	_, err = s.paperRepo.FindByFileHash(ctx, input.FileHash)
	if err == nil {
		return nil, ErrFileAlreadyExists
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	paper := &model.Paper{
		UserID:   userUUID,
		Filename: input.Filename,
		FileSize: input.FileSize,
		FileHash: input.FileHash,
		Title:    input.Title,
		Authors:  input.Authors,
		Year:     input.Year,
		Venue:    input.Venue,
		Status:   "pending",
	}

	// 先创建数据库记录（获取 UUID）
	if err := s.paperRepo.Create(ctx, paper); err != nil {
		return nil, err
	}

	// 再保存文件（使用原始扩展名）
	ext := filepath.Ext(input.Filename)
	filePath := fmt.Sprintf("%s/%s%s", s.uploadDir, paper.ID.String(), ext)
	if err := os.WriteFile(filePath, input.FileData, 0644); err != nil {
		// 文件保存失败，删除数据库记录（补偿操作）
		s.paperRepo.DeletePaper(ctx, paper.ID)
		return nil, err
	}

	// 启动异步处理（不阻塞上传响应）
	go s.processPaper(paper.ID, filePath)

	return toPaperDTO(paper), nil
}

func (s *PaperService) ListByUser(ctx context.Context, userID string) ([]dto.PaperDTO, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	papers, err := s.paperRepo.FindByUserID(ctx, userUUID)
	if err != nil {
		return nil, err
	}
	result := []dto.PaperDTO{}
	for _, paper := range papers {
		result = append(result, *toPaperDTO(&paper))
	}
	return result, nil
}

func (s *PaperService) GetByID(ctx context.Context, userID, paperID string) (*dto.PaperDTO, error) {
	paperUUID, err := uuid.Parse(paperID)
	if err != nil {
		return nil, ErrInvalidInput
	}

	paper, err := s.paperRepo.FindByID(ctx, paperUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPaperNotFound
		}
		return nil, err
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	if paper.UserID != userUUID {
		return nil, ErrPaperNotFound
	}

	return toPaperDTO(paper), nil
}

func (s *PaperService) Delete(ctx context.Context, userID, paperID string) error {
	paperUUID, err := uuid.Parse(paperID)
	if err != nil {
		return ErrInvalidInput
	}

	paper, err := s.paperRepo.FindByID(ctx, paperUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPaperNotFound
		}
		return err
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return ErrInvalidInput
	}
	if paper.UserID != userUUID {
		return ErrPaperNotFound
	}

	// 删除数据库记录
	if err := s.paperRepo.DeletePaper(ctx, paperUUID); err != nil {
		return err
	}

	// 删除文件（尝试所有可能的扩展名）
	for _, ext := range []string{".pdf", ".md", ".txt"} {
		filePath := fmt.Sprintf("%s/%s%s", s.uploadDir, paperUUID.String(), ext)
		os.Remove(filePath)
	}

	return nil
}

// 论文上传后的异步处理流水线
func (s *PaperService) processPaper(paperID uuid.UUID, filePath string) {
	// 防止 panic 扩散
	defer func() {
		if r := recover(); r != nil {
			log.Printf("处理论文 %s 时发生 panic: %v", paperID, r)
			s.updateStatus(context.Background(), paperID, "failed")
		}
	}()

	// 使用 background context，因为请求可能已经结束
	ctx := context.Background()

	// 根据文件扩展名选择处理路径
	ext := strings.ToLower(filepath.Ext(filePath))

	var sections []common.Section

	switch ext {
	case ".pdf":
		sections = s.processPDF(ctx, filePath, paperID)
	case ".md", ".txt":
		sections = s.processMarkdown(ctx, filePath, paperID)
	default:
		s.updateStatus(ctx, paperID, "failed")
		log.Printf("论文 %s 不支持的文件类型: %s", paperID, ext)
		return
	}

	if len(sections) == 0 {
		s.updateStatus(ctx, paperID, "failed")
		return
	}

	// 打印解析结果到日志
	log.Printf("论文 %s 解析完成，识别出 %d 个章节:", paperID, len(sections))
	for i, sec := range sections {
		log.Printf("  [%d] type=%s, title=%s, page=%d, contentLen=%d",
			i, sec.Type, sec.Title, sec.StartPage, len(sec.Content))
	}

	// ========== 切片 + Embedding ==========

	// 先切片
	s.updateStatus(ctx, paperID, "chunking")
	chunkResults := chunker.ChunkSections(sections)

	if len(chunkResults) == 0 {
		s.updateStatus(ctx, paperID, "failed")
		log.Printf("论文 %s 切片结果为空", paperID)
		return
	}

	log.Printf("论文 %s 切片完成，共 %d 个 chunks", paperID, len(chunkResults))

	// 开始抽特征
	s.updateStatus(ctx, paperID, "embedding")
	texts := make([]string, len(chunkResults))
	for i, cr := range chunkResults {
		texts[i] = cr.Content
	}

	// 分批并发调用 Embedding API
	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(s.embeddingMaxConcurrency)

	allEmbeddings := make([][]float32, len(texts))

	for batchStart := 0; batchStart < len(texts); batchStart += s.embeddingBatchSize {
		start := batchStart
		end := min(start+s.embeddingBatchSize, len(texts))

		g.Go(func() error {
			batch := texts[start:end]
			embeddings, err := s.embeddingClient.Embed(gCtx, batch)
			if err != nil {
				return fmt.Errorf("Embedding 批次 [%d:%d] 失败: %w", start, end, err)
			}
			for i, emb := range embeddings {
				allEmbeddings[start+i] = emb
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		s.updateStatus(ctx, paperID, "failed")
		log.Printf("论文 %s Embedding 失败: %v", paperID, err)
		return
	}

	// 构建 Chunk 模型并写入数据库
	chunks := make([]model.Chunk, len(chunkResults))
	for i, cr := range chunkResults {
		sectionType := cr.SectionType
		sectionTitle := cr.SectionTitle
		pageNumber := cr.PageNumber

		chunks[i] = model.Chunk{
			PaperID:      paperID,
			ChunkIndex:   i,
			Content:      cr.Content,
			TokenCount:   cr.TokenCount,
			Embedding:    pgvector.NewVector(allEmbeddings[i]),
			SectionType:  &sectionType,
			SectionTitle: &sectionTitle,
			PageNumber:   &pageNumber,
		}
	}

	if err := s.chunkRepo.BatchCreate(ctx, chunks); err != nil {
		s.updateStatus(ctx, paperID, "failed")
		log.Printf("论文 %s chunks 写入数据库失败: %v", paperID, err)
		return
	}

	// 更新论文状态和 chunk 数量
	s.paperRepo.UpdateChunkCount(ctx, paperID, len(chunks))
	s.updateStatus(ctx, paperID, "completed")
	log.Printf("论文 %s 处理完成，共写入 %d 个 chunks", paperID, len(chunks))
}

// processPDF 提取 PDF 并解析章节
func (s *PaperService) processPDF(ctx context.Context, filePath string, paperID uuid.UUID) []common.Section {
	s.updateStatus(ctx, paperID, "extracting")

	pdfContent, err := extractor.ExtractPDF(filePath)
	if err != nil {
		s.updateStatus(ctx, paperID, "failed")
		log.Printf("论文 %s PDF 提取失败: %v", paperID, err)
		return nil
	}

	// 检查是否提取出了有效文本
	if strings.TrimSpace(pdfContent.FullText) == "" {
		s.updateStatus(ctx, paperID, "failed")
		log.Printf("论文 %s PDF 未提取出文本（可能是扫描件）", paperID)
		return nil
	}

	// 用提取到的元数据更新论文记录
	s.updateMetadataFromPDF(ctx, paperID, pdfContent.Metadata)

	s.updateStatus(ctx, paperID, "parsing")
	return parser.ParseSections(pdfContent.Pages)
}

// processMarkdown 提取 Markdown/文本并解析章节
func (s *PaperService) processMarkdown(ctx context.Context, filePath string, paperID uuid.UUID) []common.Section {
	s.updateStatus(ctx, paperID, "extracting")

	mdContent, err := extractor.ExtractMarkdown(filePath)
	if err != nil {
		s.updateStatus(ctx, paperID, "failed")
		log.Printf("论文 %s Markdown 提取失败: %v", paperID, err)
		return nil
	}

	// 用提取到的元数据更新论文记录
	s.updateMetadataFromMarkdown(ctx, paperID, mdContent.Metadata)

	s.updateStatus(ctx, paperID, "parsing")
	return mdContent.Sections
}

// updateMetadataFromMarkdown 用 Markdown 元数据更新论文记录
func (s *PaperService) updateMetadataFromMarkdown(ctx context.Context, paperID uuid.UUID, meta extractor.MarkdownMetadata) {
	if meta.Title != "" {
		s.paperRepo.UpdateMetadataIfEmpty(ctx, paperID, map[string]interface{}{"title": meta.Title})
	}
}

// updateStatus 更新论文的处理状态
func (s *PaperService) updateStatus(ctx context.Context, paperID uuid.UUID, status string) {
	err := s.paperRepo.UpdateStatus(ctx, paperID, status)
	if err != nil {
		log.Printf("更新论文 %s 状态为 %s 失败: %v", paperID, status, err)
	}
}

// updateMetadataFromPDF 用 PDF 元数据更新论文记录
func (s *PaperService) updateMetadataFromPDF(ctx context.Context, paperID uuid.UUID, meta extractor.PDFMetadata) {
	updates := make(map[string]interface{})

	if meta.Title != "" {
		updates["title"] = meta.Title
	}
	if meta.Author != "" {
		updates["authors"] = meta.Author
	}

	if len(updates) > 0 {
		s.paperRepo.UpdateMetadataIfEmpty(ctx, paperID, updates)
	}
}

func toPaperDTO(paper *model.Paper) *dto.PaperDTO {
	return &dto.PaperDTO{
		ID:         paper.ID.String(),
		Filename:   paper.Filename,
		FileSize:   paper.FileSize,
		Title:      paper.Title,
		Authors:    paper.Authors,
		Year:       paper.Year,
		Venue:      paper.Venue,
		Status:     paper.Status,
		ChunkCount: paper.ChunkCount,
		CreatedAt:  paper.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:  paper.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
