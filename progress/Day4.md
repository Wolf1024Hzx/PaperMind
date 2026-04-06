# Day 4 进度记录

> 日期：2026-04-06 | 时间：10:30 - 14:02

## 今日学习内容

### 1. pgvector 向量存储

| 要点 | 说明 |
|------|------|
| 类型定义 | `vector(1024)` 固定维度 |
| Go 库 | `github.com/pgvector/pgvector-go`，`pgvector.NewVector([]float32)` |
| 索引 | HNSW（分层可导航小世界图），`vector_cosine_ops` 余弦距离 |
| 查询 | `ORDER BY embedding <=> $1` 余弦相似度排序 |

### 2. 切片策略（两级切片）

```
[]Section（结构解析输出）
        │
        ▼
  第一级：按 Section 切分（保持语义完整）
        │
        ▼
  第二级：对过长 Section，按 token 数 + overlap 细分
        │
        ▼
  []ChunkResult
```

| 参数 | 值 | 说明 |
|------|-----|------|
| MaxChunkSize | 512 | 单 chunk 上限 |
| ChunkOverlap | 64 | 相邻 chunk 重叠 |
| MinChunkSize | 30 | 过短则跳过 |

**面试要点**：通用 RAG 盲切会把方法描述切成两半，先按 Section 保持语义完整。

### 3. errgroup 并发控制

```go
g, ctx := errgroup.WithContext(ctx)
g.SetLimit(4)  // 最多 4 个并发

for batch := range batches {
    g.Go(func() error {
        embeddings, err := client.Embed(ctx, batch)
        // ...
        return nil
    })
}

if err := g.Wait(); err != nil {
    // 任一失败则整体失败
}
```

**优势**：任一 goroutine 出错，通过 context 取消其他请求。

### 4. 依赖容器模式（分层初始化）

```
config.Load()
      ↓
   Container
      ↓
 ┌──┴──┬──────┬───────┬─────────┐
 DB   Redis  Repos  Services  Handlers
      ↓
   app.New() + app.Run()
```

**初始化顺序**：基础设施 → Repositories → Services → Handlers → 注册路由

### 5. .env 配置文件

`godotenv.Load()` 加载 `.env` 文件，敏感信息不入代码库。

---

## 今日工程实现内容

### Chunk 数据模型

**文件**：`internal/model/chunk.go` ✅ 我自己手写

| 字段 | 说明 |
|------|------|
| `embedding` | `pgvector.Vector` 类型，GORM tag `type:vector(1024)` |
| `section_type` / `section_title` / `page_number` | 论文结构元数据 |

**注意**：`PageNumber` 用 `*int` 指针类型（Markdown 上传时可能为 nil）。

### Chunk Repository

**文件**：`internal/repository/chunk_repo.go` ✅ 我自己手写

| 方法 | 说明 |
|------|------|
| `BatchCreate` | 批量插入 chunks |
| `FindByPaperID` | 按 paper_id 查询，`Order("chunk_index ASC")` |
| `DeleteByPaperID` | 论文删除时联动删除 chunks |

### 结构感知切片器

**文件**：`internal/pkg/chunker/paper_chunker.go` ✅ 我自己手写

核心函数：
- `ChunkSections(sections []Section) []ChunkResult` — 两级切片
- `splitByTokens(text, chunkSize, overlap)` — 按空格分词后切分

**O(n²) 问题修复**：用户初版每次切片都调用 `strings.Fields()`，改为一开始调用一次，后续用索引切片。

### Embedding Service

**文件**：`internal/service/embedding_service.go` ✅ Agent 编写

**接口抽象**：
```
EmbeddingClient
    ├── MockEmbeddingClient（测试用，确定性向量）
    └── QwenEmbeddingClient（阿里云 API）
```

| 实现 | 说明 |
|------|------|
| Mock | 返回固定向量 `[0, 0.001, 0.002, ...]` |
| Qwen | 调用 `dashscope.aliyuncs.com` 多模态 Embedding API |

**API 关键信息**：

| 配置项 | 值 |
|--------|-----|
| 端点 | `dashscope.aliyuncs.com/api/v1/services/embeddings/multimodal-embedding` |
| 模型 | `qwen3-vl-embedding` |
| 维度 | 1024（固定） |
| 单次限制 | 20 条文本 |

### 依赖容器重构

**文件**：`internal/app/container.go` ✅ 我自己手写（参考 Agent 建议）

新增初始化逻辑：
- `initRepositories()` — 新增 ChunkRepository
- `initServices()` — 根据 `EmbeddingType` 切换 mock/qwen 实现

### 应用主体

**文件**：`internal/app/app.go` ✅ Agent 编写

- `New(cfg)` — 创建 Container，初始化所有依赖
- `Run()` — 启动 Gin 服务器 + 优雅关闭（监听 SIGINT/SIGTERM）

### 配置文件支持

**文件**：`internal/config/config.go` ✅ 我自己手写

新增配置项：

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `EMBEDDING_TYPE` | mock | mock / qwen |
| `ALIYUN_API_KEY` | - | 阿里云 API Key |
| `EMBEDDING_MODEL` | qwen3-vl-embedding | 模型名称 |
| `EMBEDDING_BATCH_SIZE` | 20 | 单批条数 |
| `EMBEDDING_MAX_CONCURRENCY` | 4 | 并发数 |

**文件**：`.env.example` ✅ Agent 编写

### main.go 简化

**文件**：`cmd/server/main.go` ✅ 我自己手写

简化为 15 行：加载配置 → 创建 app → 启动服务。

---

## 集成到 PaperService

**文件**：`internal/service/paper_service.go` ✅ 我自己手写（修改）

在 `processPaper` 流水线中新增：

```
extracting → parsing → chunking → embedding → completed
```

| 阶段 | 操作 |
|------|------|
| chunking | `ChunkSections()` 切片 |
| embedding | `errgroup` 并发调用 Embedding API |
| 入库 | `BatchCreate()` 写入 chunks 表 |

---

## 验证结果

### Mock Embedding 测试

上传论文 → chunks 表有数据 → embedding 列非空（固定向量值）

### 真实 API 测试

切换 `EMBEDDING_TYPE=qwen`：

```sql
SELECT chunk_index, array_length(embedding::real[], 1) FROM chunks LIMIT 5;
-- 结果: 1024（向量维度正确）
```

向量值非 Mock 的 0, 0.001, 0.002 模式，确认 API 调用成功。

上传 "Attention Is All You Need" Markdown 版本 → 生成 23 个 chunks。

---

## 今日算法题

（待补充）

---

## 后续计划

- 实现向量检索接口（pgvector 相似度查询）
- 实现 RAG 问答接口（Prompt 拼装 + LLM 调用）