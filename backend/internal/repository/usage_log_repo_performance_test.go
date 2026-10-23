package repository

import (
	"context"
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

type performanceWindowStart struct{ start *time.Time }

func (m performanceWindowStart) Match(v driver.Value) bool {
	t, ok := v.(time.Time)
	if ok {
		*m.start = t
	}
	return ok
}

type performanceWindowEnd struct{ start *time.Time }

func (m performanceWindowEnd) Match(v driver.Value) bool {
	t, ok := v.(time.Time)
	return ok && t.Sub(*m.start) == time.Minute && time.Since(t) < time.Second
}

func TestDashboardPerformance(t *testing.T) {
	for _, tc := range []struct {
		name              string
		empty             bool
		rowError          bool
		errorQueryFailure bool
	}{
		{name: "models and exact one minute totals"}, {name: "empty window", empty: true}, {name: "row failure", rowError: true}, {name: "error count failure", errorQueryFailure: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := newSQLMock(t)
			repo := &usageLogRepository{sql: db}
			rows := sqlmock.NewRows([]string{"model", "requests", "tokens"})
			if !tc.empty {
				rows.AddRow("claude-haiku-4-5-20251001", 7, 1234).AddRow("claude-sonnet-4-5-20250929", 2, 456)
			}
			if tc.rowError {
				rows.RowError(1, errors.New("interrupted"))
			}
			var start time.Time
			mock.ExpectQuery(`(?s)SELECT COALESCE\(NULLIF\(TRIM\(requested_model\), ''\), model\) AS model, COUNT\(\*\),.*SUM\(input_tokens \+ output_tokens\).*WHERE created_at >= \$1 AND created_at < \$2.*GROUP BY.*ORDER BY COUNT\(\*\) DESC, model ASC`).
				WithArgs(performanceWindowStart{&start}, performanceWindowEnd{&start}).WillReturnRows(rows).RowsWillBeClosed()
			if !tc.rowError {
				latency := sqlmock.NewRows([]string{"model", "avg", "count"})
				if !tc.empty {
					latency.AddRow("claude-haiku-4-5-20251001", 200, 3).AddRow("claude-sonnet-4-5-20250929", 1000, 1)
				}
				mock.ExpectQuery(`(?s)SELECT model, AVG\(first_token_ms\), COUNT\(\*\).*ORDER BY id DESC LIMIT 100.*WHERE first_token_ms IS NOT NULL AND first_token_ms >= 0`).WillReturnRows(latency).RowsWillBeClosed()
				countQuery := mock.ExpectQuery(`(?s)SELECT COUNT\(\*\) FILTER \(WHERE created_at >= \$3\), COUNT\(\*\) FROM ops_error_logs.*created_at >= \$1 AND created_at < \$2.*is_count_tokens = FALSE AND status_code >= 400`).WithArgs(performanceSameTime{&start, time.Minute - time.Hour}, performanceSameTime{&start, time.Minute}, performanceSameTime{&start, 0})
				if tc.errorQueryFailure {
					countQuery.WillReturnError(errors.New("error log query failed"))
				} else {
					count := 12
					if tc.empty {
						count = 0
					}
					countQuery.WillReturnRows(sqlmock.NewRows([]string{"count", "hour_count"}).AddRow(count, count*5))
					modelRows := sqlmock.NewRows([]string{"model", "errors", "usage"})
					if !tc.empty {
						modelRows.AddRow("haiku", 15, 75).AddRow("error-only", 10, 0).AddRow("success-only", 0, 0)
					}
					mock.ExpectQuery(`(?s)SELECT model, SUM\(errors\), SUM\(requests\).*FROM ops_error_logs.*status_code >= 400.*UNION ALL.*FROM usage_logs.*GROUP BY model`).
						WithArgs(sqlmock.AnyArg(), performanceSameTime{&start, time.Minute}).
						WillReturnRows(modelRows).RowsWillBeClosed()

				}

			}
			got, err := repo.GetDashboardPerformance(context.Background())
			require.NoError(t, mock.ExpectationsWereMet())
			if tc.rowError || tc.errorQueryFailure {
				require.Error(t, err)
				require.Nil(t, got)
				return
			}
			require.NoError(t, err)
			require.Equal(t, 60, got.WindowSeconds)
			require.Equal(t, start.Add(time.Minute), got.UpdatedAt)
			require.NotNil(t, got.Models)
			if tc.empty {
				require.Zero(t, got.ErrorCount)
				require.Zero(t, got.HourErrorCount)
				require.Zero(t, got.TodayErrorCount)
				require.Zero(t, got.TodayErrorRate)
				require.Empty(t, got.Models)
				require.Nil(t, got.AvgFirstTokenMS)
				require.Zero(t, got.FirstTokenSamples)
				require.Zero(t, got.RPM)
				require.Zero(t, got.TPM)
				return
			}
			require.EqualValues(t, 12, got.ErrorCount)
			require.EqualValues(t, 60, got.HourErrorCount)
			require.EqualValues(t, 25, got.TodayErrorCount)
			require.Equal(t, 25.0, got.TodayErrorRate)
			require.Len(t, got.TodayModels, 3)
			require.InDelta(t, 100.0/6, got.TodayModels[0].ErrorRate, 0.0001)
			require.Equal(t, 100.0, got.TodayModels[1].ErrorRate)
			require.Zero(t, got.TodayModels[2].ErrorRate)
			require.Equal(t, 400.0, *got.AvgFirstTokenMS)
			require.EqualValues(t, 4, got.FirstTokenSamples)
			require.Equal(t, 200.0, *got.Models[0].AvgFirstTokenMS)
			require.EqualValues(t, 9, got.RPM)
			require.EqualValues(t, 1690, got.TPM)
			require.Len(t, got.Models, 2)
			require.EqualValues(t, 7, got.Models[0].RPM)
			require.EqualValues(t, 1234, got.Models[0].TPM)
		})
	}
}

type performanceSameTime struct {
	start  *time.Time
	offset time.Duration
}

func (m performanceSameTime) Match(v driver.Value) bool {
	value, ok := v.(time.Time)
	return ok && value.Equal(m.start.Add(m.offset))
}
