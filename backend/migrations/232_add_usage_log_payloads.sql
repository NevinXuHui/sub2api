-- 232_add_usage_log_payloads.sql
-- 使用记录请求/响应正文侧边车表（不改宽 usage_logs 分区表）

CREATE TABLE IF NOT EXISTS usage_log_payloads (
    usage_log_id BIGINT NOT NULL,
    request_id VARCHAR(255) NOT NULL,
    api_key_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,

    request_body TEXT,
    response_body TEXT,

    request_truncated BOOLEAN NOT NULL DEFAULT FALSE,
    response_truncated BOOLEAN NOT NULL DEFAULT FALSE,

    request_bytes INT NOT NULL DEFAULT 0,
    response_bytes INT NOT NULL DEFAULT 0,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- 去重键对齐 usage_logs (request_id, api_key_id)
    CONSTRAINT usage_log_payloads_unique_request UNIQUE (request_id, api_key_id)
);

-- usage_log_id 索引用于按 ID 查询和清理
CREATE INDEX IF NOT EXISTS idx_usage_log_payloads_usage_log_id
    ON usage_log_payloads(usage_log_id);

-- created_at 索引用于清理任务按时间删除
CREATE INDEX IF NOT EXISTS idx_usage_log_payloads_created_at
    ON usage_log_payloads(created_at);

-- user_id 索引用于用户查询
CREATE INDEX IF NOT EXISTS idx_usage_log_payloads_user_id
    ON usage_log_payloads(user_id);

COMMENT ON TABLE usage_log_payloads IS '使用记录请求/响应正文（sidecar，避免加宽 usage_logs）';
COMMENT ON COLUMN usage_log_payloads.usage_log_id IS '关联 usage_logs.id（无 FK，usage_logs 可能分区）';
COMMENT ON COLUMN usage_log_payloads.request_body IS '请求正文（截断后，base64 图片替换为占位符）';
COMMENT ON COLUMN usage_log_payloads.response_body IS '响应正文（客户端可见，流式为截断后的原始流）';
COMMENT ON COLUMN usage_log_payloads.request_truncated IS '请求是否被截断';
COMMENT ON COLUMN usage_log_payloads.response_truncated IS '响应是否被截断';
COMMENT ON COLUMN usage_log_payloads.request_bytes IS '请求原始字节数（截断前）';
COMMENT ON COLUMN usage_log_payloads.response_bytes IS '响应原始字节数（截断前）';
