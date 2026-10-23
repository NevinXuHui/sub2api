//go:build unit

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/migrations"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

// Opt-in real PostgreSQL test. The DSN must point at a disposable test database;
// each invocation creates its own schema and cleans up only that schema.
func TestAccountDailyDiscountPostgres(t *testing.T) {
	dsn := os.Getenv("SUB2API_DISCOUNT_TEST_DSN")
	if dsn == "" {
		t.Skip("set SUB2API_DISCOUNT_TEST_DSN to an isolated PostgreSQL database")
	}
	root, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer root.Close()
	schema := fmt.Sprintf("discount_test_%d", time.Now().UnixNano())
	_, err = root.Exec(`CREATE SCHEMA ` + schema)
	require.NoError(t, err)
	defer root.Exec(`DROP SCHEMA ` + schema + ` CASCADE`)
	db, err := sql.Open("postgres", dsn+" search_path="+schema)
	require.NoError(t, err)
	defer db.Close()
	db.SetMaxOpenConns(15)
	_, err = db.Exec(`CREATE TABLE users(id BIGINT PRIMARY KEY,balance NUMERIC(20,8),updated_at TIMESTAMPTZ,deleted_at TIMESTAMPTZ);
 CREATE TABLE usage_billing_dedup(id BIGSERIAL,request_id TEXT,api_key_id BIGINT,request_fingerprint TEXT,UNIQUE(request_id,api_key_id));
 CREATE TABLE usage_billing_dedup_archive(request_id TEXT,api_key_id BIGINT,request_fingerprint TEXT);
 CREATE TABLE usage_logs(account_id BIGINT,request_id TEXT,api_key_id BIGINT,created_at TIMESTAMPTZ,actual_cost NUMERIC(20,8));
 INSERT INTO users(id,balance) VALUES(1,1000);`)
	require.NoError(t, err)
	migration, err := migrations.FS.ReadFile("246_account_daily_discount_receipts.sql")
	require.NoError(t, err)
	_, err = db.Exec(string(migration))
	require.NoError(t, err)
	_, err = db.Exec(string(migration))
	require.NoError(t, err) // migration is repeatable
	repo := &usageBillingRepository{db: db}
	policy := &service.AccountDailyDiscountSettings{Enabled: true, Tiers: []service.AccountDailyDiscountTier{{100, 8}, {200, 6}}}
	at := time.Date(2026, 10, 22, 12, 0, 0, 0, time.Local)
	command := func(id string, account int64, cost float64) *service.UsageBillingCommand {
		return &service.UsageBillingCommand{RequestID: id, APIKeyID: 1, UserID: 1, AccountID: account, BalanceCost: cost, DailyDiscount: policy, BillingAt: at}
	}
	_, err = db.Exec(`INSERT INTO usage_logs VALUES(10,'old',1,$1,100)`, at)
	require.NoError(t, err)
	r, err := repo.Apply(context.Background(), command("boundary", 10, 10))
	require.NoError(t, err)
	require.Equal(t, 1.0, r.DailyDiscount.Factor)
	second := command("discounted", 10, 10)
	r, err = repo.Apply(context.Background(), second)
	require.NoError(t, err)
	require.Equal(t, .8, r.DailyDiscount.Factor)
	require.Equal(t, 8.0, r.DailyDiscount.ActualCost)
	require.Equal(t, 10.0, second.BalanceCost, "billing must not mutate the retry command")
	// Arrival of the asynchronous usage row does not double-count its receipt.
	_, err = db.Exec(`INSERT INTO usage_logs VALUES(10,'boundary',1,$1,10),(10,'discounted',1,$1,8)`, at)
	require.NoError(t, err)
	policy.Tiers = []service.AccountDailyDiscountTier{{100, 8}, {125, 6}}
	r, err = repo.Apply(context.Background(), command("no-double-count", 10, 10))
	require.NoError(t, err)
	require.Equal(t, .8, r.DailyDiscount.Factor)
	r, err = repo.Apply(context.Background(), command("higher-tier", 10, 10))
	require.NoError(t, err)
	require.Equal(t, .6, r.DailyDiscount.Factor)
	// Replays keep the original discount even after disabling the feature.
	policy.Enabled = false
	r, err = repo.Apply(context.Background(), second)
	require.NoError(t, err)
	require.False(t, r.Applied)
	require.Equal(t, 8.0, r.DailyDiscount.ActualCost)
	policy.Enabled = true
	r, err = repo.Apply(context.Background(), command("other-account", 11, 10))
	require.NoError(t, err)
	require.Equal(t, 1.0, r.DailyDiscount.Factor)
	tomorrow := command("tomorrow", 10, 10)
	tomorrow.BillingAt = at.AddDate(0, 0, 1)
	r, err = repo.Apply(context.Background(), tomorrow)
	require.NoError(t, err)
	require.Equal(t, 1.0, r.DailyDiscount.Factor)
	// The first of 20 simultaneous settlements crosses the boundary at full price;
	// every subsequent settlement must see its committed receipt, even without logs.
	policy.Tiers = []service.AccountDailyDiscountTier{{100, 8}}
	_, err = db.Exec(`INSERT INTO usage_logs VALUES(12,'old-concurrent',1,$1,100)`, at)
	require.NoError(t, err)
	var wg sync.WaitGroup
	outcomes := make(chan *service.UsageBillingApplyResult, 20)
	failures := make(chan error, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			v, e := repo.Apply(context.Background(), command(fmt.Sprintf("parallel-%d", i), 12, 1))
			if e != nil {
				failures <- e
			} else {
				outcomes <- v
			}
		}(i)
	}
	wg.Wait()
	close(outcomes)
	close(failures)
	for e := range failures {
		require.NoError(t, e)
	}
	full, discounted := 0, 0
	for v := range outcomes {
		if v.DailyDiscount.Factor == 1 {
			full++
		} else if v.DailyDiscount.Factor == .8 {
			discounted++
		}
	}
	require.Equal(t, 1, full)
	require.Equal(t, 19, discounted)
	// Failed balance deductions must roll back the receipt and dedup claim too.
	invalid := command("rollback", 13, 10)
	invalid.UserID = 999
	_, err = repo.Apply(context.Background(), invalid)
	require.Error(t, err)
	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM account_daily_discount_receipts WHERE request_id='rollback'`).Scan(&count))
	require.Zero(t, count)
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM usage_billing_dedup WHERE request_id='rollback'`).Scan(&count))
	require.Zero(t, count)
}
