//go:build unit

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestAccountDailyDiscountScalesCustomerFieldsOnly(t *testing.T) {
	for _, subscription := range []bool{false, true} {
		t.Run(map[bool]string{false: "balance", true: "subscription"}[subscription], func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()
			mock.ExpectBegin()
			tx, err := db.BeginTx(context.Background(), nil)
			require.NoError(t, err)
			mock.ExpectExec(`SELECT pg_advisory_xact_lock`).WithArgs(sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectQuery(`(?s)SELECT.*SUM\(u.actual_cost\).*NOT EXISTS.*SUM\(actual_cost\)`).WithArgs(int64(3), sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"revenue"}).AddRow("150.00000000"))
			at := time.Now()
			mock.ExpectExec(`INSERT INTO account_daily_discount_receipts`).WithArgs("req", int64(2), int64(3), at, .8, 8.0).WillReturnResult(sqlmock.NewResult(0, 1))
			cmd := &service.UsageBillingCommand{RequestID: "req", APIKeyID: 2, UserID: 1, AccountID: 3, BillingAt: at, BalanceCost: 10, APIKeyQuotaCost: 10, APIKeyRateLimitCost: 10, AccountQuotaCost: 50, DailyDiscount: &service.AccountDailyDiscountSettings{Enabled: true, Tiers: []service.AccountDailyDiscountTier{{ThresholdUSD: 100, Discount: 8}}}}
			if subscription {
				cmd.BalanceCost = 0
				cmd.SubscriptionCost = 10
			}
			cmd.Normalize()
			fingerprint := cmd.RequestFingerprint
			result := &service.UsageBillingApplyResult{}
			require.NoError(t, applyAccountDailyDiscount(context.Background(), tx, cmd, result))
			require.Equal(t, 8.0, cmd.BalanceCost+cmd.SubscriptionCost)
			require.Equal(t, 8.0, cmd.APIKeyQuotaCost)
			require.Equal(t, 8.0, cmd.APIKeyRateLimitCost)
			require.Equal(t, 50.0, cmd.AccountQuotaCost)
			require.Equal(t, fingerprint, cmd.RequestFingerprint)
			require.Equal(t, 8.0, result.DailyDiscount.ActualCost)
			mock.ExpectRollback()
			require.NoError(t, tx.Rollback())
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
