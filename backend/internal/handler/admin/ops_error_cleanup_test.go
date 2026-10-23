package admin

import (
	"context"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type errorCleanupRepo struct {
	service.OpsRepository
	calls int
}

func (r *errorCleanupRepo) ClearRequestErrors(context.Context, time.Time) (int64, error) {
	r.calls++
	return 9, nil
}
func TestClearRequestErrorsRequiresAuthConfirmationAndCutoff(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		auth       bool
		status     int
	}{
		{"unauthenticated", `{"confirm":true,"before":"2026-01-01T00:00:00Z"}`, false, 401},
		{"unconfirmed", `{"before":"2026-01-01T00:00:00Z"}`, true, 400},
		{"missing cutoff", `{"confirm":true}`, true, 400},
		{"invalid cutoff", `{"confirm":true,"before":"bad"}`, true, 400},
		{"valid", `{"confirm":true,"before":"2026-01-01T00:00:00Z"}`, true, 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			repo := &errorCleanupRepo{}
			h := NewOpsHandler(service.NewOpsService(repo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil))
			r := gin.New()
			if tc.auth {
				r.Use(func(c *gin.Context) {
					c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 99})
					c.Next()
				})
			}
			r.POST("/clear", h.ClearRequestErrors)
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/clear", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)
			if w.Code != tc.status {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
			if (repo.calls > 0) != (tc.status == 200) {
				t.Fatal("unexpected cleanup call")
			}
		})
	}
}
