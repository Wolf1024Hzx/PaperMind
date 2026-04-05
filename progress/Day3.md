# Day 3 进度记录

> 日期：2026-04-05 | 时间：9:30 - 14:30

## 今日学习内容

### 1. GORM 表名推断机制

| 方式 | 说明 |
|------|------|
| 默认推断 | Struct 名转 snake_case，`Paper` → `papers` |
| TableName() 方法 | 显式指定，优先级更高 |

GORM 通过传入的 struct 类型推断表名，不需要在 Repository 里显式指定。

### 2. GORM 查询方法

| 方法 | 用途 | 返回 |
|------|------|------|
| `Create(&entity)` | 创建记录 | error |
| `First(&entity, condition)` | 查单个 | 写入 entity |
| `Find(&entities, condition)` | 查多个 | 写入 slice |
| `Model(&Entity{}).Where().Updates(map)` | 更新 | error |
| `Delete(&Entity{}, condition)` | 删除 | error |

关键点：`Find` 查多个时必须传指针 `&papers`，否则无法写入结果。

### 3. GORM 条件列名

Where 条件中的列名使用**数据库真实列名**（snake_case）：

```go
Where("user_id = ?", userID)   // ✅ 推荐
Where("FileHash = ?", hash)    // ✅ 可行（GORM 自动映射）
Where("fileHash = ?", hash)    // ❌ 错误
```

### 4. Go 命名惯例

缩写词（ID、URL、HTTP）应保持大小写一致：`userID` 而不是 `userId`。

### 5. Gin multipart/form-data 文件上传

```go
file, err := c.FormFile("file")        // 获取上传文件
fileContent, err := file.Open()        // 打开文件
fileData := make([]byte, file.Size)    // 读取内容
```

### 6. SHA-256 文件哈希计算

```go
hash := sha256.Sum256(fileData)
fileHash := hex.EncodeToString(hash[:])  // 转为十六进制字符串
```

## 今日工程实现内容

### papers 表设计

根据项目设计文档创建 papers 表，包含：
- 文件信息：`filename`、`file_size`、`file_hash`（去重）
- 论文元数据：`title`、`authors`、`year`、`venue`、`abstract`
- 处理状态：`status`、`chunk_count`

索引：`user_id`、`file_hash`、`year`（支持年份过滤检索）

### model/paper.go

参考 model/user.go 实现，关键点：
- `FileSize` 类型为 `int64`（对应 SQL BIGINT）
- `Year` 用 `*int` 表示可空字段
- 实现 `TableName()` 方法返回 `"papers"`

### repository/paper_repo.go

和 agent 持续对话，参考 user_repo.go 手动实现，包含方法：

| 方法 | 用途 |
|------|------|
| `Create` | 创建论文记录 |
| `FindByID` | 根据 ID 查单个 |
| `FindByUserID` | 查用户的所有论文 |
| `FindByFileHash` | 根据 hash 查重 |
| `UpdatePaperInfo` | 更新论文信息 |
| `DeletePaper` | 删除论文 |

### service/paper_service.go

同上，逐函数与 agent 对话实现，包含方法：

| 方法 | 用途 |
|------|------|
| `UploadPaper` | 上传论文（去重 + 创建记录 + 保存文件） |
| `ListByUser` | 获取用户论文列表 |
| `GetByID` | 获取论文详情（验证所有权） |
| `Delete` | 删除论文（验证所有权 + 删库 + 删文件） |

新增 `service/errors.go` 集中管理错误常量。

### handler/paper.go

Handler 不复杂，直接让 agent 写好，包含接口：

| 方法 | 路径 | 功能 |
|------|------|------|
| `UploadPaper` | POST /api/v1/papers | 上传 PDF |
| `ListPapers` | GET /api/v1/papers | 论文列表 |
| `GetPaperByID` | GET /api/v1/papers/:id | 论文详情 |
| `DeletePaper` | DELETE /api/v1/papers/:id | 删除论文 |

### main.go 注册 PaperHandler

- 添加 `os.MkdirAll` 创建上传目录
- 注册 PaperHandler 到路由

### config/config.go 新增 UploadDir

配置项：`UPLOAD_DIR` 环境变量，默认值 `./uploads/papers`

### 接口测试

所有接口已测试通过，上传接口返回示例：

```json
{
  "id": "4da54006-e9d8-4978-b0a1-913a1d7e6797",
  "filename": "ProX_Net.pdf",
  "fileSize": 13172862,
  "status": "pending",
  "chunkCount": 0
}
```

## 今日算法题

今天打了两道题，一道简单一道中等，都是没看答案 ac

[移动零](https://leetcode.cn/problems/move-zeroes/description/?envType=study-plan-v2&envId=top-100-liked)

[无重复字符的最长子串](https://leetcode.cn/problems/longest-substring-without-repeating-characters/description/?envType=study-plan-v2&envId=top-100-liked)