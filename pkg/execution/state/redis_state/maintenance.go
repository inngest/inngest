package redis_state

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/google/uuid"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/redis/rueidis"
)

// Only maintenance callers use this script. The normal dequeue Lua is unchanged.
// The comparison and lease checks execute in the SAME Redis operation as HDEL.
const maintenanceDequeueGuard = `
local observed = redis.call("HGET", KEYS[1], ARGV[1])
if not observed then return 1 end
if observed ~= ARGV[7] then return 2 end
local item = cjson.decode(observed)
if (item.leaseID ~= nil and item.leaseID ~= cjson.null) or
   (item.cl ~= nil and item.cl ~= cjson.null) or
   redis.call("ZCARD", KEYS[18]) ~= 0 then return 2 end
`

var maintenanceDequeueScript = sync.OnceValue(func() *rueidis.Lua {
	// Reuse the normal index/pointer cleanup verbatim, not a second copy that
	// could diverge. Expand includes just as the ordinary script loader does.
	source, err := embedded.ReadFile("lua/queue/dequeue.lua")
	if err != nil {
		panic(err)
	}
	code := string(source)
	for _, match := range include.FindAllStringSubmatch(code, -1) {
		part, err := embedded.ReadFile("lua/includes/" + match[1])
		if err != nil {
			panic(err)
		}
		code = strings.ReplaceAll(code, match[0], string(part))
	}
	return rueidis.NewLuaScript(maintenanceDequeueGuard + code)
})

func (q *queue) MaintenanceLeaseCount(ctx context.Context, scope osqueue.Scope) (int64, error) {
	if scope.FunctionID == uuid.Nil {
		return 0, fmt.Errorf("maintenance function scope is required")
	}
	rc := q.RedisClient.unshardedRc
	// No expiry filter: an expired execution lease is still unsafe to retire.
	return rc.Do(ctx, rc.B().Zcard().Key(q.RedisClient.kg.PartitionScavengerIndex(scope.FunctionID.String())).Build()).AsInt64()
}

func (q *queue) DequeueMaintenance(ctx context.Context, expected osqueue.QueueItem) error {
	rc := q.RedisClient.unshardedRc
	raw, err := rc.Do(ctx, rc.B().Hget().Key(q.RedisClient.kg.QueueItem()).Field(expected.ID).Build()).ToString()
	if rueidis.IsRedisNil(err) {
		return osqueue.ErrQueueItemNotFound
	}
	if err != nil {
		return err
	}
	var current osqueue.QueueItem
	if err := json.Unmarshal([]byte(raw), &current); err != nil {
		return err
	}
	current.Data.JobID = &current.ID // match LoadQueueItem normalization
	currentJSON, err := json.Marshal(current)
	if err != nil {
		return err
	}
	expectedJSON, err := json.Marshal(expected)
	if err != nil {
		return err
	}
	if !bytes.Equal(currentJSON, expectedJSON) || current.LeaseID != nil || current.CapacityLease != nil {
		return fmt.Errorf("maintenance job changed or retains a lease")
	}
	return q.dequeue(ctx, expected, maintenanceDequeueScript(), []string{raw})
}

var _ osqueue.MaintenanceShard = (*queue)(nil)
