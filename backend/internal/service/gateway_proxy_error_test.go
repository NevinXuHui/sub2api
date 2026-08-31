package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestIsTransientGatewayProxyError(t *testing.T) {
	nginx502 := []byte("<html>\r\n<head><title>502 Bad Gateway</title></head>\r\n<body>\r\n<center><h1>502 Bad Gateway</h1></center>\r\n<hr><center>nginx</center>\r\n</body>\r\n</html>\r\n")
	json502 := []byte(`{"type":"error","error":{"type":"api_error","message":"Internal server error"}}`)

	require.True(t, isTransientGatewayProxyError(http.StatusBadGateway, nginx502))
	require.True(t, isTransientGatewayProxyError(http.StatusBadGateway, nil))
	require.True(t, isTransientGatewayProxyError(http.StatusServiceUnavailable, []byte("<!DOCTYPE html><html></html>")))
	require.True(t, isTransientGatewayProxyError(http.StatusGatewayTimeout, []byte("504 Gateway Time-out")))
	require.False(t, isTransientGatewayProxyError(http.StatusBadGateway, json502))
	require.False(t, isTransientGatewayProxyError(http.StatusForbidden, nginx502))
	require.False(t, isTransientGatewayProxyError(http.StatusOK, nginx502))
}

func TestWithTransientProxyRetryMarksNginx502(t *testing.T) {
	nginx502 := []byte("<html><head><title>502 Bad Gateway</title></head><body><center>nginx</center></body></html>")
	err := withTransientProxyRetry(&UpstreamFailoverError{
		StatusCode:             http.StatusBadGateway,
		ResponseBody:           nginx502,
		RetryableOnSameAccount: false,
	})
	require.True(t, err.RetryableOnSameAccount)
	require.True(t, err.RequestScopedTransient)
	require.Equal(t, transientProxySameAccountRetries, err.SameAccountRetryMax)

	jsonErr := withTransientProxyRetry(&UpstreamFailoverError{
		StatusCode:   http.StatusBadGateway,
		ResponseBody: []byte(`{"error":"boom"}`),
	})
	require.False(t, jsonErr.RetryableOnSameAccount)
	require.False(t, jsonErr.RequestScopedTransient)
	require.Zero(t, jsonErr.SameAccountRetryMax)
}

func TestGatewayService_Forward_Nginx502MarksRequestScopedSameAccountRetry(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	body := []byte(`{"model":"claude-3-5-sonnet-latest","stream":false,"messages":[{"role":"user","content":"hello"}]}`)
	parsed, err := ParseGatewayRequest(NewRequestBodyRef(body), PlatformAnthropic)
	require.NoError(t, err)

	nginxBody := []byte("<html>\r\n<head><title>502 Bad Gateway</title></head>\r\n<body>\r\n<center><h1>502 Bad Gateway</h1></center>\r\n<hr><center>nginx</center>\r\n</body>\r\n</html>\r\n")
	upstream := &anthropicHTTPUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusBadGateway,
		Header:     http.Header{"Content-Type": []string{"text/html"}},
		Body:       io.NopCloser(bytes.NewReader(nginxBody)),
	}}
	cfg := &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}}
	svc := &GatewayService{
		cfg:                  cfg,
		responseHeaderFilter: compileResponseHeaderFilter(cfg),
		httpUpstream:         upstream,
		rateLimitService:     &RateLimitService{},
		deferredService:      &DeferredService{},
	}
	account := &Account{
		ID:          1374,
		Name:        "kiro-pool",
		Platform:    PlatformAnthropic,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":   "k",
			"base_url":  "https://kiro.local",
			"pool_mode": true,
		},
		Status:      StatusActive,
		Schedulable: true,
	}

	result, err := svc.Forward(context.Background(), c, account, parsed)
	require.Error(t, err)
	require.Nil(t, result)
	var failoverErr *UpstreamFailoverError
	require.True(t, errors.As(err, &failoverErr))
	require.Equal(t, http.StatusBadGateway, failoverErr.StatusCode)
	require.True(t, failoverErr.RetryableOnSameAccount)
	require.True(t, failoverErr.RequestScopedTransient)
	require.Equal(t, transientProxySameAccountRetries, failoverErr.SameAccountRetryMax)
}

func TestNewGrokUpstreamFailoverErrorNginx502(t *testing.T) {
	nginx502 := []byte("<html><head><title>502 Bad Gateway</title></head><body><center>nginx</center></body></html>")
	account := &Account{Platform: PlatformGrok, Type: AccountTypeAPIKey, Credentials: map[string]any{"pool_mode": true}}
	err := newGrokUpstreamFailoverError(account, http.StatusBadGateway, nginx502, nil)
	require.True(t, err.RetryableOnSameAccount)
	require.True(t, err.RequestScopedTransient)
	require.Equal(t, transientProxySameAccountRetries, err.SameAccountRetryMax)
}
