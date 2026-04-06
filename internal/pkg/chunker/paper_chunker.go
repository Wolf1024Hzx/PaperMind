package chunker

import (
	"strings"

	"wolfden.website/papermind/internal/pkg/common"
)

const (
	MaxChunkSize = 512
	ChunkOverlap = 64
	MinChunkSize = 30
)

type ChunkResult struct {
	Content      string
	TokenCount   int
	SectionType  string
	SectionTitle string
	PageNumber   int
}

// ChunkSections 将论文的章节列表切分为适合 Embedding 的文本块
// 策略：两级切片
//
//	第一级：按 Section 切分（保持语义完整）
//	第二级：对过长的 Section，按 token 数 + overlap 细分
func ChunkSections(sections []common.Section) []ChunkResult {
	chunks := make([]ChunkResult, 0)

	for _, section := range sections {
		tokenCount := countTokens(section.Content)

		if tokenCount <= MaxChunkSize {
			// Section 足够短，整体作为一个 chunk
			if tokenCount >= MinChunkSize {
				chunks = append(chunks, newChunkResult(
					section.Content,
					section.Type,
					section.Title,
					section.StartPage,
				))
			}
		} else {
			// Section 太长，按 token 数细分
			subChunks := splitByTokens(section.Content, MaxChunkSize, ChunkOverlap)
			for _, subContent := range subChunks {
				if countTokens(subContent) < MinChunkSize {
					continue
				}
				chunks = append(chunks, newChunkResult(
					subContent,
					section.Type,
					section.Title,
					section.StartPage,
				))
			}
		}
	}

	return chunks
}

// countTokens 简易 token 计数（按空格分词）
// 对英文论文足够准确，生产环境可换 tiktoken
func countTokens(text string) int {
	return len(strings.Fields(text))
}

func newChunkResult(content, sectionType, sectionTitle string, pageNumber int) ChunkResult {
	return ChunkResult{
		Content:      content,
		TokenCount:   countTokens(content),
		SectionType:  sectionType,
		SectionTitle: sectionTitle,
		PageNumber:   pageNumber,
	}
}

// splitByTokens 将长文本按 token 数切分，支持 overlap
func splitByTokens(text string, chunkSize int, overlap int) []string {
	words := strings.Fields(text)
	if len(words) <= chunkSize {
		return []string{text}
	}

	step := chunkSize - overlap // 每次前进的步长
	result := make([]string, 0)

	for start := 0; start < len(words); start += step {
		end := min(start+chunkSize, len(words))

		result = append(result, strings.Join(words[start:end], " "))

		if end == len(words) {
			break
		}
	}

	return result
}
