package handler

import (
	"net/http"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

// GetUsagePayload 用户获取自己的使用记录请求/响应正文
// GET /api/v1/usage/:id/payload
func (h *UsageHandler) GetUsagePayload(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	// 校验用户可见开关
	if h.settingService == nil || !h.settingService.IsUserUsagePayloadViewAllowed(c.Request.Context()) {
		response.Forbidden(c, "Usage payload view is disabled")
		return
	}

	usageID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || usageID <= 0 {
		response.BadRequest(c, "Invalid usage ID")
		return
	}

	// 校验归属
	usageLog, err := h.usageService.GetByID(c.Request.Context(), usageID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if usageLog.UserID != subject.UserID {
		response.Forbidden(c, "Not authorized to access this record")
		return
	}

	// 查询 payload
	payload, err := h.usageService.GetPayloadByUsageLogID(c.Request.Context(), usageID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to retrieve payload")
		return
	}

	response.Success(c, dto.UsagePayloadFromService(payload))
}
