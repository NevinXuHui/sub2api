package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const (
	ctxKeyUsagePayloadTee = "usage_payload_tee"
	ctxKeyUsagePayloadMax = "usage_payload_max"
)

// limitedResponseTee 把写入客户端的字节有界拷贝一份；超出上限仍完整转发给客户端。
type limitedResponseTee struct {
	gin.ResponseWriter
	buf      *service.LimitedBuffer
	maxBytes int
}

func newLimitedResponseTee(w gin.ResponseWriter, maxBytes int) *limitedResponseTee {
	return &limitedResponseTee{
		ResponseWriter: w,
		buf:            service.NewLimitedBuffer(maxBytes),
		maxBytes:       maxBytes,
	}
}

func (w *limitedResponseTee) Write(p []byte) (int, error) {
	n, err := w.ResponseWriter.Write(p)
	if n > 0 {
		_, _ = w.buf.Write(p[:n])
	}
	return n, err
}

func (w *limitedResponseTee) WriteString(s string) (int, error) {
	n, err := w.ResponseWriter.WriteString(s)
	if n > 0 {
		_, _ = w.buf.Write([]byte(s[:n]))
	}
	return n, err
}

func (w *limitedResponseTee) Captured() []byte {
	result := service.CaptureUsagePayload(w.buf.Bytes(), w.maxBytes)
	return []byte(result.Body)
}

// UsagePayloadCaptureMiddleware 仅在采集开启时包装 ResponseWriter。关闭时零拷贝。
func UsagePayloadCaptureMiddleware(settingService *service.SettingService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if settingService == nil {
			c.Next()
			return
		}
		enabled, maxBytes := settingService.UsagePayloadCaptureConfig(c.Request.Context())
		if !enabled || maxBytes <= 0 {
			c.Next()
			return
		}
		tee := newLimitedResponseTee(c.Writer, maxBytes)
		c.Writer = tee
		c.Set(ctxKeyUsagePayloadTee, tee)
		c.Set(ctxKeyUsagePayloadMax, maxBytes)
		c.Next()
	}
}

func captureUsageRequestPayload(c *gin.Context, body []byte) []byte {
	if c == nil {
		return nil
	}
	raw, ok := c.Get(ctxKeyUsagePayloadMax)
	if !ok {
		return nil
	}
	maxBytes, _ := raw.(int)
	return service.ExtractPayloadIfEnabled(true, maxBytes, body)
}

func snapshotUsageResponsePayload(c *gin.Context) []byte {
	if c == nil {
		return nil
	}
	raw, ok := c.Get(ctxKeyUsagePayloadTee)
	if !ok {
		return nil
	}
	tee, ok := raw.(*limitedResponseTee)
	if !ok || tee == nil {
		return nil
	}
	return tee.Captured()
}

func snapshotUsagePayloads(c *gin.Context, requestBody []byte) (req, resp []byte) {
	return captureUsageRequestPayload(c, requestBody), snapshotUsageResponsePayload(c)
}
