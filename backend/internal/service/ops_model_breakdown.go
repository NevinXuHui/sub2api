package service

import "time"

// OpsModelBreakdownItem contains aggregated metrics for a single model.
type OpsModelBreakdownItem struct {
	Model string `json:"model"`

	// Request counts
	RequestCount   int64 `json:"request_count"`
	SuccessCount   int64 `json:"success_count"`
	ErrorCount     int64 `json:"error_count"`
	ErrorRate      float64 `json:"error_rate"` // 0-1

	// Token metrics
	TotalInputTokens  int64 `json:"total_input_tokens"`
	TotalOutputTokens int64 `json:"total_output_tokens"`
	TotalTokens       int64 `json:"total_tokens"`

	// Performance metrics (milliseconds)
	AvgDurationMs   *float64 `json:"avg_duration_ms"`
	P50DurationMs   *float64 `json:"p50_duration_ms"`
	P95DurationMs   *float64 `json:"p95_duration_ms"`
	P99DurationMs   *float64 `json:"p99_duration_ms"`

	// Time to first token (milliseconds, streaming only)
	AvgTTFTMs       *float64 `json:"avg_ttft_ms"`
	P99TTFTMs       *float64 `json:"p99_ttft_ms"`

	// Throughput (tokens/sec for streaming)
	AvgTokensPerSec *float64 `json:"avg_tokens_per_sec"`

	// QPS
	QPS float64 `json:"qps"`
}

// OpsModelBreakdownResponse contains per-model metrics breakdown.
type OpsModelBreakdownResponse struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Platform  string    `json:"platform"`
	GroupID   *int64    `json:"group_id"`

	Items []*OpsModelBreakdownItem `json:"items"`
	Total int64                    `json:"total"` // Total number of distinct models
}

// OpsModelBreakdownFilter defines filters for model breakdown query.
type OpsModelBreakdownFilter struct {
	StartTime time.Time
	EndTime   time.Time
	Platform  string
	GroupID   *int64

	// Optional: limit to top N models by request count
	TopN int
}
