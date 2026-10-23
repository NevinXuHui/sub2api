//go:build unit

package repository

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestNodeFailureBudgetRollingWindowAndRecovery(t *testing.T) {
	ctx := context.Background()
	server := miniredis.RunT(t)
	now := time.Now().UTC().Truncate(time.Second)
	server.SetTime(now)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	cache := NewNodeFailureBudgetCache(client)
	for i := 1; i <= 99; i++ {
		count, trip, err := cache.RecordNodeFailure(ctx, 7, fmt.Sprint(i), now, service.DefaultNodeFailureSettings())
		require.NoError(t, err)
		require.EqualValues(t, i, count)
		require.False(t, trip)
	}
	count, trip, err := cache.RecordNodeFailure(ctx, 7, "99", now, service.DefaultNodeFailureSettings())
	require.NoError(t, err)
	require.EqualValues(t, 99, count)
	require.False(t, trip)
	count, trip, err = cache.RecordNodeFailure(ctx, 8, "other-node", now, service.DefaultNodeFailureSettings())
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
	require.False(t, trip)
	// Exact rolling boundary: previous failures aged out, even though the keys
	// still exist. This is not a fixed clock-minute counter.
	server.SetTime(now.Add(time.Minute))
	count, trip, err = cache.RecordNodeFailure(ctx, 7, "100", now.Add(time.Minute), service.DefaultNodeFailureSettings())
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
	require.False(t, trip)
	for i := 101; i <= 200; i++ {
		count, trip, err = cache.RecordNodeFailure(ctx, 7, fmt.Sprint(i), now.Add(time.Minute), service.DefaultNodeFailureSettings())
		require.NoError(t, err)
	}
	require.EqualValues(t, 101, count)
	require.True(t, trip)
	// A failed database write releases the claim without erasing evidence.
	require.NoError(t, cache.FinishNodeFailureTrip(ctx, 7, "200", false))
	count, trip, err = cache.RecordNodeFailure(ctx, 7, "201", now.Add(time.Minute), service.DefaultNodeFailureSettings())
	require.NoError(t, err)
	require.EqualValues(t, 102, count)
	require.True(t, trip)
	require.NoError(t, cache.FinishNodeFailureTrip(ctx, 7, "201", true))
	// Errors arriving from old in-flight requests cannot disable a recovered node.
	count, trip, err = cache.RecordNodeFailure(ctx, 7, "late", now, service.DefaultNodeFailureSettings())
	require.NoError(t, err)
	require.Zero(t, count)
	require.False(t, trip)
	server.SetTime(now.Add(time.Minute + time.Second))
	count, trip, err = cache.RecordNodeFailure(ctx, 7, "fresh-after-manual-recovery", now.Add(time.Minute+time.Second), service.DefaultNodeFailureSettings())
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
	require.False(t, trip)
}

func TestNodeFailureBudgetAtomicAcrossInstances(t *testing.T) {
	server := miniredis.RunT(t)
	now := time.Now()
	server.SetTime(now)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	first, second := NewNodeFailureBudgetCache(client), NewNodeFailureBudgetCache(client)
	var wg sync.WaitGroup
	var trips atomic.Int64
	var failures atomic.Int64
	for i := 0; i < 120; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			cache := first
			if i%2 == 0 {
				cache = second
			}
			_, trip, err := cache.RecordNodeFailure(context.Background(), 1, fmt.Sprint(i), now, service.DefaultNodeFailureSettings())
			if err != nil {
				failures.Add(1)
			}
			if trip {
				trips.Add(1)
			}
		}(i)
	}
	wg.Wait()
	require.Zero(t, failures.Load())
	require.EqualValues(t, 1, trips.Load())
}

func TestNodeFailureBudgetConfigurableWindowAndThreshold(t *testing.T) {
	ctx := context.Background()
	server := miniredis.RunT(t)
	now := time.Now()
	server.SetTime(now)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	cache := NewNodeFailureBudgetCache(client)
	settings := service.NodeFailureSettings{FailureRatePercent: 40, WindowMinutes: 2, FailureThreshold: 1, CooldownMinutes: 3}
	count, trip, err := cache.RecordNodeFailure(ctx, 1, "a", now, settings)
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
	require.False(t, trip)
	server.SetTime(now.Add(90 * time.Second))
	count, trip, err = cache.RecordNodeFailure(ctx, 1, "b", now.Add(90*time.Second), settings)
	require.NoError(t, err)
	require.EqualValues(t, 2, count)
	require.True(t, trip) // A 1-minute hard-coded window would fail this assertion.
	require.NoError(t, cache.FinishNodeFailureTrip(ctx, 1, "b", false))
	server.SetTime(now.Add(210 * time.Second))
	count, trip, err = cache.RecordNodeFailure(ctx, 1, "c", now.Add(210*time.Second), settings)
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
	require.False(t, trip)
}

func TestNodeFailureBudgetRequiresBothStrictThresholds(t *testing.T) {
	ctx := context.Background()
	server := miniredis.RunT(t)
	now := time.Now()
	server.SetTime(now)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	cache := NewNodeFailureBudgetCache(client)
	settings := service.DefaultNodeFailureSettings()
	// 101/253 is below 40%, despite more than 100 failures.
	for i := 0; i < 152; i++ {
		require.NoError(t, cache.RecordNodeSuccess(ctx, 1, fmt.Sprintf("s%d", i), now, settings))
	}
	for i := 1; i <= 101; i++ {
		count, trip, err := cache.RecordNodeFailure(ctx, 1, fmt.Sprintf("f%d", i), now, settings)
		require.NoError(t, err)
		require.EqualValues(t, i, count)
		require.False(t, trip)
	}
	// Retrying a failed request does not inflate either side of the fraction.
	require.NoError(t, cache.RecordNodeSuccess(ctx, 1, "f101", now, settings))
	count, trip, err := cache.RecordNodeFailure(ctx, 1, "f101", now, settings)
	require.NoError(t, err)
	require.EqualValues(t, 101, count)
	require.False(t, trip)
	// 102/254 > 40% and failures > 100.
	count, trip, err = cache.RecordNodeFailure(ctx, 1, "f102", now, settings)
	require.NoError(t, err)
	require.EqualValues(t, 102, count)
	require.True(t, trip)
	// Exactly 40% does not trip, even when the failure count is over 100.
	for i := 0; i < 153; i++ {
		require.NoError(t, cache.RecordNodeSuccess(ctx, 2, fmt.Sprintf("s%d", i), now, settings))
	}
	for i := 1; i <= 102; i++ {
		_, trip, err = cache.RecordNodeFailure(ctx, 2, fmt.Sprintf("f%d", i), now, settings)
		require.NoError(t, err)
		require.False(t, trip)
	}
	// Exactly 100 failures at 100% also does not trip.
	for i := 1; i <= 100; i++ {
		_, trip, err = cache.RecordNodeFailure(ctx, 3, fmt.Sprint(i), now, settings)
		require.NoError(t, err)
		require.False(t, trip)
	}
}

func TestNodeFailureProbeLeaseAcrossInstances(t *testing.T) {
	ctx := context.Background()
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	a, b := NewNodeFailureBudgetCache(client), NewNodeFailureBudgetCache(client)
	allowed, err := a.ClaimNodeFailureProbe(ctx, 1, "generation", 3*time.Minute)
	require.NoError(t, err)
	require.True(t, allowed)
	allowed, err = b.ClaimNodeFailureProbe(ctx, 1, "generation", 3*time.Minute)
	require.NoError(t, err)
	require.False(t, allowed)
	server.FastForward(3 * time.Minute)
	allowed, err = b.ClaimNodeFailureProbe(ctx, 1, "generation", 3*time.Minute)
	require.NoError(t, err)
	require.True(t, allowed)
}
