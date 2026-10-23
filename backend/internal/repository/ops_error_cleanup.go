package repository

import (
	"context"
	"fmt"
	"time"
)

// Match the request-error list, including streaming policy errors but not
// recovered upstream failures. A fixed ID boundary protects concurrent inserts.
const clearRequestErrorsBatchSQL = `DELETE FROM ops_error_logs WHERE id IN (
 SELECT id FROM ops_error_logs
 WHERE id <= $1 AND created_at < $2
 AND (COALESCE(status_code, 0) >= 400 OR error_type = 'cyber_policy')
 ORDER BY id LIMIT 2000
)`

func (r *opsRepository) ClearRequestErrors(ctx context.Context, before time.Time) (int64, error) {
	if before.IsZero() {
		return 0, fmt.Errorf("cleanup cutoff is required")
	}
	var ceiling int64
	if err := r.db.QueryRowContext(ctx, "SELECT COALESCE(MAX(id), 0) FROM ops_error_logs").Scan(&ceiling); err != nil {
		return 0, err
	}
	var total int64
	for {
		result, err := r.db.ExecContext(ctx, clearRequestErrorsBatchSQL, ceiling, before.UTC())
		if err != nil {
			return total, err
		}
		n, err := result.RowsAffected()
		if err != nil {
			return total, err
		}
		total += n
		if n == 0 {
			return total, nil
		}
	}
}
