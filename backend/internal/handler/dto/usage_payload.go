package dto

import "github.com/Wei-Shaw/sub2api/internal/service"

// UsagePayload 使用记录请求/响应正文 DTO
type UsagePayload struct {
	RequestBody       string `json:"request_body"`
	ResponseBody      string `json:"response_body"`
	RequestTruncated  bool   `json:"request_truncated"`
	ResponseTruncated bool   `json:"response_truncated"`
	RequestBytes      int    `json:"request_bytes"`
	ResponseBytes     int    `json:"response_bytes"`
	Captured          bool   `json:"captured"` // 是否采集（未采集返回空正文 + captured=false）
}

// UsagePayloadFromService 从 service 层转换
func UsagePayloadFromService(payload *service.UsageLogPayload) *UsagePayload {
	if payload == nil {
		return &UsagePayload{Captured: false}
	}

	return &UsagePayload{
		RequestBody:       payload.RequestBody,
		ResponseBody:      payload.ResponseBody,
		RequestTruncated:  payload.RequestTruncated,
		ResponseTruncated: payload.ResponseTruncated,
		RequestBytes:      payload.RequestBytes,
		ResponseBytes:     payload.ResponseBytes,
		Captured:          true,
	}
}
