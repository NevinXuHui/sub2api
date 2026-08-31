package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

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
	if err != nil {
		return fmt.Errorf("insert usage payload: %w", err)
	}
	return nil
}

// GetPayloadByUsageLogID 根据 usage_log_id 查询正文
func (r *usagePayloadRepository) GetPayloadByUsageLogID(ctx context.Context, usageLogID int64) (*service.UsageLogPayload, error) {
	query := `
		SELECT usage_log_id, request_id, api_key_id, user_id,
			request_body, response_body,
			request_truncated, response_truncated,
			request_bytes, response_bytes,
			created_at
		FROM usage_log_payloads
		WHERE usage_log_id = $1
		LIMIT 1
	`

	var payload service.UsageLogPayload
	err := scanSingleRow(ctx, r.sql, query, []any{usageLogID},
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
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query usage payload: %w", err)
	}
	return &payload, nil
}

// DeletePayloadsBefore 按 created_at cutoff（unix 秒）删除 sidecar，避免正文比用量行活得更久
func (r *usagePayloadRepository) DeletePayloadsBefore(ctx context.Context, before int64) (int64, error) {
	cutoff := time.Unix(before, 0).UTC()
	res, err := r.sql.ExecContext(ctx, `DELETE FROM usage_log_payloads WHERE created_at < $1`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("delete usage payloads: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("delete usage payloads rows affected: %w", err)
	}
	return affected, nil
}
