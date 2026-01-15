package constraintapi

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/util"
	"github.com/jonboulle/clockwork"
	"github.com/karlseguin/ccache/v3"
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

	LimitingConstraintCacheTTLConcurrency = 5 * time.Second
	LimitingConstraintCacheTTLThrottle    = time.Second
)

var enableDebugLogs = false

type limitingConstraintCacheItem struct {
	constraint ConstraintItem
	retryAfter time.Time
}

type redisCapacityManager struct {
	keyGenerator

	// Until fully rolled out, the Constraint API will use the existing data stores
	// for accessing and modifying existing constraint state, as well as lease-related state.
	//
	// This means, we need to connect to all queue shards, as well as the instance
	// responsible for storing rate limit state.
	//
	// In a future release, we will gracefully migrate all constraint and lease state to a
	// dedicated horizontally-scalable and account-sharded backing data store.
	queueShards     map[string]rueidis.Client
	rateLimitClient rueidis.Client

	clock clockwork.Clock

	rateLimitKeyPrefix  string
	queueStateKeyPrefix string

	numScavengerShards int

	enableDebugLogs                      bool
	enableHighCardinalityInstrumentation EnableHighCardinalityInstrumentation

	lifecycles []ConstraintAPILifecycleHooks

	operationIdempotencyTTL       time.Duration
	constraintCheckIdempotencyTTL time.Duration
	checkIdempotencyTTL           time.Duration

	limitingConstraintCache        *ccache.Cache[*limitingConstraintCacheItem]
	limitingConstraintCacheTTLFunc func(c ConstraintItem) time.Duration
}

type RedisCapacityManagerOption func(m *redisCapacityManager)

func WithQueueShards(shards map[string]rueidis.Client) RedisCapacityManagerOption {
	return func(m *redisCapacityManager) {
		m.queueShards = shards
	}
}

func WithQueueStateKeyPrefix(prefix string) RedisCapacityManagerOption {
	return func(m *redisCapacityManager) {
		m.queueStateKeyPrefix = prefix
	}
}

func WithRateLimitClient(client rueidis.Client) RedisCapacityManagerOption {
	return func(m *redisCapacityManager) {
		m.rateLimitClient = client
	}
}

func WithRateLimitKeyPrefix(prefix string) RedisCapacityManagerOption {
	return func(m *redisCapacityManager) {
		m.rateLimitKeyPrefix = prefix
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

func WithNumScavengerShards(numShards int) RedisCapacityManagerOption {
	return func(m *redisCapacityManager) {
		m.numScavengerShards = numShards
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

func WithLimitingConstraintCacheTTLFunc(f func(c ConstraintItem) time.Duration) RedisCapacityManagerOption {
	return func(m *redisCapacityManager) {
		m.limitingConstraintCacheTTLFunc = f
	}
}

func DefaultLimitingConstraintCacheTTLFunc(c ConstraintItem) time.Duration {
	switch c.Kind {
	case ConstraintKindConcurrency:
		return time.Second
	case ConstraintKindThrottle:
		return time.Second
	case ConstraintKindRateLimit:
		return time.Second
	default:
		return time.Second
	}
}

func NewRedisCapacityManager(
	options ...RedisCapacityManagerOption,
) (*redisCapacityManager, error) {
	manager := &redisCapacityManager{
		keyGenerator:                  keyGenerator{},
		operationIdempotencyTTL:       OperationIdempotencyTTL,
		constraintCheckIdempotencyTTL: ConstraintCheckIdempotencyTTL,
		checkIdempotencyTTL:           CheckIdempotencyTTL,
		limitingConstraintCache: ccache.New(
			ccache.Configure[*limitingConstraintCacheItem]().
				MaxSize(10_000).
				ItemsToPrune(500),
		),
		limitingConstraintCacheTTLFunc: DefaultLimitingConstraintCacheTTLFunc,
	}

	for _, rcmo := range options {
		rcmo(manager)
	}

	if manager.rateLimitClient == nil || manager.queueShards == nil {
		return nil, fmt.Errorf("missing clients")
	}

	if manager.clock == nil {
		manager.clock = clockwork.NewRealClock()
	}

	if manager.numScavengerShards == 0 {
		manager.numScavengerShards = 64
	}

	manager.keyGenerator.queueStateKeyPrefix = manager.queueStateKeyPrefix
	manager.keyGenerator.rateLimitKeyPrefix = manager.rateLimitKeyPrefix

	return manager, nil
}

func accountScope(accountID uuid.UUID) string {
	return fmt.Sprintf("a:%s", accountID)
}

// keyScavengerShard represents the top-level sharded sorted set containing individual accounts
func (r *redisCapacityManager) keyScavengerShard(prefix string, shard int) string {
	return fmt.Sprintf("{%s}:css:%d", prefix, shard)
}

// keyAccountLeases represents active leases for the account
func (r *redisCapacityManager) keyAccountLeases(prefix string, accountID uuid.UUID) string {
	return fmt.Sprintf("{%s}:%s:leaseq", prefix, accountScope(accountID))
}

// keyRequestState returns the key storing per-operation request details
func (r *redisCapacityManager) keyRequestState(prefix string, accountID uuid.UUID, requestID ulid.ULID) string {
	return fmt.Sprintf("{%s}:%s:rs:%s", prefix, accountScope(accountID), requestID)
}

// keyOperationIdempotency returns the operation idempotency key for operation retries
func (r *redisCapacityManager) keyOperationIdempotency(prefix string, accountID uuid.UUID, operation, idempotencyKey string) string {
	return fmt.Sprintf("{%s}:%s:ik:op:%s:%s", prefix, accountScope(accountID), operation, util.XXHash(idempotencyKey))
}

func keyConstraintCheckIdempotency(prefix string, accountID uuid.UUID, idempotencyKey string) string {
	return fmt.Sprintf("{%s}:%s:ik:cc:%s", prefix, accountScope(accountID), util.XXHash(idempotencyKey))
}

// keyConstraintCheckIdempotency returns the operation idempotency key for constraint check retries
func (r *redisCapacityManager) keyConstraintCheckIdempotency(prefix string, accountID uuid.UUID, idempotencyKey string) string {
	return keyConstraintCheckIdempotency(prefix, accountID, idempotencyKey)
}

// keyLeaseDetails returns the key to the hash including the lease idempotency key, lease run ID, and operation idempotency key
func (r *redisCapacityManager) keyLeaseDetails(prefix string, accountID uuid.UUID, leaseID ulid.ULID) string {
	return fmt.Sprintf("{%s}:%s:ld:%s", prefix, accountScope(accountID), leaseID)
}

type keyGenerator struct {
	rateLimitKeyPrefix  string
	queueStateKeyPrefix string
}

func (r *keyGenerator) KeyInProgressLeasesAccount(accountID uuid.UUID) string {
	return ConcurrencyConstraint{
		Scope: enums.ConcurrencyScopeAccount,
		Mode:  enums.ConcurrencyModeStep,
	}.InProgressLeasesKey(r.queueStateKeyPrefix, accountID, uuid.Nil, uuid.Nil)
}

func (r *keyGenerator) KeyInProgressLeasesFunction(accountID uuid.UUID, fnID uuid.UUID) string {
	return ConcurrencyConstraint{
		Scope: enums.ConcurrencyScopeFn,
		Mode:  enums.ConcurrencyModeStep,
	}.InProgressLeasesKey(r.queueStateKeyPrefix, accountID, uuid.Nil, fnID)
}

func (r *keyGenerator) KeyInProgressLeasesCustom(accountID uuid.UUID, scope enums.ConcurrencyScope, entityID uuid.UUID, keyExpressionHash, evaluatedKeyHash string) string {
	return ConcurrencyConstraint{
		Scope:             scope,
		Mode:              enums.ConcurrencyModeStep,
		KeyExpressionHash: keyExpressionHash,
		EvaluatedKeyHash:  evaluatedKeyHash,
	}.InProgressLeasesKey(r.queueStateKeyPrefix, accountID, entityID, entityID)
}

func (r *keyGenerator) KeyConstraintCheckIdempotency(mi MigrationIdentifier, accountID uuid.UUID, leaseIdempotencyKey string) string {
	prefix := r.queueStateKeyPrefix
	if mi.IsRateLimit {
		prefix = r.rateLimitKeyPrefix
	}

	return keyConstraintCheckIdempotency(prefix, accountID, leaseIdempotencyKey)
}

// clientAndPrefix returns the Redis client and Lua key prefix for the first stage of the Constraint API.
//
// Since we are colocating lease data with the existing state, we will have to use the
// same Redis hash tag to avoid Lua errors and inconsistencies on the old and new scripts.
//
// This is essentially required for backward- and forward-compatibility.
func (r *redisCapacityManager) clientAndPrefix(m MigrationIdentifier) (string, rueidis.Client, error) {
	// TODO: Once we support new data stores, we can return those clients here, including a per-account hash tag prefix, e.g. <accountID>

	if m.IsRateLimit {
		return r.rateLimitKeyPrefix, r.rateLimitClient, nil
	}

	shard, ok := r.queueShards[m.QueueShard]
	if !ok {
		return "", nil, fmt.Errorf("unknown queue shard %q", m.QueueShard)
	}

	return r.queueStateKeyPrefix, shard, nil
}
