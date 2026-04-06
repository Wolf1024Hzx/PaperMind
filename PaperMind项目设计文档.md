# PaperMind — 面向科研场景的论文知识检索与问答系统

## 项目设计文档

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
上传论文 → PDF文本提取 → 结构解析 → 按Section切片 → Embedding向量化 → 入库

用户提问 → 意图识别 → 向量检索 → Prompt拼装 → LLM调用 → 返回回答（附引用来源）
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

## 2. 技术选型

|层级|技术|选型理由|
|---|---|---|
|**语言**|Go 1.22+|目标岗位要求；并发模型适合批量论文处理|
|**Web 框架**|Gin|国内 Go 后端岗最主流框架|
|**ORM**|GORM v2|管理业务元数据，自动迁移方便快速迭代|
|**数据库**|PostgreSQL 15+ pgvector|一套数据库同时存业务数据和向量|
|**缓存**|Redis 7+|JWT 会话存储、后续可用于缓存和限流|
|**Embedding**|通义千问 text-embedding-v3|国产模型成本低，中文效果好|
|**LLM**|通义千问|性价比高，国内访问稳定，使用 OpenAI 兼容格式|

### 为什么选 PostgreSQL + pgvector 而不是 Milvus

> **面试话术**："科研团队的论文库规模通常在几百到几千篇（对应几万到几十万 chunk），pgvector 的 HNSW 索引完全够用。在设计上我做了 Repository 层的接口抽象，向量检索逻辑封装在 `VectorRepo` 接口后面，如果后续数据量增长，可以无缝切换到 Milvus。"

---

## 3. 项目目录结构

```
internal/
├── config/           # 配置加载（环境变量）
├── handler/          # HTTP Handler 层（Gin）
│   ├── paper.go      # 论文上传/列表/删除
│   ├── chat.go       # 问答对话接口
│   └── auth.go       # 注册/登录
├── service/          # 业务逻辑层
│   ├── paper_service.go      # 论文处理流水线
│   ├── chat_service.go       # 问答编排（意图识别→检索→LLM）
│   ├── embedding_service.go  # Embedding API 封装
│   └── llm_service.go        # LLM API 封装
├── repository/       # 数据访问层
│   ├── paper_repo.go         # 论文 CRUD
│   ├── chunk_repo.go         # Chunk CRUD
│   ├── vector_repo.go        # 向量检索（原生 SQL）
│   └── conversation_repo.go  # 对话记录 CRUD
├── model/            # 数据模型定义
├── dto/              # 请求/响应类型定义
├── middleware/       # JWT 认证中间件
└── pkg/              # 内部工具包
    ├── chunker/      # 结构感知切片
    ├── extractor/    # PDF/Markdown 文本提取
    ├── parser/       # 论文章节结构识别
    └── prompt/       # Prompt 模板
```

> **面试要点**：与通用 RAG 项目的差异在于 `parser/section_parser.go`（论文结构识别）和 `chunker/paper_chunker.go`（结构感知切片）。

---

## 4. 数据库设计

### 表结构概览

```
users 1──N papers 1──N chunks
  │
  └── 1──N conversations 1──N messages
```

### chunks 表（核心差异化设计）

```sql
CREATE TABLE chunks (
    id           UUID PRIMARY KEY,
    paper_id     UUID NOT NULL REFERENCES papers(id),
    chunk_index  INT NOT NULL,
    content      TEXT NOT NULL,
    token_count  INT NOT NULL,
    embedding    vector(1024) NOT NULL,  -- 向量

    -- 论文结构元数据（核心差异化字段）
    section_type  VARCHAR(64),   -- abstract / method / experiment / ...
    section_title VARCHAR(256),  -- 原始章节标题
    page_number   INT,           -- 所在页码

    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- 向量检索索引
CREATE INDEX idx_chunks_embedding ON chunks
    USING hnsw (embedding vector_cosine_ops);

-- 支持按章节类型过滤
CREATE INDEX idx_chunks_section_type ON chunks(section_type);
```

> **面试考点**：为什么 `section_type` 要建索引？用户问"这篇论文的实验结果是什么"时，可以先用 `section_type = 'experiment'` 过滤再做向量检索，缩小搜索范围、提高精度。这是"结构化过滤 + 向量检索"的混合检索策略。

### messages 表

```sql
CREATE TABLE messages (
    id              UUID PRIMARY KEY,
    conversation_id UUID NOT NULL REFERENCES conversations(id),
    role            VARCHAR(16) NOT NULL,  -- user / assistant
    content         TEXT NOT NULL,
    references_data JSONB,  -- 引用来源（注意：references 是保留字）
    token_usage     JSONB,  -- token 统计
    created_at      TIMESTAMP NOT NULL DEFAULT NOW()
);
```

---

## 5. 核心功能模块

### 5.1 论文结构解析（场景差异化核心）

**目标**：识别论文的 Section 结构（Abstract、Introduction、Method 等）。

**实现**：基于规则的章节识别器，匹配 `"1. Introduction"`、`"3 Methodology"`、`"IV. EXPERIMENTS"` 等格式。

**面试追问**：
- Q: "准确率多少？" A: "标准 ACL/NeurIPS 论文 85-90%，不标准的降级为 `other`，不影响功能。"
- Q: "为什么不用 LLM 识别？" A: "成本。每篇论文上万 token，规则匹配零成本。"

### 5.2 结构感知切片（面试必须讲清楚）

**两级切片策略**：
1. 第一级：按 Section 切分（保持语义完整）
2. 第二级：对过长 Section，按 token 数 + overlap 细分

**参数**：max_chunk_size=512, chunk_overlap=64, min_chunk_size=30

**为什么比通用"固定 token 盲切"好**：
1. 不会把一段方法描述切成两半
2. 每个 chunk 携带 `section_type`，检索时可精确过滤
3. Abstract 短（200-300 token）不会被切碎

### 5.3 Embedding 向量化

**实现要点**：
- 使用 `errgroup` 并发调用 API，`g.SetLimit(4)` 控制并发数
- 批量请求，每批最多 20 条文本
- 维度对齐：chunks 表的 `vector(1024)` 与模型输出一致

### 5.4 向量检索（结构化过滤 + 向量检索）

**核心 SQL**：
```sql
SELECT c.*, p.title AS paper_title, 
       1 - (c.embedding <=> $1::vector) AS similarity
FROM chunks c
JOIN papers p ON c.paper_id = p.id
WHERE p.user_id = $2           -- 用户隔离
  AND c.section_type = $3      -- 可选：章节过滤
ORDER BY c.embedding <=> $1::vector  -- 距离升序 = 相似度降序
LIMIT 5;
```

> **安全设计**：强制 `WHERE p.user_id = $2`，确保用户只能检索自己上传的论文。

> **面试话术**："纯向量检索的问题是用户问'实验结果'可能检索到 Introduction 里的文字。我通过结构化过滤先缩小范围到 Experiment 章节，再做向量检索，精度提升明显。"

### 5.5 意图识别 + Prompt 模板

**意图识别**：基于关键词规则匹配（"对比"、"区别"、"vs" 等 → compare 模式）

**两种 Prompt 模板**：
| 模式 | TopK | 特点 |
|------|------|------|
| extract | 5 | 单论文知识提取，标注引用来源 |
| compare | 10 | 跨论文对比，要求结构化输出（表格） |

### 5.6 多轮对话

**实现要点**：
1. 从 messages 表取历史消息
2. **权限校验**：`FindByIDAndUserID` 验证对话归属，防止用户访问他人对话
3. 将历史拼入 LLM 的 messages 数组

**安全设计**：`FindByIDAndUserID` 和 `DeleteByIDAndUserID` 确保用户只能访问/删除自己的对话。

---

## 6. API 接口

|方法|路径|说明|状态|
|---|---|---|---|
|POST|`/api/v1/auth/register`|用户注册|✅|
|POST|`/api/v1/auth/login`|用户登录，返回 JWT|✅|
|POST|`/api/v1/papers/upload`|上传论文|✅|
|GET|`/api/v1/papers`|获取论文列表|✅|
|GET|`/api/v1/papers/:id`|获取论文详情|✅|
|DELETE|`/api/v1/papers/:id`|删除论文|✅|
|POST|`/api/v1/chat`|发送问题，获取 RAG 回答|✅|
|GET|`/api/v1/conversations`|获取对话列表|✅|
|GET|`/api/v1/conversations/:id/messages`|获取消息历史|✅|
|DELETE|`/api/v1/conversations/:id`|删除对话|✅|

---

## 7. 安全设计

| 场景 | 实现方式 |
|------|----------|
| 论文列表/详情/删除 | `WHERE user_id = ?` 或验证 `paper.UserID` |
| 向量检索 | `WHERE p.user_id = $2` 强制过滤 |
| 追问历史 | `FindByIDAndUserID` 验证对话归属 |
| 对话删除 | `DeleteByIDAndUserID` 带用户校验 |

---

## 8. 部署方案

使用 Docker Compose 部署 PostgreSQL (pgvector)、Redis、后端服务。

---

## 9. 开发进度

```
Step 1-5: ✅ 已完成（用户认证、论文处理、切片向量化、检索问答、对话管理）
Step 6:   待实现（中间件：CORS、限流、日志）
Step 7:   待实现（前端开发）
Step 8:   待实现（容器化部署）
```

---

## 10. 面试高频追问清单

### RAG 原理类

1. 什么是 RAG？为什么不直接把论文全部喂给大模型？
2. Embedding 是什么？为什么能用向量表示语义？
3. 余弦相似度和 L2 距离有什么区别？为什么用余弦？
4. Chunk 太大或太小分别会有什么问题？

### 场景设计类（PaperMind 专属，高区分度）

5. **你的切片策略和通用 RAG 有什么区别？** → 两级切片：先按 Section 保持语义完整，再按 token 细分。
6. **Section 识别准确率怎么样？** → 标准论文 85-90%，不标准降级为 other，不影响功能。
7. **跨论文对比模式的 Prompt 怎么设计的？** → TopK 更大，Prompt 要求结构化对比输出（表格）。
8. **如果检索出的内容和问题不相关怎么办？** → 设相似度阈值过滤；进阶方案引入 Re-rank。
9. **为什么用规则做 Section 识别而不是 LLM？** → 成本。规则零成本。

### 工程实现类

10. 上传一篇 50 页论文的处理流程？瓶颈在哪？ → 提取→解析→切片→Embedding，瓶颈在 API 网络 I/O，用 Goroutine 并发优化。
11. 两个用户同时上传论文怎么处理？ → 每个上传启动独立 Goroutine，user_id + paper_id 隔离。
12. HNSW 索引原理？ → 分层可导航小世界图，O(log n) 近似搜索。
13. 为什么选 pgvector 不选 Milvus？ → 科研场景数据量小，pgvector 够用且运维简单。
14. **用户数据如何隔离？** → 检索时 `WHERE user_id = ?`；追问时 `FindByIDAndUserID` 验证；删除时 `DeleteByIDAndUserID` 校验。
15. **为什么用原生 SQL 做向量检索？** → pgvector 的 `<=>` 运算符不在 GORM 标准方言中。

### 高并发 & 扩展类

16. 如果实验室所有人都用，瓶颈在哪？ → LLM API 延迟和成本。方案：Redis 缓存、限流、流式输出。
17. 要支持图表问答怎么做？ → 多模态 Embedding，用 CLIP 等视觉模型向量化图片。

---

_文档版本：v3.0（精简版）| 最后更新：2026-04-06_