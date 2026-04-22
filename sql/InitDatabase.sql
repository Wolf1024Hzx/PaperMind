CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

SELECT extname FROM pg_extension;

-- users 表
CREATE TABLE users (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username    VARCHAR(64)  NOT NULL UNIQUE,
    email       VARCHAR(128) NOT NULL UNIQUE,
    password    VARCHAR(256) NOT NULL,
    created_at  TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP    NOT NULL DEFAULT NOW()
);

-- papers 表（论文元数据 + 文件信息 + 处理状态）
CREATE TABLE papers (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id      UUID         NOT NULL REFERENCES users(id),

    -- 文件信息
    filename     VARCHAR(256) NOT NULL,          -- 原始文件名
    file_size    BIGINT       NOT NULL,          -- 字节数
    file_hash    VARCHAR(64)  NOT NULL,          -- SHA-256，用于去重

    -- 论文元数据（后续功能的区分度关键）
    title        VARCHAR(512),                   -- 论文标题（从 PDF 提取或用户手动填写）
    authors      VARCHAR(512),                   -- 作者列表，逗号分隔
    year         INT,                            -- 发表年份
    venue        VARCHAR(256),                   -- 发表会议/期刊（如 NeurIPS 2024）
    abstract     TEXT,                           -- 摘要（从论文提取）

    -- 处理状态
    chunk_count  INT          NOT NULL DEFAULT 0,
    status       VARCHAR(32)  NOT NULL DEFAULT 'pending',
    -- status 枚举: pending → extracting → chunking → embedding → completed → failed

    created_at   TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMP    NOT NULL DEFAULT NOW()
);

-- 索引
CREATE INDEX idx_papers_user_id ON papers(user_id);
CREATE INDEX idx_papers_file_hash ON papers(file_hash);    -- 全局按文件哈希查询（如管理功能）
CREATE INDEX idx_papers_year ON papers(year);               -- 支持按年份筛选检索范围

-- 复合唯一索引：确保每个用户的文件去重，同时优化查询性能
CREATE UNIQUE INDEX idx_papers_user_file_hash ON papers(user_id, file_hash);

-- chunks 表（论文切片 + 向量）
CREATE TABLE chunks (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    paper_id      UUID          NOT NULL REFERENCES papers(id) ON DELETE CASCADE,
    chunk_index   INT           NOT NULL,
    content       TEXT          NOT NULL,
    token_count   INT           NOT NULL,
    embedding     vector(1024)  NOT NULL,

    -- 论文结构元数据
    section_type  VARCHAR(64),
    section_title VARCHAR(256),
    page_number   INT,

    created_at    TIMESTAMP     NOT NULL DEFAULT NOW()
);

-- 向量检索索引（HNSW，余弦距离）
CREATE INDEX idx_chunks_embedding ON chunks
    USING hnsw (embedding vector_cosine_ops);

-- 业务索引
CREATE INDEX idx_chunks_paper_id ON chunks(paper_id);
CREATE INDEX idx_chunks_section_type ON chunks(section_type);

-- conversations 表
CREATE TABLE conversations (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID         NOT NULL REFERENCES users(id),
    title       VARCHAR(256),
    mode        VARCHAR(32)  NOT NULL DEFAULT 'extract',
    created_at  TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP    NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_conversations_user_id ON conversations(user_id);

-- messages 表
CREATE TABLE messages (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    conversation_id UUID        NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    role            VARCHAR(16) NOT NULL,
    content         TEXT        NOT NULL,
    references_data JSONB,
    token_usage     JSONB,
    created_at      TIMESTAMP   NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_messages_conversation_id ON messages(conversation_id);

