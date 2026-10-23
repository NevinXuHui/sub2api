package service

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAnthropicStreamErrorStatus(t *testing.T) {
	wrap := func(kind, message string) string {
		b, err := json.Marshal(map[string]any{"type": "error", "error": map[string]string{"type": kind, "message": message}})
		require.NoError(t, err)
		return string(b)
	}
	overload := wrap("overloaded_error", "Upstream service overloaded. Retry later.")
	for _, tc := range []struct {
		name, body string
		status     int
	}{
		{"plain overload", overload, 529},
		{"temporary refresh overload", wrap("overloaded_error", "Token 准备暂时不可用: 刷新出口正在忙碌"), 529},
		{"service unavailable", wrap("service_unavailable_error", "refresh unavailable"), http.StatusServiceUnavailable},
		{"nested service unavailable", wrap("api_error", wrap("service_unavailable_error", "refresh unavailable")), http.StatusServiceUnavailable},
		{"nested overload", wrap("api_error", overload), 529},
		{"deep overload", wrap("api_error", wrap("api_error", overload)), 529},
		{"wrapped permission", wrap("api_error", wrap("permission_error", "denied")), http.StatusForbidden},
		{"wrapped invalid parameters", wrap("api_error", wrap("invalid_request_error", "bad query")), http.StatusBadRequest},
		{"prose is not structured status", wrap("api_error", "403 Forbidden; upstream overloaded"), http.StatusBadGateway},
		{"transport", wrap("stream_read_error", "connection reset by peer"), http.StatusBadGateway},
		{"malformed nested body", wrap("api_error", `{"error":`), http.StatusBadGateway},
	} {
		t.Run(tc.name, func(t *testing.T) { require.Equal(t, tc.status, anthropicStreamErrorStatus([]byte(tc.body))) })
	}
	body := overload
	for i := 0; i < 20; i++ {
		body = wrap("api_error", body)
	}
	require.Equal(t, http.StatusBadGateway, anthropicStreamErrorStatus([]byte(body)), "untrusted nesting must be bounded")
}
