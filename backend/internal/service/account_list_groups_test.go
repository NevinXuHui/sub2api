package service

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestParseAccountListGroups(t *testing.T) {
	for _, tc := range []struct {
		raw    string
		single int64
		multi  []int64
	}{
		{"", 0, nil}, {"7", 7, nil}, {"ungrouped", -1, nil},
		{"7,9,7", 0, []int64{7, 9}}, {"7,ungrouped", 0, []int64{7, -1}},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			ctx, id, err := ParseAccountListGroups(context.Background(), tc.raw)
			require.NoError(t, err)
			require.Equal(t, tc.single, id)
			require.Equal(t, tc.multi, AccountListGroups(ctx))
		})
	}
	for _, raw := range []string{"7,", "7,no", "-2", "7,0"} {
		_, _, err := ParseAccountListGroups(context.Background(), raw)
		require.Error(t, err)
	}
}
