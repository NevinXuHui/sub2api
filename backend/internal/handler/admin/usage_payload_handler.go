package admin

import (
	"net/http"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

// GetUsagePayload 管理员获取使用记录请求/响应正文
// GET /api/v1/admin/usage/:id/payload
func (h *UsageHandler) GetUsagePayload(c *gin.Context) {
	usageID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || usageID <= 0 {
		response.BadRequest(c, "Invalid usage ID")
		return
	}

	// 查询 payload（管理员无需校验归属）
	payload, err := h.usageService.GetPayloadByUsageLogID(c.Request.Context(), usageID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to retrieve payload")
		return
	}

	response.Success(c, dto.UsagePayloadFromService(payload))
}
