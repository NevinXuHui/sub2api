package repository

import (
	"context"
	"log/slog"
	"maps"
	"time"

	dbaccount "github.com/Wei-Shaw/sub2api/ent/account"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
)

// Store concurrency=1 during the cooldown. Existing scheduler versions honor
// both this limit and temp_unschedulable_until, without a new cache format.
func (r *accountRepository) ParkNodeForFailureBudget(ctx context.Context, account *service.Account, until time.Time, reason string) (bool, error) {
	original := account.Concurrency
	if previous := service.GetNodeFailureRecovery(account); previous != nil {
		original = previous.OriginalConcurrency
	}
	extra := maps.Clone(account.Extra)
	if extra == nil {
		extra = map[string]any{}
	}
	extra[service.NodeFailureRecoveryKey] = service.NodeFailureRecovery{OriginalConcurrency: original, ResumeAfter: until, Generation: uuid.NewString()}
	changed, err := r.client.Account.Update().Where(
		dbaccount.IDEQ(account.ID), dbaccount.StatusEQ(service.StatusActive), dbaccount.SchedulableEQ(true), dbaccount.UpdatedAtEQ(account.UpdatedAt),
	).SetConcurrency(1).SetTempUnschedulableUntil(until).SetTempUnschedulableReason(reason).SetExtra(extra).Save(ctx)
	if err != nil || changed == 0 {
		return false, err
	}
	r.syncNodeFailureBudget(ctx, account.ID)
	return true, nil
}

func (r *accountRepository) RestoreNodeAfterFailureProbe(ctx context.Context, account *service.Account) (bool, error) {
	state := service.GetNodeFailureRecovery(account)
	if state == nil {
		return false, nil
	}
	extra := maps.Clone(account.Extra)
	delete(extra, service.NodeFailureRecoveryKey)
	changed, err := r.client.Account.Update().Where(
		dbaccount.IDEQ(account.ID), dbaccount.StatusEQ(service.StatusActive), dbaccount.SchedulableEQ(true), dbaccount.ConcurrencyEQ(1), dbaccount.UpdatedAtEQ(account.UpdatedAt),
	).SetConcurrency(state.OriginalConcurrency).ClearTempUnschedulableUntil().ClearTempUnschedulableReason().SetExtra(extra).Save(ctx)
	if err != nil || changed == 0 {
		return false, err
	}
	r.syncNodeFailureBudget(ctx, account.ID)
	return true, nil
}

func (r *accountRepository) syncNodeFailureBudget(ctx context.Context, id int64) {
	if err := enqueueSchedulerOutbox(ctx, r.sql, service.SchedulerOutboxEventAccountChanged, &id, nil, nil); err != nil {
		slog.Warn("node_failure_budget_outbox_failed", "account_id", id, "error", err)
	}
	r.syncSchedulerAccountSnapshot(ctx, id)
}
