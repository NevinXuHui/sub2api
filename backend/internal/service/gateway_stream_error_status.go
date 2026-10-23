package service

import (
	"net/http"

	"github.com/tidwall/gjson"
)

// Gateways may wrap an Anthropic error in api_error.message as encoded JSON.
// Only structured error types determine status; prose mentioning 403/overload
// must never turn a transport failure into an authorization failure.
func anthropicStreamErrorStatus(body []byte) int {
	payload := gjson.ParseBytes(body)
	status := http.StatusBadGateway
	for depth := 0; depth < 5; depth++ {
		kind := payload.Get("error.type").String()
		if kind == "" {
			kind = payload.Get("type").String()
		}
		switch kind {
		case "permission_error":
			status = http.StatusForbidden
		case "authentication_error":
			status = http.StatusUnauthorized
		case "service_unavailable_error":
			status = http.StatusServiceUnavailable
		case "overloaded_error":
			status = 529
		case "rate_limit_error":
			status = http.StatusTooManyRequests
		case "invalid_request_error", "request_too_large":
			status = http.StatusBadRequest
		}
		nested := payload.Get("error.message")
		if !nested.Exists() {
			nested = payload.Get("message")
		}
		if nested.Type != gjson.String || !gjson.Valid(nested.String()) {
			break
		}
		next := gjson.Parse(nested.String())
		if !next.IsObject() {
			break
		}
		payload = next
	}
	return status
}
