# Day 2 进度记录

> 日期：2026-04-04 | 时间：16:23 - 22:30

## 今日学习内容

### 1. 项目架构对比（Gin vs Spring Boot）

| Go (Gin/GORM) | Spring Boot |
|---------------|-------------|
| `model/` | `@Entity` (数据库实体) |
| `repository/` | `@Mapper` (MyBatis) |
| `service/` | `@Service` |
| `handler/` | `@Controller` |
| `middleware/` | `Interceptor` / `Filter` |

关键理解：`model` 是 Entity 不是 DTO，`repository` 类似 MyBatis Mapper。

### 2. Struct Tag

Go 使用 struct tag 实现类似 Java 注解的功能：

| Tag | 含义 |
|-----|------|
| `gorm:"column:password"` | 数据库列名映射 |
| `json:"-"` | JSON 序列化时忽略（密码字段安全） |
| `json:"username"` | JSON 字段名映射 |

### 3. Gin Handler 响应方式

| 特性 | Spring Boot | Gin |
|------|-------------|-----|
| 响应方式 | `return ResponseEntity` | `c.JSON(statusCode, data)` |
| 错误处理 | `@ExceptionHandler` | 手动 `if err != nil` |
| 中断流程 | 异常自动中断 | 必须手动 `return` |

### 4. 新增功能接口流程

```
数据库建表 → model/ (Entity) → repository/ (DAO) → service/ (业务) → handler/ (API) → main.go (组装)
```

### 5. Go JSON 解析行为

- 缺省字段使用**零值**（string → `""`, int → `0`, bool → `false`）
- 不会报错，需要手动校验或使用 `binding:"required"` tag

## 今日工程实现内容

### JWT 迁移到 Redis

**背景：** 原 JWT 无状态，无法主动让 token 失效。Redis 已初始化但未使用。

**方案：** 白名单模式
- 签发 token → 存入 Redis（key = token, value = userID）
- 验证 token → 查 Redis → 解析 JWT
- 登出 → 删除 Redis 中的 token

**修改文件：**

| 文件 | 改动 |
|------|------|
| `service/redis_service.go` | 新增，Redis 工具封装，自动带前缀 |
| `service/auth_service.go` | 添加 redisService，createToken 存 Redis，新增 Logout |
| `middleware/jwt.go` | 直接依赖 RedisService，不调用 AuthService |
| `handler/auth.go` | 新增 Logout handler，RegisterRoutes 传入 redisService |
| `cmd/server/main.go` | 创建 RedisService，传入中间件 |

**调用拓扑：**

```
config.JWTSecret
      ↓
  ┌───┴───┐
  ↓       ↓
AuthService  Middleware
  ↓       ↓
RedisService
```

**关键设计：**
- 中间件和 Service 平级，中间件不"越级"调用 Service
- jwtSecret 从 config 直接获取，不经过 AuthService
- TokenClaims 放 service 包避免循环依赖

### Token 自动刷新

中间件验证时，剩余时间 < TTL/2 则自动刷新 TTL。

### Register 不再返回 Token

注册只返回用户信息，需单独登录获取 token。

### 接口测试完成

| 接口 | 状态 |
|------|------|
| `POST /api/v1/auth/register` | ✅ |
| `POST /api/v1/auth/login` | ✅ |
| `POST /api/v1/auth/logout` | ✅ |
| `GET /api/v1/users/me` | ✅ |
| `PUT /api/v1/users/me` | ✅ |
| `DELETE /api/v1/users/me` | ✅ |

验证：登录后 token 存入 Redis，登出后删除，中间件正常工作。

## 今日总结

**收获：**

1. Go 项目架构与 Spring Boot 分层思想一致，无依赖注入框架
2. struct tag 实现类似 Java 注解功能
3. Gin Handler 写入 `gin.Context` 返回响应，必须手动 `return`
4. Go JSON 解析默认宽松，缺省字段使用零值
5. 中间件和 Service 应平级，中间件不应"越级"调用 Service
6. Go 不允许循环依赖，需合理规划类型位置

**实现功能：**

- JWT 迁移到 Redis（白名单模式）
- 登出接口（主动让 token 失效）
- Token 自动刷新（剩余 < TTL/2 时刷新）
- 注册不再自动返回 token
- 完成所有接口联调测试

**问题与解决：**

- 循环依赖：TokenClaims 放 service 包，middleware 引用
- Redis 调试：添加日志排查写入问题

## 后续计划

- 开始实现论文相关功能
- 设计 `papers` 表结构