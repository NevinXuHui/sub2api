package service

import (
	"context"
	"errors"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestNodeFailureSettingsPersistValidateAndRefresh(t *testing.T) {
	ctx := context.Background()
	repo := &panelRateLimitSettingRepo{}
	first := &SettingService{settingRepo: repo}
	second := &SettingService{settingRepo: repo}
	got, err := first.GetNodeFailureSettings(ctx)
	require.NoError(t, err)
	require.Equal(t, DefaultNodeFailureSettings(), got)
	_, err = second.GetNodeFailureSettings(ctx)
	require.NoError(t, err)
	changed := NodeFailureSettings{FailureRatePercent: 40, WindowMinutes: 3, FailureThreshold: 7, CooldownMinutes: 2}
	require.NoError(t, first.SetNodeFailureSettings(ctx, changed))
	got, err = first.GetNodeFailureSettings(ctx)
	require.NoError(t, err)
	require.Equal(t, changed, got)
	// Other processes refresh after their bounded cache lifetime.
	second.nodeFailureSettingsCache.expiresAt = time.Now().Add(-time.Second)
	got, err = second.GetNodeFailureSettings(ctx)
	require.NoError(t, err)
	require.Equal(t, changed, got)
	fresh := &SettingService{settingRepo: repo}
	got, err = fresh.GetNodeFailureSettings(ctx)
	require.NoError(t, err)
	require.Equal(t, changed, got)
	for _, bad := range []NodeFailureSettings{{40, 0, 7, 2}, {40, 61, 7, 2}, {40, 3, 0, 2}, {40, 3, 100001, 2}, {40, 3, 7, 0}, {40, 3, 7, 1441}, {0, 3, 7, 2}, {100, 3, 7, 2}} {
		require.Error(t, first.SetNodeFailureSettings(ctx, bad))
	}
	got, err = first.GetNodeFailureSettings(ctx)
	require.NoError(t, err)
	require.Equal(t, changed, got)
	first.nodeFailureSettingsCache.expiresAt = time.Now().Add(-time.Second)
	repo.getValueErr = errors.New("unavailable")
	require.Equal(t, changed, (&RateLimitService{settingService: first}).nodeFailureSettings(ctx))
}

func TestNodeFailureSettingsDriveTripAndProbeCooldown(t *testing.T) {
	ctx := context.Background()
	settings := &SettingService{settingRepo: &panelRateLimitSettingRepo{}}
	require.NoError(t, settings.SetNodeFailureSettings(ctx, NodeFailureSettings{FailureRatePercent: 40, WindowMinutes: 2, FailureThreshold: 1, CooldownMinutes: 3}))
	account := nodeBudgetAccount()
	repo := &nodeBudgetRepoStub{account: account}
	svc := &RateLimitService{settingService: settings, accountRepo: repo, nodeFailureBudget: &nodeBudgetStub{}}
	svc.recordNodeFailure(ctx, account, nodeFailureRequest{id: "first", started: time.Now()})
	require.Zero(t, repo.calls)
	svc.recordNodeFailure(ctx, account, nodeFailureRequest{id: "second", started: time.Now()})
	require.Equal(t, 1, repo.calls)
	require.WithinDuration(t, time.Now().Add(3*time.Minute), *account.TempUnschedulableUntil, time.Second)
	require.Contains(t, repo.reason, "最近 2 分钟累计 2 次异常")
	// Saving new settings does not move the cooldown that already started.
	until := *account.TempUnschedulableUntil
	require.NoError(t, settings.SetNodeFailureSettings(ctx, NodeFailureSettings{FailureRatePercent: 40, WindowMinutes: 2, FailureThreshold: 1, CooldownMinutes: 4}))
	require.Equal(t, until, *account.TempUnschedulableUntil)
	state := GetNodeFailureRecovery(account)
	state.ResumeAfter = time.Now().Add(-time.Second)
	account.Extra[NodeFailureRecoveryKey] = *state
	account.TempUnschedulableUntil = &state.ResumeAfter
	svc.finishNodeFailureProbe(ctx, account, nodeFailureRequest{id: "probe", started: time.Now()}, false)
	require.Equal(t, 2, repo.calls)
	require.WithinDuration(t, time.Now().Add(4*time.Minute), *account.TempUnschedulableUntil, time.Second)
	require.Contains(t, repo.reason, "停调 4 分钟")
}

func TestNodeFailureSettingsLegacyConfigAddsRate(t *testing.T) {
	repo := &panelRateLimitSettingRepo{values: map[string]string{SettingKeyNodeFailureSettings: `{"window_minutes":2,"failure_threshold":150,"cooldown_minutes":7}`}}
	svc := &SettingService{settingRepo: repo}
	value, err := svc.GetNodeFailureSettings(context.Background())
	require.NoError(t, err)
	require.Equal(t, NodeFailureSettings{WindowMinutes: 2, FailureThreshold: 150, CooldownMinutes: 7, FailureRatePercent: 40}, value)
	require.Equal(t, 3, DefaultNodeFailureSettings().CooldownMinutes)
}
