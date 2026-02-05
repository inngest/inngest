package constraintapi

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/util"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
)

const (
	pkgName = "constraintapi.redis"
)

const (
	MaximumAllowedRequestDelay = time.Second
	// OperationIdempotencyTTL represents the time the same response will be returned after a successful request.
	// Depending on the operation, this should be low (Otherwise, Acquire may return an already expired lease)
	// TODO: Figure out a reasonable operation idempotency TTL (maybe per-operation)
	OperationIdempotencyTTL       = 5 * time.Second
	CheckIdempotencyTTL           = 5 * time.Second
	ConstraintCheckIdempotencyTTL = 5 * time.Minute
)

var enableDebugLogs = false

type redisCapacityManager struct {
	// Constraint state is stored in Redis-compatible scavengerShards for the time being.
	// In the future, we will move to another data store like FoundationDB.
	shardName string
	client    rueidis.Client

	clock clockwork.Clock

	enableDebugLogs                      bool
	enableHighCardinalityInstrumentation EnableHighCardinalityInstrumentation

	lifecycles []ConstraintAPILifecycleHooks

	operationIdempotencyTTL       time.Duration
	constraintCheckIdempotencyTTL time.Duration
	checkIdempotencyTTL           time.Duration
}

type RedisCapacityManagerOption func(m *redisCapacityManager)

func WithShardName(name string) RedisCapacityManagerOption {
	return func(m *redisCapacityManager) {
		m.shardName = name
	}
}

func WithClient(client rueidis.Client) RedisCapacityManagerOption {
	return func(m *redisCapacityManager) {
		m.client = client
	}
}

func WithEnableHighCardinalityInstrumentation(ehci EnableHighCardinalityInstrumentation) RedisCapacityManagerOption {
	return func(m *redisCapacityManager) {
		m.enableHighCardinalityInstrumentation = ehci
	}
}

func WithClock(clock clockwork.Clock) RedisCapacityManagerOption {
	return func(m *redisCapacityManager) {
		m.clock = clock
	}
}

func WithEnableDebugLogs(enableDebugLogs bool) RedisCapacityManagerOption {
	return func(m *redisCapacityManager) {
		m.enableDebugLogs = enableDebugLogs
	}
}

func WithLifecycles(lifecycles ...ConstraintAPILifecycleHooks) RedisCapacityManagerOption {
	return func(m *redisCapacityManager) {
		m.lifecycles = lifecycles
	}
}

func WithOperationIdempotencyTTL(ttl time.Duration) RedisCapacityManagerOption {
	return func(m *redisCapacityManager) {
		m.operationIdempotencyTTL = ttl
	}
}

func WithConstraintCheckIdempotencyTTL(ttl time.Duration) RedisCapacityManagerOption {
	return func(m *redisCapacityManager) {
		m.constraintCheckIdempotencyTTL = ttl
	}
}

func WithCheckIdempotencyTTL(ttl time.Duration) RedisCapacityManagerOption {
	return func(m *redisCapacityManager) {
		m.checkIdempotencyTTL = ttl
	}
}

func NewRedisCapacityManager(
	options ...RedisCapacityManagerOption,
) (*redisCapacityManager, error) {
	manager := &redisCapacityManager{
		operationIdempotencyTTL:       OperationIdempotencyTTL,
		constraintCheckIdempotencyTTL: ConstraintCheckIdempotencyTTL,
		checkIdempotencyTTL:           CheckIdempotencyTTL,
	}

	for _, rcmo := range options {
		rcmo(manager)
	}

	if manager.client == nil {
		return nil, fmt.Errorf("missing client")
	}

	if manager.shardName == "" {
		return nil, fmt.Errorf("missing shard name")
	}

	if manager.clock == nil {
		manager.clock = clockwork.NewRealClock()
	}

	return manager, nil
}

func accountScope(accountID uuid.UUID) string {
	return fmt.Sprintf("a:%s", accountID)
}

// keyScavengerShard represents the top-level sharded sorted set containing individual accounts
func (r *redisCapacityManager) keyScavengerShard() string {
	return fmt.Sprintf("{cs}:css")
}

// keyAccountLeases represents active leases for the account
func (r *redisCapacityManager) keyAccountLeases(accountID uuid.UUID) string {
	return fmt.Sprintf("{cs}:%s:leaseq", accountScope(accountID))
}

// keyRequestState returns the key storing per-operation request details
func (r *redisCapacityManager) keyRequestState(accountID uuid.UUID, requestID ulid.ULID) string {
	return fmt.Sprintf("{cs}:%s:rs:%s", accountScope(accountID), requestID)
}

// keyOperationIdempotency returns the operation idempotency key for operation retries
func (r *redisCapacityManager) keyOperationIdempotency(accountID uuid.UUID, operation, idempotencyKey string) string {
	return fmt.Sprintf("{cs}:%s:ik:op:%s:%s", accountScope(accountID), operation, util.XXHash(idempotencyKey))
}

// keyConstraintCheckIdempotency returns the operation idempotency key for constraint check retries
func (r *redisCapacityManager) keyConstraintCheckIdempotency(accountID uuid.UUID, idempotencyKey string) string {
	return fmt.Sprintf("{cs}:%s:ik:cc:%s", accountScope(accountID), util.XXHash(idempotencyKey))
}

// keyLeaseDetails returns the key to the hash including the lease idempotency key, lease run ID, and operation idempotency key
func (r *redisCapacityManager) keyLeaseDetails(accountID uuid.UUID, leaseID ulid.ULID) string {
	return fmt.Sprintf("{cs}:%s:ld:%s", accountScope(accountID), leaseID)
}
