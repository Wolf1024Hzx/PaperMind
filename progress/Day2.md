# Day 2 进度记录

## 今日目标

- 学习 Gin、GORM 的基本用法
- 对比 Spring Boot 理解 Go Web 项目架构
- 掌握新增一组功能接口的完整流程

## 今日学习内容

### 1. 项目架构对比（Gin vs Spring Boot）

| Go (Gin/GORM) | Spring Boot |
|---------------|-------------|
| `model/` | `@Entity` (数据库实体) |
| `repository/` | `@Mapper` (MyBatis) 或 `@Repository` (JPA) |
| `service/` | `@Service` |
| `handler/` | `@Controller` / `@RestController` |
| `middleware/` | `Interceptor` / `Filter` |

**关键理解：**

- `model` 是 Entity（数据库实体），不是 DTO
- DTO 在 Go 项目通常放在 `dto/`、`request/`、`response/` 目录
- `repository` 类似 MyBatis 的 Mapper，负责数据访问

### 2. Struct Tag（结构体标签）

Go 使用 struct tag 实现类似 Java 注解的功能：

```go
type User struct {
    Password  string  `gorm:"column:password" json:"-"`
}
```

| Tag | 含义 |
|-----|------|
| `gorm:"column:password"` | 数据库列名映射（类似 `@Column(name = "password")`） |
| `json:"-"` | JSON 序列化时忽略此字段（类似 `@JsonIgnore`） |
| `json:"username"` | JSON 中字段名为 `username`（类似 `@JsonProperty("username")`） |

**`json:"-"` 的安全意义：** 密码字段永远不应出现在 API 响应中，用 `-` 排除。

**json tag 的双向作用：**

- 响应（序列化）：结构体 → JSON
- 请求（反序列化）：JSON → 结构体（`c.ShouldBindJSON`）

### 3. 结构体嵌入（Embedding）

```go
type TokenClaims struct {
    UserID   string `json:"sub"`
    Username string `json:"username"`
    jwt.RegisteredClaims  // 嵌入
}
```

嵌入后 `TokenClaims` 直接拥有 `jwt.RegisteredClaims` 的所有字段：

```go
claims.Subject      // 直接访问，不用 claims.RegisteredClaims.Subject
claims.ExpiresAt    // 同上
```

对比 Java：

| Go 嵌入 | Java 继承 |
|---------|-----------|
| 组合，字段提升 | 继承，字段继承 |
| `struct` 嵌入 `struct` | `class extends class` |

### 4. Gin Handler 响应方式

Gin 不像 Spring Boot 那样 `return ResponseEntity`，而是写入 `gin.Context`：

```go
func (h *AuthHandler) Login(c *gin.Context) {
    // 1. 绑定请求
    var request loginRequest
    if err := c.ShouldBindJSON(&request); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"message": "请求体格式错误"})
        return  // 必须手动 return
    }

    // 2. 调用 service
    result, err := h.authService.Login(ctx, input)
    if err != nil {
        h.handleServiceError(c, err)
        return  // 必须手动 return
    }

    // 3. 成功响应
    c.JSON(http.StatusOK, result)
}
```

| 特性 | Spring Boot | Gin |
|------|-------------|-----|
| 响应方式 | `return ResponseEntity` | `c.JSON(statusCode, data)` |
| 错误处理 | `@ExceptionHandler` 全局捕获 | 手动 `if err != nil` |
| 中断流程 | 异常自动中断 | 必须手动 `return` |

**gin.Context 包含：**

- 请求信息：`c.ShouldBindJSON()`, `c.Query()`, `c.Param()`
- 响应写入：`c.JSON()`, `c.Status()`
- 中间件数据：`c.Set()`, `c.GetString()`

### 5. 路由注册流程

Gin 的路由注册是手动的，不像 Spring Boot 自动扫描：

```go
// main.go
healthHandler := handler.NewHealthHandler()
authHandler := handler.NewAuthHandler(authService)

router := gin.Default()
healthHandler.RegisterRoutes(router)

api := router.Group("/api/v1")
authHandler.RegisterRoutes(api)

// handler/auth.go
func (h *AuthHandler) RegisterRoutes(router *gin.RouterGroup) {
    authGroup := router.Group("/auth")
    authGroup.POST("/register", h.Register)
    authGroup.POST("/login", h.Login)

    userGroup := router.Group("/users")
    userGroup.Use(middleware.RequireAuth(h.authService))
    userGroup.GET("/me", h.GetCurrentUser)
}
```

| 框架 | 路由注册方式 |
|------|-------------|
| Spring Boot | 自动扫描 `@Controller`、`@RequestMapping`（反射 + 注解） |
| Express.js | 手动 `app.get('/path', handler)` |
| Gin | 手动 `RegisterRoutes()`，内部调用 `group.POST()` |

### 6. 新增一组功能接口的完整流程

假设新增 `paper`（论文）相关接口：

```
步骤1: 数据库          → 创建表 papers
步骤2: internal/model/ → 创建 paper.go (Entity)
步骤3: internal/repository/ → 创建 paper_repo.go (数据访问层)
步骤4: internal/service/ → 创建 paper_service.go (业务逻辑层)
步骤5: internal/handler/ → 创建 paper.go (API处理层)
步骤6: cmd/server/main.go → 组装依赖 + 注册路由
```

代码骨架：

```go
// 1. model/paper.go
type Paper struct {
    ID        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
    Title     string    `gorm:"column:title" json:"title"`
    AuthorID  uuid.UUID `gorm:"column:author_id" json:"authorId"`
}

// 2. repository/paper_repo.go
type PaperRepository struct { db *gorm.DB }
func (r *PaperRepository) Create(ctx context.Context, paper *model.Paper) error
func (r *PaperRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Paper, error)

// 3. service/paper_service.go
type PaperService struct { paperRepo *repository.PaperRepository }
func (s *PaperService) CreatePaper(ctx context.Context, input CreatePaperInput) (*PaperResult, error)

// 4. handler/paper.go
type PaperHandler struct { paperService *service.PaperService }
func (h *PaperHandler) RegisterRoutes(router *gin.RouterGroup) {
    papers := router.Group("/papers")
    papers.Use(middleware.RequireAuth(h.authService))  // JWT 验证
    papers.POST("", h.Create)
    papers.GET("/:id", h.GetByID)
}

// 5. main.go
paperRepo := repository.NewPaperRepository(db)
paperService := service.NewPaperService(paperRepo)
paperHandler := handler.NewPaperHandler(paperService)
paperHandler.RegisterRoutes(api)
```

### 7. 依赖注入对比

| Go (手动) | Spring Boot (自动) |
|-----------|-------------------|
| `main.go` 手动组装 | `@Autowired` 自动注入 |
| 显式、无魔法 | 反射 + 注解 |

```go
// Go - 手动依赖注入
userRepo := repository.NewUserRepository(db)
authService := service.NewAuthService(userRepo, cfg.JWTSecret, cfg.JWTTTL)
authHandler := handler.NewAuthHandler(authService)
authHandler.RegisterRoutes(api)
```

## 今日总结

Day 2 主要学习 Gin、GORM 与 Spring Boot 的对比理解。核心收获：

1. Go 项目架构与 Spring Boot 分层架构思想一致，只是没有依赖注入框架
2. struct tag 实现类似 Java 注解的功能
3. 结构体嵌入 = 组合（字段提升，类似继承）
4. Gin Handler 通过写入 `gin.Context` 返回响应，必须手动 `return`
5. 路由注册是手动的，封装在 `RegisterRoutes` 方法中
6. 掌握了新增一组接口的完整流程

Go 是"手动挡"，Spring Boot 是"自动挡"。Go 需要手动处理每一步，但更透明、更可控。

## 后续计划

- 继续学习项目其他模块
- 开始实现论文相关功能