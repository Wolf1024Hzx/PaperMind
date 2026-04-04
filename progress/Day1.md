# Day 1 进度记录

## 今日目标

- 完成本地开发环境打通
- 配置 PostgreSQL + pgvector + Redis
- 初始化 Go 后端骨架
- 引入 Gin + GORM
- 实现 users 表相关的认证与基础接口

## 今日完成内容

### 1. 本地环境与数据库

- 使用 WSL 内 Docker 运行 PostgreSQL 与 Redis
- PostgreSQL 已切换为支持 pgvector 的镜像
- 已成功执行以下扩展初始化：

```sql
CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
```

- 已创建 `users` 表
- Windows 上的 Go 程序已能成功连接 PostgreSQL 与 Redis

### 2. 项目骨架初始化

- 初始化 Go 项目模块：`wolfden.website/papermind`
- 增加后端核心依赖：
  - Gin
  - GORM
  - PostgreSQL Driver
  - JWT
  - bcrypt
  - Redis Client
  - UUID

### 3. 已实现的后端结构

- `cmd/server/main.go`
  - 程序入口
  - 初始化配置
  - 连接 PostgreSQL
  - 连接 Redis
  - 注册 HTTP 路由

- `internal/config/config.go`
  - 统一读取环境变量
  - 生成 PostgreSQL DSN

- `internal/model/user.go`
  - `users` 表模型映射

- `internal/repository/user_repo.go`
  - 用户数据访问层

- `internal/service/auth_service.go`
  - 注册
  - 登录
  - JWT 签发
  - JWT 解析
  - 获取当前用户
  - 更新当前用户
  - 删除当前用户

- `internal/middleware/jwt.go`
  - JWT 鉴权中间件

- `internal/handler/health.go`
  - 健康检查接口

- `internal/handler/auth.go`
  - 注册接口
  - 登录接口
  - 当前用户查询/更新/删除接口

## 当前可用接口

- `GET /healthz`
- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- `GET /api/v1/users/me`
- `PUT /api/v1/users/me`
- `DELETE /api/v1/users/me`

## 当前状态

- 服务已成功启动
- 已验证 `/healthz` 可正常返回 200
- Gin 路由注册正常
- PostgreSQL 与 Redis 初始化连接成功
- user 相关接口代码已完成，尚未逐条手动联调

## 今日验证情况

- 已执行 `go mod tidy`
- 已执行 `gofmt -w .\cmd .\internal`
- 已执行 `go test ./...`
- 已执行 `go vet ./...`
- 启动 `go run .\cmd\server` 成功

## 遇到的问题与处理

### pgvector 扩展缺失

问题：

- 原 PostgreSQL 容器镜像不包含 pgvector
- 执行 `CREATE EXTENSION vector` 报错

处理：

- 删除旧数据目录 `pg_data`
- 切换为支持 pgvector 的 PostgreSQL 镜像
- 重建容器后重新执行扩展初始化 SQL

结果：

- `vector` 与 `uuid-ossp` 扩展均已启用

## 后续计划

### 优先完成

- 手动联调 users 相关接口：
  - 注册
  - 登录
  - 获取当前用户
  - 更新当前用户
  - 删除当前用户

### 联调通过后进入 Step 2

- 开始实现论文上传接口
- 设计并落地 `papers` 表对应的 Go 代码
- 准备 PDF 文件上传与保存逻辑

## 总结

Day 1 已完成 Step 1 的基础部分：本地环境、数据库扩展、服务骨架、认证主链路代码均已落地。当前项目已经从“设计文档阶段”进入“可运行的后端开发阶段”，接下来可以继续推进 users 接口联调，并进入论文上传模块开发。
