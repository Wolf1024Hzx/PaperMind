# Day 3 进度记录

> 日期：2026-04-05 | 时间：9:30 - 16:15

## 今日学习内容

### 1. GORM 表名推断机制

| 方式 | 说明 |
|------|------|
| 默认推断 | Struct 名转 snake_case，`Paper` → `papers` |
| TableName() 方法 | 显式指定，优先级更高 |

### 2. GORM 查询方法

| 方法 | 用途 | 返回 |
|------|------|------|
| `Create(&entity)` | 创建记录 | error |
| `First(&entity, cond)` | 查单个 | 写入 entity |
| `Find(&slice, cond)` | 查多个 | 写入 slice |
| `Model().Where().Updates()` | 更新 | error |
| `Delete(&Entity{}, cond)` | 删除 | error |

关键：`Find` 查多个时必须传指针 `&papers`。

### 3. GORM 条件列名

使用**数据库真实列名**（snake_case）：
- ✅ `Where("user_id = ?", userID)`
- ❌ `Where("fileHash = ?", hash)`（大小写不一致）

### 4. Go 命名惯例

缩写词（ID、URL、HTTP）保持大小写一致：`userID` 而不是 `userId`。

### 5. Gin 文件上传

`c.FormFile("file")` 获取上传文件，`file.Open()` 打开读取。

### 6. SHA-256 文件哈希

`sha256.Sum256(fileData)` + `hex.EncodeToString()` 去重。

### 7. Goroutine 异步处理

上传接口立即返回，后台异步处理 PDF：
- 使用 `context.Background()` 防止请求结束导致 ctx 失效
- `defer recover()` 防止 panic 扩散

### 8. PDF 文本提取库（ledongthuc/pdf）

`Open()` 打开 → `NumPage()` 获取页数 → `Page(i).GetTextByRow()` 按行提取文本。

### 9. 论文章节识别

通过正则匹配标题格式：`1. Introduction`、`3.2 Method`、`IV. Experiments`。
章节类型映射：abstract / introduction / related_work / method / experiment / conclusion / other。

## 今日工程实现内容

### papers 表设计

| 字段 | 说明 |
|------|------|
| `filename`、`file_size`、`file_hash` | 文件信息（去重） |
| `title`、`authors`、`year`、`venue`、`abstract` | 论文元数据 |
| `status`、`chunk_count` | 处理状态 |

索引：`user_id`、`file_hash`、`year`。

### 论文 CRUD 实现

**新增文件：**

| 文件 | 说明 |
|------|------|
| `model/paper.go` | Paper 实体，FileSize 为 int64，Year 用 *int |
| `repository/paper_repo.go` | CRUD + UpdateStatus + UpdateMetadataIfEmpty |
| `service/paper_service.go` | 业务逻辑，异步处理 PDF |
| `handler/paper.go` | 上传、列表、详情、删除接口 |
| `config/config.go` | 新增 UploadDir 配置 |

**修改文件：**

| 文件 | 改动 |
|------|------|
| `main.go` | 创建上传目录、注册 PaperHandler |

### PDF 文本提取与章节解析

**新增文件：**

| 文件 | 说明 |
|------|------|
| `internal/pkg/extractor/pdf.go` | PDF 文本提取 + 元数据提取 |
| `internal/pkg/parser/section_parser.go` | 论文章节结构识别 |

**处理流程：**

```
上传 PDF → 保存文件 → 创建记录 → status: pending
                                    ↓
                          [异步 Goroutine]
                                    ↓
              extracting → PDF 文本提取 → parsing → 结构解析
                                    ↓
              completed → 日志输出章节列表
```

### 接口测试

| 接口 | 状态 |
|------|------|
| `POST /api/v1/papers` (上传) | ✅ |
| `GET /api/v1/papers` (列表) | ✅ |
| `GET /api/v1/papers/:id` (详情) | ✅ |
| `DELETE /api/v1/papers/:id` | ✅ |

### Markdown 上传通道（新增）

**背景：** PDF 文本提取质量差是行业难题，添加 Markdown 作为替代方案。

**改动文件：**

| 文件 | 说明 |
|------|------|
| `pkg/common/section_type.go` | 新增，Section 结构体 + 类型常量 + MatchSectionType |
| `parser/section_parser.go` | 改用 common 包 |
| `extractor/markdown.go` | 新增，Markdown/.txt 提取 + 章节识别 |
| `handler/paper.go` | 放宽校验，允许 .md/.txt |
| `service/paper_service.go` | 按扩展名分支处理 |

**处理流程：**

```
上传入口 → 判断扩展名 → PDF: ExtractPDF → ParseSections
                      → Markdown: ExtractMarkdown → []Section（一步完成）
                      → .txt: 按空行切分段落
```

**设计要点：**
- `Section` 结构体移到 common 包避免循环依赖
- Markdown `#` `##` `###` 直接标识章节，识别准确率接近 100%

## 遇到的问题

### 1. PDF 文本提取单词粘连

ledongthuc/pdf 库 `GetTextByRow()` 返回单词无空格，尝试根据标点加空格但效果有限，部分单词被错误拆分。

### 2. 章节标题识别适配困难

| 格式 | 示例 | 状态 |
|------|------|------|
| NeurIPS/ICML | `1 Introduction` | ✅ 可匹配 |
| 部分期刊 | `Introduction`（无编号） | ⚠️ 需白名单 |
| 双栏论文 | 文本顺序混乱 | ❌ 无法处理 |

### 3. 其他问题

| 问题 | 说明 |
|------|------|
| 双栏格式 | 文本提取顺序混乱，左右栏交错 |
| PDF 元数据 | 很多论文 Info 字段为空或不准确 |
| 自定义字体编码 | 提取出乱码或空文本 |

## 今日算法题

今天打了两道题，一道简单一道中等，都是没看答案 ac

[移动零](https://leetcode.cn/problems/move-zeroes/description/?envType=study-plan-v2&envId=top-100-liked)

[无重复字符的最长子串](https://leetcode.cn/problems/longest-substring-without-repeating-characters/description/?envType=study-plan-v2&envId=top-100-liked)

## 今日总结

**收获：**

1. GORM 查询方法：`First` 写入单个实体，`Find` 写入切片，`Updates` 需配合 `Model`
2. Gin 文件上传：`c.FormFile("file")` + `file.Open()` + 读取字节
3. SHA-256 文件哈希去重：`sha256.Sum256()` + `hex.EncodeToString()`
4. Goroutine 异步处理：使用 `context.Background()` 防止请求结束导致 ctx 失效
5. defer/recover 防止 panic 扩散到主进程
6. Go 循环依赖解决：把共享类型（如 Section）移到独立的 common 包

**实现功能：**

- 论文 CRUD（上传、列表、详情、删除）
- PDF 文本提取 + 章节结构解析（异步流水线）
- Markdown/.txt 上传通道（替代 PDF 提取质量差的问题）

**问题与解决：**

- PDF 单词粘连 → 添加 Markdown 上传通道绕过问题
- 循环依赖 → Section 结构体移到 common 包
- 章节类型判断重复 → MatchSectionType 移到 common 包复用

**工程判断：**

PDF 解析质量差是行业已知难题，不是代码 bug。务实方案：添加 Markdown 通道，快速推进核心 RAG Pipeline。面试时诚实说明：系统支持两种输入格式，复杂排版可手动整理成 Markdown。

## 后续计划

- 实现文本切片（Chunking）
- 实现 Embedding 向量化
- 实现向量检索（pgvector）