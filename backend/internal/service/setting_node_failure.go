package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

const SettingKeyNodeFailureSettings = "node_failure_settings_v1"
const nodeFailureSettingsTTL = 5 * time.Second

type NodeFailureSettings struct {
	FailureRatePercent int `json:"failure_rate_percent" binding:"required,min=1,max=99"`
	WindowMinutes      int `json:"window_minutes" binding:"required,min=1,max=60"`
	FailureThreshold   int `json:"failure_threshold" binding:"required,min=1,max=100000"`
	CooldownMinutes    int `json:"cooldown_minutes" binding:"required,min=1,max=1440"`
}

type cachedNodeFailureSettings struct {
	settings  NodeFailureSettings
	expiresAt time.Time
}

func DefaultNodeFailureSettings() NodeFailureSettings {
	return NodeFailureSettings{WindowMinutes: 1, FailureThreshold: NodeFailureThreshold, CooldownMinutes: 3, FailureRatePercent: 40}
}

func (v NodeFailureSettings) Validate() error {
	if v.FailureRatePercent < 1 || v.FailureRatePercent > 99 {
		return fmt.Errorf("failure_rate_percent must be between 1 and 99")
	}
	if v.WindowMinutes < 1 || v.WindowMinutes > 60 {
		return fmt.Errorf("window_minutes must be between 1 and 60")
	}
	if v.FailureThreshold < 1 || v.FailureThreshold > 100000 {
		return fmt.Errorf("failure_threshold must be between 1 and 100000")
	}
	if v.CooldownMinutes < 1 || v.CooldownMinutes > 1440 {
		return fmt.Errorf("cooldown_minutes must be between 1 and 1440")
	}
	return nil
}

func (s *SettingService) GetNodeFailureSettings(ctx context.Context) (NodeFailureSettings, error) {
	if s == nil || s.settingRepo == nil {
		return DefaultNodeFailureSettings(), nil
	}
	s.nodeFailureSettingsMu.Lock()
	defer s.nodeFailureSettingsMu.Unlock()
	if cached := s.nodeFailureSettingsCache; cached != nil && time.Now().Before(cached.expiresAt) {
		return cached.settings, nil
	}
	value, err := s.settingRepo.GetValue(ctx, SettingKeyNodeFailureSettings)
	if err != nil && !errors.Is(err, ErrSettingNotFound) {
		return NodeFailureSettings{}, err
	}
	settings := DefaultNodeFailureSettings()
	if value != "" {
		stored := DefaultNodeFailureSettings()
		if json.Unmarshal([]byte(value), &stored) == nil && stored.Validate() == nil {
			settings = stored
		}
	}
	s.nodeFailureSettingsCache = &cachedNodeFailureSettings{settings: settings, expiresAt: time.Now().Add(nodeFailureSettingsTTL)}
	return settings, nil
}

func (s *SettingService) SetNodeFailureSettings(ctx context.Context, settings NodeFailureSettings) error {
	if err := settings.Validate(); err != nil {
		return err
	}
	data, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	s.nodeFailureSettingsMu.Lock()
	defer s.nodeFailureSettingsMu.Unlock()
	if err := s.settingRepo.Set(ctx, SettingKeyNodeFailureSettings, string(data)); err != nil {
		return err
	}
	s.nodeFailureSettingsCache = &cachedNodeFailureSettings{settings: settings, expiresAt: time.Now().Add(nodeFailureSettingsTTL)}
	return nil
}

func (s *RateLimitService) nodeFailureSettings(ctx context.Context) NodeFailureSettings {
	settings, err := s.settingService.GetNodeFailureSettings(ctx)
	if err == nil {
		return settings
	}
	// Keep the last valid policy during a temporary settings-store outage.
	if s.settingService != nil {
		s.settingService.nodeFailureSettingsMu.Lock()
		defer s.settingService.nodeFailureSettingsMu.Unlock()
		if cached := s.settingService.nodeFailureSettingsCache; cached != nil {
			return cached.settings
		}
	}
	return DefaultNodeFailureSettings()
}
