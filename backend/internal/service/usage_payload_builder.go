package service

import "strings"

// buildUsagePayload 从 usage log + input payload 构造 UsageLogPayload
func buildUsagePayload(log *UsageLog, input *recordUsageCoreInput) *UsageLogPayload {
	if log == nil || input == nil || log.ID == 0 {
		return nil
	}

	reqResult := CaptureUsagePayload(input.RequestPayload, len(input.RequestPayload))
	respResult := CaptureUsagePayload(input.ResponsePayload, len(input.ResponsePayload))

	return &UsageLogPayload{
		UsageLogID: log.ID,
		RequestID:  log.RequestID,
		APIKeyID:   log.APIKeyID,
		UserID:     log.UserID,

		RequestBody:  reqResult.Body,
		ResponseBody: respResult.Body,

		RequestTruncated:  reqResult.Truncated,
		ResponseTruncated: respResult.Truncated,

		RequestBytes:  reqResult.Bytes,
		ResponseBytes: respResult.Bytes,
	}
}

// buildOpenAIUsagePayload 从 OpenAI usage log + input payload 构造 UsageLogPayload
func buildOpenAIUsagePayload(log *UsageLog, reqPayload, respPayload []byte) *UsageLogPayload {
	if log == nil || log.ID == 0 {
		return nil
	}

	reqResult := CaptureUsagePayload(reqPayload, len(reqPayload))
	respResult := CaptureUsagePayload(respPayload, len(respPayload))

	return &UsageLogPayload{
		UsageLogID: log.ID,
		RequestID:  log.RequestID,
		APIKeyID:   log.APIKeyID,
		UserID:     log.UserID,

		RequestBody:  reqResult.Body,
		ResponseBody: respResult.Body,

		RequestTruncated:  reqResult.Truncated,
		ResponseTruncated: respResult.Truncated,

		RequestBytes:  reqResult.Bytes,
		ResponseBytes: respResult.Bytes,
	}
}

// shouldWritePayload 判断是否需要写入 payload（去除 nil/空切片）
func shouldWritePayload(reqPayload, respPayload []byte) bool {
	hasReq := len(reqPayload) > 0 && strings.TrimSpace(string(reqPayload)) != ""
	hasResp := len(respPayload) > 0 && strings.TrimSpace(string(respPayload)) != ""
	return hasReq || hasResp
}
