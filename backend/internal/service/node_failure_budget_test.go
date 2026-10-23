package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type nodeBudgetStub struct {
	seen     map[string]bool
	count    int64
	total    int64
	trips    int
	finished []bool
}

func (c *nodeBudgetStub) RecordNodeFailure(_ context.Context, id int64, request string, _ time.Time, settings NodeFailureSettings) (int64, bool, error) {
	if c.seen == nil {
		c.seen = map[string]bool{}
	}
	if !c.seen[request] {
		c.seen[request] = true
		c.count++
		c.total++
	}
	return c.count, c.count > int64(settings.FailureThreshold) && c.count*100 > c.total*int64(settings.FailureRatePercent), nil
}
func (c *nodeBudgetStub) ClaimNodeFailureProbe(context.Context, int64, string, time.Duration) (bool, error) {
	return true, nil
}
func (c *nodeBudgetStub) RecordNodeSuccess(_ context.Context, _ int64, request string, _ time.Time, _ NodeFailureSettings) error {
	if c.seen == nil {
		c.seen = map[string]bool{}
	}
	if !c.seen[request] {
		c.seen[request] = true
		c.total++
	}
	return nil
}
func (c *nodeBudgetStub) FinishNodeFailureTrip(_ context.Context, _ int64, _ string, disabled bool) error {
	c.trips++
	c.finished = append(c.finished, disabled)
	return nil
}

type nodeBudgetRepoStub struct {
	AccountRepository
	account *Account
	calls   int
	reason  string
	fail    bool
}

func (r *nodeBudgetRepoStub) GetByID(context.Context, int64) (*Account, error) { return r.account, nil }
func (r *nodeBudgetRepoStub) ParkNodeForFailureBudget(_ context.Context, account *Account, until time.Time, reason string) (bool, error) {
	r.calls++
	r.reason = reason
	if r.fail {
		return false, errors.New("database unavailable")
	}
	original := account.Concurrency
	if state := GetNodeFailureRecovery(account); state != nil {
		original = state.OriginalConcurrency
	}
	r.account.Concurrency = 1
	r.account.TempUnschedulableUntil = &until
	r.account.TempUnschedulableReason = reason
	r.account.Extra = map[string]any{NodeFailureRecoveryKey: NodeFailureRecovery{OriginalConcurrency: original, ResumeAfter: until, Generation: fmt.Sprint(r.calls)}}
	return true, nil
}
func (r *nodeBudgetRepoStub) RestoreNodeAfterFailureProbe(_ context.Context, account *Account) (bool, error) {
	r.calls++
	if r.fail {
		return false, errors.New("database unavailable")
	}
	state := GetNodeFailureRecovery(account)
	r.account.Concurrency = state.OriginalConcurrency
	r.account.TempUnschedulableUntil = nil
	r.account.TempUnschedulableReason = ""
	r.account.Extra = nil
	return true, nil
}
func nodeBudgetAccount() *Account {
	return &Account{ID: 17, Platform: PlatformAnthropic, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 50, Credentials: map[string]any{"base_url": "https://kiro.example.com", "pool_mode": true}}
}

func TestNodeFailureBudgetCountsRequestsAndParksPoolAccount(t *testing.T) {
	account := nodeBudgetAccount()
	cache := &nodeBudgetStub{}
	repo := &nodeBudgetRepoStub{account: account}
	svc := &GatewayService{rateLimitService: &RateLimitService{accountRepo: repo, nodeFailureBudget: cache}}
	failure := &UpstreamFailoverError{StatusCode: 503, ResponseBody: []byte(`{"error":{"type":"api_error","message":"no available accounts"}}`)}
	for i := 0; i < 100; i++ {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		observe := svc.beginNodeFailureObservation(c, account)
		observe(context.Background(), nil, failure)
		// A same-account handler retry must reuse the request identity.
		svc.beginNodeFailureObservation(c, account)(context.Background(), nil, failure)
	}
	require.EqualValues(t, 100, cache.count)
	require.Zero(t, repo.calls)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	svc.beginNodeFailureObservation(c, account)(context.Background(), nil, failure)
	require.Equal(t, 1, repo.calls)
	require.Equal(t, StatusActive, account.Status)
	require.True(t, account.Schedulable)
	require.False(t, account.IsSchedulable())
	require.Equal(t, 1, account.Concurrency)
	require.Equal(t, 50, GetNodeFailureRecovery(account).OriginalConcurrency)
	require.WithinDuration(t, time.Now().Add(3*time.Minute), *account.TempUnschedulableUntil, time.Second)
	require.Contains(t, repo.reason, "1 分钟累计 101 次异常")
	require.Equal(t, []bool{true}, cache.finished)
}

func TestNodeFailureBudgetIgnoresClientAndRequestErrors(t *testing.T) {
	account := nodeBudgetAccount()
	cache := &nodeBudgetStub{}
	svc := &GatewayService{rateLimitService: &RateLimitService{nodeFailureBudget: cache}}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	observe := svc.beginNodeFailureObservation(c, account)
	observe(context.Background(), nil, nil)
	observe(context.Background(), nil, context.Canceled)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	observe(ctx, nil, &UpstreamFailoverError{StatusCode: 502})
	observe(context.Background(), &ForwardResult{ClientDisconnect: true}, errors.New("stream read error: EOF"))
	for _, body := range []string{
		`{"error":{"type":"invalid_request_error","message":"bad tool history"}}`,
		`{"error":{"type":"api_error","message":"Unsupported content block image"}}`,
		`{"error":{"type":"api_error","message":"Model output was blocked by content filtering policy"}}`,
	} {
		observe(context.Background(), nil, &UpstreamFailoverError{StatusCode: 502, ResponseBody: []byte(body)})
	}
	observe(context.Background(), nil, &UpstreamFailoverError{StatusCode: 400})
	observe(context.Background(), nil, errors.New("local request validation"))
	c.Set(nodeFailureClientDisconnectedKey, true)
	c.Set(nodeFailureSSEKey, `{"type":"error","error":{"type":"api_error"}}`)
	observe(context.Background(), nil, errors.New("stream usage incomplete: missing terminal event"))
	require.Zero(t, cache.count)
}

func TestNodeFailureBudgetReleaseOnPersistenceFailure(t *testing.T) {
	account := nodeBudgetAccount()
	cache := &nodeBudgetStub{count: 100, total: 100}
	repo := &nodeBudgetRepoStub{account: account, fail: true}
	svc := &RateLimitService{accountRepo: repo, nodeFailureBudget: cache}
	svc.recordNodeFailure(context.Background(), account, nodeFailureRequest{id: "failure", started: time.Now()})
	require.Equal(t, []bool{false}, cache.finished)
	require.True(t, account.Schedulable)
	require.Equal(t, StatusActive, account.Status)
}

func TestNodeFailureBudgetForwardTransportFailure(t *testing.T) {
	upstream := &anthropicHTTPUpstreamRecorder{err: io.ErrUnexpectedEOF}
	svc := newForwardPartialUsageServiceForTest(upstream)
	cache := &nodeBudgetStub{}
	svc.rateLimitService.nodeFailureBudget = cache
	account := nodeBudgetAccount()
	account.Credentials["api_key"] = "test-key"
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	parsed, err := ParseGatewayRequest(NewRequestBodyRef([]byte(`{"model":"claude-sonnet-4-5","stream":true,"max_tokens":10,"messages":[{"role":"user","content":"hello"}]}`)), PlatformAnthropic)
	require.NoError(t, err)
	_, err = svc.Forward(context.Background(), c, account, parsed)
	require.Error(t, err)
	require.EqualValues(t, 1, cache.count, "forward error: %v", err)
}

func TestNodeFailureClassification(t *testing.T) {
	for _, err := range []error{errors.New("stream usage incomplete: missing terminal event"), errors.New("stream read error: EOF"), errors.New("stream data interval timeout"), &sseStreamErrorEventError{RawData: `{"error":{"type":"overloaded_error"}}`}} {
		require.True(t, isNodeFailure(err, 17, nil), "%v", err)
	}
	require.False(t, isNodeFailure(errors.New("local validation"), 17, []*OpsUpstreamErrorEvent{{AccountID: 18, UpstreamStatusCode: 502}}))
	require.False(t, nodeFailureEligible(&Account{Platform: PlatformAnthropic, Type: AccountTypeOAuth}))
	official := nodeBudgetAccount()
	official.Credentials["base_url"] = "https://api.anthropic.com"
	require.False(t, nodeFailureEligible(official))
}

func TestNodeFailureBudgetForwardSSE(t *testing.T) {
	for _, passthrough := range []bool{false, true} {
		for _, tc := range []struct {
			name, frame string
			want        int64
		}{
			{"upstream unavailable", `{"type":"error","error":{"type":"api_error","message":"no available accounts"}}`, 1},
			{"bad request", `{"type":"error","error":{"type":"invalid_request_error","message":"bad tool history"}}`, 0},
			{"normal response", `{"type":"message_stop"}`, 0},
		} {
			t.Run(fmt.Sprintf("%v/%s", passthrough, tc.name), func(t *testing.T) {
				body := "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"test\",\"type\":\"message\",\"role\":\"assistant\",\"content\":[],\"usage\":{\"input_tokens\":1,\"output_tokens\":0}}}\n\ndata: " + tc.frame + "\n\n"
				upstream := &anthropicHTTPUpstreamRecorder{resp: &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"text/event-stream"}}, Body: io.NopCloser(strings.NewReader(body))}}
				svc := newForwardPartialUsageServiceForTest(upstream)
				cache := &nodeBudgetStub{}
				svc.rateLimitService.nodeFailureBudget = cache
				account := nodeBudgetAccount()
				account.Credentials["api_key"] = "test-key"
				account.Extra = map[string]any{"anthropic_passthrough": passthrough}
				c, _ := gin.CreateTestContext(httptest.NewRecorder())
				c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
				parsed, err := ParseGatewayRequest(NewRequestBodyRef([]byte(`{"model":"claude-sonnet-4-5","stream":true,"max_tokens":10,"messages":[{"role":"user","content":"hello"}]}`)), PlatformAnthropic)
				require.NoError(t, err)
				_, err = svc.Forward(context.Background(), c, account, parsed)
				require.Equal(t, tc.want, cache.count, "forward error: %v", err)
				require.NotNil(t, upstream.lastReq, "test must reach the upstream")
			})
		}
	}
}

func TestNodeFailureProbeCooldownRecoveryAndStaleCompletions(t *testing.T) {
	for _, success := range []bool{false, true} {
		t.Run(fmt.Sprint(success), func(t *testing.T) {
			account := nodeBudgetAccount()
			repo := &nodeBudgetRepoStub{account: account}
			until := time.Now().Add(-time.Second)
			_, err := repo.ParkNodeForFailureBudget(context.Background(), account, until, "test trip")
			require.NoError(t, err)
			require.True(t, account.IsSchedulable())
			require.Equal(t, 1, account.Concurrency)
			selected := *account // retain the generation selected before mutation
			svc := &GatewayService{rateLimitService: &RateLimitService{accountRepo: repo, nodeFailureBudget: &nodeBudgetStub{}}}
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			observe := svc.beginNodeFailureObservation(c, &selected)
			if success {
				observe(context.Background(), &ForwardResult{UpstreamHeaders: http.Header{}}, nil)
			} else {
				observe(context.Background(), nil, &UpstreamFailoverError{StatusCode: 502})
			}
			require.Equal(t, 2, repo.calls)
			if success {
				require.Equal(t, 50, account.Concurrency)
				require.Nil(t, GetNodeFailureRecovery(account))
				require.True(t, account.IsSchedulable())
			} else {
				require.Equal(t, 1, account.Concurrency)
				require.Equal(t, 50, GetNodeFailureRecovery(account).OriginalConcurrency)
				require.False(t, account.IsSchedulable())
				require.WithinDuration(t, time.Now().Add(3*time.Minute), *account.TempUnschedulableUntil, time.Second)
			}
			// An old-generation completion cannot restore or re-park this node.
			observe(context.Background(), &ForwardResult{UpstreamHeaders: http.Header{}}, nil)
			require.Equal(t, 2, repo.calls)
		})
	}
}

func TestNodeFailureProbeDoesNotRecoverEarlyOrOverrideManualDisable(t *testing.T) {
	account := nodeBudgetAccount()
	repo := &nodeBudgetRepoStub{account: account}
	until := time.Now().Add(3 * time.Minute)
	_, err := repo.ParkNodeForFailureBudget(context.Background(), account, until, "test trip")
	require.NoError(t, err)
	svc := &RateLimitService{accountRepo: repo}
	svc.finishNodeFailureProbe(context.Background(), account, nodeFailureRequest{started: time.Now()}, true)
	require.Equal(t, 1, repo.calls)
	account.Schedulable = false
	svc.finishNodeFailureProbe(context.Background(), account, nodeFailureRequest{started: until.Add(time.Second)}, true)
	require.Equal(t, 1, repo.calls)
	require.False(t, account.Schedulable)
	require.Equal(t, 1, account.Concurrency)
}

func TestNodeFailureProbeIncompleteResponseKeepsCooldown(t *testing.T) {
	for _, name := range []string{"local response", "client canceled", "invalid request"} {
		t.Run(name, func(t *testing.T) {
			account := nodeBudgetAccount()
			repo := &nodeBudgetRepoStub{account: account}
			_, err := repo.ParkNodeForFailureBudget(context.Background(), account, time.Now().Add(-time.Second), "test")
			require.NoError(t, err)
			svc := &GatewayService{rateLimitService: &RateLimitService{accountRepo: repo, nodeFailureBudget: &nodeBudgetStub{}}}
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			observe := svc.beginNodeFailureObservation(c, account)
			ctx := context.Background()
			var resultErr error
			if name == "client canceled" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
				resultErr = context.Canceled
			}
			if name == "invalid request" {
				resultErr = &UpstreamFailoverError{StatusCode: 400}
			}
			observe(ctx, &ForwardResult{}, resultErr)
			require.False(t, account.IsSchedulable())
			require.Equal(t, 1, account.Concurrency)
			require.WithinDuration(t, time.Now().Add(3*time.Minute), *account.TempUnschedulableUntil, time.Second)
		})
	}
}

func TestNodeFailureProbeSkipsInternalRetry(t *testing.T) {
	account := nodeBudgetAccount()
	account.Extra = make(map[string]any)
	account.Extra[NodeFailureRecoveryKey] = NodeFailureRecovery{OriginalConcurrency: 50, ResumeAfter: time.Now().Add(-time.Second), Generation: "probe"}
	svc := &GatewayService{}
	for _, status := range []int{401, 403, 429, 500, 502, 503, 529} {
		require.False(t, svc.shouldRetryUpstreamError(account, status))
	}
}
