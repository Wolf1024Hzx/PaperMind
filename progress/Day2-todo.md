# Day 2 待办事项

## 今日目标

- 完成 Day 1 遗留：users 接口手动联调验证
- 实现论文上传接口
- 实现 PDF 文本提取
- 实现论文元数据提取
- 实现论文章节结构解析（Section 识别）

---

## 任务清单

### 第一部分：users 接口联调（优先完成）

|序号|任务|验证方式|状态|
|---|---|---|---|
|1|启动服务，验证 `/healthz` 返回 200|curl 或浏览器访问|⬜|
|2|测试注册接口 `POST /api/v1/auth/register`|curl 发送请求，检查数据库记录|⬜|
|3|测试登录接口 `POST /api/v1/auth/login`|curl 发送请求，验证 JWT 返回|⬜|
|4|测试获取当前用户 `GET /api/v1/users/me`|带 JWT 的 curl 请求|⬜|
|5|测试更新当前用户 `PUT /api/v1/users/me`|带 JWT 的 curl 请求|⬜|
|6|测试删除当前用户 `DELETE /api/v1/users/me`|带 JWT 的 curl 请求|⬜|

### 第二部分：论文上传接口

|序号|任务|说明|状态|
|---|---|---|---|
|7|设计并创建 `papers` 表模型|参考设计文档 4.2 节|⬜|
|8|创建 `internal/model/paper.go`|定义 Paper struct|⬜|
|9|创建 `internal/repository/paper_repo.go`|论文 CRUD 操作|⬜|
|10|创建 `internal/service/paper_service.go`|论文上传业务逻辑|⬜|
|11|创建 `internal/handler/paper.go`|HTTP Handler|⬜|
|12|实现 `POST /api/v1/papers/upload` 接口|multipart/form-data 上传 PDF|⬜|

**papers 表核心字段**：

```
id, user_id, filename, file_size, file_hash,
title, authors, year, venue, abstract,
chunk_count, status, created_at, updated_at
```

**status 状态机**：`pending → extracting → chunking → embedding → completed → failed`

### 第三部分：PDF 文本提取

|序号|任务|说明|状态|
|---|---|---|---|
|13|引入 PDF 解析库|`github.com/ledongthuc/pdf`|⬜|
|14|创建 `internal/pkg/extractor/pdf.go`|PDF 文本提取封装|⬜|
|15|实现按页提取文本逻辑|返回每页文本内容 + 页码|⬜|
|16|实现 SHA-256 文件哈希计算|用于论文去重|⬜|

### 第四部分：论文元数据提取

|序号|任务|说明|状态|
|---|---|---|---|
|17|从 PDF metadata 提取 title/author|优先级最高|⬜|
|18|正则匹配首页标题（降级方案）|处理 metadata 缺失情况|⬜|
|19|前端预留手动填写入口（暂不实现）|最终降级|⬜|

### 第五部分：论文章节结构解析（核心差异化）

|序号|任务|说明|状态|
|---|---|---|---|
|20|创建 `internal/pkg/parser/section_parser.go`|章节识别器|⬜|
|21|定义 SectionTypes 映射|abstract/intro/method/experiment/conclusion 等|⬜|
|22|实现正则匹配章节标题|识别 "1. Introduction" 等格式|⬜|
|23|输出章节列表（Type + Title + Content + Page）|日志验证|⬜|

---

## 验证目标

完成今日任务后，应能：

1. ✅ 所有 users 接口可正常调用（注册 → 登录 → 获取/更新/删除用户）
2. ✅ 上传一篇 PDF 论文，返回 paper_id 和初始状态
3. ✅ 在日志中看到论文文本提取结果（每页内容）
4. ✅ 在日志中看到识别出的章节列表（如 "Introduction", "Method", "Experiment"）

---

## 技术要点备忘

### PDF 解析库选择

```go
import "github.com/ledongthuc/pdf"

// 基本用法
f, r, err := pdf.Open("paper.pdf")
// 逐页提取
for i := 1; i <= r.NumPage(); i++ {
    p := r.Page(i)
    text, _ := p.GetPlainText()
}
```

### Section 识别正则参考

```go
// 匹配格式: "1. Introduction", "3 Methodology", "IV. EXPERIMENTS"
headingPattern := regexp.MustCompile(
    `^(?:\d+\.?\s+|[IVX]+\.?\s+)?` +
    `([A-Z][A-Za-z\s:]+)$`,
)
```

### 标准章节类型

```go
var SectionTypes = map[string][]string{
    "abstract":     {"abstract"},
    "introduction": {"introduction", "intro"},
    "related_work": {"related work", "background", "literature review"},
    "method":       {"method", "methodology", "approach", "proposed method"},
    "experiment":   {"experiment", "evaluation", "results"},
    "conclusion":   {"conclusion", "summary", "future work"},
}
```

---

## 文件创建清单

今日需创建的文件：

```
internal/model/paper.go
internal/repository/paper_repo.go
internal/service/paper_service.go
internal/handler/paper.go
internal/pkg/extractor/pdf.go
internal/pkg/parser/section_parser.go
```

---

## 预计耗时

|部分|预计时间|
|---|---|
|users 接口联调|1 小时|
|论文上传接口|2 小时|
|PDF 提取|1.5 小时|
|章节结构解析|2 小时|
|联调验证|1 小时|
|**合计**|~7.5 小时|

---

## 可能遇到的问题

1. **PDF 解析库编译问题**：`ledongthuc/pdf` 可能需要特定环境，备选 `pdfcpu`
2. **章节识别准确率**：非标准格式论文可能识别失败，降级为 `other` 类型
3. **文件存储路径**：需确定 PDF 文件本地存储位置（如 `uploads/papers/`）

---

_创建日期：2026-04-04 | 计划周期：Day 2_