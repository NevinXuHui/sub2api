package service

import (
	"context"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"log"
	"time"
)

func (s *OpsService) ClearRequestErrors(ctx context.Context, before time.Time, operatorID int64) (int64, error) {
	if err := s.RequireMonitoringEnabled(ctx); err != nil {
		return 0, err
	}
	if operatorID <= 0 || before.IsZero() || before.After(time.Now()) {
		return 0, infraerrors.BadRequest("OPS_ERROR_CLEANUP_INVALID", "Invalid cleanup cutoff or operator")
	}
	if s.opsRepo == nil {
		return 0, infraerrors.ServiceUnavailable("OPS_REPO_UNAVAILABLE", "Ops repository not available")
	}
	deleted, err := s.opsRepo.ClearRequestErrors(ctx, before)
	log.Printf("[OpsErrorCleanup] operator=%d before=%s deleted=%d failed=%t", operatorID, before.UTC().Format(time.RFC3339Nano), deleted, err != nil)
	if err != nil {
		return 0, infraerrors.InternalServer("OPS_ERROR_CLEANUP_FAILED", "Failed to clear error records; some batches may already have been deleted").WithCause(err)
	}
	return deleted, nil
}
