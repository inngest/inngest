package redis_state

import (
	"context"
	"fmt"

	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
)

const singletonMigrationCleanupLimit = 100

func singletonMigrationKeys(kg QueueKeyGenerator, scope osqueue.Scope) (hold, pending string) {
	hold = kg.QueuePrefix() + ":singleton-migration:" + scope.FunctionID.String()
	return hold, hold + ":pending"
}

var beginSingletonMigration = rueidis.NewLuaScript(`
local owner = redis.call("GET", KEYS[1])
if owner and owner ~= ARGV[1] then return -1 end
redis.call("SET", KEYS[1], ARGV[1])
return 0
`)

var singletonMigrationSourceDrained = rueidis.NewLuaScript(`
for i = 1, #KEYS do
  if redis.call("ZCARD", KEYS[i]) ~= 0 then return 0 end
end
return 1
`)

// SingletonMigrationSourceDrained includes ready jobs, all leases (even expired
// leases awaiting scavenging), and backlog pointers. RunningCount deliberately
// excludes expired leases and therefore cannot prove a migration source empty.
// This is a source-only check: destination work may still be executing.
func (q *queue) SingletonMigrationSourceDrained(ctx context.Context, scope osqueue.Scope) (bool, error) {
	if scope.IsSystem {
		return false, fmt.Errorf("singleton migration requires a function scope")
	}
	if err := scope.Validate(); err != nil {
		return false, err
	}
	kg := q.RedisClient.kg
	keys := []string{
		kg.PartitionQueueSet(enums.PartitionTypeDefault, scope.FunctionID.String(), ""),
		kg.PartitionScavengerIndex(scope.FunctionID.String()),
		kg.ShadowPartitionSet(scope.FunctionID.String()),
	}
	result, err := singletonMigrationSourceDrained.Exec(ctx, q.RedisClient.unshardedRc, keys, nil).AsInt64()
	if err != nil {
		return false, fmt.Errorf("check singleton migration source: %w", err)
	}
	return result == 1, nil
}

var finishSingletonMigration = rueidis.NewLuaScript(`
local owner = redis.call("GET", KEYS[1])
if not owner then return 1 end
if owner ~= ARGV[1] then return -1 end
for i = 2, #ARGV do
  local reverse = KEYS[2 * i - 1]
  local runIndex = KEYS[2 * i]
  local singleton = redis.call("GET", reverse)
  if singleton and redis.call("ZCARD", runIndex) == 0 then
    if redis.call("GET", singleton) == ARGV[i] then
      redis.call("DEL", singleton)
    end
    redis.call("DEL", reverse)
  end
  redis.call("SREM", KEYS[2], ARGV[i])
end
if redis.call("SCARD", KEYS[2]) ~= 0 then return 0 end
redis.call("DEL", KEYS[1], KEYS[2])
return 1
`)

func validateSingletonMigration(scope osqueue.Scope, migrationID string) error {
	if scope.IsSystem {
		return fmt.Errorf("singleton migration requires a function scope")
	}
	if err := scope.Validate(); err != nil {
		return err
	}
	if migrationID == "" {
		return fmt.Errorf("singleton migration ID is required")
	}
	return nil
}

func (q *queue) BeginSingletonMigration(ctx context.Context, scope osqueue.Scope, migrationID string) error {
	if err := validateSingletonMigration(scope, migrationID); err != nil {
		return err
	}
	hold, _ := singletonMigrationKeys(q.RedisClient.kg, scope)
	result, err := beginSingletonMigration.Exec(ctx, q.RedisClient.unshardedRc, []string{hold}, []string{migrationID}).AsInt64()
	if err != nil {
		return fmt.Errorf("begin singleton migration: %w", err)
	}
	if result == -1 {
		return osqueue.ErrSingletonMigrationConflict
	}
	return nil
}

func (q *queue) FinishSingletonMigration(ctx context.Context, scope osqueue.Scope, migrationID string) (bool, error) {
	if err := validateSingletonMigration(scope, migrationID); err != nil {
		return false, err
	}
	kg := q.RedisClient.kg
	hold, pending := singletonMigrationKeys(kg, scope)
	rc := q.RedisClient.unshardedRc
	runs, err := rc.Do(ctx, rc.B().Srandmember().Key(pending).Count(singletonMigrationCleanupLimit).Build()).AsStrSlice()
	if err != nil {
		return false, fmt.Errorf("read deferred singleton releases: %w", err)
	}
	keys, args := []string{hold, pending}, []string{migrationID}
	for _, run := range runs {
		id, err := ulid.Parse(run)
		if err != nil {
			return false, fmt.Errorf("invalid deferred singleton run: %w", err)
		}
		keys = append(keys, kg.SingletonRunKey(run), kg.RunIndex(id))
		args = append(args, run)
	}
	result, err := finishSingletonMigration.Exec(ctx, rc, keys, args).AsInt64()
	if err != nil {
		return false, fmt.Errorf("finish singleton migration: %w", err)
	}
	if result == -1 {
		return false, osqueue.ErrSingletonMigrationConflict
	}
	return result == 1, nil
}

var _ osqueue.SingletonMigrationOperations = (*queue)(nil)
