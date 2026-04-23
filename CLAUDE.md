# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

PaperMind is a paper knowledge retrieval and Q&A system for research scenarios. It supports uploading papers, cross-paper Q&A, and cross-paper comparison. Built with Go (Gin + GORM) backend and React frontend, using PostgreSQL with pgvector for vector search and Redis for JWT whitelist.

## Commands

### Backend (Go)
```bash
# Run server
go run cmd/server/main.go

# Run all tests
go test ./...

# Run specific test
go test ./internal/repository/... -v
go test ./internal/service/... -v
```

### Frontend (React)
```bash
cd web

# Development server
npm run dev

# Build for production
npm run build

# Lint
npm run lint
```

### Docker
```bash
# Start dependencies (PostgreSQL + Redis)
docker-compose up -d postgres redis
```

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        Client (React)                       │
│         论文管理 / 问答对话 / 跨论文对比 / 用户认证          │
└──────────────────────────┬──────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────┐
│                    Go Backend (Gin + GORM)                  │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────┐    │
│  │   Auth   │ │  Paper   │ │   Chat   │ │   Pipeline   │    │
│  │ JWT+白名单 │ │ 上传/解析 │ │ 多轮问答  │ │ 切片/向量化   │    │
│  └────┬─────┘ └────┬─────┘ └────┬─────┘ └──────┬───────┘    │
└───────┼────────────┼────────────┼──────────────┼────────────┘
        │            │            │              │
   ┌────▼────┐  ┌────▼────────────▼─────┐  ┌────▼─────────┐
   │  Redis  │  │  PostgreSQL+pgvector  │  │   LLM API    │
   │ JWT白名单 │  │   papers / chunks    │  │  Embedding + │
   │         │  │   conversations       │  │  Chat 接口   │
   └─────────┘  └───────────────────────┘  └──────────────┘
```

### Backend Structure (`internal/`)
- **config/**: Environment variable loading
- **app/**: Application initialization and dependency injection container
- **model/**: GORM models (User, Paper, Chunk, Conversation, Message)
- **dto/**: Request/Response DTOs for API layer
- **repository/**: Database access layer (includes `vector_repo.go` for pgvector operations)
- **service/**: Business logic layer
- **handler/**: HTTP handlers (Gin)
- **middleware/**: JWT authentication middleware
- **pkg/**: Shared utilities:
  - `chunker/`: Two-level chunking for papers
  - `extractor/`: PDF/Markdown extraction
  - `parser/`: Section parsing
  - `prompt/`: LLM prompt templates

### Frontend Structure (`web/src/`)
- **stores/**: Zustand stores for state management (`authStore.ts`, `paperStore.ts`, `conversationStore.ts`)
- **api/**: API client (`client.ts`) and endpoint modules
- **hooks/**: Custom React hooks
- **pages/**: Route components
- **components/**: Reusable UI components

## Key Patterns

### Authentication Flow
- JWT stored in Redis whitelist on login (key = token, value = userID)
- Middleware validates JWT signature AND checks Redis existence
- Logout removes token from Redis (immediate invalidation)
- Frontend handles 401 responses by clearing auth state and redirecting to `/login`

### Paper Processing Pipeline
Paper upload triggers async goroutine pipeline:
```
Upload → PDF/MD extraction → Section parsing → Chunking → Embedding → DB storage
```
Status tracked in `paper.status`: pending → extracting → parsing → chunking → embedding → completed/failed

### Vector Search
- Uses pgvector cosine distance (`<=>` operator)
- Two modes: extraction (single paper, top-k=5) vs comparison (multi-paper, top-k=10)
- All queries enforce `user_id` filtering for data isolation

### API Error Handling
- Backend returns 401 for invalid/expired tokens
- Frontend `apiFetch` in `client.ts` intercepts 401 and triggers logout + redirect

## Environment Variables

Required in `.env`:
- `DB_*`: PostgreSQL connection
- `REDIS_*`: Redis connection
- `ALIYUN_API_KEY`: LLM API key
- `EMBEDDING_MODEL`: Default `qwen3-vl-embedding`
- `LLM_MODEL`: Default `qwen3.5-plus`
- `JWT_SECRET`: JWT signing secret
