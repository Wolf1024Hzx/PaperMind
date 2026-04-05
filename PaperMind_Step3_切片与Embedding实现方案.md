# PaperMind Step 3 实现方案：文本切片与 Embedding 向量化

## 概述

本文档覆盖从"结构解析输出 []Section"到"向量写入 chunks 表"的完整实现。这是 RAG Pipeline 中最核心的环节，面试必问，务必理解每一步。

---

## 1. 整体流程

```
[]Section（已完成，来自 Step 2）
        │
        ▼
  ┌─────────────┐
  │ 更新状态为    │
  │ chunking     │
  └──────┬──────┘
         │
         ▼
  ┌─────────────┐
  │ 结构感知切片  │  ← Section 短则整块保留，长则按 token 数细分
  └──────┬──────┘
         │
         ▼
  ┌─────────────┐
  │ 更新状态为    │
  │ embedding    │
  └──────┬──────┘
         │
         ▼
  ┌─────────────┐
  │ 批量调用      │  ← 并发调用 Embedding API，每批最多 25 条
  │ Embedding API │
  └──────┬──────┘
         │
         ▼
  ┌─────────────┐
  │ 写入 chunks  │  ← chunk 文本 + 向量 + section 元数据 一起入库
  │ 表           │
  └──────┬──────┘
         │
         ▼
  ┌─────────────┐
  │ 更新 paper   │
  │ status →     │
  │ completed    │
  │ chunk_count  │
  └─────────────┘
```

---

## 2. 新增文件清单

```
internal/
├── model/
│   └── chunk.go                # Chunk 数据模型
├── repository/
│   └── chunk_repo.go           # Chunk 数据访问层
├── pkg/
│   └── chunker/
│       └── paper_chunker.go    # 结构感知切片器
└── service/
    └── embedding_service.go    # Embedding API 封装
```

需要修改的文件：

```
internal/service/paper_service.go   # processPaper 中接入切片和 Embedding
scripts/init.sql                     # 添加 chunks 表建表语句
```

---

## 3. chunks 表建表

在 PostgreSQL 中执行（或加入 init.sql）：

```sql
CREATE TABLE chunks (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    paper_id     UUID          NOT NULL REFERENCES papers(id) ON DELETE CASCADE,
    chunk_index  INT           NOT NULL,
    content      TEXT          NOT NULL,
    token_count  INT           NOT NULL,
    embedding    vector(1024)  NOT NULL,

    -- 论文结构元数据
    section_type  VARCHAR(64),
    section_title VARCHAR(256),
    page_number   INT,

    created_at   TIMESTAMP     NOT NULL DEFAULT NOW()
);

-- 向量检索索引（HNSW，余弦距离）
CREATE INDEX idx_chunks_embedding ON chunks
    USING hnsw (embedding vector_cosine_ops);

-- 业务索引
CREATE INDEX idx_chunks_paper_id ON chunks(paper_id);
CREATE INDEX idx_chunks_section_type ON chunks(section_type);
```

**关于向量维度：** 这里写 `vector(1024)` 是示例值。实际维度取决于你选的 Embedding 模型。通义千问 text-embedding-v3 支持 1024/768/512 维度，建议用 1024。如果用 OpenAI text-embedding-3-small 则是 1536 维。确定模型后统一改这个数字。

---

## 4. Chunk 数据模型

文件：`internal/model/chunk.go`

```go
package model

import (
    "time"

    "github.com/google/uuid"
    "github.com/pgvector/pgvector-go"
)

type Chunk struct {
    ID           uuid.UUID      `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
    PaperID      uuid.UUID      `gorm:"type:uuid;not null;index" json:"paperId"`
    ChunkIndex   int            `gorm:"not null" json:"chunkIndex"`
    Content      string         `gorm:"type:text;not null" json:"content"`
    TokenCount   int            `gorm:"not null" json:"tokenCount"`
    Embedding    pgvector.Vector `gorm:"type:vector(1024);not null" json:"-"`

    // 论文结构元数据
    SectionType  *string `gorm:"column:section_type" json:"sectionType,omitempty"`
    SectionTitle *string `gorm:"column:section_title" json:"sectionTitle,omitempty"`
    PageNumber   *int    `gorm:"column:page_number" json:"pageNumber,omitempty"`

    CreatedAt    time.Time `gorm:"autoCreateTime" json:"createdAt"`
}

func (Chunk) TableName() string {
    return "chunks"
}
```

**需要安装 pgvector-go：**

```bash
go get github.com/pgvector/pgvector-go
```

---

## 5. 结构感知切片器

文件：`internal/pkg/chunker/paper_chunker.go`

### 5.1 Token 计数

用最简单的方式：按空格分词计数。这对英文论文足够准确，面试时如果被问可以说"生产环境可以用 tiktoken 做精确计数，当前用空格分词作为近似"。

```go
package chunker

import (
    "strings"

    "wolfden.website/papermind/internal/pkg/common"
)

// 切片参数
const (
    MaxChunkSize = 512  // 单个 chunk 的目标 token 上限
    ChunkOverlap = 64   // 相邻 chunk 的重叠 token 数
    MinChunkSize = 30   // 低于此长度的 chunk 合并到上一个
)

// ChunkResult 表示一个切片结果
type ChunkResult struct {
    Content      string
    TokenCount   int
    SectionType  string
    SectionTitle string
    PageNumber   int
}

// countTokens 简易 token 计数（按空格分词）
func countTokens(text string) int {
    if strings.TrimSpace(text) == "" {
        return 0
    }
    return len(strings.Fields(text))
}
```

### 5.2 核心切片逻辑

```go
// ChunkSections 将论文的章节列表切分为适合 Embedding 的文本块
// 核心策略：两级切片
//   第一级：按 Section 切分（保持语义完整）
//   第二级：对过长的 Section，按 token 数 + overlap 细分
func ChunkSections(sections []common.Section) []ChunkResult {
    var results []ChunkResult

    for _, section := range sections {
        content := strings.TrimSpace(section.Content)
        if content == "" {
            continue
        }

        tokenCount := countTokens(content)

        if tokenCount <= MaxChunkSize {
            // Section 足够短，整体作为一个 chunk
            if tokenCount < MinChunkSize {
                // 过短的 section（如只有标题没有实质内容），跳过
                continue
            }
            results = append(results, ChunkResult{
                Content:      content,
                TokenCount:   tokenCount,
                SectionType:  section.Type,
                SectionTitle: section.Title,
                PageNumber:   section.StartPage,
            })
        } else {
            // Section 太长，按 token 数细分
            subChunks := splitByTokens(content, MaxChunkSize, ChunkOverlap)
            for _, sc := range subChunks {
                tc := countTokens(sc)
                if tc < MinChunkSize {
                    continue
                }
                results = append(results, ChunkResult{
                    Content:      sc,
                    TokenCount:   tc,
                    SectionType:  section.Type,
                    SectionTitle: section.Title,
                    PageNumber:   section.StartPage,
                })
            }
        }
    }

    return results
}
```

### 5.3 按 token 细分函数

```go
// splitByTokens 将长文本按 token 数切分，支持 overlap
func splitByTokens(text string, chunkSize int, overlap int) []string {
    words := strings.Fields(text)
    if len(words) <= chunkSize {
        return []string{text}
    }

    var chunks []string
    step := chunkSize - overlap // 每次前进的步长

    for i := 0; i < len(words); i += step {
        end := i + chunkSize
        if end > len(words) {
            end = len(words)
        }

        chunk := strings.Join(words[i:end], " ")
        chunks = append(chunks, chunk)

        // 如果已经到末尾，退出
        if end == len(words) {
            break
        }
    }

    return chunks
}
```

### 5.4 面试必背：为什么这样切

**为什么先按 Section 再按 token？**
→ 通用 RAG 直接按固定 token 盲切，可能把一段方法描述切成两半。我先按 Section 保持语义完整，只有 Section 太长时才细分。

**Overlap 有什么用？**
→ 如果关键信息恰好在两个 chunk 的边界，没有 overlap 会丢失上下文。64 token 的 overlap 让相邻 chunk 有交集，提高检索命中率。

**MaxChunkSize 为什么选 512？**
→ 太大会导致 Embedding 语义模糊（一个向量里塞太多主题），太小会丢失上下文。512 是业界常用的平衡点。

---

## 6. Embedding Service

文件：`internal/service/embedding_service.go`

### 6.1 接口定义

```go
package service

import (
    "context"
)

// EmbeddingClient 定义 Embedding API 的统一接口
// 方便后续切换不同的 Embedding 提供商
type EmbeddingClient interface {
    Embed(ctx context.Context, texts []string) ([][]float32, error)
    Dimension() int // 返回向量维度，用于校验
}
```

### 6.2 通义千问实现

```go
package service

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"
)

type QwenEmbeddingClient struct {
    apiKey     string
    model      string
    dimension  int
    httpClient *http.Client
}

func NewQwenEmbeddingClient(apiKey string) *QwenEmbeddingClient {
    return &QwenEmbeddingClient{
        apiKey:    apiKey,
        model:     "text-embedding-v3",
        dimension: 1024,
        httpClient: &http.Client{
            Timeout: 30 * time.Second,
        },
    }
}

func (c *QwenEmbeddingClient) Dimension() int {
    return c.dimension
}

// Embed 调用通义千问 Embedding API
// texts 长度建议不超过 25（API 限制）
func (c *QwenEmbeddingClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
    // 构建请求体
    reqBody := map[string]interface{}{
        "model": c.model,
        "input": map[string]interface{}{
            "texts": texts,
        },
        "parameters": map[string]interface{}{
            "dimension": c.dimension,
        },
    }

    bodyBytes, err := json.Marshal(reqBody)
    if err != nil {
        return nil, fmt.Errorf("序列化请求失败: %w", err)
    }

    // 创建 HTTP 请求
    url := "https://dashscope.aliyuncs.com/api/v1/services/embeddings/text-embedding/text-embedding"
    req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
    if err != nil {
        return nil, fmt.Errorf("创建请求失败: %w", err)
    }

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+c.apiKey)

    // 发送请求
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("请求 Embedding API 失败: %w", err)
    }
    defer resp.Body.Close()

    respBytes, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("读取响应失败: %w", err)
    }

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("Embedding API 返回错误 %d: %s", resp.StatusCode, string(respBytes))
    }

    // 解析响应
    var result qwenEmbeddingResponse
    if err := json.Unmarshal(respBytes, &result); err != nil {
        return nil, fmt.Errorf("解析响应失败: %w", err)
    }

    // 按 text_index 排序提取向量
    embeddings := make([][]float32, len(texts))
    for _, item := range result.Output.Embeddings {
        if item.TextIndex < len(embeddings) {
            embeddings[item.TextIndex] = item.Embedding
        }
    }

    // 校验每个向量都有值
    for i, emb := range embeddings {
        if len(emb) == 0 {
            return nil, fmt.Errorf("第 %d 个文本的 Embedding 为空", i)
        }
    }

    return embeddings, nil
}

// 通义千问 Embedding API 响应结构
type qwenEmbeddingResponse struct {
    Output struct {
        Embeddings []struct {
            TextIndex int       `json:"text_index"`
            Embedding []float32 `json:"embedding"`
        } `json:"embeddings"`
    } `json:"output"`
    Usage struct {
        TotalTokens int `json:"total_tokens"`
    } `json:"usage"`
}
```

### 6.3 备选：DeepSeek 实现

如果通义千问注册麻烦，也可以用其他支持 Embedding 的 API。接口是统一的 `EmbeddingClient`，换一个实现即可。你也可以先用一个 mock 实现来跑通流程：

```go
// MockEmbeddingClient 用于开发阶段测试，生成随机向量
type MockEmbeddingClient struct {
    dim int
}

func NewMockEmbeddingClient(dim int) *MockEmbeddingClient {
    return &MockEmbeddingClient{dim: dim}
}

func (c *MockEmbeddingClient) Dimension() int {
    return c.dim
}

func (c *MockEmbeddingClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
    results := make([][]float32, len(texts))
    for i := range texts {
        vec := make([]float32, c.dim)
        for j := range vec {
            vec[j] = float32(j) * 0.001 // 简单填充，非随机
        }
        results[i] = vec
    }
    return results, nil
}
```

**建议：先用 Mock 跑通整条链路（切片→入库→检索），确认数据能正确写入 chunks 表后，再接真实 API。** 这样能把"链路不通"和"API 调用失败"两类问题分开排查。

---

## 7. Chunk Repository

文件：`internal/repository/chunk_repo.go`

```go
package repository

import (
    "context"

    "github.com/google/uuid"
    "gorm.io/gorm"

    "wolfden.website/papermind/internal/model"
)

type ChunkRepository struct {
    db *gorm.DB
}

func NewChunkRepository(db *gorm.DB) *ChunkRepository {
    return &ChunkRepository{db: db}
}

// BatchCreate 批量插入 chunks
func (r *ChunkRepository) BatchCreate(ctx context.Context, chunks []model.Chunk) error {
    if len(chunks) == 0 {
        return nil
    }
    // GORM 的 Create 支持切片批量插入
    return r.db.WithContext(ctx).Create(&chunks).Error
}

// FindByPaperID 获取某篇论文的所有 chunks
func (r *ChunkRepository) FindByPaperID(ctx context.Context, paperID uuid.UUID) ([]model.Chunk, error) {
    var chunks []model.Chunk
    err := r.db.WithContext(ctx).
        Where("paper_id = ?", paperID).
        Order("chunk_index ASC").
        Find(&chunks).Error
    return chunks, err
}

// DeleteByPaperID 删除某篇论文的所有 chunks（论文删除时联动）
func (r *ChunkRepository) DeleteByPaperID(ctx context.Context, paperID uuid.UUID) error {
    return r.db.WithContext(ctx).
        Where("paper_id = ?", paperID).
        Delete(&model.Chunk{}).Error
}
```

---

## 8. 集成到 paper_service.go

在 `processPaper` 方法中，接在结构解析之后：

```go
// ========== 阶段 3: 切片 ==========
s.updateStatus(ctx, paperID, "chunking")

chunkResults := chunker.ChunkSections(sections)

if len(chunkResults) == 0 {
    s.updateStatus(ctx, paperID, "failed")
    log.Printf("论文 %s 切片结果为空", paperID)
    return
}

log.Printf("论文 %s 切片完成，共 %d 个 chunks", paperID, len(chunkResults))

// ========== 阶段 4: Embedding ==========
s.updateStatus(ctx, paperID, "embedding")

// 提取所有 chunk 的文本
texts := make([]string, len(chunkResults))
for i, cr := range chunkResults {
    texts[i] = cr.Content
}

// 分批调用 Embedding API（每批最多 25 条）
batchSize := 25
allEmbeddings := make([][]float32, len(texts))

g, gCtx := errgroup.WithContext(ctx)
g.SetLimit(4) // 最多 4 个并发请求

for batchStart := 0; batchStart < len(texts); batchStart += batchSize {
    start := batchStart
    end := start + batchSize
    if end > len(texts) {
        end = len(texts)
    }

    g.Go(func() error {
        batch := texts[start:end]
        embeddings, err := s.embeddingClient.Embed(gCtx, batch)
        if err != nil {
            return fmt.Errorf("Embedding 批次 [%d:%d] 失败: %w", start, end, err)
        }
        // 写入对应位置
        for i, emb := range embeddings {
            allEmbeddings[start+i] = emb
        }
        return nil
    })
}

if err := g.Wait(); err != nil {
    s.updateStatus(ctx, paperID, "failed")
    log.Printf("论文 %s Embedding 失败: %v", paperID, err)
    return
}

// ========== 阶段 5: 写入数据库 ==========
chunks := make([]model.Chunk, len(chunkResults))
for i, cr := range chunkResults {
    sectionType := cr.SectionType
    sectionTitle := cr.SectionTitle
    pageNumber := cr.PageNumber

    chunks[i] = model.Chunk{
        PaperID:      paperID,
        ChunkIndex:   i,
        Content:      cr.Content,
        TokenCount:   cr.TokenCount,
        Embedding:    pgvector.NewVector(allEmbeddings[i]),
        SectionType:  &sectionType,
        SectionTitle: &sectionTitle,
        PageNumber:   &pageNumber,
    }
}

if err := s.chunkRepo.BatchCreate(ctx, chunks); err != nil {
    s.updateStatus(ctx, paperID, "failed")
    log.Printf("论文 %s chunks 写入数据库失败: %v", paperID, err)
    return
}

// 更新论文状态和 chunk 数量
s.paperRepo.UpdateChunkCount(ctx, paperID, len(chunks))
s.updateStatus(ctx, paperID, "completed")
log.Printf("论文 %s 处理完成，共写入 %d 个 chunks", paperID, len(chunks))
```

### 需要在 paper_repo.go 新增的方法

```go
// UpdateChunkCount 更新论文的 chunk 数量
func (r *PaperRepository) UpdateChunkCount(ctx context.Context, paperID uuid.UUID, count int) error {
    return r.db.WithContext(ctx).
        Model(&model.Paper{}).
        Where("id = ?", paperID).
        Update("chunk_count", count).Error
}
```

### PaperService 结构体需要新增的依赖

```go
type PaperService struct {
    paperRepo       *repository.PaperRepository
    chunkRepo       *repository.ChunkRepository    // 新增
    embeddingClient EmbeddingClient                 // 新增
    uploadDir       string
}
```

在 `main.go` 中组装时传入：

```go
chunkRepo := repository.NewChunkRepository(db)
embeddingClient := service.NewMockEmbeddingClient(1024) // 先用 Mock，跑通后换真实 API
paperService := service.NewPaperService(paperRepo, chunkRepo, embeddingClient, cfg.UploadDir)
```

---

## 9. 需要安装的依赖

```bash
go get github.com/pgvector/pgvector-go
go get golang.org/x/sync/errgroup
```

---

## 10. 开发顺序建议

```
Step A: 建 chunks 表 + model/chunk.go + chunk_repo.go       （30 分钟）
  └── 验证：GORM AutoMigrate 或手动建表成功

Step B: 写切片器 paper_chunker.go                            （30 分钟）
  └── 验证：写个简单的 main_test.go，传入几个 Section，
      打印输出的 ChunkResult 列表，确认切片逻辑正确

Step C: 写 EmbeddingClient 接口 + Mock 实现                   （15 分钟）
  └── 验证：调用 Mock，确认返回正确维度的向量

Step D: 集成到 processPaper，用 Mock Embedding 跑通全链路       （30 分钟）
  └── 验证：上传论文后，chunks 表中有数据，
      每条记录的 embedding 列非空，section_type 正确

Step E: 注册通义千问 API Key，实现真实 EmbeddingClient          （30 分钟）
  └── 验证：用真实 API 替换 Mock，上传论文后 chunks 表数据正常

Step F: 在 paper 详情接口中返回 chunk 信息                     （15 分钟）
  └── 验证：GET /api/v1/papers/:id 返回 chunkCount 和 chunks 列表
```

**总预估时间：2.5 - 3 小时**（不包括 API 注册和排查网络问题的时间）

---

## 11. 通义千问 API Key 获取

1. 访问 `dashscope.console.aliyun.com`
2. 注册/登录阿里云账号
3. 开通 DashScope 服务（免费额度足够测试）
4. 在控制台创建 API Key
5. 将 API Key 配置到 `.env` 文件中：`EMBEDDING_API_KEY=sk-xxxxx`

如果注册阿里云麻烦，也可以用 DeepSeek 或 OpenAI 兼容的 API，只需要改 URL 和请求格式，接口保持 `EmbeddingClient` 不变。

---

## 12. config.go 新增配置项

```go
type Config struct {
    // ... 已有配置 ...
    EmbeddingAPIKey string // 环境变量: EMBEDDING_API_KEY
    EmbeddingModel  string // 环境变量: EMBEDDING_MODEL，默认 "text-embedding-v3"
}
```

---

## 13. 验证方法

### 上传论文后检查 chunks 表

```sql
-- 查看 chunks 数量
SELECT COUNT(*) FROM chunks WHERE paper_id = 'xxx';

-- 查看每个 chunk 的基本信息
SELECT chunk_index, section_type, section_title, token_count,
       LEFT(content, 80) AS content_preview
FROM chunks
WHERE paper_id = 'xxx'
ORDER BY chunk_index;

-- 验证 embedding 列非空
SELECT chunk_index, array_length(embedding::real[], 1) AS dim
FROM chunks
WHERE paper_id = 'xxx'
LIMIT 5;
```

### 预期结果示例

上传 Attention Is All You Need 的 Markdown 版本后：

```
chunk_index | section_type  | section_title              | token_count
-----------+---------------+----------------------------+------------
0          | other         | Attention Is All You Need  | 89
1          | abstract      | Abstract                   | 156
2          | introduction  | 1 Introduction             | 312
3          | related_work  | 2 Background               | 298
4          | method        | 3 Model Architecture       | 118
5          | other         | 3.2 Attention              | 487
6          | other         | 3.2 Attention              | 325   ← 该 section 被切成两个 chunk
...
```

---

## 14. 面试追问准备

完成这一步后确保你能回答：

1. **为什么 MaxChunkSize 选 512？**
   → 太大语义模糊、太小丢失上下文，512 是平衡点。可以根据实际检索效果调参。

2. **Overlap 设多少合适？为什么？**
   → 设了 64 token，约为 chunk 大小的 12%。防止边界信息丢失。太大会增加存储和计算成本。

3. **Embedding 为什么要分批？**
   → API 有单次请求长度限制（通常 25 条），且分批后可以并发调用提高速度。

4. **并发调用 Embedding 用了什么？**
   → `errgroup.Group` + `SetLimit(4)`，最多 4 个并发请求。errgroup 的好处是任何一个 goroutine 出错，可以通过 context 取消其他正在执行的请求。

5. **如果 Embedding API 调用失败了怎么办？**
   → 论文状态更新为 failed，记录错误日志。用户可以重新触发处理（删除重传或加一个重试接口）。

6. **Token 计数为什么用空格分词而不是 tiktoken？**
   → 空格分词是近似方案，实现简单零依赖。对英文论文误差在 10% 以内，不影响切片质量。生产环境可以换 tiktoken 做精确计数。

7. **pgvector 的 HNSW 索引是什么？**
   → 分层可导航小世界图，通过多层图结构实现近似最近邻搜索。建索引时间较长但查询快，适合中小数据量。比暴力扫描快数个量级，精度损失极小。
