package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

// ClearRequestErrors is admin-only; no GET/list operation deletes records.
func (h *OpsHandler) ClearRequestErrors(c *gin.Context) {
	if h.opsService == nil {
		response.Error(c, http.StatusServiceUnavailable, "Ops service not available")
		return
	}
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Error(c, http.StatusUnauthorized, "Unauthorized")
		return
	}
	var req struct {
		Confirm bool      `json:"confirm"`
		Before  time.Time `json:"before"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || !req.Confirm {
		response.BadRequest(c, "Explicit confirmation and a valid cutoff are required")
		return
	}
	deleted, err := h.opsService.ClearRequestErrors(c.Request.Context(), req.Before, subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": deleted})
}
