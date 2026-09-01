package service

import (
	"bytes"
	"net/http"
)

// transientProxySameAccountRetries covers a typical container restart window
// (~30s) with handler exponential backoff (0.5+1+2+4+8+8s).
const transientProxySameAccountRetries = 6

// isTransientGatewayProxyError reports reverse-proxy / empty 502/503/504 pages
// (nginx HTML during kiro-rs or grok2api restarts). JSON upstream errors are
// not matched so real 5xx still follow account failover.
func isTransientGatewayProxyError(statusCode int, body []byte) bool {
	switch statusCode {
	case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
	default:
		return false
	}
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return true
	}
	lower := bytes.ToLower(trimmed)
	if bytes.HasPrefix(lower, []byte("<!doctype html")) || bytes.HasPrefix(lower, []byte("<html")) {
		return true
	}
	if bytes.Contains(lower, []byte("<center>nginx</center")) {
		return true
	}
	if bytes.Contains(lower, []byte("502 bad gateway")) ||
		bytes.Contains(lower, []byte("503 service temporarily unavailable")) ||
		bytes.Contains(lower, []byte("504 gateway time-out")) {
		return true
	}
	return false
}

func withTransientProxyRetry(err *UpstreamFailoverError) *UpstreamFailoverError {
	if err == nil || !isTransientGatewayProxyError(err.StatusCode, err.ResponseBody) {
		return err
	}
	err.RetryableOnSameAccount = true
	err.RequestScopedTransient = true
	err.SameAccountRetryMax = transientProxySameAccountRetries
	return err
}

func newGrokUpstreamFailoverError(account *Account, statusCode int, body []byte, headers http.Header) *UpstreamFailoverError {
	_ = account
	return withTransientProxyRetry(&UpstreamFailoverError{
		StatusCode:      statusCode,
		ResponseBody:    body,
		ResponseHeaders: headers,
	})
}
