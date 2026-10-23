package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

type nodeFailureBudgetCache struct{ rdb *redis.Client }

func NewNodeFailureBudgetCache(rdb *redis.Client) service.NodeFailureBudgetCache {
	return &nodeFailureBudgetCache{rdb: rdb}
}

func nodeFailureKeys(id int64) []string {
	prefix := fmt.Sprintf("node_failure_budget:v2:{%d}:", id)
	return []string{prefix + "events", prefix + "trip", prefix + "cutoff", prefix + "requests"}
}

var recordNodeFailureScript = redis.NewScript(`
 local tm = redis.call('TIME')
 local now = tonumber(tm[1])*1000 + math.floor(tonumber(tm[2])/1000)
 local window = tonumber(ARGV[3])
 local cutoff = tonumber(redis.call('GET', KEYS[3]) or '0')
 if tonumber(ARGV[2]) <= cutoff then return {0,0} end
 redis.call('ZREMRANGEBYSCORE', KEYS[1], '-inf', now-window)
 redis.call('ZREMRANGEBYSCORE', KEYS[4], '-inf', now-window)
 local sampleTime = tonumber(redis.call('ZSCORE', KEYS[4], ARGV[1])) or now
 if ARGV[6] == '1' then redis.call('ZADD', KEYS[1], 'NX', sampleTime, ARGV[1]) end
 redis.call('PEXPIRE', KEYS[1], window*2)
 redis.call('ZADD', KEYS[4], 'NX', sampleTime, ARGV[1])
 redis.call('PEXPIRE', KEYS[4], window*2)
 local count = redis.call('ZCARD', KEYS[1])
 local total = redis.call('ZCARD', KEYS[4])
 if ARGV[6] == '1' and count > tonumber(ARGV[4]) and count*100 > total*tonumber(ARGV[5]) then
  if redis.call('SET', KEYS[2], ARGV[1], 'NX', 'PX', 5000) then return {count,1} end
 end
 return {count,0}
`)

var finishNodeFailureScript = redis.NewScript(`
 if redis.call('GET', KEYS[2]) ~= ARGV[1] then return 0 end
 if ARGV[2] == '1' then
  local tm = redis.call('TIME')
  local now = tonumber(tm[1])*1000 + math.floor(tonumber(tm[2])/1000)
  -- Ignore already in-flight requests after manual recovery. New requests
  -- start a clean window; no timer re-enables the database account.
  redis.call('SET', KEYS[3], now, 'PX', 86400000)
  redis.call('DEL', KEYS[1], KEYS[4])
 end
 redis.call('DEL', KEYS[2])
 return 1
`)

func (c *nodeFailureBudgetCache) RecordNodeFailure(ctx context.Context, id int64, requestID string, started time.Time, settings service.NodeFailureSettings) (int64, bool, error) {
	result, err := recordNodeFailureScript.Run(ctx, c.rdb, nodeFailureKeys(id), requestID, started.UnixMilli(), (time.Duration(settings.WindowMinutes) * time.Minute).Milliseconds(), settings.FailureThreshold, settings.FailureRatePercent, 1).Int64Slice()
	if err != nil {
		return 0, false, err
	}
	if len(result) != 2 {
		return 0, false, fmt.Errorf("invalid node failure budget result")
	}
	return result[0], result[1] == 1, nil
}
func (c *nodeFailureBudgetCache) FinishNodeFailureTrip(ctx context.Context, id int64, requestID string, disabled bool) error {
	success := 0
	if disabled {
		success = 1
	}
	return finishNodeFailureScript.Run(ctx, c.rdb, nodeFailureKeys(id), requestID, success).Err()
}

// A request that fails and later succeeds on a same-node retry remains one
// failed sample. Success cannot erase an already observed node failure.
func (c *nodeFailureBudgetCache) RecordNodeSuccess(ctx context.Context, id int64, requestID string, started time.Time, settings service.NodeFailureSettings) error {
	return recordNodeFailureScript.Run(ctx, c.rdb, nodeFailureKeys(id), requestID, started.UnixMilli(), (time.Duration(settings.WindowMinutes) * time.Minute).Milliseconds(), settings.FailureThreshold, settings.FailureRatePercent, 0).Err()
}

func (c *nodeFailureBudgetCache) ClaimNodeFailureProbe(ctx context.Context, id int64, generation string, interval time.Duration) (bool, error) {
	key := fmt.Sprintf("node_failure_budget:v2:{%d}:probe:%s", id, generation)
	return c.rdb.SetNX(ctx, key, "1", interval).Result()
}
