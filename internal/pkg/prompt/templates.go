package prompt

import (
	"fmt"
	"strings"
)

// ExtractSystemPrompt 知识提取模式的系统提示词
const ExtractSystemPrompt = `你是一个论文阅读助手。请严格根据以下论文片段回答用户的问题。
规则：
1. 只基于提供的论文内容回答，不要引入外部知识。
2. 如果提供的内容无法回答问题，明确告知"在现有论文库中未找到相关信息"。
3. 回答时标注引用来源，格式为 [来源1]、[来源2]。
4. 使用学术化但易懂的语言。`

// CompareSystemPrompt 跨论文对比模式的系统提示词
const CompareSystemPrompt = `你是一个论文对比分析助手。请根据以下来自不同论文的片段，对用户提出的对比问题进行分析。
规则：
1. 以结构化的方式呈现对比结果（可以使用表格）。
2. 明确指出各方法/论文的异同点。
3. 每个观点都要标注引用来源 [来源N]。
4. 如果某方面信息不足以对比，如实说明。`

// Reference 表示一条引用来源信息
type Reference struct {
	Index        int    // 来源编号，如 1、2、3
	PaperTitle   string // 论文标题
	Authors      string // 作者
	SectionTitle string // 章节标题
	PageNumber   *int   // 页码（可能为空）
	Content      string // chunk 内容
}

// BuildUserPrompt 构建用户提问的 Prompt（包含检索结果）
func BuildUserPrompt(question string, refs []Reference) string {
	var builder strings.Builder

	builder.WriteString("以下是从论文库中检索到的相关片段：\n\n")

	for _, ref := range refs {
		fmt.Fprintf(&builder, "[来源%d] 论文：《%s》", ref.Index, ref.PaperTitle)
		if ref.Authors != "" {
			fmt.Fprintf(&builder, "（%s）", ref.Authors)
		}
		builder.WriteString("\n")

		if ref.SectionTitle != "" {
			fmt.Fprintf(&builder, "章节：%s", ref.SectionTitle)
			if ref.PageNumber != nil {
				fmt.Fprintf(&builder, "（第 %d 页）", *ref.PageNumber)
			}
			builder.WriteString("\n")
		}

		fmt.Fprintf(&builder, "内容：%s\n\n", ref.Content)
	}

	fmt.Fprintf(&builder, "用户问题：%s\n\n请基于以上论文内容回答：", question)

	return builder.String()
}
