package repository

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

type healthErrorBody struct{ err error }

func (b *healthErrorBody) Read([]byte) (int, error) { return 0, b.err }
func (b *healthErrorBody) Close() error             { return nil }

func TestOpenAIHTTP2BodyFailuresActivateFutureFallback(t *testing.T) {
	for _, route := range []string{"http://proxy.local:8080", directProxyKey} {
		t.Run(route, func(t *testing.T) {
			svc := &httpUpstreamService{cfg: &config.Config{Gateway: config.GatewayConfig{OpenAIHTTP2: config.GatewayOpenAIHTTP2Config{
				Enabled: true, AllowProxyFallbackToHTTP1: true, FallbackErrorThreshold: 2, FallbackWindowSeconds: 60, FallbackTTLSeconds: 600,
			}}}}
			entry := &upstreamClientEntry{client: &http.Client{}, proxyKey: route, protocolMode: upstreamProtocolModeOpenAIH2}
			for i := 0; i < 2; i++ {
				original := &healthErrorBody{err: errors.New("http2: client connection lost")}
				body := svc.observeOpenAIHTTP2Body(context.Background(), service.HTTPUpstreamProfileOpenAI, entry, original)
				_, err := body.Read(make([]byte, 10))
				require.ErrorIs(t, err, original.err)
				// A subsequent read must not count the same broken stream twice.
				_, _ = body.Read(make([]byte, 10))
				require.Equal(t, i == 1, svc.isOpenAIHTTP2FallbackActive(entry.proxyKey))
			}
			if route == directProxyKey {
				require.Equal(t, upstreamProtocolModeOpenAIH1Fallback, svc.resolveProtocolMode(service.HTTPUpstreamProfileOpenAI, directProxyKey, nil))
			}
		})
	}
}

func TestOpenAIHTTP2BodyHealthIgnoresCancellationAndSuccessfulEOF(t *testing.T) {
	for _, tc := range []struct {
		name   string
		cause  error
		cancel bool
	}{
		{"normal EOF", io.EOF, false}, {"cancel", context.Canceled, false},
		{"deadline", context.DeadlineExceeded, false}, {"cancelled connection", errors.New("http2: client connection lost"), true},
		{"unrelated payload error", errors.New("invalid compressed stream"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := &httpUpstreamService{cfg: &config.Config{Gateway: config.GatewayConfig{OpenAIHTTP2: config.GatewayOpenAIHTTP2Config{
				Enabled: true, AllowProxyFallbackToHTTP1: true, FallbackErrorThreshold: 1,
			}}}}
			entry := &upstreamClientEntry{client: &http.Client{}, proxyKey: "http://proxy.local:8080", protocolMode: upstreamProtocolModeOpenAIH2}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tc.cancel {
				cancel()
			}
			body := svc.observeOpenAIHTTP2Body(ctx, service.HTTPUpstreamProfileOpenAI, entry, &healthErrorBody{err: tc.cause})
			_, _ = body.Read(make([]byte, 10))
			require.False(t, svc.isOpenAIHTTP2FallbackActive(entry.proxyKey))
		})
	}
	// Non-H2 callers retain the original reader and its behavior.
	svc := &httpUpstreamService{}
	body := io.NopCloser(strings.NewReader("unchanged"))
	require.Equal(t, body, svc.observeOpenAIHTTP2Body(context.Background(), service.HTTPUpstreamProfileDefault, &upstreamClientEntry{}, body))
}
