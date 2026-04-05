package common

import "strings"

// Section 表示论文中的一个章节
type Section struct {
	Type      string // abstract / introduction / related_work / method / experiment / conclusion / other
	Title     string // 原始章节标题文本，如 "3.2 Multi-Head Attention"
	Content   string // 该章节的全部文本内容
	StartPage int    // 该章节起始页码
}

// SectionType 章节类型常量
const (
	TypeAbstract     = "abstract"
	TypeIntroduction = "introduction"
	TypeRelatedWork  = "related_work"
	TypeMethod       = "method"
	TypeExperiment   = "experiment"
	TypeDiscussion   = "discussion"
	TypeConclusion   = "conclusion"
	TypeOther        = "other"
)

// SectionKeywords 章节标题关键词映射
var SectionKeywords = map[string][]string{
	TypeAbstract:     {"abstract"},
	TypeIntroduction: {"introduction", "intro"},
	TypeRelatedWork:  {"related work", "background", "literature review", "prior work", "preliminary", "preliminaries"},
	TypeMethod:       {"method", "methodology", "approach", "proposed method", "model", "framework", "architecture", "our approach", "proposed approach", "system design"},
	TypeExperiment:   {"experiment", "evaluation", "results", "empirical", "setup", "experimental setup", "experimental results"},
	TypeDiscussion:   {"discussion", "analysis", "ablation"},
	TypeConclusion:   {"conclusion", "summary", "future work", "concluding remarks"},
}

// MatchSectionType 根据标题文本匹配章节类型
func MatchSectionType(title string) string {
	lowerTitle := strings.ToLower(strings.TrimSpace(title))
	for sectionType, keywords := range SectionKeywords {
		for _, keyword := range keywords {
			if strings.Contains(lowerTitle, keyword) {
				return sectionType
			}
		}
	}
	return TypeOther
}