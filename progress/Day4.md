# Day 4 进度记录

> 日期：2026-04-06 | 时间：10:32 - 19:57

## 今日学习内容

### 1. pgvector 向量存储

| 要点 | 说明 |
|------|------|
| 类型定义 | `vector(1024)` 固定维度 |
| Go 库 | `github.com/pgvector/pgvector-go`，`pgvector.NewVector([]float32)` |
| 索引 | HNSW（分层可导航小世界图），`vector_cosine_ops` 余弦距离 |
| 查询 | `ORDER BY embedding <=> $1` 余弦相似度排序 |

**面试要点**：`ORDER BY embedding <=> $1` 是距离升序，距离越小相似度越高。不能用 `ORDER BY similarity DESC` 因为那是投影列。

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

### 3. 向量检索（原生 SQL + pgvector）

**为什么用原生 SQL**：pgvector 的 `<=>` 运算符不在 GORM 标准方言中，原生 SQL 更可控。

```sql
SELECT c.*, p.title, 1 - (c.embedding <=> $1::vector) AS similarity
FROM chunks c JOIN papers p ON c.paper_id = p.id
WHERE p.user_id = $2
ORDER BY c.embedding <=> $1::vector  -- 距离升序 = 相似度降序
LIMIT $3
```

### 4. 用户数据隔离

| 功能 | 隔离方式 |
|------|----------|
| 论文列表/详情/删除 | `WHERE user_id = ?` 或验证 `paper.UserID` |
| 向量检索 | `WHERE p.user_id = $2` 强制过滤 |
| 追问历史 | `FindByIDAndUserID` 验证对话归属 |
| 对话删除 | `DeleteByIDAndUserID` 带用户校验 |

**面试要点**：检索时必须带 `user_id` 过滤，否则用户 A 能检索到用户 B 上传的论文内容。

### 5. errgroup 并发控制

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

### 6. LLM API（OpenAI 兼容格式）

阿里云千问支持 OpenAI 兼容模式，比原生 DashScope 格式更简洁：

| 项目 | 值 |
|------|-----|
| 端点 | `dashscope.aliyuncs.com/compatible-mode/v1/chat/completions` |
| 请求格式 | `{"messages": [{"role": "system", "content": "..."}, ...]}` |
| 响应格式 | `{"choices": [{"message": {"content": "..."}}]}` |

### 7. 意图识别（规则匹配）

```go
func detectMode(question string) string {
    compareKeywords := []string{"对比", "区别", "差异", "vs", "compare", ...}
    for _, kw := range compareKeywords {
        if strings.Contains(strings.ToLower(question), kw) {
            return "compare"
        }
    }
    return "extract"
}
```

**面试要点**：零成本、零延迟，关键词匹配对明确意图足够准确。

### 8. Prompt 模板设计

| 模式 | TopK | Prompt 特点 |
|------|------|-------------|
| extract | 5 | 单论文知识提取，标注引用来源 |
| compare | 10 | 跨论文对比，要求结构化输出（表格） |

### 9. 对话管理接口设计

| 接口 | 路径 | 安全设计 |
|------|------|----------|
| 发送问题 | `POST /api/v1/chat` | 检索时 `user_id` 过滤 |
| 获取对话列表 | `GET /api/v1/conversations` | 只返回当前用户的对话 |
| 获取消息历史 | `GET /api/v1/conversations/:id/messages` | `FindByIDAndUserID` 验证归属 |
| 删除对话 | `DELETE /api/v1/conversations/:id` | `DeleteByIDAndUserID` 带用户校验 |

### 10. 依赖容器模式（分层初始化）

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

### 11. .env 配置文件

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

### Conversation / Message 数据模型

**文件**：`internal/model/conversation.go` ✅ Agent 直接编写

| 字段 | 说明 |
|------|------|
| `Title` | `*string` 指针类型（允许为空） |
| `Mode` | extract / compare |

**文件**：`internal/model/message.go` ✅ Agent 直接编写

| 字段 | 说明 |
|------|------|
| `ReferencesData` | `*json.RawMessage`（PostgreSQL 保留字 `references`，用 `references_data`） |
| `TokenUsage` | `*json.RawMessage` |

### Chunk Repository

**文件**：`internal/repository/chunk_repo.go` ✅ 我自己手写

| 方法 | 说明 |
|------|------|
| `BatchCreate` | 批量插入 chunks |
| `FindByPaperID` | 按 paper_id 查询，`Order("chunk_index ASC")` |
| `DeleteByPaperID` | 论文删除时联动删除 chunks |

### Vector Repository

**文件**：`internal/repository/vector_repo.go` ✅ 用户要求详细指导，Agent 给出原生 SQL 方案

核心逻辑：
- 参数化查询：`$1` 向量、`$2` user_id、动态 `$N` 过滤条件
- 支持 `PaperIDs`、`SectionTypes`、`YearFrom` 可选过滤
- 结果包含 `similarity = 1 - distance`

**Bug 修复**：用户主动发现 ORDER BY 方向错误（按 similarity 排返回最低相似度结果），改为 `ORDER BY embedding <=> $1`。

**安全修复**：用户主动发现 SQL 无 `user_id` 过滤，Agent 添加 `WHERE p.user_id = $2`。

### Conversation Repository

**文件**：`internal/repository/conversation_repo.go` ✅ Agent 直接编写

| 方法 | 说明 |
|------|------|
| `Create` | 创建对话 |
| `FindByUserID` | 查询用户对话（按 `updated_at DESC`） |
| `CreateMessage` | 创建消息 |
| `FindMessages` | 查询消息（按 `created_at ASC`） |
| `FindByIDAndUserID` | 权限校验（用户主动发现缺失） |
| `DeleteByIDAndUserID` | 删除带校验（用户主动发现缺失） |
| `UpdateUpdatedAt` | 更新时间戳（用户主动发现 Bug） |

### 结构感知切片器

**文件**：`internal/pkg/chunker/paper_chunker.go` ✅ 我自己手写

核心函数：
- `ChunkSections(sections []Section) []ChunkResult` — 两级切片
- `splitByTokens(text, chunkSize, overlap)` — 按空格分词后切分

**O(n²) 问题修复**：用户初版每次切片都调用 `strings.Fields()`，改为一开始调用一次，后续用索引切片。

### Prompt 模板

**文件**：`internal/pkg/prompt/templates.go` ✅ Agent 编写框架，用户优化 `fmt.Fprintf` 实现

两种模板：
- `ExtractSystemPrompt` — 知识提取模式
- `CompareSystemPrompt` — 跨论文对比模式
- `BuildUserPrompt()` — 渲染检索结果 + 问题

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

### LLM Service

**文件**：`internal/service/llm_service.go` ✅ 用户要求直接编写（非核心逻辑）

接口抽象：
```
LLMClient
    ├── MockLLMClient（测试用）
    └── QwenLLMClient（OpenAI 兼容格式）
```

### Chat Service

**文件**：`internal/service/chat_service.go` ✅ 用户要求详细指导

核心流程 8 步：
1. 意图识别 → extract / compare
2. 问题向量化 → Embedding API
3. 向量检索 → Top-K + user_id 过滤
4. 查询历史 → 追问时验证权限
5. 构建 Prompt → 选择模板 + 渲染
6. 调用 LLM → 发送 messages 数组
7. 保存记录 → conversations + messages
8. 返回结果 → answer + references

**安全修复**：用户主动发现追问时无权限校验，Agent 添加 `FindByIDAndUserID` 验证。

### Chat Handler

**文件**：`internal/handler/chat.go` ✅ 用户要求直接编写（参考 paper handler）

初始实现 `POST /api/v1/chat`，后续扩展三个对话管理接口。

**文件更新**：`internal/app/container.go` — 注入 ChatService + ChatHandler

### 依赖容器重构

**文件**：`internal/app/container.go` ✅ 我自己手写（参考 Agent 建议）

新增初始化逻辑：
- `initRepositories()` — 新增 ChunkRepository、ConversationRepository、VectorRepository
- `initServices()` — 根据 `EmbeddingType` 切换 mock/qwen 实现，新增 LLM/Chat Service

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
| `LLM_TYPE` | qwen | mock / qwen |
| `LLM_MODEL` | qwen3.5-plus | LLM 模型 |
| `RETRIEVAL_TOP_K_EXTRACT` | 5 | 提取模式检索数 |
| `RETRIEVAL_TOP_K_COMPARE` | 10 | 对比模式检索数 |

**文件**：`.env.example` ✅ Agent 编写

**注意**：LLM 和 Embedding 复用 `ALIYUN_API_KEY`。

### main.go 简化

**文件**：`cmd/server/main.go` ✅ 我自己手写

简化为 15 行：加载配置 → 创建 app → 启动服务。

### 集成到 PaperService

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

## 今日算法题



## 今日总结

### 收获

- pgvector 向量检索用原生 SQL，`ORDER BY embedding <=> $1` 是距离升序
- 用户数据隔离：检索、追问、删除都要验证 `user_id`
- LLM 用 OpenAI 兼容格式比原生 DashScope 更简洁
- 对话列表排序依赖 `updated_at`，追加消息时必须更新
- 两级切片策略保持语义完整性
- errgroup 实现并发控制，任一失败整体取消

### 实现功能列表

| 功能 | 状态 |
|------|------|
| 论文切片与向量化流水线 | ✅ 已验证 |
| Mock Embedding 测试 | ✅ 已验证 |
| Qwen Embedding API 集成 | ✅ 已验证（向量维度 1024） |
| `POST /api/v1/chat` | ✅ 已验证（30s 响应正常） |
| `GET /api/v1/conversations` | ✅ 已验证 |
| `GET /api/v1/conversations/:id/messages` | ✅ 已验证 |
| `DELETE /api/v1/conversations/:id` | ✅ 已验证 |
| 用户隔离（向量检索） | ✅ 用户主动发现 |
| 对话权限校验（追问） | ✅ 用户主动发现 |
| 对话权限校验（删除） | ✅ 用户主动发现 |
| updated_at 时间更新 | ✅ 用户主动发现 |

---

## 后续计划

- Step 6：中间件完善（CORS、限流、日志）
- Redis 缓存热门问答（后续优化）
- Step 7：前端开发
- Step 8：容器化部署