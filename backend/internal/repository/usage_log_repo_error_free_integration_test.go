//go:build integration

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"
)

// VALUES fixtures exercise the production SQL without modifying any tables.
func TestDashboardErrorFreeTolerance(t *testing.T) {
	for _, tc := range []struct {
		name string
		ages []int
		want int
	}{
		{"none", nil, -1}, {"one", []int{10}, -1}, {"two", []int{10, 20}, -1},
		{"third resets", []int{10, 20, 30}, 10},
		{"exact thirty minutes", []int{10, 20, 1810}, 10},
		{"outside thirty minutes", []int{10, 20, 1811}, -1},
		{"two sparse errors preserve prior reset", []int{10, 20, 4000, 4010, 4020}, 4000},
		{"simultaneous failures", []int{10, 10, 10}, 10},
		{"fourth resets again", []int{5, 10, 20, 30}, 5},
		{"tie completes earlier pair", []int{10, 10, 1800}, 10},
		{"dense history followed by isolated failure", []int{10, 4000, 4010, 4020, 4030}, 4000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			now := time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC)
			rows := []string{"($1::timestamptz - INTERVAL '1 second', true, 500)", "($1::timestamptz - INTERVAL '2 seconds', false, 200)", "($1::timestamptz + INTERVAL '1 second', false, 500)"}
			for _, age := range tc.ages {
				rows = append(rows, fmt.Sprintf("($1::timestamptz - INTERVAL '%d seconds', false, 502)", age))
			}
			query := "WITH ops_error_logs(created_at, is_count_tokens, status_code) AS (VALUES " + strings.Join(rows, ",") + ") " + dashboardErrorFreeResetQuery
			var reset sql.NullTime
			require.NoError(t, integrationDB.QueryRowContext(context.Background(), query, now).Scan(&reset))
			if tc.want < 0 {
				require.False(t, reset.Valid)
			} else {
				require.True(t, reset.Valid)
				require.Equal(t, now.Add(-time.Duration(tc.want)*time.Second), reset.Time)
			}
		})
	}
}
