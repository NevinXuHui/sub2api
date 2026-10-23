package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"testing"
)

func TestParseMinFirstTokenMS(t *testing.T) {
	for _, tc := range []struct {
		raw     string
		want    int
		invalid bool
	}{
		{"", -1, false}, {"0", 0, false}, {"2500", 2500, false}, {"-1", 0, true}, {"2.5", 0, true}, {"NaN", 0, true}, {"2147483648", 0, true},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("GET", "/?min_first_token_ms="+tc.raw, nil)
			got, err := parseMinFirstTokenMS(c)
			if tc.invalid {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tc.want < 0 {
				require.Nil(t, got)
			} else {
				require.Equal(t, tc.want, *got)
			}
		})
	}
}
func TestUsageStatsCacheSeparatesMinFirstToken(t *testing.T) {
	zero, threshold := 0, 2500
	a := usageStatsCacheKey(usagestats.UsageLogFilters{})
	b := usageStatsCacheKey(usagestats.UsageLogFilters{MinFirstTokenMS: &zero})
	c := usageStatsCacheKey(usagestats.UsageLogFilters{MinFirstTokenMS: &threshold})
	require.NotEqual(t, a, b)
	require.NotEqual(t, b, c)
}
