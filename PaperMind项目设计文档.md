# PaperMind — 面向科研场景的论文知识检索与问答系统

## 项目设计文档（Project Design Document）

---

## 1. 项目概述

### 1.1 项目定位

PaperMind 是一个面向科研工作者的论文知识检索与问答系统，使用 Go 语言构建后端。研究人员可以将自己研究方向的论文批量上传，系统自动解析论文结构、构建向量化知识库；随后可以通过自然语言提问，实现**单论文知识提取**和**跨论文方法对比**两种核心能力，所有回答都会标注引用来源（精确到论文名称、章节与页码）。

### 1.2 为什么做这个项目

> **面试话术（核心，务必自然地讲出来）**：
> 
> "我在硕士开题调研阶段需要精读几十篇论文，经常遇到两个痛点：第一，读完一篇论文过几天就忘了具体细节，需要反复翻找；第二，要对比多篇论文的方法差异时，需要同时打开好几个 PDF 来回跳转，效率很低。市面上的通用 RAG 工具（比如直接用 ChatGPT 上传 PDF）存在三个问题——上下文窗口有限不能一次性处理大量论文、没有持久化的知识库、也不能做结构化的跨论文对比。所以我用 Go 做了这个系统来解决自己的实际需求。"

**技术层面的动机**（面试官追问"为什么用 RAG"时回答）：

- **token 限制**：直接把 30 篇论文全文喂给大模型，token 成本极高且超出上下文窗口。RAG 通过"先检索相关片段、再生成回答"大幅降低 token 消耗。
- **幻觉控制**：大模型直接回答科研问题容易编造不存在的实验数据。RAG 要求回答必须基于检索到的原文，配合 Prompt 约束，能有效降低幻觉。
- **知识持久化**：论文上传一次就永久入库，后续随时检索，不需要每次对话都重新上传。

### 1.3 核心数据流

```
┌──────────┐     ┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  上传论文  │────▶│  PDF 文本提取 │────▶│  论文结构解析  │────▶│  按 Section   │
│  (PDF)    │     │  + 元数据提取 │     │ (章节识别)    │     │  智能切片     │
└──────────┘     └──────────────┘     └──────────────┘     └──────┬───────┘
                                                                  │
                                                           ┌──────▼───────┐
                                                           │  Embedding   │
                                                           │  向量化 & 入库│
                                                           └──────────────┘

┌──────────┐     ┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  用户提问  │────▶│  意图识别     │────▶│  向量检索     │────▶│  Prompt 拼装  │
│          │     │ (单文提取/    │     │  Top-K       │     │  + LLM 调用   │
│          │     │  跨文对比)    │     └──────────────┘     └──────┬───────┘
└──────────┘     └──────────────┘                                 │
                                                           ┌──────▼───────┐
                                                           │  返回回答     │
                                                           │ (附论文来源、 │
                                                           │  章节、页码)  │
                                                           └──────────────┘
```

### 1.4 与通用 RAG 工具的区分点（面试核心卖点）

|维度|通用 RAG 工具|PaperMind|
|---|---|---|
|切片策略|固定 token 数盲切|感知论文结构，按 Section 切片，保持语义完整|
|元数据|仅文件名|论文标题、作者、年份、Section 名称、页码|
|问答模式|单一模式|知识提取模式 + 跨论文对比模式（不同 Prompt 模板）|
|引用溯源|"来自 chunk_037"|"来自《Attention Is All You Need》Method 章节，第 5 页"|
|目标用户|所有人|科研工作者，解决论文调研中的具体痛点|

---

## 2. 技术选型详解

### 2.1 总览

|层级|技术|版本建议|选型理由|
|---|---|---|---|
|**语言**|Go|1.22+|目标岗位要求；并发模型天然适合批量论文处理这种 I/O 密集场景|
|**Web 框架**|Gin|v1.9+|国内 Go 后端岗最主流框架，面试官熟悉，生态成熟|
|**ORM**|GORM|v2|管理业务元数据（用户、论文、对话），自动迁移方便快速迭代|
|**关系型数据库**|PostgreSQL|15+|配合 pgvector 扩展，一个数据库同时承担业务数据和向量存储|
|**向量检索**|pgvector|0.7+|轻量方案，避免引入独立向量数据库增加运维复杂度|
|**缓存**|Redis|7+|缓存热门问答、接口限流、论文处理进度状态|
|**Embedding 模型**|通义千问 text-embedding-v3|-|国产模型成本低（约 0.0007 元/千 token），中文效果好|
|**LLM**|DeepSeek Chat / 通义千问|-|性价比高，国内访问稳定，面试加分|
|**容器化**|Docker + docker-compose|-|一键启动全部服务|
|**反向代理**|Nginx Proxy Manager|-|已有经验，直接复用|
|**前端**|React + React Router + Zustand|

### 2.2 为什么选 PostgreSQL + pgvector 而不是 Milvus

这是面试高频追问。准备好以下回答：

- **pgvector 优势**：一套数据库同时存论文元数据和向量，运维简单；一个科研团队的论文库规模通常在几百到几千篇（对应几万到几十万 chunk），pgvector 的 HNSW 索引完全够用；支持事务，论文元数据和向量数据的一致性有保障。
- **Milvus 适用场景**：当向量数据量达到千万级以上、需要分布式部署时才需要。对于这个项目的科研场景，远不到这个量级。
- **面试话术**："在设计上我做了 Repository 层的接口抽象，向量检索逻辑封装在 `VectorRepo` 接口后面，如果后续数据量增长，可以无缝切换到 Milvus 而不影响上层业务代码。"

### 2.3 为什么选国产大模型 API

- 成本：DeepSeek / 通义千问的 API 价格远低于 OpenAI，个人科研项目预算有限。
- 网络：国内直连，无需代理，部署更稳定。
- 面试加分：说明你对国内大模型生态有了解。
- 降级方案：代码中设计统一的 `LLMClient` 接口，可以随时切换到 OpenAI 或本地部署的开源模型。

---

## 3. 项目目录结构

```
papermind/
├── cmd/
│   └── server/
│       └── main.go                 # 程序入口
├── internal/
│   ├── config/
│   │   └── config.go               # 配置加载（Viper）
│   ├── handler/                     # HTTP Handler 层（Gin）
│   │   ├── paper.go                # 论文上传 / 列表 / 删除
│   │   ├── chat.go                 # 问答对话接口
│   │   └── health.go               # 健康检查
│   ├── service/                     # 业务逻辑层
│   │   ├── paper_service.go        # 论文处理：上传 → 提取 → 结构解析 → 切片 → 向量化
│   │   ├── retrieval_service.go    # 检索逻辑：问题向量化 → Top-K 检索 → 可选过滤
│   │   ├── chat_service.go         # 对话逻辑：意图识别 → Prompt 选择 → LLM 调用
│   │   └── embedding_service.go    # Embedding API 封装
│   ├── repository/                  # 数据访问层（GORM）
│   │   ├── paper_repo.go           # 论文 & chunk 的 CRUD
│   │   ├── conversation_repo.go    # 对话记录的 CRUD
│   │   └── vector_repo.go          # 向量检索（pgvector 相关 SQL）
│   ├── model/                       # 数据模型定义
│   │   ├── paper.go                # 论文模型
│   │   ├── chunk.go                # 切片模型（含 section 元数据）
│   │   ├── conversation.go
│   │   └── message.go
│   ├── middleware/
│   │   ├── cors.go
│   │   ├── ratelimit.go
│   │   └── auth.go
│   └── pkg/                         # 内部工具包
│       ├── chunker/
│       │   ├── chunker.go          # 通用切片接口
│       │   └── paper_chunker.go    # 论文结构感知切片（核心差异化）
│       ├── extractor/
│       │   └── pdf.go              # PDF 文本 + 元数据提取
│       ├── parser/
│       │   └── section_parser.go   # 论文章节结构识别
│       ├── llm/
│       │   ├── client.go           # LLM 接口定义
│       │   ├── deepseek.go
│       │   └── qwen.go
│       └── prompt/
│           └── templates.go        # 多场景 Prompt 模板（提取/对比）
├── web/                             # 前端代码（React）
│   ├── src/
│   │   ├── pages/
│   │   │   ├── ChatPage.tsx        # 问答页面
│   │   │   ├── PaperListPage.tsx   # 论文库管理页
│   │   │   └── PaperDetailPage.tsx # 单篇论文详情（章节列表 + chunk 预览）
│   │   ├── components/
│   │   │   ├── MessageBubble.tsx   # 消息气泡（含引用来源折叠面板）
│   │   │   ├── PaperUploader.tsx   # 论文上传组件
│   │   │   └── ReferenceCard.tsx   # 引用来源卡片（论文名 + 章节 + 页码）
│   │   ├── router/
│   │   │   └── index.tsx           # React Router 路由配置
│   │   ├── store/
│   │   │   └── useChatStore.ts     # Zustand 状态管理
│   │   ├── App.tsx
│   │   └── main.tsx
│   └── package.json
├── deployments/
│   ├── Dockerfile
│   ├── Dockerfile.frontend
│   └── docker-compose.yml
├── configs/
│   └── config.yaml
├── scripts/
│   └── init.sql                     # 数据库初始化
├── go.mod
├── go.sum
└── README.md
```

> **与通用 RAG 项目的目录差异**：多了 `parser/section_parser.go`（论文结构识别）和 `chunker/paper_chunker.go`（结构感知切片），Prompt 模板也按场景拆分。面试时可以指着目录结构讲"我做了哪些场景定制"。

---

## 4. 数据库设计

### 4.1 初始化脚本（PostgreSQL）

```sql
-- 启用 pgvector 扩展
CREATE EXTENSION IF NOT EXISTS vector;

-- 启用 uuid 生成
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
```

### 4.2 表结构

#### users 表

```sql
CREATE TABLE users (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username    VARCHAR(64)  NOT NULL UNIQUE,
    email       VARCHAR(128) NOT NULL UNIQUE,
    password    VARCHAR(256) NOT NULL,          -- bcrypt 哈希
    created_at  TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP    NOT NULL DEFAULT NOW()
);
```

#### papers 表（核心变化：论文专属元数据）

```sql
CREATE TABLE papers (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id      UUID         NOT NULL REFERENCES users(id),

    -- 文件信息
    filename     VARCHAR(256) NOT NULL,          -- 原始文件名
    file_size    BIGINT       NOT NULL,          -- 字节数
    file_hash    VARCHAR(64)  NOT NULL,          -- SHA-256，用于去重

    -- 论文元数据（区分度的关键）
    title        VARCHAR(512),                   -- 论文标题（从 PDF 提取或用户手动填写）
    authors      VARCHAR(512),                   -- 作者列表，逗号分隔
    year         INT,                            -- 发表年份
    venue        VARCHAR(256),                   -- 发表会议/期刊（如 NeurIPS 2024）
    abstract     TEXT,                           -- 摘要（从论文提取）

    -- 处理状态
    chunk_count  INT          NOT NULL DEFAULT 0,
    status       VARCHAR(32)  NOT NULL DEFAULT 'pending',
    -- status 枚举: pending → extracting → chunking → embedding → completed → failed
    -- （比通用项目更细的状态机，方便前端展示处理进度）

    created_at   TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMP    NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_papers_user_id ON papers(user_id);
CREATE INDEX idx_papers_file_hash ON papers(file_hash);
CREATE INDEX idx_papers_year ON papers(year);           -- 支持按年份筛选检索范围
```

> **面试话术**："和通用的文档上传不同，我为论文场景设计了专门的元数据字段——标题、作者、年份、会议。这些字段一方面用于前端展示，另一方面可以在检索时做过滤条件，比如用户可以限定只从 2023 年以后的论文中检索。"

#### chunks 表（核心变化：Section 级元数据）

```sql
CREATE TABLE chunks (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    paper_id     UUID          NOT NULL REFERENCES papers(id) ON DELETE CASCADE,
    chunk_index  INT           NOT NULL,          -- 在论文中的全局顺序

    content      TEXT          NOT NULL,           -- 原始文本内容
    token_count  INT           NOT NULL,           -- token 数量
    embedding    vector(1536)  NOT NULL,           -- 向量

    -- 论文结构元数据（核心差异化字段）
    section_type VARCHAR(64),                      -- abstract / introduction / related_work /
                                                   -- method / experiment / conclusion / other
    section_title VARCHAR(256),                    -- 原始章节标题（如 "3.2 Multi-Head Attention"）
    page_number  INT,                              -- 所在页码

    created_at   TIMESTAMP     NOT NULL DEFAULT NOW()
);

-- 向量检索索引
CREATE INDEX idx_chunks_embedding ON chunks
    USING hnsw (embedding vector_cosine_ops);

CREATE INDEX idx_chunks_paper_id ON chunks(paper_id);
CREATE INDEX idx_chunks_section_type ON chunks(section_type);  -- 支持按章节类型过滤
```

> **面试考点**：为什么 `section_type` 要建索引？因为用户问"这篇论文的实验结果是什么"时，可以先用 `section_type = 'experiment'` 过滤再做向量检索，缩小搜索范围、提高检索精度和速度。这是一种"结构化过滤 + 向量检索"的混合检索策略。

#### conversations 表

```sql
CREATE TABLE conversations (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID         NOT NULL REFERENCES users(id),
    title       VARCHAR(256),                     -- 对话标题（可由 LLM 自动生成）
    mode        VARCHAR(32)  NOT NULL DEFAULT 'extract',
    -- mode 枚举: extract（单论文提取）/ compare（跨论文对比）
    created_at  TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP    NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_conversations_user_id ON conversations(user_id);
```

#### messages 表

```sql
CREATE TABLE messages (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    conversation_id UUID        NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    role            VARCHAR(16) NOT NULL,          -- user / assistant
    content         TEXT        NOT NULL,

    -- 引用来源（带论文级信息，而不是简单的 chunk_id）
    references      JSONB,
    -- 示例值:
    -- [
    --   {
    --     "chunk_id": "xxx",
    --     "paper_title": "Attention Is All You Need",
    --     "authors": "Vaswani et al.",
    --     "section": "3.2 Multi-Head Attention",
    --     "page": 5,
    --     "similarity": 0.94,
    --     "content_preview": "Multi-head attention allows the model to jointly attend to..."
    --   }
    -- ]

    token_usage     JSONB,                         -- {"prompt_tokens": 500, "completion_tokens": 200}
    created_at      TIMESTAMP   NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_messages_conversation_id ON messages(conversation_id);
```

### 4.3 ER 关系图

```
users 1──N papers 1──N chunks
  │
  └── 1──N conversations 1──N messages
```

---

## 5. 核心功能模块详解

### 5.1 模块一：论文上传与文本提取

**接口**：`POST /api/v1/papers/upload`

**流程**：

```
1. 用户通过 multipart/form-data 上传 PDF 文件
2. 校验文件类型（仅允许 pdf）和大小（限制 50MB，论文 PDF 通常较大）
3. 计算文件 SHA-256 哈希，检查是否重复上传
4. 在 papers 表创建记录，状态设为 pending
5. 异步触发处理流水线：提取 → 解析结构 → 切片 → 向量化
```

**文本提取**：

|步骤|说明|Go 库推荐|
|---|---|---|
|PDF 文本提取|从 PDF 中提取每页的文本内容|`github.com/ledongthuc/pdf` 或 `github.com/pdfcpu/pdfcpu`|
|元数据提取|尝试从 PDF metadata 中提取 title / author|`github.com/pdfcpu/pdfcpu` 的 Info API|
|页码追踪|记录每段文本来自哪一页|提取时按页遍历，维护 page_number 映射|

**论文元数据的提取策略**：

```
优先级：
1. PDF 内嵌 metadata（很多论文 PDF 会包含 title 和 author）
2. 正则匹配第一页文本的标题位置（通常是正文第一行，字号最大）
3. 用户手动填写（前端提供编辑入口）

面试话术："自动提取不可能 100% 准确，所以我设计了三级降级策略，
           最终兜底是让用户手动修正，优先保证数据质量。"
```

> **注意**：PDF 提取是 RAG 项目的常见痛点。简单 PDF（纯文字论文）用 Go 库即可；扫描件 PDF 需要 OCR，先不做。面试时可以说"当前版本支持可复制文本的论文 PDF，扫描件 OCR 作为后续迭代方向"。

### 5.2 模块二：论文结构解析（场景差异化核心）

**这是 PaperMind 与通用 RAG 工具最大的区别，面试必须能讲清楚。**

**目标**：将提取出的论文纯文本，识别出其 Section 结构（Abstract、Introduction、Method 等）。

**实现方式：基于规则的章节识别器** (`parser/section_parser.go`)

```go
// 论文章节的标准类型
var SectionTypes = map[string][]string{
    "abstract":     {"abstract"},
    "introduction": {"introduction", "intro"},
    "related_work": {"related work", "background", "literature review", "prior work"},
    "method":       {"method", "methodology", "approach", "proposed method", "model", "framework", "architecture"},
    "experiment":   {"experiment", "evaluation", "results", "empirical"},
    "discussion":   {"discussion", "analysis"},
    "conclusion":   {"conclusion", "summary", "future work"},
}

// 识别逻辑：
// 1. 逐行扫描文本
// 2. 匹配形如 "1. Introduction" 或 "3 Methodology" 或 "## Method" 的行
// 3. 将匹配到的标题与 SectionTypes 做模糊匹配，确定 section_type
// 4. 两个标题之间的文本归属于前一个 section
```

**实现伪代码**：

```go
type Section struct {
    Type      string   // abstract / method / experiment / ...
    Title     string   // 原始标题文本（如 "3.2 Multi-Head Attention"）
    Content   string   // 该 section 的全部文本
    StartPage int      // 起始页码
}

func ParseSections(pages []PageText) []Section {
    var sections []Section
    // 正则匹配常见的论文章节标题格式
    // 例如: "1. Introduction", "3 Methodology", "IV. EXPERIMENTS"
    headingPattern := regexp.MustCompile(
        `^(?:\d+\.?\s+|[IVX]+\.?\s+)?` +  // 可选的编号
        `([A-Z][A-Za-z\s:]+)$`,             // 首字母大写的标题
    )
    
    for _, page := range pages {
        for _, line := range strings.Split(page.Text, "\n") {
            trimmed := strings.TrimSpace(line)
            if headingPattern.MatchString(trimmed) {
                sectionType := matchSectionType(trimmed)
                sections = append(sections, Section{
                    Type:      sectionType,
                    Title:     trimmed,
                    StartPage: page.Number,
                })
            }
        }
    }
    // 填充每个 section 的 content（两个标题之间的文本）
    fillSectionContent(sections, pages)
    return sections
}
```

> **面试追问准备**：
> 
> Q："这个规则匹配准确率能到多少？" A："对于标准的 ACL/NeurIPS/CVPR 等会议论文，标题格式比较规范，准确率可以做到 85-90%。对于格式不标准的论文，会降级到 `other` 类型，不影响功能，只是丢失了 section 级的过滤能力。后续可以引入一个轻量级的标题分类模型来提升准确率。"
> 
> Q："为什么不直接用 LLM 来识别章节？" A："出于成本考虑。每篇论文可能有上万 token，用 LLM 做结构识别意味着每篇论文入库都要额外消耗一次 LLM 调用。规则匹配是零成本的，优先用规则，解决不了的再考虑 LLM 兜底。"

### 5.3 模块三：论文结构感知切片（Chunking）

**切片策略**（与通用 RAG 的核心区别）：

```
两级切片：
第一级：按 Section 切分（由 5.2 的结构解析器完成）
第二级：对于过长的 Section，再按固定 token 数 + overlap 细分

参数：
- max_chunk_size:  512 tokens（单个 chunk 的上限）
- chunk_overlap:    64 tokens（相邻 chunk 的重叠部分）
- min_chunk_size:   50 tokens（过短的 chunk 合并到上一个）
```

**实现伪代码**：

```go
func ChunkPaper(sections []Section) []Chunk {
    var chunks []Chunk
    
    for _, section := range sections {
        tokens := Tokenize(section.Content)
        
        if len(tokens) <= maxChunkSize {
            // Section 足够短，整体作为一个 chunk
            chunks = append(chunks, Chunk{
                Content:      section.Content,
                SectionType:  section.Type,
                SectionTitle: section.Title,
                PageNumber:   section.StartPage,
            })
        } else {
            // Section 太长，按 token 数细分，但保留 section 元数据
            subChunks := splitByTokens(tokens, maxChunkSize, chunkOverlap)
            for _, sc := range subChunks {
                chunks = append(chunks, Chunk{
                    Content:      sc,
                    SectionType:  section.Type,
                    SectionTitle: section.Title,
                    PageNumber:   section.StartPage, // 可进一步精确到子页
                })
            }
        }
    }
    
    // 过滤过短的 chunk（如只有标题没有内容的 section）
    return filterMinSize(chunks, minChunkSize)
}
```

**为什么这样切比通用的"固定 token 盲切"好**：

1. 不会把一段方法描述切成两半，section 内的语义完整性有保障。
2. 每个 chunk 携带 `section_type` 和 `section_title`，检索时可以做精确过滤。
3. Abstract 通常很短（200-300 token），不会被切碎；Experiment 通常很长，才需要细分。

**面试加分项**（初版不实现，但能讲）：

- 对每个 chunk 生成一句话摘要（"Contextual Chunking"），拼在 chunk 前面再做 Embedding，提升检索质量。
- 对论文中的表格单独提取为结构化数据，用 Markdown 表格格式存储。

### 5.4 模块四：Embedding 向量化

**接口封装**：

```go
type EmbeddingClient interface {
    Embed(ctx context.Context, texts []string) ([][]float32, error)
}
```

**对接通义千问 Embedding API**：

```
POST https://dashscope.aliyuncs.com/api/v1/services/embeddings/text-embedding/text-embedding

Request Body:
{
    "model": "text-embedding-v3",
    "input": {
        "texts": ["chunk 文本 1", "chunk 文本 2", ...]
    },
    "parameters": {
        "dimension": 1536
    }
}
```

**关键工程细节**：

- **批量请求**：API 通常支持一次最多 25 条文本。一篇 20 页论文大约产生 40-80 个 chunk，需要分 2-4 批。
- **并发处理**：用 `errgroup.Group` + `g.SetLimit(4)` 并发发送多个批次，提高入库速度。
- **限流保护**：用 `golang.org/x/time/rate` 做令牌桶限流，防止触发 API 频率限制。
- **维度对齐**：chunks 表的 `vector(1536)` 必须和模型输出维度一致。

```go
g, ctx := errgroup.WithContext(ctx)
g.SetLimit(4) // 最多 4 个并发 Embedding 请求

for _, batch := range chunkBatches {
    batch := batch
    g.Go(func() error {
        embeddings, err := embeddingClient.Embed(ctx, batch.Texts)
        if err != nil {
            return fmt.Errorf("embedding batch %d: %w", batch.Index, err)
        }
        batch.Embeddings = embeddings
        return nil
    })
}

if err := g.Wait(); err != nil {
    // 更新论文状态为 failed，记录错误
}
```

### 5.5 模块五：检索（结构化过滤 + 向量检索）

**这是另一个场景差异化的关键点。**

**普通向量检索**（通用 RAG 的做法）：

```sql
SELECT * FROM chunks ORDER BY embedding <=> $1 LIMIT 5;
```

**PaperMind 的增强检索**（结构化过滤 + 向量检索）：

```sql
-- 场景1：用户问"这篇论文的实验结果是什么"
-- 先按 paper_id + section_type 过滤，再向量检索
SELECT id, paper_id, content, section_title, page_number,
       1 - (embedding <=> $1) AS similarity
FROM chunks
WHERE paper_id = $2
  AND section_type IN ('experiment', 'discussion')
ORDER BY embedding <=> $1
LIMIT 5;

-- 场景2：跨论文对比，从所有论文的 method 章节中检索
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

**Go 代码中的检索参数设计**：

```go
type RetrievalRequest struct {
    Query        string   // 用户问题
    PaperIDs     []string // 可选：限定搜索哪些论文
    SectionTypes []string // 可选：限定搜索哪些章节类型
    YearFrom     *int     // 可选：最早年份
    YearTo       *int     // 可选：最晚年份
    TopK         int      // 返回数量，默认 5
}
```

> **面试话术**："纯向量检索的一个问题是，用户问'这篇论文的实验结果'时，可能检索出 Introduction 里的某段文字，因为它恰好提到了相关关键词。我通过结构化过滤先缩小范围到 Experiment 章节，再做向量检索，实测下来检索精度提升明显。"

### 5.6 模块六：意图识别 + Prompt 拼装 + LLM 调用

**意图识别**（两种问答模式的路由）：

```go
type QueryMode string

const (
    ModeExtract QueryMode = "extract"  // 单论文知识提取
    ModeCompare QueryMode = "compare"  // 跨论文对比
)

// 简单规则识别（初版足够，不需要训练模型）
func DetectMode(question string) QueryMode {
    compareKeywords := []string{
        "对比", "区别", "差异", "不同", "比较", "哪个更好",
        "compare", "difference", "versus", "vs",
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
3. 回答时标注引用来源，格式为 [来源1]、[来源2]。
4. 使用学术化但易懂的语言。`

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
const CompareSystemPrompt = `你是一个论文对比分析助手。请根据以下来自不同论文的片段，对用户提出的对比问题进行分析。
规则：
1. 以结构化的方式呈现对比结果（可以使用表格）。
2. 明确指出各方法/论文的异同点。
3. 每个观点都要标注引用来源 [来源N]。
4. 如果某方面信息不足以对比，如实说明。`

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
3. 根据模式选择检索参数（对比模式 TopK 设大一些，如 10）
4. 执行向量检索
5. 根据模式选择对应的 Prompt 模板，渲染填充
6. 构建 messages 数组，调用 LLM API
7. 解析回答，连同引用信息存入 messages 表
8. 返回给前端
```

**关键细节**：

- **token 预算控制**：检索结果加上 Prompt 不能超出模型上下文窗口。假设用 DeepSeek（32K），给回答留 2K，检索内容控制在 4K 以内。对比模式需要更多 chunk（来自不同论文），预算要适当放宽。
- **流式输出（加分项）**：用 Gin 的 `c.Stream()` + SSE 实现流式返回，面试要能讲原理，初版可不实现。
- **超时控制**：设 `context.WithTimeout(ctx, 30*time.Second)`。

### 5.7 模块七：对话记忆（多轮对话）

**场景示例**：

```
用户："Transformer 论文的 Attention 机制是什么？"
AI：（基于检索回答）
用户："它和 BERT 的 Attention 有什么不同？"  ← 需要知道"它"指 Transformer
```

**实现方案**（初版简单方案）：

```go
// 每次请求时从 messages 表取最近 N 轮历史
// 将历史拼入 LLM 的 messages 数组
messages := []ChatMessage{
    {Role: "system", Content: systemPrompt},
    // 最近 3 轮历史对话
    {Role: "user",      Content: "Transformer 论文的 Attention 机制是什么？"},
    {Role: "assistant",  Content: "根据《Attention Is All You Need》..."},
    // 本轮：检索结果 + 新问题
    {Role: "user", Content: renderedPromptWithReferences},
}
```

**历史轮次的 token 预算**：固定预留 2K token 给历史消息。如果超了，截断最早的消息。

**进阶方案（面试可讲）**：用 LLM 对历史对话生成一段摘要（如"用户之前在了解 Transformer 的 Attention 机制"），注入 Prompt 开头，节省 token。

---

## 6. API 接口设计

### 6.1 完整接口列表

|方法|路径|说明|认证|
|---|---|---|---|
|POST|`/api/v1/auth/register`|用户注册|否|
|POST|`/api/v1/auth/login`|用户登录，返回 JWT|否|
|POST|`/api/v1/papers/upload`|上传论文 PDF|是|
|GET|`/api/v1/papers`|获取论文列表（支持按年份、关键词筛选）|是|
|GET|`/api/v1/papers/:id`|获取论文详情（元数据 + 章节列表 + 处理状态）|是|
|PUT|`/api/v1/papers/:id`|更新论文元数据（手动修正标题/作者等）|是|
|DELETE|`/api/v1/papers/:id`|删除论文及其所有 chunk|是|
|POST|`/api/v1/chat`|发送问题，获取 RAG 回答|是|
|GET|`/api/v1/conversations`|获取对话列表|是|
|GET|`/api/v1/conversations/:id/messages`|获取某对话的消息历史|是|
|DELETE|`/api/v1/conversations/:id`|删除对话|是|
|GET|`/api/v1/health`|健康检查|否|

### 6.2 核心接口请求/响应示例

#### 上传论文

```
POST /api/v1/papers/upload
Content-Type: multipart/form-data
Authorization: Bearer <jwt_token>

Form Fields:
  file: (binary)
  title: "Attention Is All You Need"    // 可选，不填则自动提取
  authors: "Vaswani et al."             // 可选
  year: 2017                            // 可选
  venue: "NeurIPS 2017"                 // 可选

Response 200:
{
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "filename": "attention-is-all-you-need.pdf",
    "title": "Attention Is All You Need",
    "authors": "Vaswani et al.",
    "year": 2017,
    "status": "extracting",
    "created_at": "2026-04-05T10:30:00Z"
}
```

#### 发送问题

```
POST /api/v1/chat
Content-Type: application/json
Authorization: Bearer <jwt_token>

Request Body:
{
    "conversation_id": null,           // null 表示新建对话
    "question": "Transformer 和 BERT 的 Attention 机制有什么不同？",
    "paper_ids": ["paper-1", "paper-2"],  // 可选：限定论文范围
    "section_types": ["method"],          // 可选：限定章节范围
    "year_from": 2017                     // 可选：最早年份
}

Response 200:
{
    "conversation_id": "conv-uuid-xxx",
    "mode": "compare",
    "answer": "Transformer 和 BERT 的 Attention 机制存在以下核心区别：\n\n| 维度 | Transformer | BERT |\n|...|...|...|\n\n[来源1][来源2]...",
    "references": [
        {
            "chunk_id": "chunk-uuid-1",
            "paper_title": "Attention Is All You Need",
            "authors": "Vaswani et al.",
            "year": 2017,
            "section_title": "3.2 Multi-Head Attention",
            "page_number": 5,
            "similarity": 0.94,
            "content_preview": "Multi-head attention allows the model to jointly attend to..."
        },
        {
            "chunk_id": "chunk-uuid-2",
            "paper_title": "BERT: Pre-training of Deep Bidirectional Transformers",
            "authors": "Devlin et al.",
            "year": 2019,
            "section_title": "3.1 Model Architecture",
            "page_number": 4,
            "similarity": 0.91,
            "content_preview": "BERT uses a multi-layer bidirectional Transformer encoder..."
        }
    ],
    "token_usage": {
        "prompt_tokens": 1500,
        "completion_tokens": 400
    }
}
```

---

## 7. 缓存与性能优化

### 7.1 Redis 缓存策略

|缓存内容|Key 设计|TTL|说明|
|---|---|---|---|
|热门问答|`cache:qa:{sha256(question+paper_ids)}`|30 分钟|相同问题 + 相同论文范围直接返回缓存|
|论文处理进度|`paper:status:{paper_id}`|10 分钟|前端轮询处理进度时减轻数据库压力|
|用户会话|`session:{user_id}`|24 小时|JWT 补充验证|

### 7.2 接口限流

用 Redis `ZSET` + 滑动窗口实现：

```
规则：
- 未登录接口：10 次/分钟（按 IP）
- 已登录普通接口：60 次/分钟（按 user_id）
- Chat 接口：10 次/分钟（按 user_id），LLM API 消耗较贵需限流
- Upload 接口：5 次/分钟（按 user_id），防止频繁上传占用资源
```

### 7.3 Goroutine 并发优化

论文上传后的处理流水线：

```
论文处理流水线（异步 Goroutine）：
1. [extracting]  文本提取 + 元数据提取
2. [chunking]    结构解析 + 智能切片
3. [embedding]   批量向量化（并发）
   ├── Goroutine 1: Embed chunk[0:25]
   ├── Goroutine 2: Embed chunk[25:50]
   └── Goroutine 3: Embed chunk[50:75]
4. [completed]   写入数据库，更新状态

每步更新 papers.status，前端可轮询展示进度条。
```

---

## 8. 鉴权设计

### 8.1 JWT 方案

```
登录流程：
1. 用户提交 username + password
2. 后端校验密码（bcrypt 比对）
3. 生成 JWT（有效期 24 小时），payload 包含 user_id
4. 返回 token 给前端

请求鉴权：
1. 前端在 Header 中带上 Authorization: Bearer <token>
2. Gin 中间件解析并验证 JWT
3. 将 user_id 写入 Gin Context，供后续 Handler 使用
```

Go 库推荐：`github.com/golang-jwt/jwt/v5`

---

## 9. 前端设计（简易版）

### 9.1 页面规划

三个页面：

**页面一：论文库管理页**

- 论文上传区域（拖拽 PDF）
- 已上传论文列表，每条显示：标题、作者、年份、会议、切片数量、处理状态
- 处理中的论文显示进度条（extracting → chunking → embedding → completed）
- 点击论文可进入详情页
- 支持编辑元数据（修正标题/作者）、删除

**页面二：论文详情页**

- 顶部：论文元数据（标题、作者、年份、会议、摘要）
- 下方：章节列表，每个 section 展示类型标签 + 标题 + chunk 数量
- 点击某个 chunk 可展开查看原文片段

**页面三：问答对话页**

- 左侧：对话列表（带模式标签：提取 / 对比）
- 右侧上方：可选的检索范围设置（选择论文、章节类型、年份范围）
- 右侧中部：聊天界面，AI 回答下方展示引用来源卡片：
    - 论文标题 + 作者 + 年份
    - 章节名称 + 页码
    - 相似度分数
    - 点击展开查看原文片段
- 右侧底部：输入框 + 发送按钮

### 9.2 技术方案

- React + React Router
- UI 样式：Tailwind CSS
- HTTP 请求：axios
- 状态管理：Zustand

---

## 10. 部署方案

### 10.1 docker-compose.yml

```yaml
version: '3.8'

services:
  postgres:
    image: pgvector/pgvector:pg16
    environment:
      POSTGRES_DB: papermind
      POSTGRES_USER: papermind
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - pgdata:/var/lib/postgresql/data
      - ./scripts/init.sql:/docker-entrypoint-initdb.d/init.sql
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U papermind"]
      interval: 5s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    command: redis-server --requirepass ${REDIS_PASSWORD}
    volumes:
      - redisdata:/data
    ports:
      - "6379:6379"

  backend:
    build:
      context: .
      dockerfile: deployments/Dockerfile
    environment:
      - DB_HOST=postgres
      - DB_PORT=5432
      - DB_USER=papermind
      - DB_PASSWORD=${DB_PASSWORD}
      - DB_NAME=papermind
      - REDIS_ADDR=redis:6379
      - REDIS_PASSWORD=${REDIS_PASSWORD}
      - LLM_API_KEY=${LLM_API_KEY}
      - EMBEDDING_API_KEY=${EMBEDDING_API_KEY}
      - JWT_SECRET=${JWT_SECRET}
    ports:
      - "8080:8080"
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_started

  frontend:
    image: nginx:alpine
    volumes:
      - ./web/dist:/usr/share/nginx/html
      - ./deployments/nginx.conf:/etc/nginx/conf.d/default.conf
    ports:
      - "3000:80"
    depends_on:
      - backend

volumes:
  pgdata:
  redisdata:
```

### 10.2 后端 Dockerfile

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o papermind ./cmd/server

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/papermind .
COPY --from=builder /app/configs ./configs
EXPOSE 8080
CMD ["./papermind"]
```

### 10.3 部署到 PVE + 公网访问

```
1. 在 PVE 中创建 LXC 容器或 VM，安装 Docker
2. 将项目代码推到 GitHub，在容器内 git clone
3. 配置 .env 文件（API Key 等敏感信息）
4. docker-compose up -d
5. 在 Nginx Proxy Manager 中配置：
   - api.wolfden.website → 反向代理到 backend:8080
   - papermind.wolfden.website → 反向代理到 frontend:3000
   - 开启 SSL（Let's Encrypt 自动签发）
6. 测试公网访问
```

---

## 11. Go 依赖包清单

```
核心依赖：
github.com/gin-gonic/gin              # Web 框架
gorm.io/gorm                          # ORM
gorm.io/driver/postgres               # PostgreSQL 驱动
github.com/pgvector/pgvector-go       # pgvector 类型支持
github.com/redis/go-redis/v9          # Redis 客户端
github.com/golang-jwt/jwt/v5          # JWT
github.com/spf13/viper                # 配置管理
golang.org/x/sync/errgroup            # 并发控制
golang.org/x/time/rate                # 限流
golang.org/x/crypto/bcrypt            # 密码哈希
github.com/google/uuid                # UUID 生成

文本处理：
github.com/ledongthuc/pdf             # PDF 文本提取
github.com/pdfcpu/pdfcpu              # PDF 元数据提取
github.com/pkoukk/tiktoken-go         # token 计数

可选：
github.com/swaggo/gin-swagger         # Swagger 文档
go.uber.org/zap                       # 结构化日志
```

---

## 12. 开发顺序建议

```
Step 1: 项目初始化 & 基础 CRUD                     （Day 1）
  ├── 初始化 Go 项目，引入 Gin + GORM
  ├── 配置 PostgreSQL + pgvector
  ├── 实现 users 表 CRUD + JWT 注册登录
  └── 验证：能用 curl 注册、登录、拿到 token

Step 2: 论文上传 & PDF 解析                         （Day 2）
  ├── 实现论文上传接口
  ├── 实现 PDF 文本提取 + 元数据提取
  ├── 实现论文结构解析（Section 识别）
  └── 验证：上传论文 PDF，能在日志中看到识别出的章节列表

Step 3: 结构感知切片 & Embedding                    （Day 3）
  ├── 实现两级切片逻辑（按 Section → 按 token）
  ├── 对接 Embedding API，并发处理
  ├── 将 chunk + 向量 + 元数据写入 chunks 表
  └── 验证：上传论文后，chunks 表有数据，每条含 section_type 和 embedding

Step 4: 检索 & 问答                                 （Day 4）
  ├── 实现结构化过滤 + 向量检索
  ├── 实现意图识别（提取/对比模式）
  ├── 实现两套 Prompt 模板 + LLM 调用
  └── 验证：curl 提问，能拿到带论文引用来源的回答

Step 5: 对话管理 & 缓存                             （Day 5）
  ├── 实现 conversations + messages 的 CRUD
  ├── 实现多轮对话上下文拼装
  ├── 引入 Redis 缓存
  └── 验证：多轮追问能理解上下文

Step 6: 中间件完善                                  （Day 6）
  ├── CORS、限流、统一错误处理、结构化日志
  ├── 论文元数据编辑接口
  └── 验证：限流生效，日志格式规范

Step 7: 前端开发                                    （Day 7）
  ├── 论文库管理页（上传 + 列表 + 进度 + 元数据编辑）
  ├── 问答对话页（检索范围选择 + 聊天 + 引用来源卡片）
  ├── 前后端联调
  └── 验证：浏览器中完成"上传论文 → 提问 → 看到带引用的回答"

Step 8: 容器化 & 部署                               （Day 8）
  ├── Dockerfile + docker-compose.yml
  ├── 部署到 PVE + Nginx Proxy Manager + HTTPS
  └── 验证：papermind.wolfden.website 公网可用
```

---

## 13. 面试高频追问清单

### RAG 原理类

1. 什么是 RAG？为什么不直接把论文全部喂给大模型？
2. Embedding 是什么？为什么能用向量表示语义？
3. 余弦相似度和 L2 距离有什么区别？为什么用余弦？
4. Chunk 太大或太小分别会有什么问题？

### 场景设计类（PaperMind 专属，高区分度）

5. **你的切片策略和通用 RAG 有什么区别？为什么这样设计？** → 讲"两级切片"：先按 Section 保持语义完整，再按 token 细分。通用 RAG 盲切容易把一段方法描述切成两半。
6. **Section 识别的准确率怎么样？不准怎么办？** → 标准会议论文 85-90%，不标准的降级为 other，不影响功能。后续可用 LLM 兜底。
7. **跨论文对比模式的 Prompt 怎么设计的？和普通模式有什么不同？** → 对比模式 TopK 更大（从多篇论文取），Prompt 要求结构化对比输出（表格），引用标注更细。
8. **如果检索出的内容和问题不相关，怎么办？** → 设相似度阈值过滤低质量结果；进阶方案引入 Re-rank（先粗检索 Top-20，再用 Cross-Encoder 精排 Top-5）。
9. **为什么用规则做 Section 识别而不是 LLM？** → 成本。每篇论文上万 token，LLM 识别要额外消耗一次调用。规则是零成本，性价比高。

### 工程实现类

10. 上传一篇 50 页论文，你的处理流程是什么？耗时瓶颈在哪？ → 流水线讲清（提取 → 解析 → 切片 → Embedding），瓶颈在 Embedding API 网络 I/O，用 Goroutine 并发优化。
11. 如果两个用户同时上传论文，系统如何处理？ → 每个上传启动独立 Goroutine，数据库通过 user_id + paper_id 隔离，互不影响。
12. HNSW 索引的原理？和暴力搜索相比优劣？ → 分层可导航小世界图，O(log n) 近似搜索 vs O(n) 暴力扫描。牺牲极小精度换取数量级速度提升。
13. 为什么选 pgvector 不选 Milvus？ → 科研场景数据量小（几十万 chunk），pgvector 够用且运维简单。接口已抽象，可切换。

### 高并发 & 扩展类

14. 如果实验室所有人都用，系统瓶颈在哪？ → LLM API 调用延迟和成本。方案：Redis 缓存热门问答、接口限流、流式输出改善体验。
15. 如果要支持论文中的图表问答，你会怎么做？ → 多模态 Embedding。提取论文图片，用视觉 Embedding 模型（如 CLIP）向量化，和文本向量统一检索。
16. Agent 和 RAG 的关系？ → Agent 可以把 RAG 作为一个 Tool 调用。比如一个 Research Agent 可以先用 RAG Tool 检索论文，再用 Web Search Tool 查最新进展，最后综合回答。

---

_文档版本：v2.0（论文助手场景版）| 最后更新：2026-03-31_
