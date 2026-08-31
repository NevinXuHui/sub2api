package service

import (
	"context"
	"time"
)

// UsageLogPayload 使用记录请求/响应正文（sidecar）
type UsageLogPayload struct {
	UsageLogID int64
	RequestID  string
	APIKeyID   int64
	UserID     int64

	RequestBody  string
	ResponseBody string

	RequestTruncated  bool
	ResponseTruncated bool

	RequestBytes  int
	ResponseBytes int

	CreatedAt time.Time
}

// UsagePayloadRepository payload 仓储接口
type UsagePayloadRepository interface {
	InsertPayload(ctx context.Context, payload *UsageLogPayload) error
	GetPayloadByUsageLogID(ctx context.Context, usageLogID int64) (*UsageLogPayload, error)
	DeletePayloadsBefore(ctx context.Context, before int64) (int64, error)
}
