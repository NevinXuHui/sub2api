package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (r *opsRepository) GetModelBreakdown(ctx context.Context, filter *service.OpsModelBreakdownFilter) (*service.OpsModelBreakdownResponse, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil ops repository")
	}
	if filter == nil {
		return nil, fmt.Errorf("nil filter")
	}
	if filter.StartTime.IsZero() || filter.EndTime.IsZero() {
		return nil, fmt.Errorf("start_time/end_time required")
	}
	if filter.StartTime.After(filter.EndTime) {
		return nil, fmt.Errorf("start_time must be <= end_time")
	}

	dashboardFilter := &service.OpsDashboardFilter{
		StartTime: filter.StartTime.UTC(),
		EndTime:   filter.EndTime.UTC(),
		Platform:  strings.TrimSpace(strings.ToLower(filter.Platform)),
		GroupID:   filter.GroupID,
	}

	join, where, baseArgs, next := buildUsageWhere(dashboardFilter, dashboardFilter.StartTime, dashboardFilter.EndTime, 1)

	// Calculate time window in seconds for QPS
	timeWindowSec := filter.EndTime.Sub(filter.StartTime).Seconds()
	if timeWindowSec <= 0 {
		timeWindowSec = 1
	}

	baseCTE := fmt.Sprintf(`
WITH stats AS (
  SELECT
    ul.model AS model,
    COUNT(*)::bigint AS request_count,
    COUNT(CASE WHEN ul.error_phase IS NULL OR ul.error_phase = '' THEN 1 END)::bigint AS success_count,
    COUNT(CASE WHEN ul.error_phase IS NOT NULL AND ul.error_phase != '' THEN 1 END)::bigint AS error_count,
    COALESCE(SUM(ul.input_tokens + COALESCE(ul.cache_creation_tokens, 0) + COALESCE(ul.cache_read_tokens, 0)), 0)::bigint AS total_input_tokens,
    COALESCE(SUM(ul.output_tokens), 0)::bigint AS total_output_tokens,
    ROUND(AVG(ul.duration_ms)::numeric, 2)::float8 AS avg_duration_ms,
    ROUND(PERCENTILE_CONT(0.50) WITHIN GROUP (ORDER BY ul.duration_ms)::numeric, 2)::float8 AS p50_duration_ms,
    ROUND(PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY ul.duration_ms)::numeric, 2)::float8 AS p95_duration_ms,
    ROUND(PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY ul.duration_ms)::numeric, 2)::float8 AS p99_duration_ms,
    ROUND(AVG(ul.first_token_ms)::numeric, 2)::float8 AS avg_ttft_ms,
    ROUND(PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY ul.first_token_ms)::numeric, 2)::float8 AS p99_ttft_ms,
    ROUND(
      AVG(
        CASE
          WHEN ul.duration_ms > 0 AND ul.output_tokens > 0
          THEN ul.output_tokens * 1000.0 / ul.duration_ms
        END
      )::numeric,
      2
    )::float8 AS avg_tokens_per_sec,
    COUNT(*)::float8 / %f AS qps
  FROM usage_logs ul
  %s
  %s
  GROUP BY ul.model
)
`, timeWindowSec, join, where)

	countSQL := baseCTE + `SELECT COUNT(*) FROM stats`
	var total int64
	if err := r.db.QueryRowContext(ctx, countSQL, baseArgs...).Scan(&total); err != nil {
		return nil, err
	}

	querySQL := baseCTE + `
SELECT
  model,
  request_count,
  success_count,
  error_count,
  total_input_tokens,
  total_output_tokens,
  avg_duration_ms,
  p50_duration_ms,
  p95_duration_ms,
  p99_duration_ms,
  avg_ttft_ms,
  p99_ttft_ms,
  avg_tokens_per_sec,
  qps
FROM stats
ORDER BY request_count DESC, model ASC`

	args := make([]any, 0, len(baseArgs)+1)
	args = append(args, baseArgs...)

	if filter.TopN > 0 {
		querySQL += fmt.Sprintf("\nLIMIT $%d", next)
		args = append(args, filter.TopN)
	}

	rows, err := r.db.QueryContext(ctx, querySQL, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	items := make([]*service.OpsModelBreakdownItem, 0, 32)
	for rows.Next() {
		item := &service.OpsModelBreakdownItem{}
		var avgDuration, p50Duration, p95Duration, p99Duration sql.NullFloat64
		var avgTTFT, p99TTFT, avgTPS sql.NullFloat64

		if err := rows.Scan(
			&item.Model,
			&item.RequestCount,
			&item.SuccessCount,
			&item.ErrorCount,
			&item.TotalInputTokens,
			&item.TotalOutputTokens,
			&avgDuration,
			&p50Duration,
			&p95Duration,
			&p99Duration,
			&avgTTFT,
			&p99TTFT,
			&avgTPS,
			&item.QPS,
		); err != nil {
			return nil, err
		}

		// Calculate error rate
		if item.RequestCount > 0 {
			item.ErrorRate = float64(item.ErrorCount) / float64(item.RequestCount)
		}

		// Total tokens
		item.TotalTokens = item.TotalInputTokens + item.TotalOutputTokens

		// Assign nullable float fields
		if avgDuration.Valid {
			v := avgDuration.Float64
			item.AvgDurationMs = &v
		}
		if p50Duration.Valid {
			v := p50Duration.Float64
			item.P50DurationMs = &v
		}
		if p95Duration.Valid {
			v := p95Duration.Float64
			item.P95DurationMs = &v
		}
		if p99Duration.Valid {
			v := p99Duration.Float64
			item.P99DurationMs = &v
		}
		if avgTTFT.Valid {
			v := avgTTFT.Float64
			item.AvgTTFTMs = &v
		}
		if p99TTFT.Valid {
			v := p99TTFT.Float64
			item.P99TTFTMs = &v
		}
		if avgTPS.Valid {
			v := avgTPS.Float64
			item.AvgTokensPerSec = &v
		}

		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	resp := &service.OpsModelBreakdownResponse{
		StartTime: dashboardFilter.StartTime,
		EndTime:   dashboardFilter.EndTime,
		Platform:  dashboardFilter.Platform,
		GroupID:   dashboardFilter.GroupID,
		Items:     items,
		Total:     total,
	}
	return resp, nil
}
