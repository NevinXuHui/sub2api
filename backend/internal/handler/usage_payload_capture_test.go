package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type recordingWriter struct {
	gin.ResponseWriter
	buf bytes.Buffer
}

func (w *recordingWriter) Write(p []byte) (int, error) {
	n, err := w.ResponseWriter.Write(p)
	if n > 0 {
		_, _ = w.buf.Write(p[:n])
	}
	return n, err
}

func (w *recordingWriter) WriteString(s string) (int, error) {
	n, err := w.ResponseWriter.WriteString(s)
	if n > 0 {
		_, _ = w.buf.WriteString(s[:n])
	}
	return n, err
}

func TestLimitedResponseTee_DoesNotTruncateClient(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	inner := &recordingWriter{ResponseWriter: c.Writer}
	tee := newLimitedResponseTee(inner, 8)

	payload := []byte(strings.Repeat("x", 32))
	n, err := tee.Write(payload)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if n != len(payload) {
		t.Fatalf("Write n=%d want %d", n, len(payload))
	}
	if inner.buf.Len() != len(payload) {
		t.Fatalf("client got %d bytes, want full %d", inner.buf.Len(), len(payload))
	}
	captured := tee.Captured()
	if len(captured) > 8 {
		t.Fatalf("captured %d bytes, want <= 8", len(captured))
	}
}

func TestCaptureHelpers_NoopWhenDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(`{"a":1}`)))
	if got := captureUsageRequestPayload(c, []byte(`{"a":1}`)); got != nil {
		t.Fatalf("expected nil request payload, got %q", got)
	}
	if got := snapshotUsageResponsePayload(c); got != nil {
		t.Fatalf("expected nil response payload, got %q", got)
	}
}
