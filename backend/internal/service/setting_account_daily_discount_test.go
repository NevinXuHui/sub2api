package service

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestAccountDailyDiscountTierSelection(t *testing.T) {
	settings := AccountDailyDiscountSettings{Enabled: true, Tiers: []AccountDailyDiscountTier{{200, 6}, {100, 8}}}
	for _, tc := range []struct {
		revenue string
		factor  float64
	}{{"0", 1}, {"100", 1}, {"100.00000001", .8}, {"200", .8}, {"200.00000001", .6}, {"500", .6}} {
		t.Run(tc.revenue, func(t *testing.T) {
			require.Equal(t, tc.factor, settings.Factor(decimal.RequireFromString(tc.revenue)))
		})
	}
	settings.Enabled = false
	require.Equal(t, 1.0, settings.Factor(decimal.NewFromInt(500)))
}
func TestAccountDailyDiscountSettingsPersistenceAndValidation(t *testing.T) {
	ctx := context.Background()
	repo := &panelRateLimitSettingRepo{}
	svc := &SettingService{settingRepo: repo}
	defaults, err := svc.GetAccountDailyDiscountSettings(ctx)
	require.NoError(t, err)
	require.False(t, defaults.Enabled)
	require.Empty(t, defaults.Tiers)
	settings := AccountDailyDiscountSettings{Enabled: true, Tiers: []AccountDailyDiscountTier{{200, 6}, {100, 8}}}
	require.NoError(t, svc.SetAccountDailyDiscountSettings(ctx, settings))
	settings.Tiers[0].Discount = 1 // Caller cannot mutate the cached policy.
	got, err := svc.GetAccountDailyDiscountSettings(ctx)
	require.NoError(t, err)
	require.Equal(t, []AccountDailyDiscountTier{{100, 8}, {200, 6}}, got.Tiers)
	got.Tiers[0].Discount = 2
	fresh := &SettingService{settingRepo: repo}
	got, err = fresh.GetAccountDailyDiscountSettings(ctx)
	require.NoError(t, err)
	require.Equal(t, 8.0, got.Tiers[0].Discount)
	for _, tiers := range [][]AccountDailyDiscountTier{nil, {{100, 8}, {100, 6}}, {{-1, 8}}, {{100, 0}}, {{100, 11}}, {{math.Inf(1), 8}}, {{100, math.NaN()}}, {{100, 8.001}}, {{0.000000001, 8}}} {
		require.Error(t, svc.SetAccountDailyDiscountSettings(ctx, AccountDailyDiscountSettings{Enabled: true, Tiers: tiers}))
	}
	fresh.accountDailyDiscountCache.expiresAt = time.Now().Add(-time.Second)
	require.NoError(t, svc.SetAccountDailyDiscountSettings(ctx, AccountDailyDiscountSettings{Tiers: []AccountDailyDiscountTier{}}))
	got, err = fresh.GetAccountDailyDiscountSettings(ctx)
	require.NoError(t, err)
	require.False(t, got.Enabled)
}

type discountBillingResultStub struct {
	UsageBillingRepository
	result  *UsageBillingApplyResult
	command *UsageBillingCommand
}

func (r *discountBillingResultStub) Apply(_ context.Context, c *UsageBillingCommand) (*UsageBillingApplyResult, error) {
	r.command = c
	return r.result, nil
}
func TestAccountDailyDiscountReceiptUpdatesUsageAndCost(t *testing.T) {
	for _, applied := range []bool{false, true} {
		t.Run(map[bool]string{true: "new charge", false: "replayed charge"}[applied], func(t *testing.T) {
			settings := &SettingService{settingRepo: &panelRateLimitSettingRepo{}}
			require.NoError(t, settings.SetAccountDailyDiscountSettings(context.Background(), AccountDailyDiscountSettings{Enabled: true, Tiers: []AccountDailyDiscountTier{{100, 8}}}))
			p := &postUsageBillingParams{Cost: &CostBreakdown{TotalCost: 100, ActualCost: 10}, User: &User{ID: 1}, APIKey: &APIKey{ID: 2}, Account: &Account{ID: 3}}
			usage := &UsageLog{ActualCost: 10, TotalCost: 100, RateMultiplier: .1}
			repo := &discountBillingResultStub{result: &UsageBillingApplyResult{Applied: applied, DailyDiscount: &AccountDailyDiscountReceipt{Factor: .8, ActualCost: 8}}}
			ok, err := applyUsageBilling(context.Background(), "discount-request", usage, p, &billingDeps{settingService: settings, deferredService: &DeferredService{}}, repo)
			require.NoError(t, err)
			require.Equal(t, applied, ok)
			require.Equal(t, 8.0, p.Cost.ActualCost)
			require.Equal(t, 8.0, usage.ActualCost)
			require.InDelta(t, .08, usage.RateMultiplier, 1e-12)
			require.Equal(t, 100.0, p.Cost.TotalCost)
			require.Equal(t, 100.0, usage.TotalCost)
			require.True(t, repo.command.DailyDiscount.Enabled)
			require.Equal(t, 10.0, repo.command.BalanceCost, "repository receives original charge for stable idempotency")
		})
	}
}
