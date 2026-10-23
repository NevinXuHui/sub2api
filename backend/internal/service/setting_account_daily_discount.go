package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/shopspring/decimal"
)

const SettingKeyAccountDailyDiscount = "account_daily_discount_v1"

type AccountDailyDiscountTier struct {
	ThresholdUSD float64 `json:"threshold_usd"`
	Discount     float64 `json:"discount"` // 8 means 80% of the existing user multiplier.
}

type AccountDailyDiscountSettings struct {
	Enabled bool                       `json:"enabled"`
	Tiers   []AccountDailyDiscountTier `json:"tiers"`
}

type cachedAccountDailyDiscount struct {
	value     AccountDailyDiscountSettings
	expiresAt time.Time
}

func (v AccountDailyDiscountSettings) Clone() AccountDailyDiscountSettings {
	v.Tiers = append([]AccountDailyDiscountTier{}, v.Tiers...)
	return v
}

func (v AccountDailyDiscountSettings) Validate() error {
	if len(v.Tiers) > 20 || (v.Enabled && len(v.Tiers) == 0) {
		return fmt.Errorf("configure 1–20 discount tiers before enabling")
	}
	seen := map[float64]bool{}
	for _, tier := range v.Tiers {
		if math.IsNaN(tier.ThresholdUSD) || math.IsInf(tier.ThresholdUSD, 0) || tier.ThresholdUSD < 0 || tier.ThresholdUSD > 1e9 {
			return fmt.Errorf("threshold_usd must be between 0 and 1000000000")
		}
		if math.IsNaN(tier.Discount) || math.IsInf(tier.Discount, 0) || tier.Discount <= 0 || tier.Discount > 10 {
			return fmt.Errorf("discount must be greater than 0 and at most 10")
		}
		if decimal.NewFromFloat(tier.ThresholdUSD).Round(8).InexactFloat64() != tier.ThresholdUSD || decimal.NewFromFloat(tier.Discount).Round(2).InexactFloat64() != tier.Discount {
			return fmt.Errorf("threshold supports 8 decimal places; discount supports 2")
		}
		if seen[tier.ThresholdUSD] {
			return fmt.Errorf("income thresholds must be unique")
		}
		seen[tier.ThresholdUSD] = true
	}
	return nil
}

// Factor selects one highest crossed threshold; tiers never multiply together.
func (v AccountDailyDiscountSettings) Factor(revenue decimal.Decimal) float64 {
	factor, highest := 1.0, -1.0
	if !v.Enabled {
		return factor
	}
	for _, tier := range v.Tiers {
		if tier.ThresholdUSD > highest && revenue.GreaterThan(decimal.NewFromFloat(tier.ThresholdUSD)) {
			highest = tier.ThresholdUSD
			factor = decimal.NewFromFloat(tier.Discount).Div(decimal.NewFromInt(10)).InexactFloat64()
		}
	}
	return factor
}

func (s *SettingService) GetAccountDailyDiscountSettings(ctx context.Context) (AccountDailyDiscountSettings, error) {
	defaults := AccountDailyDiscountSettings{Tiers: []AccountDailyDiscountTier{}}
	if s == nil || s.settingRepo == nil {
		return defaults, nil
	}
	s.accountDailyDiscountMu.Lock()
	defer s.accountDailyDiscountMu.Unlock()
	if c := s.accountDailyDiscountCache; c != nil && time.Now().Before(c.expiresAt) {
		return c.value.Clone(), nil
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyAccountDailyDiscount)
	if err != nil && !errors.Is(err, ErrSettingNotFound) {
		return defaults, err
	}
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), &defaults); err != nil {
			return defaults, err
		}
		if err := defaults.Validate(); err != nil {
			return defaults, err
		}
	}
	s.accountDailyDiscountCache = &cachedAccountDailyDiscount{value: defaults.Clone(), expiresAt: time.Now().Add(5 * time.Second)}
	return defaults.Clone(), nil
}

func (s *SettingService) SetAccountDailyDiscountSettings(ctx context.Context, v AccountDailyDiscountSettings) error {
	if err := v.Validate(); err != nil {
		return err
	}
	v = v.Clone()
	sort.Slice(v.Tiers, func(i, j int) bool { return v.Tiers[i].ThresholdUSD < v.Tiers[j].ThresholdUSD })
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	s.accountDailyDiscountMu.Lock()
	defer s.accountDailyDiscountMu.Unlock()
	if err := s.settingRepo.Set(ctx, SettingKeyAccountDailyDiscount, string(raw)); err != nil {
		return err
	}
	s.accountDailyDiscountCache = &cachedAccountDailyDiscount{value: v, expiresAt: time.Now().Add(5 * time.Second)}
	return nil
}

// Persisted alongside the actual money transaction, including undiscounted
// samples while enabled, so asynchronous usage-log writes cannot shift a tier.
type AccountDailyDiscountReceipt struct {
	Factor     float64
	ActualCost float64
}
