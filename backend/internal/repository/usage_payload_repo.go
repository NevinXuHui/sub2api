package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type usagePayloadRepository struct {
	sql sqlExecutor
}

// NewUsagePayloadRepository 创建 payload 仓储
func NewUsagePayloadRepository(sqlDB *sql.DB) service.UsagePayloadRepository {
	return &usagePayloadRepository{sql: sqlDB}
}

// InsertPayload 插入请求/响应正文，ON CONFLICT DO NOTHING 去重
func (r *usagePayloadRepository) InsertPayload(ctx context.Context, payload *service.UsageLogPayload) error {
	if payload == nil {
		return nil
	}

	query := `
		INSERT INTO usage_log_payloads (
			usage_log_id, request_id, api_key_id, user_id,
			request_body, response_body,
			request_truncated, response_truncated,
			request_bytes, response_bytes,
			created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
		ON CONFLICT (request_id, api_key_id) DO NOTHING
	`

	_, err := r.sql.ExecContext(ctx, query,
		payload.UsageLogID,
		payload.RequestID,
		payload.APIKeyID,
		payload.UserID,
		payload.RequestBody,
		payload.ResponseBody,
		payload.RequestTruncated,
		payload.ResponseTruncated,
		payload.RequestBytes,
		payload.ResponseBytes,
	)

	return err
}

// GetPayloadByUsageLogID 按 usage_log_id 查询 payload
func (r *usagePayloadRepository) GetPayloadByUsageLogID(ctx context.Context, usageLogID int64) (*service.UsageLogPayload, error) {
	query := `
		SELECT
			usage_log_id, request_id, api_key_id, user_id,
			request_body, response_body,
			request_truncated, response_truncated,
			request_bytes, response_bytes,
			created_at
		FROM usage_log_payloads
		WHERE usage_log_id = $1
		LIMIT 1
	`

	var payload service.UsageLogPayload
	err := r.sql.QueryRowContext(ctx, query, usageLogID).Scan(
		&payload.UsageLogID,
		&payload.RequestID,
		&payload.APIKeyID,
		&payload.UserID,
		&payload.RequestBody,
		&payload.ResponseBody,
		&payload.RequestTruncated,
		&payload.ResponseTruncated,
		&payload.RequestBytes,
		&payload.ResponseBytes,
		&payload.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // 未采集返回 nil，不是错误
	}
	if err != nil {
		return nil, fmt.Errorf("query usage_log_payloads: %w", err)
	}

	return &payload, nil
}

// DeletePayloadsBefore 删除指定时间之前的 payloads（配合 usage_logs 清理）
func (r *usagePayloadRepository) DeletePayloadsBefore(ctx context.Context, before int64) (int64, error) {
	query := `DELETE FROM usage_log_payloads WHERE created_at < to_timestamp($1)`
	result, err := r.sql.ExecContext(ctx, query, before)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
