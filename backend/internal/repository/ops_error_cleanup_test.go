package repository

import (
	"context"
	"github.com/DATA-DOG/go-sqlmock"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestClearRequestErrorsBoundedBatches(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	before := time.Now().UTC()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COALESCE(MAX(id), 0) FROM ops_error_logs")).WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(9000))
	for _, n := range []int64{2000, 17, 0} {
		mock.ExpectExec(regexp.QuoteMeta(clearRequestErrorsBatchSQL)).WithArgs(int64(9000), before).WillReturnResult(sqlmock.NewResult(0, n))
	}
	count, err := (&opsRepository{db: db}).ClearRequestErrors(context.Background(), before)
	if err != nil || count != 2017 {
		t.Fatalf("count=%d err=%v", count, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
	for _, guard := range []string{"id <= $1", "created_at < $2", "COALESCE(status_code, 0) >= 400", "error_type = 'cyber_policy'", "LIMIT 2000"} {
		if !strings.Contains(clearRequestErrorsBatchSQL, guard) {
			t.Fatalf("missing guard %s", guard)
		}
	}
	if _, err := (&opsRepository{db: db}).ClearRequestErrors(context.Background(), time.Time{}); err == nil {
		t.Fatal("zero cutoff accepted")
	}
}
