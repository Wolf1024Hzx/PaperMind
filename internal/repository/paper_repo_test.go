package repository

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"wolfden.website/papermind/internal/model"
)

// setupTestDB 创建测试用的 mock 数据库连接
func setupTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: sqlDB,
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("创建 gorm DB 失败: %v", err)
	}

	return gormDB, mock
}

func TestFindByUserIDAndFileHash_Found(t *testing.T) {
	db, mock := setupTestDB(t)
	repo := NewPaperRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	fileHash := "abc123def456"
	paperID := uuid.New()

	rows := sqlmock.NewRows([]string{"id", "user_id", "filename", "file_size", "file_hash", "title", "authors", "year", "venue", "abstract", "chunk_count", "status", "created_at", "updated_at"}).
		AddRow(paperID, userID, "test.pdf", 1024, fileHash, "Test Paper", "Author", 2024, "Venue", "Abstract", 0, "pending", nil, nil)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "papers" WHERE user_id = $1 AND file_hash = $2 ORDER BY "papers"."id" LIMIT $3`)).
		WithArgs(userID, fileHash, 1).
		WillReturnRows(rows)

	paper, err := repo.FindByUserIDAndFileHash(ctx, userID, fileHash)
	if err != nil {
		t.Errorf("期望找到论文，但返回错误: %v", err)
	}
	if paper == nil {
		t.Error("期望返回论文，但返回 nil")
	}
	if paper.FileHash != fileHash {
		t.Errorf("期望 FileHash=%s，实际=%s", fileHash, paper.FileHash)
	}
	if paper.UserID != userID {
		t.Errorf("期望 UserID=%s，实际=%s", userID, paper.UserID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("未满足的期望: %v", err)
	}
}

func TestFindByUserIDAndFileHash_NotFound(t *testing.T) {
	db, mock := setupTestDB(t)
	repo := NewPaperRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	fileHash := "abc123def456"

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "papers" WHERE user_id = $1 AND file_hash = $2 ORDER BY "papers"."id" LIMIT $3`)).
		WithArgs(userID, fileHash, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "filename", "file_size", "file_hash", "title", "authors", "year", "venue", "abstract", "chunk_count", "status", "created_at", "updated_at"}))

	_, err := repo.FindByUserIDAndFileHash(ctx, userID, fileHash)
	if err == nil {
		t.Error("期望返回 gorm.ErrRecordNotFound，但返回 nil")
	}
	if err != gorm.ErrRecordNotFound {
		t.Errorf("期望错误为 gorm.ErrRecordNotFound，实际为: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("未满足的期望: %v", err)
	}
}

func TestFindByUserIDAndFileHash_DifferentUsers(t *testing.T) {
	db, mock := setupTestDB(t)
	repo := NewPaperRepository(db)
	ctx := context.Background()

	userA := uuid.New()
	userB := uuid.New()
	fileHash := "same-file-hash"
	paperID := uuid.New()

	// 用户A的论文存在
	rowsA := sqlmock.NewRows([]string{"id", "user_id", "filename", "file_size", "file_hash", "title", "authors", "year", "venue", "abstract", "chunk_count", "status", "created_at", "updated_at"}).
		AddRow(paperID, userA, "test.pdf", 1024, fileHash, "Test Paper", "Author", 2024, "Venue", "Abstract", 0, "pending", nil, nil)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "papers" WHERE user_id = $1 AND file_hash = $2 ORDER BY "papers"."id" LIMIT $3`)).
		WithArgs(userA, fileHash, 1).
		WillReturnRows(rowsA)

	// 用户B查询相同文件哈希，应该返回未找到
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "papers" WHERE user_id = $1 AND file_hash = $2 ORDER BY "papers"."id" LIMIT $3`)).
		WithArgs(userB, fileHash, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "filename", "file_size", "file_hash", "title", "authors", "year", "venue", "abstract", "chunk_count", "status", "created_at", "updated_at"}))

	// 用户A应该找到论文
	paperA, err := repo.FindByUserIDAndFileHash(ctx, userA, fileHash)
	if err != nil {
		t.Errorf("用户A期望找到论文，但返回错误: %v", err)
	}
	if paperA == nil {
		t.Error("用户A期望返回论文，但返回 nil")
	}

	// 用户B不应该找到论文
	_, err = repo.FindByUserIDAndFileHash(ctx, userB, fileHash)
	if err == nil {
		t.Error("用户B期望返回 gorm.ErrRecordNotFound，但返回 nil")
	}
	if err != gorm.ErrRecordNotFound {
		t.Errorf("用户B期望错误为 gorm.ErrRecordNotFound，实际为: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("未满足的期望: %v", err)
	}
}

func TestCreate(t *testing.T) {
	db, mock := setupTestDB(t)
	repo := NewPaperRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	paperID := uuid.New()

	paper := &model.Paper{
		ID:       paperID,
		UserID:   userID,
		Filename: "test.pdf",
		FileSize: 1024,
		FileHash: "abc123",
		Status:   "pending",
	}

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "papers"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(paperID))
	mock.ExpectCommit()

	err := repo.Create(ctx, paper)
	if err != nil {
		t.Errorf("创建论文失败: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("未满足的期望: %v", err)
	}
}
