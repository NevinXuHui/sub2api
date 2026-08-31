package service

import (
	"context"
	"log/slog"
	"strconv"
	"sync/atomic"
	"time"

	"golang.org/x/sync/singleflight"
)

type cachedUsagePayloadCapture struct {
	enabled   bool
	maxBytes  int
	expiresAt int64
}

var (
	usagePayloadCaptureCache atomic.Value // *cachedUsagePayloadCapture
	usagePayloadCaptureSF    singleflight.Group
)

const (
	usagePayloadCaptureCacheTTL = 60 * time.Second
	usagePayloadCaptureErrorTTL = 5 * time.Second
	usagePayloadCaptureDBTimeout = 5 * time.Second
)

// InvalidateUsagePayloadCaptureCache 设置更新后立刻丢弃采集开关缓存。
func (s *SettingService) InvalidateUsagePayloadCaptureCache() {
	usagePayloadCaptureSF.Forget("usage_payload_capture")
	usagePayloadCaptureCache.Store(&cachedUsagePayloadCapture{expiresAt: 0})
}

// UsagePayloadCaptureConfig 热路径读取采集开关与上限。失败默认关闭，避免关采集时打 DB。
func (s *SettingService) UsagePayloadCaptureConfig(ctx context.Context) (enabled bool, maxBytes int) {
	if cached, ok := usagePayloadCaptureCache.Load().(*cachedUsagePayloadCapture); ok {
		if time.Now().UnixNano() < cached.expiresAt {
			return cached.enabled, cached.maxBytes
		}
	}

	type cfg struct {
		enabled  bool
		maxBytes int
	}
	result, err, _ := usagePayloadCaptureSF.Do("usage_payload_capture", func() (any, error) {
		if cached, ok := usagePayloadCaptureCache.Load().(*cachedUsagePayloadCapture); ok {
			if time.Now().UnixNano() < cached.expiresAt {
				return cfg{cached.enabled, cached.maxBytes}, nil
			}
		}
		dbCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), usagePayloadCaptureDBTimeout)
		defer cancel()
		values, err := s.settingRepo.GetMultiple(dbCtx, []string{
			SettingKeyUsagePayloadCaptureEnabled,
			SettingKeyUsagePayloadMaxBytes,
		})
		if err != nil {
			slog.Warn("failed to get usage payload capture settings, defaulting to disabled", "error", err)
			usagePayloadCaptureCache.Store(&cachedUsagePayloadCapture{
				enabled:   false,
				maxBytes:  65536,
				expiresAt: time.Now().Add(usagePayloadCaptureErrorTTL).UnixNano(),
			})
			return cfg{false, 65536}, nil
		}
		enabled := values[SettingKeyUsagePayloadCaptureEnabled] == "true"
		maxBytes := 65536
		if raw := values[SettingKeyUsagePayloadMaxBytes]; raw != "" {
			if parsed, parseErr := strconv.Atoi(raw); parseErr == nil {
				maxBytes = ClampPayloadMaxBytes(parsed)
			}
		}
		usagePayloadCaptureCache.Store(&cachedUsagePayloadCapture{
			enabled:   enabled,
			maxBytes:  maxBytes,
			expiresAt: time.Now().Add(usagePayloadCaptureCacheTTL).UnixNano(),
		})
		return cfg{enabled, maxBytes}, nil
	})
	if err != nil || result == nil {
		return false, 65536
	}
	parsed := result.(cfg)
	return parsed.enabled, parsed.maxBytes
}
