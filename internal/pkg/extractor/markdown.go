package extractor

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"wolfden.website/papermind/internal/pkg/common"
)

// MarkdownMetadata 表示从 Markdown 中提取的元数据
type MarkdownMetadata struct {
	Title string
}

// MarkdownContent 表示从 Markdown 文件中提取的完整内容
type MarkdownContent struct {
	Sections  []common.Section
	Metadata  MarkdownMetadata
}

// ExtractMarkdown 从 Markdown 或纯文本文件中提取章节结构和元数据
func ExtractMarkdown(filePath string) (*MarkdownContent, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	text := string(content)
	filename := filepath.Base(filePath)
	filenameWithoutExt := strings.TrimSuffix(filename, ext)

	if ext == ".txt" {
		return &MarkdownContent{
			Sections: parseTxtContent(text),
			Metadata: MarkdownMetadata{Title: filenameWithoutExt},
		}, nil
	}

	sections, title := parseMarkdownContentWithMetadata(text, filenameWithoutExt)
	return &MarkdownContent{
		Sections: sections,
		Metadata: MarkdownMetadata{Title: title},
	}, nil
}

// parseMarkdownContentWithMetadata 解析 Markdown 格式，识别标题和章节，并提取元数据
// 返回章节列表和论文标题（从第一个一级标题推断，如果没有则用 fallback）
func parseMarkdownContentWithMetadata(content string, fallbackTitle string) ([]common.Section, string) {
	lines := strings.Split(content, "\n")

	// 找出所有标题行
	type headingInfo struct {
		title     string
		level     int // 1, 2, 3
		lineIndex int
	}

	var headings []headingInfo
	var inferredTitle string

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// 检查是否是标题
		level, title := parseMarkdownHeading(trimmed)
		if level > 0 {
			headings = append(headings, headingInfo{
				title:     title,
				level:     level,
				lineIndex: i,
			})
			// 第一个一级标题作为论文标题
			if level == 1 && inferredTitle == "" {
				inferredTitle = title
			}
		}
	}

	// 如果没有推断出标题，使用 fallback
	if inferredTitle == "" {
		inferredTitle = fallbackTitle
	}

	// 如果没有识别出任何标题，整篇文档作为一个 section
	if len(headings) == 0 {
		return []common.Section{
			{
				Type:      common.TypeOther,
				Title:     "Full Document",
				Content:   strings.TrimSpace(content),
				StartPage: 1,
			},
		}, inferredTitle
	}

	// 根据标题位置切分内容
	var sections []common.Section

	// 处理第一个标题之前的内容
	if headings[0].lineIndex > 0 {
		preContent := strings.Join(lines[:headings[0].lineIndex], "\n")
		preContent = strings.TrimSpace(preContent)
		if preContent != "" {
			sections = append(sections, common.Section{
				Type:      common.TypeOther,
				Title:     "Preamble",
				Content:   preContent,
				StartPage: 1,
			})
		}
	}

	// 处理每个标题及其对应的内容
	for i, heading := range headings {
		endIndex := len(lines)
		if i+1 < len(headings) {
			endIndex = headings[i+1].lineIndex
		}

		// 提取标题到下一个标题之间的文本（不包括标题行本身）
		contentLines := lines[heading.lineIndex+1:endIndex]
		sectionContent := strings.TrimSpace(strings.Join(contentLines, "\n"))

		sectionType := common.MatchSectionType(heading.title)

		sections = append(sections, common.Section{
			Type:      sectionType,
			Title:     heading.title,
			Content:   sectionContent,
			StartPage: 1,
		})
	}

	return sections, inferredTitle
}

// parseMarkdownHeading 解析 Markdown 标题行，返回级别和标题文本
// 返回 0 表示不是标题
func parseMarkdownHeading(line string) (level int, title string) {
	// 检查 # ## ### 格式
	if strings.HasPrefix(line, "### ") {
		return 3, strings.TrimSpace(line[4:])
	}
	if strings.HasPrefix(line, "## ") {
		return 2, strings.TrimSpace(line[3:])
	}
	if strings.HasPrefix(line, "# ") {
		return 1, strings.TrimSpace(line[2:])
	}
	return 0, ""
}

// parseTxtContent 解析纯文本格式，按连续空行切分段落
func parseTxtContent(content string) []common.Section {
	// 按连续两个以上换行符切分
	paragraphRegex := regexp.MustCompile(`\n{2,}`)
	paragraphs := paragraphRegex.Split(strings.TrimSpace(content), -1)

	var sections []common.Section

	for i, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		sections = append(sections, common.Section{
			Type:      common.TypeOther,
			Title:     fmt.Sprintf("Paragraph %d", i+1),
			Content:   para,
			StartPage: 1,
		})
	}

	// 如果没有任何段落，返回空文档
	if len(sections) == 0 {
		return []common.Section{
			{
				Type:      common.TypeOther,
				Title:     "Empty Document",
				Content:   "",
				StartPage: 1,
			},
		}
	}

	return sections
}