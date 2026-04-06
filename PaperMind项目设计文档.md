# PaperMind — 面向科研场景的论文知识检索与问答系统

## 项目设计文档（Project Design Document）

---

## 1. 项目概述

### 1.1 项目定位

PaperMind 是一个面向科研工作者的论文知识检索与问答系统，使用 Go 语言构建后端。研究人员可以将自己研究方向的论文批量上传，系统自动解析论文结构、构建向量化知识库；随后可以通过自然语言提问，实现**单论文知识提取**和**跨论文方法对比**两种核心能力。

### 1.2 为什么做这个项目

> **面试话术**：
> "我在硕士开题调研阶段需要精读几十篇论文，经常遇到两个痛点：第一，读完一篇论文过几天就忘了具体细节；第二，要对比多篇论文的方法差异时效率很低。市面的通用 RAG 工具存在 token 限制、没有持久化知识库、不能做结构化跨论文对比的问题。所以我用 Go 做了这个系统。"

**技术层面动机**：
- token 限制：直接把大量论文喂给大模型超出上下文窗口
- 幻觉控制：RAG 要求回答基于检索原文，降低幻觉
- 知识持久化：论文上传一次永久入库

### 1.3 与通用 RAG 工具的区分点

|维度|通用 RAG|PaperMind|
|---|---|---|
|切片策略|固定 token 数盲切|感知论文结构，按 Section 切片|
|元数据|仅文件名|论文标题、作者、年份、Section、页码|
|问答模式|单一模式|知识提取 + 跨论文对比|
|引用溯源|"来自 chunk_037"|"来自《Attention》Method 章节，第 5 页"|

---

## 2. 技术选型

|层级|技术|选型理由|
|---|---|---|
|语言|Go 1.22+|目标岗位要求，并发模型适合 I/O 密集场景|
|Web 框架|Gin|国内 Go 后端最主流框架|
|ORM|GORM|管理业务元数据，自动迁移|
|数据库|PostgreSQL + pgvector|一套数据库同时存元数据和向量|
|缓存|Redis|缓存问答、限流、进度状态|
|Embedding|阿里云 qwen3-vl-embedding|国产成本低，中文效果好|
|LLM|DeepSeek / 通义千问|性价比高，国内访问稳定|

**为什么选 pgvector 而不是 Milvus**：
- 科研场景论文量在几百到几千篇，pgvector HNSW 索引完全够用
- 一套数据库运维简单，支持事务一致性
- 接口已抽象，可无缝切换

---

## 3. 目录结构

```
papermind/
├── cmd/server/main.go
├── internal/
│   ├── config/config.go           # .env 配置加载
│   ├── app/                       # 应用主体 + 依赖容器
│   ├── handler/                   # HTTP Handler 层
│   ├── service/                   # 业务逻辑层
│   │   ├── paper_service.go       # 论文处理流水线 ✅
│   │   ├── embedding_service.go   # Embedding API ✅
│   │   ├── retrieval_service.go   # 检索逻辑 ❌
│   │   ├── chat_service.go        # 对话逻辑 ❌
│   ├── repository/                # 数据访问层
│   │   ├── chunk_repo.go          # ✅
│   │   ├── vector_repo.go         # ❌
│   ├── model/                     # 数据模型
│   ├── pkg/
│   │   ├── chunker/               # 切片器 ✅
│   │   ├── extractor/             # PDF/Markdown 提取 ✅
│   │   ├── parser/                # 章节解析 ✅
│   │   ├── llm/                   # LLM 客户端 ❌
│   │   └── prompt/                # Prompt 模板 ❌
├── uploads/papers/
├── .env.example
├── scripts/init.sql
└── deployments/
```

---

## 4. 数据库设计

### 4.1 表结构（已建）

**chunks 表**：
```sql
CREATE TABLE chunks (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    paper_id     UUID          NOT NULL REFERENCES papers(id) ON DELETE CASCADE,
    chunk_index  INT           NOT NULL,
    content      TEXT          NOT NULL,
    token_count  INT           NOT NULL,
    embedding    vector(1024)  NOT NULL,
    section_type  VARCHAR(64),
    section_title VARCHAR(256),
    page_number   INT,
    created_at   TIMESTAMP     NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_chunks_embedding ON chunks USING hnsw (embedding vector_cosine_ops);
CREATE INDEX idx_chunks_paper_id ON chunks(paper_id);
CREATE INDEX idx_chunks_section_type ON chunks(section_type);
```

**conversations / messages 表**（待建）：
```sql
CREATE TABLE conversations (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID         NOT NULL REFERENCES users(id),
    title       VARCHAR(256),
    mode        VARCHAR(32)  NOT NULL DEFAULT 'extract',  -- extract / compare
    created_at  TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP    NOT NULL DEFAULT NOW()
);

CREATE TABLE messages (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    conversation_id UUID        NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    role            VARCHAR(16) NOT NULL,  -- user / assistant
    content         TEXT        NOT NULL,
    references      JSONB,      -- 引用来源
    token_usage     JSONB,
    created_at      TIMESTAMP   NOT NULL DEFAULT NOW()
);
```

---

## 5. 核心功能模块

### 5.1 模块一：论文上传与文本提取 ✅ 已完成

流程：`上传 → 校验 → 计算哈希去重 → 创建记录 → 异步处理`

支持 PDF 和 Markdown 格式，提取文本和元数据。

### 5.2 模块二：论文结构解析 ✅ 已完成

基于规则的章节识别器，识别 Abstract、Introduction、Method 等结构。

标准会议论文识别率 85-90%，不标准的降级为 `other`。

### 5.3 模块三：结构感知切片 ✅ 已完成

**两级切片**：先按 Section 切分，对过长 Section 再按 token + overlap 细分。

参数：MaxChunkSize=512, Overlap=64, MinChunkSize=30

### 5.4 模块四：Embedding 向量化 ✅ 已完成

`EmbeddingClient` 接口支持 Mock/Qwen 切换。

`errgroup` 并发调用，每批 20 条，最大并发 4。

---

### 5.5 模块五：检索（结构化过滤 + 向量检索）❌ 待实现

**这是场景差异化的关键点。**

**普通向量检索**：
```sql
SELECT * FROM chunks ORDER BY embedding <=> $1 LIMIT 5;
```

**PaperMind 增强检索**：
```sql
-- 场景1：限定章节类型过滤
SELECT id, paper_id, content, section_title, page_number,
       1 - (embedding <=> $1) AS similarity
FROM chunks
WHERE paper_id = $2
  AND section_type IN ('experiment', 'discussion')
ORDER BY embedding <=> $1
LIMIT 5;

-- 场景2：跨论文对比，从所有 method 章节检索
SELECT c.*, p.title AS paper_title, p.authors, p.year
FROM chunks c
JOIN papers p ON c.paper_id = p.id
WHERE p.user_id = $2
  AND c.section_type = 'method'
ORDER BY c.embedding <=> $1
LIMIT 10;

-- 场景3：限定年份范围
SELECT c.*, p.title AS paper_title
FROM chunks c
JOIN papers p ON c.paper_id = p.id
WHERE p.user_id = $2
  AND p.year >= 2023
ORDER BY c.embedding <=> $1
LIMIT 5;
```

**检索参数设计**：
```go
type RetrievalRequest struct {
    Query        string   // 用户问题
    PaperIDs     []string // 可选：限定搜索哪些论文
    SectionTypes []string // 可选：限定章节类型
    YearFrom     *int     // 可选：最早年份
    YearTo       *int     // 可选：最晚年份
    TopK         int      // 返回数量，默认 5
}
```

---

### 5.6 模块六：意图识别 + Prompt 拼装 + LLM 调用 ❌ 待实现

**意图识别**：
```go
type QueryMode string

const (
    ModeExtract QueryMode = "extract"  // 单论文知识提取
    ModeCompare QueryMode = "compare"  // 跨论文对比
)

// 简单规则识别
func DetectMode(question string) QueryMode {
    compareKeywords := []string{
        "对比", "区别", "差异", "比较", "vs",
        "compare", "difference", "versus",
    }
    for _, kw := range compareKeywords {
        if strings.Contains(strings.ToLower(question), kw) {
            return ModeCompare
        }
    }
    return ModeExtract
}
```

**Prompt 模板——知识提取模式**：
```go
const ExtractSystemPrompt = `你是一个论文阅读助手。请严格根据以下论文片段回答用户的问题。
规则：
1. 只基于提供的论文内容回答，不要引入外部知识。
2. 如果提供的内容无法回答问题，明确告知"在现有论文库中未找到相关信息"。
3. 回答时标注引用来源，格式为 [来源1]、[来源2]。`

const ExtractUserTemplate = `以下是从论文库中检索到的相关片段：
{{range $i, $ref := .References}}
[来源{{add $i 1}}] 论文：《{{$ref.PaperTitle}}》({{$ref.Authors}}, {{$ref.Year}})
章节：{{$ref.SectionTitle}}（第 {{$ref.Page}} 页）
内容：{{$ref.Content}}

{{end}}

用户问题：{{.Question}}

请基于以上论文内容回答：`
```

**Prompt 模板——跨论文对比模式**：
```go
const CompareSystemPrompt = `你是一个论文对比分析助手。请根据以下来自不同论文的片段进行对比分析。
规则：
1. 以结构化的方式呈现对比结果（可以使用表格）。
2. 明确指出各方法/论文的异同点。
3. 每个观点都要标注引用来源 [来源N]。`

const CompareUserTemplate = `以下是从多篇论文中检索到的相关片段：
{{range $i, $ref := .References}}
[来源{{add $i 1}}] 论文：《{{$ref.PaperTitle}}》({{$ref.Authors}}, {{$ref.Year}})
章节：{{$ref.SectionTitle}}
内容：{{$ref.Content}}

{{end}}

用户的对比问题：{{.Question}}

请对以上论文内容进行对比分析：`
```

**LLM 客户端接口**：
```go
type LLMClient interface {
    Chat(ctx context.Context, messages []ChatMessage) (*ChatResponse, error)
}

type ChatMessage struct {
    Role    string // system / user / assistant
    Content string
}

type ChatResponse struct {
    Content          string
    PromptTokens     int
    CompletionTokens int
}
```

**调用流程**：
```
1. 接收用户问题
2. 意图识别 → 确定 ModeExtract 或 ModeCompare
3. 根据模式选择检索参数（对比模式 TopK 设大）
4. 执行向量检索
5. 根据模式选择 Prompt 模板
6. 构建 messages 数组，调用 LLM API
7. 解析回答，存入 messages 表
8. 返回给前端
```

---

### 5.7 模块七：对话记忆 ❌ 待实现

**场景**：用户追问 "它和 BERT 有什么不同？" 时需要知道 "它" 指 Transformer。

**初版方案**：
```go
// 每次请求取最近 N 轮历史拼入 LLM messages
messages := []ChatMessage{
    {Role: "system", Content: systemPrompt},
    // 最近 3 轮历史
    {Role: "user",      Content: "Transformer 的 Attention 机制是什么？"},
    {Role: "assistant", Content: "根据《Attention Is All You Need》..."},
    // 本轮
    {Role: "user", Content: renderedPromptWithReferences},
}
```

**历史 token 预算**：预留 2K token，超了截断最早消息。

---

## 6. API 接口设计

|方法|路径|说明|状态|
|---|---|---|---|
|POST|`/api/v1/auth/register`|用户注册|✅|
|POST|`/api/v1/auth/login`|用户登录|✅|
|POST|`/api/v1/papers/upload`|上传论文|✅|
|GET|`/api/v1/papers`|论文列表|✅|
|GET|`/api/v1/papers/:id`|论文详情|✅|
|DELETE|`/api/v1/papers/:id`|删除论文|✅|
|POST|`/api/v1/chat`|RAG 问答|❌|
|GET|`/api/v1/conversations`|对话列表|❌|
|GET|`/api/v1/conversations/:id/messages`|消息历史|❌|

**Chat 接口响应示例**：
```json
{
    "conversation_id": "conv-uuid",
    "mode": "compare",
    "answer": "Transformer 和 BERT 的 Attention 存在以下区别...",
    "references": [
        {
            "paper_title": "Attention Is All You Need",
            "section_title": "3.2 Multi-Head Attention",
            "page_number": 5,
            "similarity": 0.94
        }
    ],
    "token_usage": {"prompt_tokens": 1500, "completion_tokens": 400}
}
```

---

## 7. 开发进度

|Step|内容|状态|
|---|---|---|
|1|项目初始化 + JWT 认证|✅|
|2|论文上传 + PDF/Markdown 解析 + 结构解析|✅|
|3|切片 + Embedding + 入库|✅|
|4|检索 + 问答|❌ 待开发|
|5|对话管理 + Redis 缓存|❌|
|6|中间件完善（限流、日志）|❌|
|7|前端开发|❌|
|8|容器化 + 部署到 PVE|❌|

---

## 8. 面试高频追问

**场景设计类**：
1. 切片策略和通用 RAG 的区别？→ 两级切片，先 Section 保持语义
2. Section 识别准确率？→ 标准 85-90%，不标准降级 other
3. 为什么不用 LLM 识别章节？→ 成本，规则零成本

**工程类**：
4. 上传论文处理流程？→ 提取 → 解析 → 切片 → Embedding，瓶颈在 API 网络 I/O
5. HNSW 索引原理？→ 分层可导航小世界图，O(log n) 近似搜索
6. pgvector vs Milvus？→ 科研场景量小，pgvector 够用且运维简单

---

_文档版本：v2.1 | 最后更新：2026-04-06_