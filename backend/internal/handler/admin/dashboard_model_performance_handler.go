package admin

import (
	"net/http"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

// ModelPerformanceStats represents RPM/TPM for a single model
type ModelPerformanceStats struct {
	Model string `json:"model"`
	RPM   int64  `json:"rpm"`
	TPM   int64  `json:"tpm"`
}

// GetModelPerformance returns RPM/TPM stats for each model in the last 5 minutes
// GET /api/v1/admin/dashboard/model-performance
func (h *DashboardHandler) GetModelPerformance(c *gin.Context) {
	now := time.Now().UTC()
	// Calculate RPM/TPM for the last 5 minutes
	startTime := now.Add(-5 * time.Minute)
	endTime := now

	modelStats, err := h.dashboardService.GetModelStatsWithFilters(
		c.Request.Context(),
		startTime,
		endTime,
		0, // userID
		0, // apiKeyID
		0, // accountID
		0, // groupID
		nil, // requestType
		nil, // stream
		nil, // billingType
	)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to get model performance stats")
		return
	}

	// Calculate total RPM/TPM
	var totalRequests int64
	var totalTokens int64
	for _, stat := range modelStats {
		totalRequests += stat.Requests
		totalTokens += stat.TotalTokens
	}

	// RPM/TPM are per-minute averages over 5 minutes
	totalRPM := totalRequests / 5
	totalTPM := totalTokens / 5

	// Build result list
	result := []ModelPerformanceStats{
		{
			Model: "all",
			RPM:   totalRPM,
			TPM:   totalTPM,
		},
	}

	for _, stat := range modelStats {
		rpm := stat.Requests / 5
		tpm := stat.TotalTokens / 5
		result = append(result, ModelPerformanceStats{
			Model: stat.Model,
			RPM:   rpm,
			TPM:   tpm,
		})
	}

	response.Success(c, result)
}
