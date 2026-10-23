package service

import (
	"context"
	"testing"
	"time"
)

func TestClearRequestErrorsValidationAndCutoff(t *testing.T) {
	calls := 0
	before := time.Now().Add(-time.Minute)
	repo := &opsRepoMock{ClearRequestErrorsFn: func(ctx context.Context, cutoff time.Time) (int64, error) {
		calls++
		if !cutoff.Equal(before) {
			t.Fatal("cutoff changed")
		}
		return 12, nil
	}}
	svc := NewOpsService(repo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	for _, cutoff := range []time.Time{{}, time.Now().Add(time.Hour)} {
		if _, err := svc.ClearRequestErrors(context.Background(), cutoff, 1); err == nil {
			t.Fatal("invalid cutoff accepted")
		}
	}
	if _, err := svc.ClearRequestErrors(context.Background(), before, 0); err == nil {
		t.Fatal("invalid operator accepted")
	}
	if calls != 0 {
		t.Fatal("invalid input reached repository")
	}
	n, err := svc.ClearRequestErrors(context.Background(), before, 1)
	if err != nil || n != 12 || calls != 1 {
		t.Fatalf("n=%d calls=%d err=%v", n, calls, err)
	}
}
