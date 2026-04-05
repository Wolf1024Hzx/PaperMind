package parser

import (
	"log"
	"regexp"
	"strings"

	"wolfden.website/papermind/internal/pkg/extractor"
)

// Section 表示论文中的一个章节
type Section struct {
	Type      string // abstract / introduction / related_work / method / experiment / conclusion / other
	Title     string // 原始章节标题文本，如 "3.2 Multi-Head Attention"
	Content   string // 该章节的全部文本内容
	StartPage int    // 该章节起始页码
}

// sectionKeywords 定义了论文章节类型与关键词的映射
var sectionKeywords = map[string][]string{
	"abstract":     {"abstract"},
	"introduction": {"introduction", "intro"},
	"related_work": {"related work", "background", "literature review", "prior work", "preliminary", "preliminaries"},
	"method":       {"method", "methodology", "approach", "proposed method", "model", "framework", "architecture", "our approach", "proposed approach", "system design"},
	"experiment":   {"experiment", "evaluation", "results", "empirical", "setup", "experimental setup", "experimental results"},
	"discussion":   {"discussion", "analysis", "ablation"},
	"conclusion":   {"conclusion", "summary", "future work", "concluding remarks"},
}

// headingPattern 匹配常见的论文章节标题格式
var headingPattern = regexp.MustCompile(
	`^` +
		`(?:` +
		`\d+(?:\.\d+)*\.?\s+` + // 阿拉伯数字编号: "1." "1" "3.2" "3.2."
		`|` +
		`[IVX]+\.?\s+` + // 罗马数字编号: "I." "IV" "III."
		`)?` +
		`([A-Z][A-Za-z\s:&\-]+)` + // 标题文本：首字母大写
		`$`,
)

// noNumberKeywords 允许没有编号也能作为标题的关键词（小写）
var noNumberKeywords = map[string]bool{
	"abstract": true,
}

// ParseSections 将逐页文本解析为结构化的章节列表
func ParseSections(pages []extractor.PageText) []Section {
	// 把所有页的文本按行合并，同时记录每行属于哪一页
	type lineInfo struct {
		text       string
		pageNumber int
	}

	var allLines []lineInfo
	for _, page := range pages {
		if page.Text == "" {
			continue
		}
		lines := strings.Split(page.Text, "\n")
		for _, line := range lines {
			allLines = append(allLines, lineInfo{
				text:       line,
				pageNumber: page.PageNumber,
			})
		}
	}

	if len(allLines) == 0 {
		// 整个 PDF 没有提取出任何文本
		return []Section{{
			Type:      "other",
			Title:     "Full Document",
			Content:   "",
			StartPage: 1,
		}}
	}

	// 找出所有可能的标题行
	type headingInfo struct {
		title       string
		sectionType string
		pageNumber  int
		lineIndex   int
	}

	var headings []headingInfo
	for i, line := range allLines {
		trimmed := strings.TrimSpace(line.text)
		if trimmed == "" {
			continue
		}

		// 调试：打印可能匹配的行（前100行）
		if i < 100 {
			matched := headingPattern.MatchString(trimmed)
			if matched || strings.Contains(strings.ToLower(trimmed), "introduction") ||
				strings.Contains(strings.ToLower(trimmed), "method") ||
				strings.Contains(strings.ToLower(trimmed), "abstract") {
				log.Printf("[DEBUG] line %d: matched=%v, len=%d, spaces=%d, text=%q", i, matched, len(trimmed), strings.Count(trimmed, " "), trimmed)
			}
		}

		if isLikelyHeading(trimmed) {
			sectionType := matchSectionType(trimmed)
			headings = append(headings, headingInfo{
				title:       trimmed,
				sectionType: sectionType,
				pageNumber:  line.pageNumber,
				lineIndex:   i,
			})
		}
	}

	// 如果没有识别出任何标题，整篇文档作为一个 section
	if len(headings) == 0 {
		var fullText strings.Builder
		for _, line := range allLines {
			fullText.WriteString(line.text)
			fullText.WriteString("\n")
		}
		return []Section{{
			Type:      "other",
			Title:     "Full Document",
			Content:   strings.TrimSpace(fullText.String()),
			StartPage: 1,
		}}
	}

	// 根据标题位置切分内容
	var sections []Section

	// 处理第一个标题之前的内容（如果有的话）
	if headings[0].lineIndex > 0 {
		var preContent strings.Builder
		for i := 0; i < headings[0].lineIndex; i++ {
			preContent.WriteString(allLines[i].text)
			preContent.WriteString("\n")
		}
		text := strings.TrimSpace(preContent.String())
		if text != "" {
			sections = append(sections, Section{
				Type:      "other",
				Title:     "Preamble",
				Content:   text,
				StartPage: allLines[0].pageNumber,
			})
		}
	}

	// 处理每个标题及其对应的内容
	for i, heading := range headings {
		// 确定内容的结束位置：下一个标题的起始位置，或文档末尾
		endIndex := len(allLines)
		if i+1 < len(headings) {
			endIndex = headings[i+1].lineIndex
		}

		// 提取标题到下一个标题之间的文本（不包括标题行本身）
		var contentBuilder strings.Builder
		for j := heading.lineIndex + 1; j < endIndex; j++ {
			contentBuilder.WriteString(allLines[j].text)
			contentBuilder.WriteString("\n")
		}

		sections = append(sections, Section{
			Type:      heading.sectionType,
			Title:     heading.title,
			Content:   strings.TrimSpace(contentBuilder.String()),
			StartPage: heading.pageNumber,
		})
	}

	return sections
}

// matchSectionType 将标题文本匹配到对应的 section type
func matchSectionType(title string) string {
	lowerTitle := strings.ToLower(strings.TrimSpace(title))

	for sectionType, keywords := range sectionKeywords {
		for _, keyword := range keywords {
			if strings.Contains(lowerTitle, keyword) {
				return sectionType
			}
		}
	}

	return "other"
}

// isLikelyHeading 判断一行文本是否可能是章节标题
func isLikelyHeading(line string) bool {
	trimmed := strings.TrimSpace(line)

	// 太短或太长的不太可能是标题
	if len(trimmed) < 3 || len(trimmed) > 100 {
		return false
	}

	// 检查空格数量：真正的标题应该有空格分隔单词
	spaceCount := strings.Count(trimmed, " ")
	if len(trimmed) > 30 && spaceCount < 3 {
		return false
	}

	// 如果以句号结尾，检查是否是编号型标题（如 "1. Introduction"）
	if strings.HasSuffix(trimmed, ".") {
		// 允许 "1." "1.1." 等编号结尾（不常见但可能）
		// 排除明显的句子（空格多说明是长句）
		if strings.Count(trimmed, " ") > 5 {
			return false
		}
	}

	// 检查是否匹配标题正则
	if !headingPattern.MatchString(trimmed) {
		return false
	}

	// 检查是否有编号
	hasNumbering := hasHeadingNumber(trimmed)

	// 如果没有编号，只有特定关键词（如 Abstract）才能作为标题
	if !hasNumbering {
		lowerTrimmed := strings.ToLower(trimmed)
		if !noNumberKeywords[lowerTrimmed] {
			return false
		}
	}

	return true
}

// hasHeadingNumber 检查标题是否有编号
func hasHeadingNumber(line string) bool {
	// 检查是否以数字编号开头，如 "1." "1 " "3.2" "2.1."
	// 或罗马数字开头，如 "I." "II " "III."
	if len(line) == 0 {
		return false
	}

	// 阿拉伯数字开头
	if line[0] >= '0' && line[0] <= '9' {
		return true
	}

	// 罗马数字开头 (I, V, X)
	if line[0] == 'I' || line[0] == 'V' || line[0] == 'X' {
		// 检查后面是否跟着 "." 或空格或另一个罗马数字
		if len(line) > 1 {
			next := line[1]
			if next == '.' || next == ' ' || next == 'I' || next == 'V' || next == 'X' {
				return true
			}
		}
	}

	return false
}
