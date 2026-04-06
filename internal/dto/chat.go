package dto

import (
	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
)

// ChatRequest 问答请求参数
type ChatRequest struct {
	ConversationID *uuid.UUID  `json:"conversationId,omitempty"` // 可选：追加到已有对话
	Question       string      `json:"question"`
	PaperIDs       []uuid.UUID `json:"paperIds,omitempty"`
	SectionTypes   []string    `json:"sectionTypes,omitempty"`
	YearFrom       *int        `json:"yearFrom,omitempty"`
}

// ChatResult 问答返回结果
type ChatResult struct {
	ConversationID uuid.UUID         `json:"conversationId"`
	Mode           string            `json:"mode"` // extract 或 compare
	Answer         string            `json:"answer"`
	References     []RetrievalResult `json:"references"` // 引用来源列表
	TokenUsage     map[string]int    `json:"tokenUsage"` // token 统计
}

// RetrievalRequest 向量检索请求参数
type RetrievalRequest struct {
	QueryVector  pgvector.Vector
	UserID       uuid.UUID   // 必须：只检索该用户上传的论文
	PaperIDs     []uuid.UUID // 可选：限定论文范围
	SectionTypes []string    // 可选：限定章节类型
	YearFrom     *int        // 可选：最早年份
	TopK         int
}

// RetrievalResult 向量检索结果
type RetrievalResult struct {
	ChunkID      uuid.UUID // chunk 的 ID
	PaperID      uuid.UUID // 论文 ID
	PaperTitle   string    // 论文标题
	Authors      string    // 作者
	Year         *int      // 年份（可能为空）
	SectionType  string    // 章节类型
	SectionTitle string    // 章节标题
	PageNumber   *int      // 页码（可能为空）
	Content      string    // chunk 内容
	Similarity   float64   // 相似度分数（0-1）
}
