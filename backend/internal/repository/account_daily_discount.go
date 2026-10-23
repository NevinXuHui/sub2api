package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/shopspring/decimal"
)

func loadAccountDailyDiscountReceipt(ctx context.Context, tx *sql.Tx, requestID string, apiKeyID int64) (*service.AccountDailyDiscountReceipt, error) {
	var receipt service.AccountDailyDiscountReceipt
	err := tx.QueryRowContext(ctx, `SELECT factor, actual_cost FROM account_daily_discount_receipts WHERE request_id = $1 AND api_key_id = $2`, requestID, apiKeyID).Scan(&receipt.Factor, &receipt.ActualCost)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &receipt, nil
}

func applyAccountDailyDiscount(ctx context.Context, tx *sql.Tx, cmd *service.UsageBillingCommand, result *service.UsageBillingApplyResult) error {
	policy := cmd.DailyDiscount
	if policy == nil || !policy.Enabled || cmd.AccountID <= 0 || cmd.BalanceCost+cmd.SubscriptionCost <= 0 {
		return nil
	}
	if err := policy.Validate(); err != nil {
		return err
	}
	at := cmd.BillingAt
	if at.IsZero() {
		at = time.Now()
	}
	day := timezone.StartOfDay(at)
	// Serialize only this account's settlements, across gateway replicas. A new
	// statement after this lock sees the preceding transaction's committed receipt.
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, fmt.Sprintf("account_daily_discount:%d:%s", cmd.AccountID, day.Format("2006-01-02"))); err != nil {
		return err
	}
	var revenue decimal.Decimal
	err := tx.QueryRowContext(ctx, `SELECT
 COALESCE((SELECT SUM(u.actual_cost) FROM usage_logs u
 WHERE u.account_id=$1 AND u.created_at >= $2 AND u.created_at < $3
 AND NOT EXISTS (SELECT 1 FROM account_daily_discount_receipts d WHERE d.request_id=u.request_id AND d.api_key_id=u.api_key_id)),0)
 + COALESCE((SELECT SUM(actual_cost) FROM account_daily_discount_receipts
 WHERE account_id=$1 AND billed_at >= $2 AND billed_at < $3),0)`, cmd.AccountID, day, day.AddDate(0, 0, 1)).Scan(&revenue)
	if err != nil {
		return err
	}
	factor := policy.Factor(revenue)
	scale := func(v float64) float64 {
		return decimal.NewFromFloat(v).Mul(decimal.NewFromFloat(factor)).Round(service.UsageBillingMonetaryScale).InexactFloat64()
	}
	cmd.BalanceCost = scale(cmd.BalanceCost)
	cmd.SubscriptionCost = scale(cmd.SubscriptionCost)
	cmd.APIKeyQuotaCost = scale(cmd.APIKeyQuotaCost)
	cmd.APIKeyRateLimitCost = scale(cmd.APIKeyRateLimitCost)
	// AccountQuotaCost is upstream cost, not user revenue; do not discount it.
	receipt := &service.AccountDailyDiscountReceipt{Factor: factor, ActualCost: cmd.BalanceCost + cmd.SubscriptionCost}
	_, err = tx.ExecContext(ctx, `INSERT INTO account_daily_discount_receipts (request_id,api_key_id,account_id,billed_at,factor,actual_cost) VALUES ($1,$2,$3,$4,$5,$6)`, cmd.RequestID, cmd.APIKeyID, cmd.AccountID, at, receipt.Factor, receipt.ActualCost)
	if err != nil {
		return err
	}
	result.DailyDiscount = receipt
	return nil
}
