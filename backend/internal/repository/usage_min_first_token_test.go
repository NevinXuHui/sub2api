package repository

import (
	"context"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestUsageListMinFirstToken(t *testing.T) {
	db, mock := newSQLMock(t)
	threshold := 2500
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM usage_logs WHERE first_token_ms >= \$1`).WithArgs(threshold).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`SELECT .* FROM usage_logs WHERE first_token_ms >= \$1 ORDER BY .* LIMIT \$2 OFFSET \$3`).WithArgs(threshold, 20, 0).WillReturnRows(sqlmock.NewRows([]string{"id"}))
	repo := &usageLogRepository{sql: db}
	rows, _, err := repo.ListWithFilters(context.Background(), pagination.PaginationParams{Page: 1, PageSize: 20}, UsageLogFilters{MinFirstTokenMS: &threshold, ExactTotal: true})
	require.NoError(t, err)
	require.Empty(t, rows)
	require.NoError(t, mock.ExpectationsWereMet())
}
