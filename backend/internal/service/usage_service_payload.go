package service

import "context"

// GetPayloadByUsageLogID 查询用量正文（repo 层返回 nil 不是错误，表示未采集）
func (s *UsageService) GetPayloadByUsageLogID(ctx context.Context, usageLogID int64) (*UsageLogPayload, error) {
	if s.usagePayloadRepo == nil {
		return nil, nil
	}
	return s.usagePayloadRepo.GetPayloadByUsageLogID(ctx, usageLogID)
}
