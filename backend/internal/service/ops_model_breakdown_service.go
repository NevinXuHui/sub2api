package service

import (
	"context"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

func (s *OpsService) GetModelBreakdown(ctx context.Context, filter *OpsModelBreakdownFilter) (*OpsModelBreakdownResponse, error) {
	if err := s.RequireMonitoringEnabled(ctx); err != nil {
		return nil, err
	}
	if s.opsRepo == nil {
		return nil, infraerrors.ServiceUnavailable("OPS_REPO_UNAVAILABLE", "Ops repository not available")
	}
	if filter == nil {
		return nil, infraerrors.BadRequest("OPS_FILTER_REQUIRED", "filter is required")
	}
	if filter.StartTime.IsZero() || filter.EndTime.IsZero() {
		return nil, infraerrors.BadRequest("OPS_TIME_RANGE_REQUIRED", "start_time/end_time are required")
	}
	if filter.StartTime.After(filter.EndTime) {
		return nil, infraerrors.BadRequest("OPS_TIME_RANGE_INVALID", "start_time must be <= end_time")
	}

	// Limit time range to prevent resource exhaustion
	maxDuration := 7 * 24 * time.Hour // 7 days
	if filter.EndTime.Sub(filter.StartTime) > maxDuration {
		return nil, infraerrors.BadRequest("OPS_TIME_RANGE_TOO_LARGE", "time range must be <= 7 days")
	}

	// Default TopN to 50 if not specified
	if filter.TopN == 0 {
		filter.TopN = 50
	}

	resp, err := s.opsRepo.GetModelBreakdown(ctx, filter)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
