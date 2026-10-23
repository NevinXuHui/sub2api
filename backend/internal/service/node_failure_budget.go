package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tidwall/gjson"
)

const nodeFailureClientDisconnectedKey = "node_failure_client_disconnected_v1"

const nodeFailureSSEKey = "node_failure_sse_error_v1"

const NodeFailureWindow = time.Minute
const NodeFailureThreshold = 100
const NodeFailureCooldown = 3 * time.Minute
const NodeFailureRecoveryKey = "node_failure_recovery_v1"

// Persisted with the account so restart/replica changes retain the original
// concurrency. The normal scheduler's temp-unsched expiry opens the probe.
type NodeFailureRecovery struct {
	OriginalConcurrency int       `json:"original_concurrency"`
	ResumeAfter         time.Time `json:"resume_after"`
	Generation          string    `json:"generation"`
}

func GetNodeFailureRecovery(account *Account) *NodeFailureRecovery {
	if account == nil {
		return nil
	}
	value, ok := account.Extra[NodeFailureRecoveryKey]
	if !ok {
		return nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	var state NodeFailureRecovery
	if json.Unmarshal(raw, &state) != nil || state.Generation == "" || state.ResumeAfter.IsZero() || state.OriginalConcurrency < 0 {
		return nil
	}
	return &state
}

// NodeFailureBudgetCache uses server time and a rolling window shared by all
// instances. A trip lease prevents concurrent requests from disabling twice.
type NodeFailureBudgetCache interface {
	ClaimNodeFailureProbe(ctx context.Context, accountID int64, generation string, interval time.Duration) (bool, error)
	RecordNodeSuccess(ctx context.Context, accountID int64, requestID string, started time.Time, settings NodeFailureSettings) error
	RecordNodeFailure(ctx context.Context, accountID int64, requestID string, started time.Time, settings NodeFailureSettings) (count int64, trip bool, err error)
	FinishNodeFailureTrip(ctx context.Context, accountID int64, requestID string, disabled bool) error
}

type nodeFailureAccountRecovery interface {
	ParkNodeForFailureBudget(context.Context, *Account, time.Time, string) (bool, error)
	RestoreNodeAfterFailureProbe(context.Context, *Account) (bool, error)
}

type nodeFailureRequest struct {
	id      string
	started time.Time
}

func nodeFailureEligible(account *Account) bool {
	if account == nil || account.Platform != PlatformAnthropic || account.Type != AccountTypeAPIKey {
		return false
	}
	endpoint, err := url.Parse(account.GetCredential("base_url"))
	return err == nil && endpoint.Hostname() != "" && !strings.EqualFold(endpoint.Hostname(), "api.anthropic.com")
}

// Register once at Forward's boundary. Pool retries and handler failovers on
// the same Gin request reuse the same ID, rather than inflating the count.
func (s *GatewayService) beginNodeFailureObservation(c *gin.Context, account *Account) func(context.Context, *ForwardResult, error) {
	if c == nil || s.rateLimitService == nil || s.rateLimitService.nodeFailureBudget == nil || !nodeFailureEligible(account) {
		return func(context.Context, *ForwardResult, error) {}
	}
	const key = "node_failure_budget_request_v1"
	value, exists := c.Get(key)
	request, ok := value.(nodeFailureRequest)
	if !exists || !ok {
		request = nodeFailureRequest{id: uuid.NewString(), started: time.Now()}
		c.Set(key, request)
	}
	// Snapshot the event count so an earlier account's failure cannot poison
	// this account after a local validation error.
	before := len(nodeFailureEvents(c))
	c.Set(nodeFailureSSEKey, "")
	c.Set(nodeFailureClientDisconnectedKey, false)
	return func(ctx context.Context, result *ForwardResult, err error) {
		probe := GetNodeFailureRecovery(account) != nil
		probeSuccess := false
		if probe {
			defer func() { s.rateLimitService.finishNodeFailureProbe(ctx, account, request, probeSuccess) }()
		}
		if ctx.Err() != nil || errors.Is(err, context.Canceled) || c.GetBool(nodeFailureClientDisconnectedKey) ||
			(result != nil && result.ClientDisconnect) || IsOpsRequestScopedError(c) {
			return
		}
		// Passthrough forwards SSE bytes unchanged, including error frames.
		// Observe their semantics even when HTTP/message_stop looks successful.
		if raw := c.GetString(nodeFailureSSEKey); raw != "" {
			err = &sseStreamErrorEventError{RawData: raw}
		}
		if err == nil {
			if result != nil && result.UpstreamHeaders != nil {
				if GetNodeFailureRecovery(account) != nil {
					probeSuccess = true
				} else {
					s.rateLimitService.recordNodeSuccess(ctx, account, request)
				}
			}
			return
		}
		events := nodeFailureEvents(c)
		if before > len(events) {
			before = len(events)
		}
		var failover *UpstreamFailoverError
		probeRateLimited := GetNodeFailureRecovery(account) != nil && errors.As(err, &failover) && failover.StatusCode == 429
		if !probeRateLimited && !isNodeFailure(err, account.ID, events[before:]) {
			return
		}
		if GetNodeFailureRecovery(account) != nil {
			return
		}
		s.rateLimitService.recordNodeFailure(ctx, account, request)
	}
}

func nodeFailureEvents(c *gin.Context) []*OpsUpstreamErrorEvent {
	value, _ := c.Get(OpsUpstreamErrorsKey)
	events, _ := value.([]*OpsUpstreamErrorEvent)
	return events
}

func nodeFailureStatus(status int, body string) bool {
	typ := gjson.Get(body, "error.type").String()
	message := strings.ToLower(extractUpstreamErrorMessage([]byte(body)))
	if typ == "invalid_request_error" || typ == "not_found_error" || typ == "request_too_large" ||
		strings.Contains(message, "unsupported content block") ||
		strings.Contains(message, "blocked by content filtering policy") ||
		strings.Contains(message, "工具映射不支持") {
		return false
	}
	return status == 401 || status == 402 || status == 403 || status >= 500 && status <= 599
}

func isNodeFailure(err error, accountID int64, events []*OpsUpstreamErrorEvent) bool {
	var failover *UpstreamFailoverError
	if errors.As(err, &failover) {
		return nodeFailureStatus(failover.StatusCode, string(failover.ResponseBody))
	}
	var streamError *sseStreamErrorEventError
	if errors.As(err, &streamError) {
		return nodeFailureStatus(anthropicStreamErrorStatus([]byte(streamError.RawData)), streamError.RawData)
	}
	message := err.Error()
	// These errors are emitted by the upstream body reader, not local request
	// validation. Client-disconnect errors are deliberately absent.
	if strings.HasPrefix(message, "stream read error:") ||
		message == "stream usage incomplete: missing terminal event" ||
		message == "stream data interval timeout" {
		return true
	}
	for i := len(events) - 1; i >= 0; i-- {
		ev := events[i]
		if ev != nil && ev.AccountID == accountID {
			return nodeFailureStatus(ev.UpstreamStatusCode, ev.Detail)
		}
	}
	return false
}

func (s *RateLimitService) recordNodeFailure(ctx context.Context, account *Account, request nodeFailureRequest) {
	// Request completion must not strand accepted state updates, but Redis/DB
	// outages must never hold up the response indefinitely.
	stateCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()
	settings := s.nodeFailureSettings(stateCtx)
	count, trip, err := s.nodeFailureBudget.RecordNodeFailure(stateCtx, account.ID, request.id, request.started, settings)
	if err != nil {
		slog.Warn("node_failure_budget_record_failed", "account_id", account.ID, "error", err)
		return
	}
	if !trip {
		return
	}
	disabled := false
	defer func() {
		finishCtx, finishCancel := context.WithTimeout(context.Background(), time.Second)
		defer finishCancel()
		if err := s.nodeFailureBudget.FinishNodeFailureTrip(finishCtx, account.ID, request.id, disabled); err != nil {
			slog.Warn("node_failure_budget_finish_failed", "account_id", account.ID, "error", err)
		}
	}()
	// Re-read after claiming: never overwrite a manual disable or a different
	// error reason using an old scheduler snapshot.
	current, err := s.accountRepo.GetByID(stateCtx, account.ID)
	if err != nil {
		slog.Warn("node_failure_budget_account_read_failed", "account_id", account.ID, "error", err)
		return
	}
	if current == nil || current.Status != StatusActive || !current.Schedulable {
		disabled = true
		return
	}
	if !nodeFailureEligible(current) {
		return
	}
	// An old in-flight request must not extend an already-open circuit.
	if GetNodeFailureRecovery(current) != nil {
		disabled = true
		return
	}
	reason := fmt.Sprintf("节点异常熔断：最近 %d 分钟累计 %d 次异常（大于 %d），失败率超过 %d%%，停调 %d 分钟后以 1 并发试恢复", settings.WindowMinutes, count, settings.FailureThreshold, settings.FailureRatePercent, settings.CooldownMinutes)
	disabler, ok := s.accountRepo.(nodeFailureAccountRecovery)
	if !ok {
		slog.Error("node_failure_budget_disabler_missing", "account_id", account.ID)
		return
	}
	until := time.Now().Add(time.Duration(settings.CooldownMinutes) * time.Minute)
	changed, err := disabler.ParkNodeForFailureBudget(stateCtx, current, until, reason)
	if err != nil {
		slog.Warn("node_failure_budget_disable_failed", "account_id", account.ID, "error", err)
		return
	}
	if !changed {
		return
	}
	disabled = true
	s.notifyAccountSchedulingBlocked(current, until, "node_failure_budget")
	slog.Warn("node_disabled_failure_budget", "account_id", account.ID, "count", count, "window_seconds", settings.WindowMinutes*60)
}

// Only a request selected in this generation, after the cooldown elapsed,
// can close the circuit. Pre-trip completions and manual disables cannot.
func (s *RateLimitService) finishNodeFailureProbe(ctx context.Context, account *Account, request nodeFailureRequest, success bool) {
	selectedState := GetNodeFailureRecovery(account)
	if selectedState == nil || request.started.Before(selectedState.ResumeAfter) {
		return
	}
	stateCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()
	current, err := s.accountRepo.GetByID(stateCtx, account.ID)
	if err != nil {
		slog.Warn("node_probe_account_read_failed", "account_id", account.ID, "error", err)
		return
	}
	state := GetNodeFailureRecovery(current)
	if state == nil || state.Generation != selectedState.Generation || current.Status != StatusActive || !current.Schedulable || current.Concurrency != 1 {
		return
	}
	// Respect a newer manual/rate-limit pause instead of clearing it on success.
	if success && current.TempUnschedulableUntil != nil && time.Now().Before(*current.TempUnschedulableUntil) {
		return
	}
	recovery, ok := s.accountRepo.(nodeFailureAccountRecovery)
	if !ok {
		return
	}
	var changed bool
	if success {
		changed, err = recovery.RestoreNodeAfterFailureProbe(stateCtx, current)
		if changed {
			s.notifyAccountSchedulingBlockCleared(current.ID)
		}
	} else {
		settings := s.nodeFailureSettings(stateCtx)
		until := time.Now().Add(time.Duration(settings.CooldownMinutes) * time.Minute)
		if current.TempUnschedulableUntil != nil && current.TempUnschedulableUntil.After(until) {
			until = *current.TempUnschedulableUntil
		}
		changed, err = recovery.ParkNodeForFailureBudget(stateCtx, current, until, fmt.Sprintf("节点试恢复失败：继续停调 %d 分钟，到期后以 1 并发重试", settings.CooldownMinutes))
		if changed {
			s.notifyAccountSchedulingBlocked(current, until, "node_failure_probe")
		}
	}
	if err != nil {
		slog.Warn("node_probe_update_failed", "account_id", account.ID, "error", err)
		return
	}
	if changed {
		slog.Info("node_probe_completed", "account_id", account.ID, "success", success, "original_concurrency", state.OriginalConcurrency)
	}
}

func (s *RateLimitService) recordNodeSuccess(ctx context.Context, account *Account, request nodeFailureRequest) {
	stateCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()
	settings := s.nodeFailureSettings(stateCtx)
	if err := s.nodeFailureBudget.RecordNodeSuccess(stateCtx, account.ID, request.id, request.started, settings); err != nil {
		slog.Warn("node_failure_budget_success_record_failed", "account_id", account.ID, "error", err)
	}
}

// In addition to the normal concurrency slot, share a probe lease across
// instances. A canceled probe or failed state write cannot cause rapid retries.
func (s *GatewayService) admitNodeFailureProbe(ctx context.Context, account *Account) error {
	state := GetNodeFailureRecovery(account)
	if state == nil {
		return nil
	}
	unavailable := func() error {
		return &UpstreamFailoverError{StatusCode: 503, ResponseBody: []byte(`{"error":{"type":"api_error","message":"Node recovery probe is not due or already running"}}`)}
	}
	if time.Now().Before(state.ResumeAfter) || s.rateLimitService == nil || s.rateLimitService.nodeFailureBudget == nil {
		return unavailable()
	}
	probeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	settings := s.rateLimitService.nodeFailureSettings(probeCtx)
	claimed, err := s.rateLimitService.nodeFailureBudget.ClaimNodeFailureProbe(probeCtx, account.ID, state.Generation, time.Duration(settings.CooldownMinutes)*time.Minute)
	if err != nil || !claimed {
		return unavailable()
	}
	return nil
}
