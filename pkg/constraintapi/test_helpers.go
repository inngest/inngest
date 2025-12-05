package constraintapi

import (
	"fmt"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

// TestEnvironment provides a complete test environment for constraint API testing
type TestEnvironment struct {
	Redis           *miniredis.Miniredis
	Client          rueidis.Client
	CapacityManager *redisCapacityManager
	KeyPrefix       string
	AccountID       uuid.UUID
	EnvID           uuid.UUID
	FunctionID      uuid.UUID
	t               *testing.T
}

// NewTestEnvironment creates a new test environment with Redis and capacity manager
func NewTestEnvironment(t *testing.T) *TestEnvironment {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)

	cm, err := NewRedisCapacityManager(
		WithRateLimitClient(rc),
		WithQueueShards(map[string]rueidis.Client{
			"test": rc,
		}),
		WithNumScavengerShards(1),
		WithQueueStateKeyPrefix("q:v1"),
		WithRateLimitKeyPrefix("rl"),
	)
	require.NoError(t, err)

	return &TestEnvironment{
		Redis:           r,
		Client:          rc,
		CapacityManager: cm,
		KeyPrefix:       "q:v1",
		AccountID:       uuid.New(),
		EnvID:           uuid.New(),
		FunctionID:      uuid.New(),
		t:               t,
	}
}

// Cleanup closes the test environment
func (te *TestEnvironment) Cleanup() {
	te.Client.Close()
	te.Redis.Close()
}

// RedisStateSnapshot captures the complete Redis state for comparison
type RedisStateSnapshot struct {
	Keys       []string                      `json:"keys"`
	Hashes     map[string][]string           `json:"hashes"`
	SortedSets map[string][]string           `json:"sorted_sets"`
	Strings    map[string]string             `json:"strings"`
	TTLs       map[string]int                `json:"ttls"`
	ZSetScores map[string]map[string]float64 `json:"zset_scores"`
	KeysByType map[string][]string           `json:"keys_by_type"`
}

// CaptureRedisState captures the complete current Redis state
func (te *TestEnvironment) CaptureRedisState() *RedisStateSnapshot {
	snapshot := &RedisStateSnapshot{
		Keys:       te.Redis.Keys(),
		Hashes:     make(map[string][]string),
		SortedSets: make(map[string][]string),
		Strings:    make(map[string]string),
		TTLs:       make(map[string]int),
		ZSetScores: make(map[string]map[string]float64),
		KeysByType: make(map[string][]string),
	}

	sort.Strings(snapshot.Keys)

	for _, key := range snapshot.Keys {
		keyType := te.Redis.Type(key)
		snapshot.KeysByType[keyType] = append(snapshot.KeysByType[keyType], key)

		// Capture TTL
		ttl := te.Redis.TTL(key)
		if ttl > 0 {
			snapshot.TTLs[key] = int(ttl.Seconds())
		}

		switch keyType {
		case "string":
			val, _ := te.Redis.Get(key)
			snapshot.Strings[key] = val

		case "hash":
			fields, _ := te.Redis.HKeys(key)
			sort.Strings(fields)
			snapshot.Hashes[key] = fields

		case "zset":
			members, _ := te.Redis.ZMembers(key)
			sort.Strings(members)
			snapshot.SortedSets[key] = members

			// Capture scores
			snapshot.ZSetScores[key] = make(map[string]float64)
			for _, member := range members {
				score, _ := te.Redis.ZScore(key, member)
				snapshot.ZSetScores[key][member] = score
			}
		}
	}

	return snapshot
}

// CompareRedisState compares two Redis state snapshots and reports differences
func (te *TestEnvironment) CompareRedisState(before, after *RedisStateSnapshot, operation string) {
	// Check for key count changes
	beforeCount := len(before.Keys)
	afterCount := len(after.Keys)

	if beforeCount != afterCount {
		te.t.Logf("%s: Key count changed from %d to %d", operation, beforeCount, afterCount)
	}

	// Find new keys
	beforeKeySet := make(map[string]bool)
	for _, key := range before.Keys {
		beforeKeySet[key] = true
	}

	var newKeys []string
	for _, key := range after.Keys {
		if !beforeKeySet[key] {
			newKeys = append(newKeys, key)
		}
	}

	// Find deleted keys
	afterKeySet := make(map[string]bool)
	for _, key := range after.Keys {
		afterKeySet[key] = true
	}

	var deletedKeys []string
	for _, key := range before.Keys {
		if !afterKeySet[key] {
			deletedKeys = append(deletedKeys, key)
		}
	}

	if len(newKeys) > 0 {
		te.t.Logf("%s: New keys: %v", operation, newKeys)
	}
	if len(deletedKeys) > 0 {
		te.t.Logf("%s: Deleted keys: %v", operation, deletedKeys)
	}
}

// VerifyNoResourceLeaks ensures no unexpected keys remain after operations
func (te *TestEnvironment) VerifyNoResourceLeaks(initialSnapshot *RedisStateSnapshot, allowedKeys []string) {
	currentSnapshot := te.CaptureRedisState()

	allowedKeySet := make(map[string]bool)
	for _, key := range initialSnapshot.Keys {
		allowedKeySet[key] = true
	}
	for _, key := range allowedKeys {
		allowedKeySet[key] = true
	}

	var unexpectedKeys []string
	for _, key := range currentSnapshot.Keys {
		if !allowedKeySet[key] {
			unexpectedKeys = append(unexpectedKeys, key)
		}
	}

	require.Empty(te.t, unexpectedKeys, "Found unexpected keys that may indicate resource leaks: %v", unexpectedKeys)
}

// ConstraintVerifier helps verify constraint state consistency
type ConstraintVerifier struct {
	te *TestEnvironment
}

// NewConstraintVerifier creates a new constraint state verifier
func (te *TestEnvironment) NewConstraintVerifier() *ConstraintVerifier {
	return &ConstraintVerifier{te: te}
}

// VerifyInProgressCounts checks that in-progress counts match actual Redis state
func (cv *ConstraintVerifier) VerifyInProgressCounts(constraints []ConstraintItem, expectedCounts map[string]int) {
	for i, constraint := range constraints {
		if constraint.Kind == ConstraintKindConcurrency && constraint.Concurrency != nil {
			// Check in-progress items
			itemsCount := 0
			if cv.te.Redis.Exists(constraint.Concurrency.InProgressItemKey) {
				members, _ := cv.te.Redis.ZMembers(constraint.Concurrency.InProgressItemKey)
				itemsCount = len(members)
			}

			// Check in-progress leases
			leasesCount := 0
			inProgressLeasesKey := constraint.Concurrency.InProgressLeasesKey(cv.te.KeyPrefix, cv.te.AccountID, cv.te.EnvID, cv.te.FunctionID)
			if cv.te.Redis.Exists(inProgressLeasesKey) {
				members, _ := cv.te.Redis.ZMembers(inProgressLeasesKey)
				leasesCount = len(members)
			}

			total := itemsCount + leasesCount
			constraintKey := fmt.Sprintf("constraint_%d", i)
			if expected, ok := expectedCounts[constraintKey]; ok {
				require.Equal(cv.te.t, expected, total,
					"In-progress count mismatch for constraint %d: items=%d, leases=%d, total=%d, expected=%d, all=%s",
					i, itemsCount, leasesCount, total, expected, cv.te.Redis.Dump())
			}
		}
	}
}

// VerifyLeaseDetails checks that lease details are properly stored and consistent
func (cv *ConstraintVerifier) VerifyLeaseDetails(leaseID ulid.ULID, expectedIdempotencyKey, expectedRunID string, expectedRequestID ulid.ULID) {
	leaseDetailsKey := cv.te.CapacityManager.keyLeaseDetails(cv.te.KeyPrefix, cv.te.AccountID, leaseID)

	require.True(cv.te.t, cv.te.Redis.Exists(leaseDetailsKey), "Lease details key should exist: %s", leaseDetailsKey)

	if expectedIdempotencyKey != "" {
		lik := cv.te.Redis.HGet(leaseDetailsKey, "lik")
		require.NotEmpty(cv.te.t, lik, "Lease idempotency key should be stored")
		require.Equal(cv.te.t, expectedIdempotencyKey, lik, "Lease idempotency key mismatch")
	}

	if expectedRunID != "" {
		rid := cv.te.Redis.HGet(leaseDetailsKey, "rid")
		require.NotEmpty(cv.te.t, rid, "Run ID should be stored")
		require.Equal(cv.te.t, expectedRunID, rid, "Run ID mismatch")
	}

	if !expectedRequestID.IsZero() {
		reqID := cv.te.Redis.HGet(leaseDetailsKey, "req")
		require.NotEmpty(cv.te.t, reqID, "Operation idempotency key should be stored")
		require.Equal(cv.te.t, expectedRequestID.String(), reqID, "Operation idempotency key mismatch")
	}
}

// VerifyAccountLeases checks that account leases are properly tracked
func (cv *ConstraintVerifier) VerifyAccountLeases(expectedLeases []ulid.ULID) {
	accountLeasesKey := cv.te.CapacityManager.keyAccountLeases(cv.te.KeyPrefix, cv.te.AccountID)

	if len(expectedLeases) == 0 {
		require.False(cv.te.t, cv.te.Redis.Exists(accountLeasesKey), "Account leases key should not exist when no leases expected")
		return
	}

	require.True(cv.te.t, cv.te.Redis.Exists(accountLeasesKey), "Account leases key should exist")

	members, _ := cv.te.Redis.ZMembers(accountLeasesKey)
	require.Len(cv.te.t, members, len(expectedLeases), "Account leases count mismatch")

	expectedLeaseIDs := make(map[string]bool)
	for _, leaseID := range expectedLeases {
		expectedLeaseIDs[leaseID.String()] = true
	}

	for _, member := range members {
		require.True(cv.te.t, expectedLeaseIDs[member], "Unexpected lease in account leases: %s", member)
	}
}

// VerifyScavengerShard checks that scavenger shard is properly maintained
func (cv *ConstraintVerifier) VerifyScavengerShard(expectedScore float64, shouldExist bool) {
	scavengerShardKey := cv.te.CapacityManager.keyScavengerShard(cv.te.KeyPrefix, 0)

	if !shouldExist {
		score, _ := cv.te.Redis.ZScore(scavengerShardKey, cv.te.AccountID.String())
		if score == 0 {
			return // Not in the set
		}
		require.Fail(cv.te.t, "Account should not be in scavenger shard")
		return
	}

	score, _ := cv.te.Redis.ZScore(scavengerShardKey, cv.te.AccountID.String())
	require.NotEqual(cv.te.t, 0.0, score, "Account should be in scavenger shard")
	require.Equal(cv.te.t, expectedScore, score, "Scavenger shard score mismatch")
}

// IdempotencyVerifier helps verify idempotency key behavior
type IdempotencyVerifier struct {
	te *TestEnvironment
}

// NewIdempotencyVerifier creates a new idempotency verifier
func (te *TestEnvironment) NewIdempotencyVerifier() *IdempotencyVerifier {
	return &IdempotencyVerifier{te: te}
}

// VerifyOperationIdempotency checks that operation idempotency keys are properly set
func (iv *IdempotencyVerifier) VerifyOperationIdempotency(operation, idempotencyKey string, expectedTTL int, shouldExist bool) {
	opIdempotencyKey := iv.te.CapacityManager.keyOperationIdempotency(iv.te.KeyPrefix, iv.te.AccountID, operation, idempotencyKey)

	if !shouldExist {
		require.False(iv.te.t, iv.te.Redis.Exists(opIdempotencyKey), "Operation idempotency key should not exist: %s", opIdempotencyKey)
		return
	}

	require.True(iv.te.t, iv.te.Redis.Exists(opIdempotencyKey), "Operation idempotency key should exist: %s", opIdempotencyKey)

	if expectedTTL > 0 {
		ttl := iv.te.Redis.TTL(opIdempotencyKey)
		require.True(iv.te.t, ttl > 0, "Operation idempotency key should have TTL")
		require.True(iv.te.t, int(ttl.Seconds()) <= expectedTTL, "TTL should not exceed expected value")
	}
}

// VerifyConstraintCheckIdempotency checks constraint check idempotency keys
func (iv *IdempotencyVerifier) VerifyConstraintCheckIdempotency(idempotencyKey string, expectedTTL int, shouldExist bool) {
	checkIdempotencyKey := iv.te.CapacityManager.keyConstraintCheckIdempotency(iv.te.KeyPrefix, iv.te.AccountID, idempotencyKey)

	if !shouldExist {
		require.False(iv.te.t, iv.te.Redis.Exists(checkIdempotencyKey), "Constraint check idempotency key should not exist")
		return
	}

	require.True(iv.te.t, iv.te.Redis.Exists(checkIdempotencyKey), "Constraint check idempotency key should exist")

	if expectedTTL > 0 {
		ttl := iv.te.Redis.TTL(checkIdempotencyKey)
		require.True(iv.te.t, ttl > 0, "Constraint check idempotency key should have TTL")
		require.True(iv.te.t, int(ttl.Seconds()) <= expectedTTL, "TTL should not exceed expected value")
	}
}

// RateLimitStateVerifier helps verify rate limit state
type RateLimitStateVerifier struct {
	te *TestEnvironment
}

// NewRateLimitStateVerifier creates a new rate limit state verifier
func (te *TestEnvironment) NewRateLimitStateVerifier() *RateLimitStateVerifier {
	return &RateLimitStateVerifier{te: te}
}

// VerifyRateLimitState checks that rate limit TAT values are within expected bounds
func (rv *RateLimitStateVerifier) VerifyRateLimitState(key string, expectedMinTAT, expectedMaxTAT int64) {
	if !rv.te.Redis.Exists(key) {
		rv.te.t.Logf("Rate limit key does not exist: %s", key)
		return
	}

	tatStr, _ := rv.te.Redis.Get(key)
	require.NotEmpty(rv.te.t, tatStr, "Rate limit TAT value should exist")

	tat, err := strconv.ParseInt(tatStr, 10, 64)
	require.NoError(rv.te.t, err, "TAT value should be parseable as int64")

	if expectedMinTAT > 0 {
		require.True(rv.te.t, tat >= expectedMinTAT, "TAT value %d should be >= %d", tat, expectedMinTAT)
	}

	if expectedMaxTAT > 0 {
		require.True(rv.te.t, tat <= expectedMaxTAT, "TAT value %d should be <= %d", tat, expectedMaxTAT)
	}
}

// TestDataBuilder helps build test data for various scenarios
type TestDataBuilder struct {
	te *TestEnvironment
}

// NewTestDataBuilder creates a new test data builder
func (te *TestEnvironment) NewTestDataBuilder() *TestDataBuilder {
	return &TestDataBuilder{te: te}
}

// CreateBasicRateLimitConstraint creates a basic rate limit constraint for testing
func (tb *TestDataBuilder) CreateBasicRateLimitConstraint(limit int, period int) ConstraintItem {
	return ConstraintItem{
		Kind: ConstraintKindRateLimit,
		RateLimit: &RateLimitConstraint{
			Scope:             0, // RateLimitScopeFn
			KeyExpressionHash: "test-hash",
			EvaluatedKeyHash:  "test-value",
		},
	}
}

// CreateBasicThrottleConstraint creates a basic throttle constraint for testing
func (tb *TestDataBuilder) CreateBasicThrottleConstraint(limit, burst, period int) ConstraintItem {
	return ConstraintItem{
		Kind: ConstraintKindThrottle,
		Throttle: &ThrottleConstraint{
			Scope:             2, // ThrottleScopeFn
			KeyExpressionHash: "throttle-hash",
			EvaluatedKeyHash:  "throttle-value",
		},
	}
}

// CreateBasicConstraintConfig creates a basic constraint configuration
func (tb *TestDataBuilder) CreateBasicConstraintConfig(concurrencyLimit, rateLimitAmount, rateLimitPeriod int) ConstraintConfig {
	return ConstraintConfig{
		FunctionVersion: 1,
		Concurrency: ConcurrencyConfig{
			FunctionConcurrency: concurrencyLimit,
		},
		RateLimit: []RateLimitConfig{
			{
				Scope:             0, // RateLimitScopeFn
				Limit:             rateLimitAmount,
				Period:            rateLimitPeriod,
				KeyExpressionHash: "test-hash",
			},
		},
		Throttle: []ThrottleConfig{
			{
				Scope:             2, // ThrottleScopeFn
				Limit:             rateLimitAmount,
				Burst:             rateLimitAmount / 2,
				Period:            rateLimitPeriod,
				KeyExpressionHash: "throttle-hash",
			},
		},
	}
}

// AdvanceTimeAndRedis advances both the clock and Redis time for testing
func (te *TestEnvironment) AdvanceTimeAndRedis(duration time.Duration) {
	te.Redis.FastForward(duration)
	te.Redis.SetTime(time.Now().Add(duration))
}

// LogRedisState logs the current Redis state for debugging
func (te *TestEnvironment) LogRedisState(prefix string) {
	snapshot := te.CaptureRedisState()
	te.t.Logf("%s - Redis state: %d keys", prefix, len(snapshot.Keys))

	for keyType, keys := range snapshot.KeysByType {
		if len(keys) > 0 {
			te.t.Logf("  %s keys (%d): %v", keyType, len(keys), keys)
		}
	}
}
