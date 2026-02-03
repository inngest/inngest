package constraintapi

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/util"
)

func TestConstraintEnforcement(t *testing.T) {
	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()

	type deps struct {
		cm    *redisCapacityManager
		clock clockwork.FakeClock
		r     *miniredis.Miniredis
		rc    rueidis.Client

		config      ConstraintConfig
		constraints []ConstraintItem

		acquireRequestID      ulid.ULID
		acquireIdempotencyKey string
	}

	type testCase struct {
		name string

		amount      int
		config      ConstraintConfig
		constraints []ConstraintItem
		mi          MigrationIdentifier

		beforeAcquire func(t *testing.T, deps *deps)

		afterPreAcquireCheck  func(t *testing.T, deps *deps, resp *CapacityCheckResponse)
		afterAcquire          func(t *testing.T, deps *deps, resp *CapacityAcquireResponse)
		afterPostAcquireCheck func(t *testing.T, deps *deps, resp *CapacityCheckResponse)

		expectedLeaseAmount int

		afterExtend  func(t *testing.T, deps *deps, resp *CapacityExtendLeaseResponse)
		afterRelease func(t *testing.T, deps *deps, resp *CapacityReleaseResponse)
	}

	testCases := []testCase{
		{
			name: "account concurrency",
			config: ConstraintConfig{
				FunctionVersion: 1,
				Concurrency: ConcurrencyConfig{
					AccountConcurrency: 10,
				},
			},
			constraints: []ConstraintItem{
				{
					Kind: ConstraintKindConcurrency,
					Concurrency: &ConcurrencyConstraint{
						Scope:             enums.ConcurrencyScopeAccount,
						Mode:              enums.ConcurrencyModeStep,
						InProgressItemKey: fmt.Sprintf("{q:v1}:concurrency:account:%s", accountID),
					},
				},
			},
			amount:              1,
			expectedLeaseAmount: 1,
			mi: MigrationIdentifier{
				QueueShard: "test",
			},
			afterAcquire: func(t *testing.T, deps *deps, resp *CapacityAcquireResponse) {
				cm := deps.cm
				r := deps.r
				// All keys should exist
				keys := deps.r.Keys()
				keyInProgressLeases := deps.constraints[0].Concurrency.InProgressLeasesKey(cm.queueStateKeyPrefix, accountID, envID, fnID)
				require.Len(t, keys, 8)
				require.Subset(t, []string{
					cm.keyScavengerShard(cm.queueStateKeyPrefix, 0),
					cm.keyAccountLeases(cm.queueStateKeyPrefix, accountID),
					cm.keyOperationIdempotency(cm.queueStateKeyPrefix, accountID, "acq", deps.acquireIdempotencyKey),
					cm.keyConstraintCheckIdempotency(cm.queueStateKeyPrefix, accountID, "acquire"),
					cm.keyConstraintCheckIdempotency(cm.queueStateKeyPrefix, accountID, "item0"),
					cm.keyRequestState(cm.queueStateKeyPrefix, accountID, deps.acquireRequestID),
					cm.keyLeaseDetails(cm.queueStateKeyPrefix, accountID, resp.Leases[0].LeaseID),
					keyInProgressLeases,
					fmt.Sprintf("{q:v1}:concurrency:account:%s", accountID),
				}, keys)

				// In progress leases should have a single item
				mem, err := r.ZMembers(keyInProgressLeases)
				require.NoError(t, err)
				require.Len(t, mem, 1)
				require.Contains(t, mem, resp.Leases[0].LeaseID.String())

				// Score must be lease expiry
				score, err := r.ZScore(keyInProgressLeases, resp.Leases[0].LeaseID.String())
				require.NoError(t, err)

				require.Equal(t, float64(resp.Leases[0].LeaseID.Timestamp().UnixMilli()), score)
			},
			afterExtend: func(t *testing.T, deps *deps, resp *CapacityExtendLeaseResponse) {
				cm := deps.cm
				r := deps.r
				constraints := deps.constraints
				// All keys should exist
				keys := r.Keys()
				keyInProgressLeases := constraints[0].Concurrency.InProgressLeasesKey(cm.queueStateKeyPrefix, accountID, envID, fnID)
				require.Len(t, keys, 9)
				require.Subset(t, []string{
					cm.keyScavengerShard(cm.queueStateKeyPrefix, 0),
					cm.keyAccountLeases(cm.queueStateKeyPrefix, accountID),
					cm.keyOperationIdempotency(cm.queueStateKeyPrefix, accountID, "acq", deps.acquireIdempotencyKey),
					cm.keyOperationIdempotency(cm.queueStateKeyPrefix, accountID, "ext", "extend"),
					cm.keyConstraintCheckIdempotency(cm.queueStateKeyPrefix, accountID, "acquire"),
					cm.keyConstraintCheckIdempotency(cm.queueStateKeyPrefix, accountID, "item0"),
					cm.keyRequestState(cm.queueStateKeyPrefix, accountID, deps.acquireRequestID),
					cm.keyLeaseDetails(cm.queueStateKeyPrefix, accountID, *resp.LeaseID),
					keyInProgressLeases,
					fmt.Sprintf("{q:v1}:concurrency:account:%s", accountID),
				}, keys)

				// In progress leases should have a single item
				mem, err := r.ZMembers(keyInProgressLeases)
				require.NoError(t, err)
				require.Len(t, mem, 1)
				require.Contains(t, mem, resp.LeaseID.String())

				// Score must be lease expiry
				score, err := r.ZScore(keyInProgressLeases, resp.LeaseID.String())
				require.NoError(t, err)

				require.Equal(t, float64(resp.LeaseID.Timestamp().UnixMilli()), score)
			},
			afterRelease: func(t *testing.T, deps *deps, resp *CapacityReleaseResponse) {
				cm := deps.cm
				r := deps.r

				// Keys should be cleaned up
				keys := r.Keys()
				require.Len(t, keys, 5)
				require.Subset(t, []string{
					cm.keyOperationIdempotency(cm.queueStateKeyPrefix, accountID, "acq", deps.acquireIdempotencyKey),
					cm.keyOperationIdempotency(cm.queueStateKeyPrefix, accountID, "ext", "extend"),
					cm.keyOperationIdempotency(cm.queueStateKeyPrefix, accountID, "rel", "release"),
					cm.keyConstraintCheckIdempotency(cm.queueStateKeyPrefix, accountID, "acquire"),
					cm.keyConstraintCheckIdempotency(cm.queueStateKeyPrefix, accountID, "item0"),
				}, keys)
			},
		},

		{
			name: "missing account concurrency",
			config: ConstraintConfig{
				FunctionVersion: 1,
				Concurrency: ConcurrencyConfig{
					AccountConcurrency: 1,
				},
			},
			constraints: []ConstraintItem{
				{
					Kind: ConstraintKindConcurrency,
					Concurrency: &ConcurrencyConstraint{
						Scope:             enums.ConcurrencyScopeAccount,
						Mode:              enums.ConcurrencyModeStep,
						InProgressItemKey: fmt.Sprintf("{q:v1}:concurrency:account:%s", accountID),
					},
				},
			},
			amount:              1,
			expectedLeaseAmount: 1,
			mi: MigrationIdentifier{
				QueueShard: "test",
			},
			afterAcquire: func(t *testing.T, deps *deps, resp *CapacityAcquireResponse) {
				cm := deps.cm

				res, err := cm.Acquire(context.Background(), &CapacityAcquireRequest{
					IdempotencyKey: "before-acquire-acquire-call",
					AccountID:      accountID,
					EnvID:          envID,
					FunctionID:     fnID,

					Duration: 5 * time.Second,

					Configuration:        deps.config,
					Constraints:          deps.constraints,
					Amount:               1,
					LeaseIdempotencyKeys: []string{"test1"},

					CurrentTime:     deps.clock.Now(),
					MaximumLifetime: time.Minute,

					Source: LeaseSource{
						Service:           ServiceAPI,
						Location:          CallerLocationBacklogRefill,
						RunProcessingMode: RunProcessingModeBackground,
					},

					Migration: MigrationIdentifier{
						QueueShard: "test",
					},
				})
				require.NoError(t, err)
				require.Len(t, res.Leases, 0)
				require.Len(t, res.LimitingConstraints, 1)
				require.Len(t, res.ExhaustedConstraints, 1)
				require.Equal(t, ConstraintKindConcurrency, res.ExhaustedConstraints[0].Kind)
				require.Equal(t, deps.clock.Now().Add(ConcurrencyLimitRetryAfter), res.RetryAfter)
			},
		},

		{
			name: "account concurrency limited due to legacy concurrency",
			config: ConstraintConfig{
				FunctionVersion: 1,
				Concurrency: ConcurrencyConfig{
					AccountConcurrency: 10,
				},
			},
			constraints: []ConstraintItem{
				{
					Kind: ConstraintKindConcurrency,
					Concurrency: &ConcurrencyConstraint{
						Scope:             enums.ConcurrencyScopeAccount,
						Mode:              enums.ConcurrencyModeStep,
						InProgressItemKey: fmt.Sprintf("{q:v1}:concurrency:account:%s", accountID),
					},
				},
			},
			mi: MigrationIdentifier{
				QueueShard: "test",
			},
			beforeAcquire: func(t *testing.T, deps *deps) {
				r := deps.r
				clock := deps.clock
				// Simulate existing concurrency usage (in progress item Leased by queue)
				for i := range 10 {
					_, err := r.ZAdd(
						fmt.Sprintf("{q:v1}:concurrency:account:%s", accountID),
						float64(clock.Now().Add(time.Second).UnixMilli()),
						fmt.Sprintf("queueItem%d", i),
					)
					require.NoError(t, err)
				}
			},
			amount:              1,
			expectedLeaseAmount: 0,
			afterAcquire: func(t *testing.T, deps *deps, resp *CapacityAcquireResponse) {
				require.Equal(t, 0, len(resp.Leases))
				require.Equal(t, deps.clock.Now().Add(ConcurrencyLimitRetryAfter), resp.RetryAfter)
			},
		},

		{
			name: "ignore account concurrency claimed by expired capacity lease",
			config: ConstraintConfig{
				FunctionVersion: 1,
				Concurrency: ConcurrencyConfig{
					AccountConcurrency: 10,
				},
			},
			constraints: []ConstraintItem{
				{
					Kind: ConstraintKindConcurrency,
					Concurrency: &ConcurrencyConstraint{
						Scope:             enums.ConcurrencyScopeAccount,
						Mode:              enums.ConcurrencyModeStep,
						InProgressItemKey: fmt.Sprintf("{q:v1}:concurrency:account:%s", accountID),
					},
				},
			},
			mi: MigrationIdentifier{
				QueueShard: "test",
			},
			beforeAcquire: func(t *testing.T, deps *deps) {
				r := deps.r
				clock := deps.clock
				cm := deps.cm
				// Claim capacity initially

				leaseIdempotencyKeys := make([]string, 10)
				for i := range 10 {
					leaseIdempotencyKeys[i] = fmt.Sprintf("oldItem%d", i)
				}

				var err error

				res, err := cm.Acquire(context.Background(), &CapacityAcquireRequest{
					IdempotencyKey: "before-acquire-acquire-call",
					AccountID:      accountID,
					EnvID:          envID,
					FunctionID:     fnID,

					Duration: 5 * time.Second,

					Configuration:        deps.config,
					Constraints:          deps.constraints,
					Amount:               10,
					LeaseIdempotencyKeys: leaseIdempotencyKeys,

					CurrentTime:     clock.Now(),
					MaximumLifetime: time.Minute,

					Source: LeaseSource{
						Service:           ServiceAPI,
						Location:          CallerLocationBacklogRefill,
						RunProcessingMode: RunProcessingModeBackground,
					},

					Migration: MigrationIdentifier{
						QueueShard: "test",
					},
				})
				require.NoError(t, err)
				require.Len(t, res.Leases, 10)

				// Expect in progress leases set to be populated
				mem, err := r.ZMembers(cm.KeyInProgressLeasesAccount(accountID))
				require.NoError(t, err)
				require.Len(t, mem, 10)

				// Fast forward to expire lease (but do not scavenge yet)
				clock.Advance(10 * time.Second)
				r.FastForward(10 * time.Second)
				r.SetTime(clock.Now())
			},
			amount:              10,
			expectedLeaseAmount: 10,
			afterAcquire: func(t *testing.T, deps *deps, resp *CapacityAcquireResponse) {
				r := deps.r
				cm := deps.cm
				rc := deps.rc
				clock := deps.clock

				// Even though there's an expired lease, we expect to claim 10 new leases with expired concurrency capacity
				require.Len(t, resp.Leases, 10)

				// Expect in progress leases set to be populated with expired and non-expired items
				mem, err := r.ZMembers(cm.KeyInProgressLeasesAccount(accountID))
				require.NoError(t, err)
				require.Len(t, mem, 20)

				expiry := fmt.Sprintf("%d", clock.Now().UnixMilli())

				// Count expired
				cmd := rc.B().Zcount().
					Key(cm.KeyInProgressLeasesAccount(accountID)).
					Min("-inf").
					Max(expiry).
					Build()
				count, err := rc.Do(context.Background(), cmd).ToInt64()
				require.NoError(t, err)
				require.Equal(t, int64(10), count)

				// Count active
				cmd = rc.B().Zcount().
					Key(cm.KeyInProgressLeasesAccount(accountID)).
					Min(expiry).
					Max("+inf").
					Build()
				count, err = rc.Do(context.Background(), cmd).ToInt64()
				require.NoError(t, err)
				require.Equal(t, int64(10), count)
			},
		},

		{
			name: "account concurrency partially limited due to legacy concurrency",
			config: ConstraintConfig{
				FunctionVersion: 1,
				Concurrency: ConcurrencyConfig{
					AccountConcurrency: 10,
				},
			},
			constraints: []ConstraintItem{
				{
					Kind: ConstraintKindConcurrency,
					Concurrency: &ConcurrencyConstraint{
						Scope:             enums.ConcurrencyScopeAccount,
						Mode:              enums.ConcurrencyModeStep,
						InProgressItemKey: fmt.Sprintf("{q:v1}:concurrency:account:%s", accountID),
					},
				},
			},
			mi: MigrationIdentifier{
				QueueShard: "test",
			},
			beforeAcquire: func(t *testing.T, deps *deps) {
				r := deps.r
				clock := deps.clock
				// Simulate existing concurrency usage (in progress item Leased by queue)
				for i := range 5 { // 5/10
					_, err := r.ZAdd(
						fmt.Sprintf("{q:v1}:concurrency:account:%s", accountID),
						float64(clock.Now().Add(time.Second).UnixMilli()),
						fmt.Sprintf("queueItem%d", i),
					)
					require.NoError(t, err)
				}
			},
			amount:              10,
			expectedLeaseAmount: 5,
		},

		{
			name: "function concurrency",
			config: ConstraintConfig{
				FunctionVersion: 1,
				Concurrency: ConcurrencyConfig{
					FunctionConcurrency: 10,
				},
			},
			constraints: []ConstraintItem{
				{
					Kind: ConstraintKindConcurrency,
					Concurrency: &ConcurrencyConstraint{
						Scope:             enums.ConcurrencyScopeFn,
						Mode:              enums.ConcurrencyModeStep,
						InProgressItemKey: fmt.Sprintf("{q:v1}:concurrency:p:%s", fnID),
					},
				},
			},
			amount:              1,
			expectedLeaseAmount: 1,
			mi: MigrationIdentifier{
				QueueShard: "test",
			},
			afterAcquire: func(t *testing.T, deps *deps, resp *CapacityAcquireResponse) {
				r := deps.r
				cm := deps.cm
				constraints := deps.constraints
				// All keys should exist
				keys := r.Keys()
				keyInProgressLeases := constraints[0].Concurrency.InProgressLeasesKey(cm.queueStateKeyPrefix, accountID, envID, fnID)

				t.Log(deps.acquireIdempotencyKey)
				t.Log(util.XXHash(deps.acquireIdempotencyKey))

				require.Len(t, keys, 8)
				require.Subset(t, []string{
					cm.keyScavengerShard(cm.queueStateKeyPrefix, 0),
					cm.keyAccountLeases(cm.queueStateKeyPrefix, accountID),
					cm.keyOperationIdempotency(cm.queueStateKeyPrefix, accountID, "acq", deps.acquireIdempotencyKey),
					cm.keyConstraintCheckIdempotency(cm.queueStateKeyPrefix, accountID, "acquire"),
					cm.keyConstraintCheckIdempotency(cm.queueStateKeyPrefix, accountID, "item0"),
					cm.keyRequestState(cm.queueStateKeyPrefix, accountID, deps.acquireRequestID),
					cm.keyLeaseDetails(cm.queueStateKeyPrefix, accountID, resp.Leases[0].LeaseID),
					keyInProgressLeases,
				}, keys, r.Dump())

				// In progress leases should have a single item
				mem, err := r.ZMembers(keyInProgressLeases)
				require.NoError(t, err)
				require.Len(t, mem, 1)
				require.Contains(t, mem, resp.Leases[0].LeaseID.String())

				// Score must be lease expiry
				score, err := r.ZScore(keyInProgressLeases, resp.Leases[0].LeaseID.String())
				require.NoError(t, err)

				require.Equal(t, float64(resp.Leases[0].LeaseID.Timestamp().UnixMilli()), score)
			},
			afterExtend: func(t *testing.T, deps *deps, resp *CapacityExtendLeaseResponse) {
				r := deps.r
				cm := deps.cm
				constraints := deps.constraints

				// All keys should exist
				keys := r.Keys()
				keyInProgressLeases := constraints[0].Concurrency.InProgressLeasesKey(cm.queueStateKeyPrefix, accountID, envID, fnID)
				require.Len(t, keys, 9)
				require.Subset(t, []string{
					cm.keyScavengerShard(cm.queueStateKeyPrefix, 0),
					cm.keyAccountLeases(cm.queueStateKeyPrefix, accountID),
					cm.keyOperationIdempotency(cm.queueStateKeyPrefix, accountID, "acq", deps.acquireIdempotencyKey),
					cm.keyOperationIdempotency(cm.queueStateKeyPrefix, accountID, "ext", "extend"),
					cm.keyConstraintCheckIdempotency(cm.queueStateKeyPrefix, accountID, "acquire"),
					cm.keyConstraintCheckIdempotency(cm.queueStateKeyPrefix, accountID, "item0"),
					cm.keyRequestState(cm.queueStateKeyPrefix, accountID, deps.acquireRequestID),
					cm.keyLeaseDetails(cm.queueStateKeyPrefix, accountID, *resp.LeaseID),
					keyInProgressLeases,
				}, keys)

				// In progress leases should have a single item
				mem, err := r.ZMembers(keyInProgressLeases)
				require.NoError(t, err)
				require.Len(t, mem, 1)
				require.Contains(t, mem, resp.LeaseID.String())

				// Score must be lease expiry
				score, err := r.ZScore(keyInProgressLeases, resp.LeaseID.String())
				require.NoError(t, err)

				require.Equal(t, float64(resp.LeaseID.Timestamp().UnixMilli()), score)
			},
			afterRelease: func(t *testing.T, deps *deps, resp *CapacityReleaseResponse) {
				r := deps.r
				cm := deps.cm

				// Keys should be cleaned up
				keys := r.Keys()
				require.Len(t, keys, 5)
				require.Subset(t, []string{
					cm.keyOperationIdempotency(cm.queueStateKeyPrefix, accountID, "acq", deps.acquireIdempotencyKey),
					cm.keyOperationIdempotency(cm.queueStateKeyPrefix, accountID, "ext", "extend"),
					cm.keyOperationIdempotency(cm.queueStateKeyPrefix, accountID, "rel", "release"),
					cm.keyConstraintCheckIdempotency(cm.queueStateKeyPrefix, accountID, "acquire"),
					cm.keyConstraintCheckIdempotency(cm.queueStateKeyPrefix, accountID, "item0"),
				}, keys)
			},
		},

		{
			name: "custom concurrency",
			config: ConstraintConfig{
				FunctionVersion: 1,
				Concurrency: ConcurrencyConfig{
					CustomConcurrencyKeys: []CustomConcurrencyLimit{
						{
							Mode:              enums.ConcurrencyModeStep,
							Scope:             enums.ConcurrencyScopeEnv,
							KeyExpressionHash: "static-key",
							Limit:             5,
						},
					},
				},
			},
			constraints: []ConstraintItem{
				{
					Kind: ConstraintKindConcurrency,
					Concurrency: &ConcurrencyConstraint{
						Scope:             enums.ConcurrencyScopeEnv,
						Mode:              enums.ConcurrencyModeStep,
						KeyExpressionHash: "static-key",
						EvaluatedKeyHash:  "inngest",
						InProgressItemKey: fmt.Sprintf("{q:v1}:concurrency:custom:e:%s:%s", fnID, util.XXHash("inngest")),
					},
				},
			},
			amount:              1,
			expectedLeaseAmount: 1,
			mi: MigrationIdentifier{
				QueueShard: "test",
			},
			afterAcquire: func(t *testing.T, deps *deps, resp *CapacityAcquireResponse) {
				r := deps.r
				cm := deps.cm
				constraints := deps.constraints

				// All keys should exist
				keys := r.Keys()
				keyInProgressLeases := constraints[0].Concurrency.InProgressLeasesKey(cm.queueStateKeyPrefix, accountID, envID, fnID)
				require.Len(t, keys, 8)
				require.Subset(t, []string{
					cm.keyScavengerShard(cm.queueStateKeyPrefix, 0),
					cm.keyAccountLeases(cm.queueStateKeyPrefix, accountID),
					cm.keyOperationIdempotency(cm.queueStateKeyPrefix, accountID, "acq", deps.acquireIdempotencyKey),
					cm.keyConstraintCheckIdempotency(cm.queueStateKeyPrefix, accountID, "acquire"),
					cm.keyConstraintCheckIdempotency(cm.queueStateKeyPrefix, accountID, "item0"),
					cm.keyRequestState(cm.queueStateKeyPrefix, accountID, deps.acquireRequestID),
					cm.keyLeaseDetails(cm.queueStateKeyPrefix, accountID, resp.Leases[0].LeaseID),
					keyInProgressLeases,
				}, keys)

				// In progress leases should have a single item
				mem, err := r.ZMembers(keyInProgressLeases)
				require.NoError(t, err)
				require.Len(t, mem, 1)
				require.Contains(t, mem, resp.Leases[0].LeaseID.String())

				// Score must be lease expiry
				score, err := r.ZScore(keyInProgressLeases, resp.Leases[0].LeaseID.String())
				require.NoError(t, err)

				require.Equal(t, float64(resp.Leases[0].LeaseID.Timestamp().UnixMilli()), score)
			},
			afterExtend: func(t *testing.T, deps *deps, resp *CapacityExtendLeaseResponse) {
				r := deps.r
				cm := deps.cm
				constraints := deps.constraints

				// All keys should exist
				keys := r.Keys()
				keyInProgressLeases := constraints[0].Concurrency.InProgressLeasesKey(cm.queueStateKeyPrefix, accountID, envID, fnID)
				require.Len(t, keys, 9)
				require.Subset(t, []string{
					cm.keyScavengerShard(cm.queueStateKeyPrefix, 0),
					cm.keyAccountLeases(cm.queueStateKeyPrefix, accountID),
					cm.keyOperationIdempotency(cm.queueStateKeyPrefix, accountID, "acq", deps.acquireIdempotencyKey),
					cm.keyOperationIdempotency(cm.queueStateKeyPrefix, accountID, "ext", "extend"),
					cm.keyConstraintCheckIdempotency(cm.queueStateKeyPrefix, accountID, "acquire"),
					cm.keyConstraintCheckIdempotency(cm.queueStateKeyPrefix, accountID, "item0"),
					cm.keyRequestState(cm.queueStateKeyPrefix, accountID, deps.acquireRequestID),
					cm.keyLeaseDetails(cm.queueStateKeyPrefix, accountID, *resp.LeaseID),
					keyInProgressLeases,
				}, keys)

				// In progress leases should have a single item
				mem, err := r.ZMembers(keyInProgressLeases)
				require.NoError(t, err)
				require.Len(t, mem, 1)
				require.Contains(t, mem, resp.LeaseID.String())

				// Score must be lease expiry
				score, err := r.ZScore(keyInProgressLeases, resp.LeaseID.String())
				require.NoError(t, err)

				require.Equal(t, float64(resp.LeaseID.Timestamp().UnixMilli()), score)
			},
			afterRelease: func(t *testing.T, deps *deps, resp *CapacityReleaseResponse) {
				r := deps.r
				cm := deps.cm

				// Keys should be cleaned up
				keys := r.Keys()
				require.Len(t, keys, 5)
				require.Subset(t, []string{
					cm.keyOperationIdempotency(cm.queueStateKeyPrefix, accountID, "acq", deps.acquireIdempotencyKey),
					cm.keyOperationIdempotency(cm.queueStateKeyPrefix, accountID, "ext", "extend"),
					cm.keyOperationIdempotency(cm.queueStateKeyPrefix, accountID, "rel", "release"),
					cm.keyConstraintCheckIdempotency(cm.queueStateKeyPrefix, accountID, "acquire"),
					cm.keyConstraintCheckIdempotency(cm.queueStateKeyPrefix, accountID, "item0"),
				}, keys)
			},
		},

		{
			name: "throttle allowed",
			mi: MigrationIdentifier{
				QueueShard: "test",
			},
			config: ConstraintConfig{
				FunctionVersion: 1,
				Throttle: []ThrottleConfig{
					{
						Limit:             1,
						Period:            60,
						KeyExpressionHash: "expr-hash",
					},
				},
			},
			constraints: []ConstraintItem{
				{
					Kind: ConstraintKindThrottle,
					Throttle: &ThrottleConstraint{
						KeyExpressionHash: "expr-hash",
						EvaluatedKeyHash:  "key-hash",
					},
				},
			},
			amount:              1,
			expectedLeaseAmount: 1,
			afterPreAcquireCheck: func(t *testing.T, deps *deps, resp *CapacityCheckResponse) {
				require.Equal(t, 0, resp.Usage[0].Used)
			},
			afterAcquire: func(t *testing.T, deps *deps, resp *CapacityAcquireResponse) {
			},
			afterPostAcquireCheck: func(t *testing.T, deps *deps, resp *CapacityCheckResponse) {
				require.Equal(t, 1, resp.Usage[0].Used)
			},
		},

		{
			name: "throttle rejected",
			mi: MigrationIdentifier{
				QueueShard: "test",
			},
			config: ConstraintConfig{
				FunctionVersion: 1,
				Throttle: []ThrottleConfig{
					{
						Limit:             1,
						Period:            3600,
						KeyExpressionHash: "expr-hash",
					},
				},
			},
			constraints: []ConstraintItem{
				{
					Kind: ConstraintKindThrottle,
					Throttle: &ThrottleConstraint{
						KeyExpressionHash: "expr-hash",
						EvaluatedKeyHash:  "key-hash",
					},
				},
			},
			amount:              2,
			expectedLeaseAmount: 1,
			afterPreAcquireCheck: func(t *testing.T, deps *deps, resp *CapacityCheckResponse) {
				require.Equal(t, 0, resp.Usage[0].Used)
			},
			afterAcquire: func(t *testing.T, deps *deps, resp *CapacityAcquireResponse) {
				t.Log(resp.Debug())

				require.Len(t, resp.Leases, 1)

				require.Len(t, resp.LimitingConstraints, 1)
				require.Equal(t, ConstraintKindThrottle, resp.LimitingConstraints[0].Kind)
				require.Len(t, resp.ExhaustedConstraints, 1)
				require.Equal(t, ConstraintKindThrottle, resp.ExhaustedConstraints[0].Kind)
				require.False(t, resp.RetryAfter.IsZero())
				// Next unit will be available in 1h
				require.WithinDuration(t, deps.clock.Now().Add(time.Hour), resp.RetryAfter, time.Second)
			},
			afterPostAcquireCheck: func(t *testing.T, deps *deps, resp *CapacityCheckResponse) {
				require.Equal(t, 1, resp.Usage[0].Used)
				require.Equal(t, 1, resp.Usage[0].Limit)
			},
		},

		{
			name: "throttle allowed with legacy state",
			mi: MigrationIdentifier{
				QueueShard: "test",
			},
			config: ConstraintConfig{
				FunctionVersion: 1,
				Throttle: []ThrottleConfig{
					{
						Limit:             5,
						Period:            60,
						KeyExpressionHash: "expr-hash",
					},
				},
			},
			constraints: []ConstraintItem{
				{
					Kind: ConstraintKindThrottle,
					Throttle: &ThrottleConstraint{
						KeyExpressionHash: "expr-hash",
						EvaluatedKeyHash:  "key-hash",
					},
				},
			},
			amount:              1,
			expectedLeaseAmount: 1,
			beforeAcquire: func(t *testing.T, deps *deps) {
				// Set existing legacy state
				tat := deps.clock.Now().Add(24 * time.Second).UnixMilli()
				err := deps.r.Set("{q:v1}:throttle:key-hash", strconv.Itoa(int(tat)))
				require.NoError(t, err)
			},
			afterPreAcquireCheck: func(t *testing.T, deps *deps, resp *CapacityCheckResponse) {
				require.Equal(t, 2, resp.Usage[0].Used)

				raw, err := deps.r.Get("{q:v1}:throttle:key-hash")
				require.NoError(t, err)
				parsed, err := strconv.Atoi(raw)
				require.NoError(t, err)
				tat := time.UnixMilli(int64(parsed))
				require.WithinDuration(t, deps.clock.Now().Add(24*time.Second), tat, time.Second)
			},
			afterAcquire: func(t *testing.T, deps *deps, resp *CapacityAcquireResponse) {
				require.True(t, resp.RetryAfter.IsZero())

				raw, err := deps.r.Get("{q:v1}:throttle:key-hash")
				require.NoError(t, err)
				parsed, err := strconv.Atoi(raw)
				require.NoError(t, err)
				tat := time.UnixMilli(int64(parsed))
				require.WithinDuration(t, deps.clock.Now().Add(36*time.Second), tat, time.Second)
			},
			afterPostAcquireCheck: func(t *testing.T, deps *deps, resp *CapacityCheckResponse) {
				require.Equal(t, 3, resp.Usage[0].Used)
			},
		},

		{
			name: "throttle partially rejected with legacy state",
			mi: MigrationIdentifier{
				QueueShard: "test",
			},
			config: ConstraintConfig{
				FunctionVersion: 1,
				Throttle: []ThrottleConfig{
					{
						Limit:             5,
						Period:            60,
						KeyExpressionHash: "expr-hash",
					},
				},
			},
			constraints: []ConstraintItem{
				{
					Kind: ConstraintKindThrottle,
					Throttle: &ThrottleConstraint{
						KeyExpressionHash: "expr-hash",
						EvaluatedKeyHash:  "key-hash",
					},
				},
			},
			amount:              2,
			expectedLeaseAmount: 1,
			beforeAcquire: func(t *testing.T, deps *deps) {
				// Set existing legacy state
				tat := deps.clock.Now().Add(48 * time.Second).UnixMilli()
				err := deps.r.Set("{q:v1}:throttle:key-hash", strconv.Itoa(int(tat)))
				require.NoError(t, err)
			},
			afterPreAcquireCheck: func(t *testing.T, deps *deps, resp *CapacityCheckResponse) {
				// The initial state accounts for 4 requests
				require.Equal(t, 4, resp.Usage[0].Used)
			},
			afterAcquire: func(t *testing.T, deps *deps, resp *CapacityAcquireResponse) {
				// This should be the actual value, accounting for 5 requests
				raw, err := deps.r.Get("{q:v1}:throttle:key-hash")
				require.NoError(t, err)
				parsed, err := strconv.Atoi(raw)
				require.NoError(t, err)
				tat := time.UnixMilli(int64(parsed))
				require.WithinDuration(t, deps.clock.Now().Add(60*time.Second), tat, time.Second)

				t.Log("now", deps.clock.Now())
				t.Log("retry", resp.RetryAfter)

				// Wait one "window", 12s, until the next request can happen
				require.WithinDuration(t, deps.clock.Now().Add(12*time.Second), resp.RetryAfter, time.Second)
			},
			afterPostAcquireCheck: func(t *testing.T, deps *deps, resp *CapacityCheckResponse) {
				// We are now accounting for 5 requests
				require.Equal(t, 5, resp.Usage[0].Used)
			},
		},

		{
			name: "ratelimit allowed",
			mi: MigrationIdentifier{
				IsRateLimit: true,
			},
			config: ConstraintConfig{
				FunctionVersion: 1,
				RateLimit: []RateLimitConfig{
					{
						Limit:             1,
						Period:            60,
						KeyExpressionHash: "expr-hash",
					},
				},
			},
			constraints: []ConstraintItem{
				{
					Kind: ConstraintKindRateLimit,
					RateLimit: &RateLimitConstraint{
						KeyExpressionHash: "expr-hash",
						EvaluatedKeyHash:  "key-hash",
					},
				},
			},
			amount:              1,
			expectedLeaseAmount: 1,
			afterPreAcquireCheck: func(t *testing.T, deps *deps, resp *CapacityCheckResponse) {
				t.Log(resp.Debug())
				require.Equal(t, 0, resp.Usage[0].Used)
			},
			afterAcquire: func(t *testing.T, deps *deps, resp *CapacityAcquireResponse) {
				t.Log(resp.Debug())
			},
			afterPostAcquireCheck: func(t *testing.T, deps *deps, resp *CapacityCheckResponse) {
				require.Equal(t, 1, resp.Usage[0].Used)
			},
		},

		{
			name: "rate limit rejected",
			mi: MigrationIdentifier{
				IsRateLimit: true,
			},
			config: ConstraintConfig{
				FunctionVersion: 1,
				RateLimit: []RateLimitConfig{
					{
						Limit:             1,
						Period:            60,
						KeyExpressionHash: "expr-hash",
					},
				},
			},
			constraints: []ConstraintItem{
				{
					Kind: ConstraintKindRateLimit,
					RateLimit: &RateLimitConstraint{
						KeyExpressionHash: "expr-hash",
						EvaluatedKeyHash:  "key-hash",
					},
				},
			},
			amount:              2,
			expectedLeaseAmount: 1,
			afterPreAcquireCheck: func(t *testing.T, deps *deps, resp *CapacityCheckResponse) {
				require.Equal(t, 0, resp.Usage[0].Used)
			},
			afterAcquire: func(t *testing.T, deps *deps, resp *CapacityAcquireResponse) {
				t.Log(resp.Debug())

				require.Len(t, resp.LimitingConstraints, 1)
				require.Equal(t, ConstraintKindRateLimit, resp.LimitingConstraints[0].Kind)
				require.Len(t, resp.ExhaustedConstraints, 1)
				require.Equal(t, ConstraintKindRateLimit, resp.ExhaustedConstraints[0].Kind)
				require.False(t, resp.RetryAfter.IsZero())
				// Next unit will be available in 1m
				require.WithinDuration(t, deps.clock.Now().Add(time.Minute), resp.RetryAfter, time.Second)
			},
			afterPostAcquireCheck: func(t *testing.T, deps *deps, resp *CapacityCheckResponse) {
				require.Equal(t, 1, resp.Usage[0].Used)
			},
		},

		{
			name: "rate limit allowed with legacy state",
			mi: MigrationIdentifier{
				IsRateLimit: true,
			},
			config: ConstraintConfig{
				FunctionVersion: 1,
				RateLimit: []RateLimitConfig{
					{
						Limit:             10,
						Period:            60,
						KeyExpressionHash: "expr-hash",
					},
				},
			},
			constraints: []ConstraintItem{
				{
					Kind: ConstraintKindRateLimit,
					RateLimit: &RateLimitConstraint{
						KeyExpressionHash: "expr-hash",
						EvaluatedKeyHash:  "key-hash",
					},
				},
			},
			amount:              1,
			expectedLeaseAmount: 1,
			beforeAcquire: func(t *testing.T, deps *deps) {
				// Set existing legacy state
				tat := deps.clock.Now().Add(6 * time.Second).UnixNano()
				err := deps.r.Set("{rl}:key-hash", strconv.Itoa(int(tat)))
				require.NoError(t, err)
			},
			afterPreAcquireCheck: func(t *testing.T, deps *deps, resp *CapacityCheckResponse) {
				t.Log(resp.Debug())

				require.Equal(t, 1, resp.Usage[0].Used)

				raw, err := deps.r.Get("{rl}:key-hash")
				require.NoError(t, err)
				parsed, err := strconv.Atoi(raw)
				require.NoError(t, err)
				tat := time.Unix(0, int64(parsed))
				require.WithinDuration(t, deps.clock.Now().Add(6*time.Second), tat, time.Second)
			},
			afterAcquire: func(t *testing.T, deps *deps, resp *CapacityAcquireResponse) {
				t.Log(resp.Debug())

				require.True(t, resp.RetryAfter.Before(deps.clock.Now()))

				raw, err := deps.r.Get("{rl}:key-hash")
				require.NoError(t, err)
				parsed, err := strconv.Atoi(raw)
				require.NoError(t, err)
				tat := time.Unix(0, int64(parsed))
				require.WithinDuration(t, deps.clock.Now().Add(12*time.Second), tat, time.Second)
			},
			afterPostAcquireCheck: func(t *testing.T, deps *deps, resp *CapacityCheckResponse) {
				require.Equal(t, 2, resp.Usage[0].Used)
			},
		},

		{
			name: "rate limit partially rejected with legacy state",
			mi: MigrationIdentifier{
				IsRateLimit: true,
			},
			config: ConstraintConfig{
				FunctionVersion: 1,
				RateLimit: []RateLimitConfig{
					{
						Limit:             10,
						Period:            60,
						KeyExpressionHash: "expr-hash",
					},
				},
			},
			constraints: []ConstraintItem{
				{
					Kind: ConstraintKindRateLimit,
					RateLimit: &RateLimitConstraint{
						KeyExpressionHash: "expr-hash",
						EvaluatedKeyHash:  "key-hash",
					},
				},
			},
			amount:              2,
			expectedLeaseAmount: 1,
			beforeAcquire: func(t *testing.T, deps *deps) {
				// Set existing legacy state
				tat := deps.clock.Now().Add(6 * time.Second).UnixNano()
				err := deps.r.Set("{rl}:key-hash", strconv.Itoa(int(tat)))
				require.NoError(t, err)
			},
			afterPreAcquireCheck: func(t *testing.T, deps *deps, resp *CapacityCheckResponse) {
				t.Log(resp.Debug())

				require.Equal(t, 1, resp.Usage[0].Used)

				raw, err := deps.r.Get("{rl}:key-hash")
				require.NoError(t, err)
				parsed, err := strconv.Atoi(raw)
				require.NoError(t, err)
				tat := time.Unix(0, int64(parsed))
				require.WithinDuration(t, deps.clock.Now().Add(6*time.Second), tat, time.Second)
			},
			afterAcquire: func(t *testing.T, deps *deps, resp *CapacityAcquireResponse) {
				t.Log(resp.Debug())

				require.WithinDuration(t, deps.clock.Now().Add(6*time.Second), resp.RetryAfter, time.Second)

				raw, err := deps.r.Get("{rl}:key-hash")
				require.NoError(t, err)
				parsed, err := strconv.Atoi(raw)
				require.NoError(t, err)
				tat := time.Unix(0, int64(parsed))
				require.WithinDuration(t, deps.clock.Now().Add(12*time.Second), tat, time.Second)
			},
			afterPostAcquireCheck: func(t *testing.T, deps *deps, resp *CapacityCheckResponse) {
				// We are now accounting for 2 requests
				require.Equal(t, 2, resp.Usage[0].Used)
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			r := miniredis.RunT(t)
			ctx := context.Background()

			rc, err := rueidis.NewClient(rueidis.ClientOption{
				InitAddress:  []string{r.Addr()},
				DisableCache: true,
			})
			require.NoError(t, err)
			defer rc.Close()

			clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))
			l := logger.StdlibLogger(ctx, logger.WithLoggerLevel(logger.LevelTrace))
			ctx = logger.WithStdlib(ctx, l)

			cm, err := NewRedisCapacityManager(
				WithRateLimitClient(rc),
				WithQueueShards(map[string]rueidis.Client{
					"test": rc,
				}),
				WithClock(clock),
				WithNumScavengerShards(1),
				WithQueueStateKeyPrefix("q:v1"),
				WithRateLimitKeyPrefix("rl"),
				WithEnableDebugLogs(true),
				// Do not cache check requests
				WithCheckIdempotencyTTL(0),
			)
			require.NoError(t, err)
			require.NotNil(t, cm)

			deps := &deps{
				config:      test.config,
				constraints: test.constraints,
				cm:          cm,
				clock:       clock,
				r:           r,
				rc:          rc,
			}

			if test.beforeAcquire != nil {
				test.beforeAcquire(t, deps)
			}

			checkResp, _, err := cm.Check(ctx, &CapacityCheckRequest{
				Migration:     test.mi,
				AccountID:     accountID,
				Configuration: test.config,
				Constraints:   test.constraints,
				EnvID:         envID,
				FunctionID:    fnID,
			})
			require.NoError(t, err)

			if test.afterPreAcquireCheck != nil {
				test.afterPreAcquireCheck(t, deps, checkResp)
			}

			leaseIdempotencyKeys := make([]string, test.amount)
			for i := range test.amount {
				leaseIdempotencyKeys[i] = fmt.Sprintf("item%d", i)
			}

			acquireReq := &CapacityAcquireRequest{
				Migration:            test.mi,
				AccountID:            accountID,
				IdempotencyKey:       "acquire",
				Constraints:          test.constraints,
				Amount:               test.amount,
				EnvID:                envID,
				FunctionID:           fnID,
				Configuration:        test.config,
				LeaseIdempotencyKeys: leaseIdempotencyKeys,
				LeaseRunIDs:          make(map[string]ulid.ULID),
				CurrentTime:          clock.Now(),
				Duration:             5 * time.Second,
				MaximumLifetime:      time.Hour,
				Source: LeaseSource{
					Service:           ServiceExecutor,
					Location:          CallerLocationItemLease,
					RunProcessingMode: RunProcessingModeBackground,
				},
			}

			keyPrefix, _, err := cm.clientAndPrefix(test.mi)
			require.NoError(t, err)

			_, _, _, fingerprint, err := buildRequestState(acquireReq, keyPrefix)
			require.NoError(t, err)

			deps.acquireIdempotencyKey = fmt.Sprintf("acquire-%s", fingerprint)

			acquireResp, err := cm.Acquire(ctx, acquireReq)
			require.NoError(t, err)

			if acquireResp != nil {
				deps.acquireRequestID = acquireResp.RequestID
			}

			if test.afterAcquire != nil {
				test.afterAcquire(t, deps, acquireResp)
			}

			require.Len(t, acquireResp.Leases, test.expectedLeaseAmount)

			if test.expectedLeaseAmount == 0 {
				return
			}

			checkResp, _, err = cm.Check(ctx, &CapacityCheckRequest{
				Migration:     test.mi,
				AccountID:     accountID,
				Configuration: test.config,
				Constraints:   test.constraints,
				EnvID:         envID,
				FunctionID:    fnID,
			})
			require.NoError(t, err)

			if test.afterPostAcquireCheck != nil {
				test.afterPostAcquireCheck(t, deps, checkResp)
			}

			clock.Advance(2 * time.Second)
			r.FastForward(2 * time.Second)
			r.SetTime(clock.Now())

			for _, lease := range acquireResp.Leases {
				extendResp, err := cm.ExtendLease(ctx, &CapacityExtendLeaseRequest{
					IdempotencyKey: "extend",
					LeaseID:        lease.LeaseID,
					AccountID:      accountID,
					Duration:       5 * time.Second,
					Migration:      test.mi,
				})
				require.NoError(t, err)

				if test.afterExtend != nil {
					test.afterExtend(t, deps, extendResp)
				}

				releaseResp, err := cm.Release(ctx, &CapacityReleaseRequest{
					AccountID:      accountID,
					IdempotencyKey: "release",
					Migration:      test.mi,
					LeaseID:        *extendResp.LeaseID,
				})
				require.NoError(t, err)

				if test.afterRelease != nil {
					test.afterRelease(t, deps, releaseResp)
				}
			}
		})
	}
}

func TestConcurrencyConstraint_InProgressLeasesKey(t *testing.T) {
	// Test UUIDs
	accountID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440001")
	envID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440002")
	functionID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440003")
	prefix := "test-prefix"

	tests := []struct {
		name        string
		constraint  ConcurrencyConstraint
		prefix      string
		accountID   uuid.UUID
		envID       uuid.UUID
		functionID  uuid.UUID
		expected    string
		description string
	}{
		// Basic Scope Testing
		{
			name: "account scope",
			constraint: ConcurrencyConstraint{
				Mode:  enums.ConcurrencyModeStep,
				Scope: enums.ConcurrencyScopeAccount,
			},
			prefix:      prefix,
			accountID:   accountID,
			envID:       envID,
			functionID:  functionID,
			expected:    "{test-prefix}:550e8400-e29b-41d4-a716-446655440001:state:concurrency:a:550e8400-e29b-41d4-a716-446655440001",
			description: "should use account scope ID 'a' and accountID as entityID",
		},
		{
			name: "environment scope",
			constraint: ConcurrencyConstraint{
				Mode:  enums.ConcurrencyModeStep,
				Scope: enums.ConcurrencyScopeEnv,
			},
			prefix:      prefix,
			accountID:   accountID,
			envID:       envID,
			functionID:  functionID,
			expected:    "{test-prefix}:550e8400-e29b-41d4-a716-446655440001:state:concurrency:e:550e8400-e29b-41d4-a716-446655440002",
			description: "should use environment scope ID 'e' and envID as entityID",
		},
		{
			name: "function scope",
			constraint: ConcurrencyConstraint{
				Mode:  enums.ConcurrencyModeStep,
				Scope: enums.ConcurrencyScopeFn,
			},
			prefix:      prefix,
			accountID:   accountID,
			envID:       envID,
			functionID:  functionID,
			expected:    "{test-prefix}:550e8400-e29b-41d4-a716-446655440001:state:concurrency:f:550e8400-e29b-41d4-a716-446655440003",
			description: "should use function scope ID 'f' and functionID as entityID",
		},

		// Mode Testing (should not affect key generation)
		{
			name: "step mode with function scope",
			constraint: ConcurrencyConstraint{
				Mode:  enums.ConcurrencyModeStep,
				Scope: enums.ConcurrencyScopeFn,
			},
			prefix:      prefix,
			accountID:   accountID,
			envID:       envID,
			functionID:  functionID,
			expected:    "{test-prefix}:550e8400-e29b-41d4-a716-446655440001:state:concurrency:f:550e8400-e29b-41d4-a716-446655440003",
			description: "step mode should generate same key format as other modes",
		},
		{
			name: "run mode with function scope",
			constraint: ConcurrencyConstraint{
				Mode:  enums.ConcurrencyModeRun,
				Scope: enums.ConcurrencyScopeFn,
			},
			prefix:      prefix,
			accountID:   accountID,
			envID:       envID,
			functionID:  functionID,
			expected:    "{test-prefix}:550e8400-e29b-41d4-a716-446655440001:state:concurrency:f:550e8400-e29b-41d4-a716-446655440003",
			description: "run mode should generate same key format as other modes",
		},

		// Key Expression Hash Testing
		{
			name: "no custom key hash",
			constraint: ConcurrencyConstraint{
				Mode:              enums.ConcurrencyModeStep,
				Scope:             enums.ConcurrencyScopeFn,
				KeyExpressionHash: "",
				EvaluatedKeyHash:  "",
			},
			prefix:      prefix,
			accountID:   accountID,
			envID:       envID,
			functionID:  functionID,
			expected:    "{test-prefix}:550e8400-e29b-41d4-a716-446655440001:state:concurrency:f:550e8400-e29b-41d4-a716-446655440003",
			description: "empty KeyExpressionHash should not append keyID suffix",
		},
		{
			name: "with custom key hash",
			constraint: ConcurrencyConstraint{
				Mode:              enums.ConcurrencyModeStep,
				Scope:             enums.ConcurrencyScopeFn,
				KeyExpressionHash: "expr_hash_123",
				EvaluatedKeyHash:  "eval_hash_456",
			},
			prefix:      prefix,
			accountID:   accountID,
			envID:       envID,
			functionID:  functionID,
			expected:    "{test-prefix}:550e8400-e29b-41d4-a716-446655440001:state:concurrency:f:550e8400-e29b-41d4-a716-446655440003<expr_hash_123:eval_hash_456>",
			description: "non-empty KeyExpressionHash should append keyID suffix with format <hash:evaluated>",
		},
		{
			name: "expression hash without evaluated hash",
			constraint: ConcurrencyConstraint{
				Mode:              enums.ConcurrencyModeStep,
				Scope:             enums.ConcurrencyScopeFn,
				KeyExpressionHash: "expr_hash_789",
				EvaluatedKeyHash:  "",
			},
			prefix:      prefix,
			accountID:   accountID,
			envID:       envID,
			functionID:  functionID,
			expected:    "{test-prefix}:550e8400-e29b-41d4-a716-446655440001:state:concurrency:f:550e8400-e29b-41d4-a716-446655440003<expr_hash_789:>",
			description: "KeyExpressionHash with empty EvaluatedKeyHash should still include format",
		},

		// Parameter Validation Testing
		{
			name: "empty prefix",
			constraint: ConcurrencyConstraint{
				Mode:  enums.ConcurrencyModeStep,
				Scope: enums.ConcurrencyScopeFn,
			},
			prefix:      "",
			accountID:   accountID,
			envID:       envID,
			functionID:  functionID,
			expected:    "{}:550e8400-e29b-41d4-a716-446655440001:state:concurrency:f:550e8400-e29b-41d4-a716-446655440003",
			description: "empty prefix should still generate valid key format",
		},
		{
			name: "different prefix",
			constraint: ConcurrencyConstraint{
				Mode:  enums.ConcurrencyModeStep,
				Scope: enums.ConcurrencyScopeFn,
			},
			prefix:      "production",
			accountID:   accountID,
			envID:       envID,
			functionID:  functionID,
			expected:    "{production}:550e8400-e29b-41d4-a716-446655440001:state:concurrency:f:550e8400-e29b-41d4-a716-446655440003",
			description: "different prefix should be reflected in generated key",
		},
		{
			name: "zero UUIDs",
			constraint: ConcurrencyConstraint{
				Mode:  enums.ConcurrencyModeStep,
				Scope: enums.ConcurrencyScopeFn,
			},
			prefix:      prefix,
			accountID:   uuid.Nil,
			envID:       uuid.Nil,
			functionID:  uuid.Nil,
			expected:    "{test-prefix}:00000000-0000-0000-0000-000000000000:state:concurrency:f:00000000-0000-0000-0000-000000000000",
			description: "nil UUIDs should be formatted as zero UUIDs",
		},

		// Integration Testing - Complex Combinations
		{
			name: "account scope with custom key and run mode",
			constraint: ConcurrencyConstraint{
				Mode:              enums.ConcurrencyModeRun,
				Scope:             enums.ConcurrencyScopeAccount,
				KeyExpressionHash: "account_key",
				EvaluatedKeyHash:  "account_eval",
			},
			prefix:      "prod-redis",
			accountID:   accountID,
			envID:       envID,
			functionID:  functionID,
			expected:    "{prod-redis}:550e8400-e29b-41d4-a716-446655440001:state:concurrency:a:550e8400-e29b-41d4-a716-446655440001<account_key:account_eval>",
			description: "complex combination should work correctly with all parameters",
		},
		{
			name: "environment scope with custom key",
			constraint: ConcurrencyConstraint{
				Mode:              enums.ConcurrencyModeStep,
				Scope:             enums.ConcurrencyScopeEnv,
				KeyExpressionHash: "env_custom",
				EvaluatedKeyHash:  "env_value",
			},
			prefix:      "staging",
			accountID:   accountID,
			envID:       envID,
			functionID:  functionID,
			expected:    "{staging}:550e8400-e29b-41d4-a716-446655440001:state:concurrency:e:550e8400-e29b-41d4-a716-446655440002<env_custom:env_value>",
			description: "environment scope with custom keys should generate correct format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.constraint.InProgressLeasesKey(tt.prefix, tt.accountID, tt.envID, tt.functionID)
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

func TestConcurrencyConstraint_InProgressLeasesKey_KeyFormat(t *testing.T) {
	// Additional tests to verify key format consistency
	constraint := ConcurrencyConstraint{
		Mode:  enums.ConcurrencyModeStep,
		Scope: enums.ConcurrencyScopeFn,
	}

	accountID := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	envID := uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	functionID := uuid.MustParse("ffffffff-1111-2222-3333-444444444444")

	t.Run("key format validation", func(t *testing.T) {
		key := constraint.InProgressLeasesKey("prefix", accountID, envID, functionID)

		// Verify the key follows expected pattern: prefix:accountID:state:concurrency:scopeID:entityID[keyID]
		assert.Contains(t, key, "{prefix}:")
		assert.Contains(t, key, ":11111111-2222-3333-4444-555555555555:")
		assert.Contains(t, key, ":state:concurrency:")
		assert.Contains(t, key, ":f:")
		assert.Contains(t, key, ":ffffffff-1111-2222-3333-444444444444")
	})

	t.Run("key uniqueness", func(t *testing.T) {
		// Different scopes should produce different keys
		constraintAccount := ConcurrencyConstraint{Mode: enums.ConcurrencyModeStep, Scope: enums.ConcurrencyScopeAccount}
		constraintEnv := ConcurrencyConstraint{Mode: enums.ConcurrencyModeStep, Scope: enums.ConcurrencyScopeEnv}
		constraintFn := ConcurrencyConstraint{Mode: enums.ConcurrencyModeStep, Scope: enums.ConcurrencyScopeFn}

		keyAccount := constraintAccount.InProgressLeasesKey("test", accountID, envID, functionID)
		keyEnv := constraintEnv.InProgressLeasesKey("test", accountID, envID, functionID)
		keyFn := constraintFn.InProgressLeasesKey("test", accountID, envID, functionID)

		assert.NotEqual(t, keyAccount, keyEnv, "account and environment scoped keys should be different")
		assert.NotEqual(t, keyEnv, keyFn, "environment and function scoped keys should be different")
		assert.NotEqual(t, keyAccount, keyFn, "account and function scoped keys should be different")
	})
}

func TestConstraintItem_IsFunctionLevelConstraint(t *testing.T) {
	tests := []struct {
		name        string
		constraint  ConstraintItem
		expected    bool
		description string
	}{
		// Rate Limit Constraints
		{
			name: "rate limit function scope",
			constraint: ConstraintItem{
				Kind: ConstraintKindRateLimit,
				RateLimit: &RateLimitConstraint{
					Scope: enums.RateLimitScopeFn,
				},
			},
			expected:    true,
			description: "rate limit with function scope should be function-level",
		},
		{
			name: "rate limit account scope",
			constraint: ConstraintItem{
				Kind: ConstraintKindRateLimit,
				RateLimit: &RateLimitConstraint{
					Scope: enums.RateLimitScopeAccount,
				},
			},
			expected:    false,
			description: "rate limit with account scope should not be function-level",
		},
		{
			name: "rate limit environment scope",
			constraint: ConstraintItem{
				Kind: ConstraintKindRateLimit,
				RateLimit: &RateLimitConstraint{
					Scope: enums.RateLimitScopeEnv,
				},
			},
			expected:    false,
			description: "rate limit with environment scope should not be function-level",
		},
		{
			name: "rate limit constraint with nil pointer",
			constraint: ConstraintItem{
				Kind:      ConstraintKindRateLimit,
				RateLimit: nil,
			},
			expected:    false,
			description: "rate limit with nil pointer should not be function-level",
		},

		// Throttle Constraints
		{
			name: "throttle function scope",
			constraint: ConstraintItem{
				Kind: ConstraintKindThrottle,
				Throttle: &ThrottleConstraint{
					Scope: enums.ThrottleScopeFn,
				},
			},
			expected:    true,
			description: "throttle with function scope should be function-level",
		},
		{
			name: "throttle account scope",
			constraint: ConstraintItem{
				Kind: ConstraintKindThrottle,
				Throttle: &ThrottleConstraint{
					Scope: enums.ThrottleScopeAccount,
				},
			},
			expected:    false,
			description: "throttle with account scope should not be function-level",
		},
		{
			name: "throttle environment scope",
			constraint: ConstraintItem{
				Kind: ConstraintKindThrottle,
				Throttle: &ThrottleConstraint{
					Scope: enums.ThrottleScopeEnv,
				},
			},
			expected:    false,
			description: "throttle with environment scope should not be function-level",
		},
		{
			name: "throttle constraint with nil pointer",
			constraint: ConstraintItem{
				Kind:     ConstraintKindThrottle,
				Throttle: nil,
			},
			expected:    false,
			description: "throttle with nil pointer should not be function-level",
		},

		// Concurrency Constraints
		{
			name: "concurrency function scope",
			constraint: ConstraintItem{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Scope: enums.ConcurrencyScopeFn,
				},
			},
			expected:    true,
			description: "concurrency with function scope should be function-level",
		},
		{
			name: "concurrency account scope",
			constraint: ConstraintItem{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Scope: enums.ConcurrencyScopeAccount,
				},
			},
			expected:    false,
			description: "concurrency with account scope should not be function-level",
		},
		{
			name: "concurrency environment scope",
			constraint: ConstraintItem{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Scope: enums.ConcurrencyScopeEnv,
				},
			},
			expected:    false,
			description: "concurrency with environment scope should not be function-level",
		},
		{
			name: "concurrency constraint with nil pointer",
			constraint: ConstraintItem{
				Kind:        ConstraintKindConcurrency,
				Concurrency: nil,
			},
			expected:    false,
			description: "concurrency with nil pointer should not be function-level",
		},

		// Unknown/Default Cases
		{
			name: "unknown constraint kind",
			constraint: ConstraintItem{
				Kind: ConstraintKind("unknown"),
			},
			expected:    false,
			description: "unknown constraint kind should not be function-level",
		},
		{
			name: "empty constraint kind",
			constraint: ConstraintItem{
				Kind: ConstraintKind(""),
			},
			expected:    false,
			description: "empty constraint kind should not be function-level",
		},

		// Edge Cases with Multiple Constraint Objects
		{
			name: "multiple constraint objects with function-level rate limit",
			constraint: ConstraintItem{
				Kind: ConstraintKindRateLimit,
				RateLimit: &RateLimitConstraint{
					Scope: enums.RateLimitScopeFn,
				},
				Throttle: &ThrottleConstraint{
					Scope: enums.ThrottleScopeAccount,
				},
				Concurrency: &ConcurrencyConstraint{
					Scope: enums.ConcurrencyScopeEnv,
				},
			},
			expected:    true,
			description: "should only consider the constraint type matching the Kind field",
		},
		{
			name: "multiple constraint objects with account-level throttle",
			constraint: ConstraintItem{
				Kind: ConstraintKindThrottle,
				RateLimit: &RateLimitConstraint{
					Scope: enums.RateLimitScopeFn,
				},
				Throttle: &ThrottleConstraint{
					Scope: enums.ThrottleScopeAccount,
				},
				Concurrency: &ConcurrencyConstraint{
					Scope: enums.ConcurrencyScopeFn,
				},
			},
			expected:    false,
			description: "should only consider the constraint type matching the Kind field",
		},

		// Zero Value Testing
		{
			name:        "completely zero constraint item",
			constraint:  ConstraintItem{},
			expected:    false,
			description: "zero-value constraint item should not be function-level",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.constraint.IsFunctionLevelConstraint()
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

func TestRateLimitConstraint_StateKey(t *testing.T) {
	accountID := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	envID := uuid.MustParse("66666666-7777-8888-9999-aaaaaaaaaaaa")
	keyPrefix := "test"
	evaluatedKeyHash := "abcd1234hash"

	tests := []struct {
		name        string
		constraint  *RateLimitConstraint
		expectedKey string
		description string
	}{
		{
			name: "account scope rate limit",
			constraint: &RateLimitConstraint{
				Scope:            enums.RateLimitScopeAccount,
				EvaluatedKeyHash: evaluatedKeyHash,
			},
			expectedKey: "{test}:rl:a:11111111-2222-3333-4444-555555555555:abcd1234hash",
			description: "account scope should generate account-specific key",
		},
		{
			name: "environment scope rate limit",
			constraint: &RateLimitConstraint{
				Scope:            enums.RateLimitScopeEnv,
				EvaluatedKeyHash: evaluatedKeyHash,
			},
			expectedKey: "{test}:rl:e:66666666-7777-8888-9999-aaaaaaaaaaaa:abcd1234hash",
			description: "environment scope should generate environment-specific key",
		},
		{
			name: "function scope rate limit",
			constraint: &RateLimitConstraint{
				Scope:            enums.RateLimitScopeFn,
				EvaluatedKeyHash: evaluatedKeyHash,
			},
			expectedKey: "{test}:abcd1234hash",
			description: "function scope should generate compatibility key with rate limit prefix",
		},
		{
			name: "function scope with different key hash",
			constraint: &RateLimitConstraint{
				Scope:            enums.RateLimitScopeFn,
				EvaluatedKeyHash: "xyz789different",
			},
			expectedKey: "{test}:xyz789different",
			description: "function scope key should vary with different evaluated key hash",
		},
		{
			name: "account scope with empty key hash",
			constraint: &RateLimitConstraint{
				Scope:            enums.RateLimitScopeAccount,
				EvaluatedKeyHash: "",
			},
			expectedKey: "{test}:rl:a:11111111-2222-3333-4444-555555555555:",
			description: "empty key hash should still generate valid key structure",
		},
		{
			name: "environment scope with empty key hash",
			constraint: &RateLimitConstraint{
				Scope:            enums.RateLimitScopeEnv,
				EvaluatedKeyHash: "",
			},
			expectedKey: "{test}:rl:e:66666666-7777-8888-9999-aaaaaaaaaaaa:",
			description: "empty key hash should still generate valid key structure",
		},
		{
			name: "function scope with empty key hash",
			constraint: &RateLimitConstraint{
				Scope:            enums.RateLimitScopeFn,
				EvaluatedKeyHash: "",
			},
			expectedKey: "{test}:",
			description: "function scope with empty hash should generate minimal key",
		},
		{
			name: "account scope with key expression hash",
			constraint: &RateLimitConstraint{
				Scope:             enums.RateLimitScopeAccount,
				KeyExpressionHash: "expr123",
				EvaluatedKeyHash:  evaluatedKeyHash,
			},
			expectedKey: "{test}:rl:a:11111111-2222-3333-4444-555555555555:abcd1234hash",
			description: "key expression hash should not affect state key generation",
		},
		{
			name: "invalid scope should default to function behavior",
			constraint: &RateLimitConstraint{
				Scope:            enums.RateLimitScope(999), // invalid scope value
				EvaluatedKeyHash: evaluatedKeyHash,
			},
			expectedKey: "{test}:abcd1234hash",
			description: "invalid scope should default to function scope behavior",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualKey := tt.constraint.StateKey(keyPrefix, accountID, envID)
			assert.Equal(t, tt.expectedKey, actualKey, tt.description)
		})
	}
}

// Test key uniqueness across different parameters
func TestRateLimitConstraint_StateKey_Uniqueness(t *testing.T) {
	accountID1 := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	accountID2 := uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	envID1 := uuid.MustParse("66666666-7777-8888-9999-aaaaaaaaaaaa")
	envID2 := uuid.MustParse("ffffffff-0000-1111-2222-333333333333")
	keyPrefix := "test"
	evaluatedKeyHash := "samehash"

	constraint := &RateLimitConstraint{
		Scope:            enums.RateLimitScopeAccount,
		EvaluatedKeyHash: evaluatedKeyHash,
	}

	// Test that different account IDs produce different keys
	key1 := constraint.StateKey(keyPrefix, accountID1, envID1)
	key2 := constraint.StateKey(keyPrefix, accountID2, envID1)
	assert.NotEqual(t, key1, key2, "Different account IDs should produce different keys")

	// Test that different environment IDs produce different keys for env scope
	envConstraint := &RateLimitConstraint{
		Scope:            enums.RateLimitScopeEnv,
		EvaluatedKeyHash: evaluatedKeyHash,
	}
	key3 := envConstraint.StateKey(keyPrefix, accountID1, envID1)
	key4 := envConstraint.StateKey(keyPrefix, accountID1, envID2)
	assert.NotEqual(t, key3, key4, "Different environment IDs should produce different keys for env scope")

	// Test that different key prefixes produce different keys
	key5 := constraint.StateKey(keyPrefix, accountID1, envID1)
	key6 := constraint.StateKey("prefix2", accountID1, envID1)
	assert.NotEqual(t, key5, key6, "Different key prefixes should produce different keys")

	// Test that different evaluated key hashes produce different keys
	constraint2 := &RateLimitConstraint{
		Scope:            enums.RateLimitScopeAccount,
		EvaluatedKeyHash: "differenthash",
	}
	key7 := constraint.StateKey(keyPrefix, accountID1, envID1)
	key8 := constraint2.StateKey(keyPrefix, accountID1, envID1)
	assert.NotEqual(t, key7, key8, "Different evaluated key hashes should produce different keys")
}

func TestThrottleConstraint_StateKey(t *testing.T) {
	accountID := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	envID := uuid.MustParse("66666666-7777-8888-9999-aaaaaaaaaaaa")
	keyPrefix := "test"
	evaluatedKeyHash := "xyz456hash"

	tests := []struct {
		name        string
		constraint  *ThrottleConstraint
		expectedKey string
		description string
	}{
		{
			name: "account scope throttle",
			constraint: &ThrottleConstraint{
				Scope:            enums.ThrottleScopeAccount,
				EvaluatedKeyHash: evaluatedKeyHash,
			},
			expectedKey: "{test}:throttle:a:11111111-2222-3333-4444-555555555555:xyz456hash",
			description: "account scope should generate account-specific throttle key",
		},
		{
			name: "environment scope throttle",
			constraint: &ThrottleConstraint{
				Scope:            enums.ThrottleScopeEnv,
				EvaluatedKeyHash: evaluatedKeyHash,
			},
			expectedKey: "{test}:throttle:e:66666666-7777-8888-9999-aaaaaaaaaaaa:xyz456hash",
			description: "environment scope should generate environment-specific throttle key",
		},
		{
			name: "function scope throttle",
			constraint: &ThrottleConstraint{
				Scope:            enums.ThrottleScopeFn,
				EvaluatedKeyHash: evaluatedKeyHash,
			},
			expectedKey: "{test}:throttle:xyz456hash",
			description: "function scope should generate compatibility throttle key",
		},
		{
			name: "function scope with different key hash",
			constraint: &ThrottleConstraint{
				Scope:            enums.ThrottleScopeFn,
				EvaluatedKeyHash: "different123",
			},
			expectedKey: "{test}:throttle:different123",
			description: "function scope key should vary with different evaluated key hash",
		},
		{
			name: "account scope with empty key hash",
			constraint: &ThrottleConstraint{
				Scope:            enums.ThrottleScopeAccount,
				EvaluatedKeyHash: "",
			},
			expectedKey: "{test}:throttle:a:11111111-2222-3333-4444-555555555555:",
			description: "empty key hash should still generate valid throttle key structure",
		},
		{
			name: "environment scope with empty key hash",
			constraint: &ThrottleConstraint{
				Scope:            enums.ThrottleScopeEnv,
				EvaluatedKeyHash: "",
			},
			expectedKey: "{test}:throttle:e:66666666-7777-8888-9999-aaaaaaaaaaaa:",
			description: "empty key hash should still generate valid throttle key structure",
		},
		{
			name: "function scope with empty key hash",
			constraint: &ThrottleConstraint{
				Scope:            enums.ThrottleScopeFn,
				EvaluatedKeyHash: "",
			},
			expectedKey: "{test}:throttle:",
			description: "function scope with empty hash should generate minimal throttle key",
		},
		{
			name: "account scope with key expression hash",
			constraint: &ThrottleConstraint{
				Scope:             enums.ThrottleScopeAccount,
				KeyExpressionHash: "expr456",
				EvaluatedKeyHash:  evaluatedKeyHash,
			},
			expectedKey: "{test}:throttle:a:11111111-2222-3333-4444-555555555555:xyz456hash",
			description: "key expression hash should not affect throttle state key generation",
		},
		{
			name: "invalid scope should default to function behavior",
			constraint: &ThrottleConstraint{
				Scope:            enums.ThrottleScope(999), // invalid scope value
				EvaluatedKeyHash: evaluatedKeyHash,
			},
			expectedKey: "{test}:throttle:xyz456hash",
			description: "invalid scope should default to function scope throttle behavior",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualKey := tt.constraint.StateKey(keyPrefix, accountID, envID)
			assert.Equal(t, tt.expectedKey, actualKey, tt.description)
		})
	}
}

// Test throttle key uniqueness across different parameters
func TestThrottleConstraint_StateKey_Uniqueness(t *testing.T) {
	accountID1 := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	accountID2 := uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	envID1 := uuid.MustParse("66666666-7777-8888-9999-aaaaaaaaaaaa")
	envID2 := uuid.MustParse("ffffffff-0000-1111-2222-333333333333")
	keyPrefix := "test"
	evaluatedKeyHash := "samehash"

	constraint := &ThrottleConstraint{
		Scope:            enums.ThrottleScopeAccount,
		EvaluatedKeyHash: evaluatedKeyHash,
	}

	// Test that different account IDs produce different keys
	key1 := constraint.StateKey(keyPrefix, accountID1, envID1)
	key2 := constraint.StateKey(keyPrefix, accountID2, envID1)
	assert.NotEqual(t, key1, key2, "Different account IDs should produce different throttle keys")

	// Test that different environment IDs produce different keys for env scope
	envConstraint := &ThrottleConstraint{
		Scope:            enums.ThrottleScopeEnv,
		EvaluatedKeyHash: evaluatedKeyHash,
	}
	key3 := envConstraint.StateKey(keyPrefix, accountID1, envID1)
	key4 := envConstraint.StateKey(keyPrefix, accountID1, envID2)
	assert.NotEqual(t, key3, key4, "Different environment IDs should produce different throttle keys for env scope")

	// Test that different key prefixes produce different keys
	key5 := constraint.StateKey(keyPrefix, accountID1, envID1)
	key6 := constraint.StateKey("otherprefix", accountID1, envID1)
	assert.NotEqual(t, key5, key6, "Different key prefixes should produce different throttle keys")

	// Test that different evaluated key hashes produce different keys
	constraint2 := &ThrottleConstraint{
		Scope:            enums.ThrottleScopeAccount,
		EvaluatedKeyHash: "anotherhash",
	}
	key7 := constraint.StateKey(keyPrefix, accountID1, envID1)
	key8 := constraint2.StateKey(keyPrefix, accountID1, envID1)
	assert.NotEqual(t, key7, key8, "Different evaluated key hashes should produce different throttle keys")

	// Test that throttle and rate limit keys are different for same parameters
	rateLimitConstraint := &RateLimitConstraint{
		Scope:            enums.RateLimitScopeAccount,
		EvaluatedKeyHash: evaluatedKeyHash,
	}
	throttleKey := constraint.StateKey(keyPrefix, accountID1, envID1)
	rateLimitKey := rateLimitConstraint.StateKey(keyPrefix, accountID1, envID1)
	assert.NotEqual(t, throttleKey, rateLimitKey, "Throttle and rate limit keys should be different for same parameters")
}

func TestConstraintKind_IsQueueConstraint(t *testing.T) {
	tests := []struct {
		name        string
		kind        ConstraintKind
		expected    bool
		description string
	}{
		{
			name:        "concurrency constraint is queue constraint",
			kind:        ConstraintKindConcurrency,
			expected:    true,
			description: "concurrency constraints should be considered queue constraints",
		},
		{
			name:        "throttle constraint is queue constraint",
			kind:        ConstraintKindThrottle,
			expected:    true,
			description: "throttle constraints should be considered queue constraints",
		},
		{
			name:        "rate limit constraint is not queue constraint",
			kind:        ConstraintKindRateLimit,
			expected:    false,
			description: "rate limit constraints should not be considered queue constraints",
		},
		{
			name:        "unknown constraint kind is not queue constraint",
			kind:        ConstraintKind("unknown"),
			expected:    false,
			description: "unknown constraint kinds should not be considered queue constraints",
		},
		{
			name:        "empty constraint kind is not queue constraint",
			kind:        ConstraintKind(""),
			expected:    false,
			description: "empty constraint kinds should not be considered queue constraints",
		},
		{
			name:        "invalid constraint kind is not queue constraint",
			kind:        ConstraintKind("invalid_type"),
			expected:    false,
			description: "invalid constraint kinds should not be considered queue constraints",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.kind.IsQueueConstraint()
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

// Test that IsQueueConstraint correctly identifies constraint mixability
func TestConstraintKind_IsQueueConstraint_ConstraintMixing(t *testing.T) {
	// Verify that queue constraints cannot be mixed with rate limit constraints
	// This aligns with validation logic that prevents mixing these constraint types

	queueConstraints := []ConstraintKind{
		ConstraintKindConcurrency,
		ConstraintKindThrottle,
	}

	nonQueueConstraints := []ConstraintKind{
		ConstraintKindRateLimit,
	}

	// All queue constraints should return true
	for _, kind := range queueConstraints {
		assert.True(t, kind.IsQueueConstraint(), "Queue constraint %s should return true for IsQueueConstraint()", kind)
	}

	// All non-queue constraints should return false
	for _, kind := range nonQueueConstraints {
		assert.False(t, kind.IsQueueConstraint(), "Non-queue constraint %s should return false for IsQueueConstraint()", kind)
	}

	// Verify distinct categorization
	for _, queueKind := range queueConstraints {
		for _, nonQueueKind := range nonQueueConstraints {
			assert.NotEqual(t, queueKind.IsQueueConstraint(), nonQueueKind.IsQueueConstraint(),
				"Queue constraint %s and non-queue constraint %s should have different IsQueueConstraint results",
				queueKind, nonQueueKind)
		}
	}
}

func TestConcurrencyConstraint_IsCustomKey(t *testing.T) {
	tests := []struct {
		name        string
		constraint  ConcurrencyConstraint
		expected    bool
		description string
	}{
		{
			name: "constraint with key expression hash is custom key",
			constraint: ConcurrencyConstraint{
				KeyExpressionHash: "hash123",
				EvaluatedKeyHash:  "eval456",
			},
			expected:    true,
			description: "constraints with non-empty KeyExpressionHash should be custom keys",
		},
		{
			name: "constraint with empty key expression hash is not custom key",
			constraint: ConcurrencyConstraint{
				KeyExpressionHash: "",
				EvaluatedKeyHash:  "eval456",
			},
			expected:    false,
			description: "constraints with empty KeyExpressionHash should not be custom keys",
		},
		{
			name: "constraint with only evaluated key hash is not custom key",
			constraint: ConcurrencyConstraint{
				EvaluatedKeyHash: "eval789",
			},
			expected:    false,
			description: "constraints with only EvaluatedKeyHash should not be custom keys",
		},
		{
			name: "constraint with whitespace key expression hash is custom key",
			constraint: ConcurrencyConstraint{
				KeyExpressionHash: "   ",
			},
			expected:    true,
			description: "constraints with whitespace KeyExpressionHash should still be considered custom keys",
		},
		{
			name:        "zero value constraint is not custom key",
			constraint:  ConcurrencyConstraint{},
			expected:    false,
			description: "zero-value constraints should not be custom keys",
		},
		{
			name: "constraint with all fields set is custom key",
			constraint: ConcurrencyConstraint{
				Mode:              enums.ConcurrencyModeStep,
				Scope:             enums.ConcurrencyScopeFn,
				KeyExpressionHash: "expr_hash",
				EvaluatedKeyHash:  "eval_hash",
				InProgressItemKey: "progress_key",
			},
			expected:    true,
			description: "constraints with all fields set including KeyExpressionHash should be custom keys",
		},
		{
			name: "constraint without key expression hash but with other fields is not custom key",
			constraint: ConcurrencyConstraint{
				Mode:              enums.ConcurrencyModeStep,
				Scope:             enums.ConcurrencyScopeEnv,
				EvaluatedKeyHash:  "eval_hash",
				InProgressItemKey: "progress_key",
			},
			expected:    false,
			description: "constraints with other fields but no KeyExpressionHash should not be custom keys",
		},
		{
			name: "constraint with special characters in key expression hash is custom key",
			constraint: ConcurrencyConstraint{
				KeyExpressionHash: "key-with_special.chars123",
			},
			expected:    true,
			description: "constraints with special characters in KeyExpressionHash should be custom keys",
		},
		{
			name: "constraint with numeric key expression hash is custom key",
			constraint: ConcurrencyConstraint{
				KeyExpressionHash: "12345",
			},
			expected:    true,
			description: "constraints with numeric KeyExpressionHash should be custom keys",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.constraint.IsCustomKey()
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

// Test IsCustomKey impact on key generation
func TestConcurrencyConstraint_IsCustomKey_KeyGeneration(t *testing.T) {
	accountID := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	envID := uuid.MustParse("66666666-7777-8888-9999-aaaaaaaaaaaa")
	functionID := uuid.MustParse("77777777-8888-9999-aaaa-bbbbbbbbbbbb")
	prefix := "test"

	// Test that custom key affects InProgressLeasesKey generation
	standardConstraint := ConcurrencyConstraint{
		Scope:             enums.ConcurrencyScopeFn,
		Mode:              enums.ConcurrencyModeStep,
		InProgressItemKey: "progress_key",
	}

	customConstraint := ConcurrencyConstraint{
		Scope:             enums.ConcurrencyScopeFn,
		Mode:              enums.ConcurrencyModeStep,
		KeyExpressionHash: "custom_expr",
		EvaluatedKeyHash:  "custom_eval",
		InProgressItemKey: "progress_key",
	}

	// Verify IsCustomKey returns correct values
	assert.False(t, standardConstraint.IsCustomKey(), "Standard constraint should not be custom key")
	assert.True(t, customConstraint.IsCustomKey(), "Custom constraint should be custom key")

	// Verify that custom key affects the generated keys
	standardKey := standardConstraint.InProgressLeasesKey(prefix, accountID, envID, functionID)
	customKey := customConstraint.InProgressLeasesKey(prefix, accountID, envID, functionID)

	assert.NotEqual(t, standardKey, customKey, "Standard and custom constraints should generate different keys")

	// Custom key should include the custom key information
	assert.Contains(t, customKey, "custom_expr", "Custom key should include key expression hash")
	assert.Contains(t, customKey, "custom_eval", "Custom key should include evaluated key hash")

	// Standard key should not contain custom key patterns
	assert.NotContains(t, standardKey, "<", "Standard key should not contain custom key markers")
	assert.NotContains(t, standardKey, ">", "Standard key should not contain custom key markers")
}

func TestConstraintItem_CacheKey(t *testing.T) {
	// Test UUIDs
	accountID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440001")
	envID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440002")
	functionID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440003")

	tests := []struct {
		name        string
		constraint  ConstraintItem
		expected    string
		description string
	}{
		// Concurrency Constraints - No Custom Key
		{
			name: "concurrency account scope no custom key",
			constraint: ConstraintItem{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Scope: enums.ConcurrencyScopeAccount,
				},
			},
			expected:    "550e8400-e29b-41d4-a716-446655440001:c:a",
			description: "account concurrency without custom key should use account ID",
		},
		{
			name: "concurrency env scope no custom key",
			constraint: ConstraintItem{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Scope: enums.ConcurrencyScopeEnv,
				},
			},
			expected:    "550e8400-e29b-41d4-a716-446655440001:c:e:550e8400-e29b-41d4-a716-446655440002",
			description: "env concurrency without custom key should include env ID",
		},
		{
			name: "concurrency function scope no custom key",
			constraint: ConstraintItem{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Scope: enums.ConcurrencyScopeFn,
				},
			},
			expected:    "550e8400-e29b-41d4-a716-446655440001:c:f:550e8400-e29b-41d4-a716-446655440003",
			description: "function concurrency without custom key should include function ID",
		},

		// Concurrency Constraints - With Custom Key
		{
			name: "concurrency account scope with custom key",
			constraint: ConstraintItem{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Scope:             enums.ConcurrencyScopeAccount,
					KeyExpressionHash: "expr_hash_123",
					EvaluatedKeyHash:  "eval_hash_456",
				},
			},
			expected:    "550e8400-e29b-41d4-a716-446655440001:c:a:expr_hash_123:eval_hash_456",
			description: "account concurrency with custom key should include both hashes",
		},
		{
			name: "concurrency env scope with custom key",
			constraint: ConstraintItem{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Scope:             enums.ConcurrencyScopeEnv,
					KeyExpressionHash: "expr_env",
					EvaluatedKeyHash:  "eval_env",
				},
			},
			expected:    "550e8400-e29b-41d4-a716-446655440001:c:e:expr_env:eval_env",
			description: "env concurrency with custom key should include both hashes",
		},
		{
			name: "concurrency function scope with custom key",
			constraint: ConstraintItem{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Scope:             enums.ConcurrencyScopeFn,
					KeyExpressionHash: "expr_fn",
					EvaluatedKeyHash:  "eval_fn",
				},
			},
			expected:    "550e8400-e29b-41d4-a716-446655440001:c:f:expr_fn:eval_fn",
			description: "function concurrency with custom key should include both hashes",
		},

		// Throttle Constraints - No Custom Key
		{
			name: "throttle account scope no custom key",
			constraint: ConstraintItem{
				Kind: ConstraintKindThrottle,
				Throttle: &ThrottleConstraint{
					Scope: enums.ThrottleScopeAccount,
				},
			},
			expected:    "550e8400-e29b-41d4-a716-446655440001:t:a",
			description: "account throttle without custom key should use account ID",
		},
		{
			name: "throttle env scope no custom key",
			constraint: ConstraintItem{
				Kind: ConstraintKindThrottle,
				Throttle: &ThrottleConstraint{
					Scope: enums.ThrottleScopeEnv,
				},
			},
			expected:    "550e8400-e29b-41d4-a716-446655440001:t:e:550e8400-e29b-41d4-a716-446655440002",
			description: "env throttle without custom key should include env ID",
		},
		{
			name: "throttle function scope no custom key",
			constraint: ConstraintItem{
				Kind: ConstraintKindThrottle,
				Throttle: &ThrottleConstraint{
					Scope: enums.ThrottleScopeFn,
				},
			},
			expected:    "550e8400-e29b-41d4-a716-446655440001:t:f:550e8400-e29b-41d4-a716-446655440003",
			description: "function throttle without custom key should include function ID",
		},

		// Throttle Constraints - With Custom Key
		{
			name: "throttle account scope with custom key",
			constraint: ConstraintItem{
				Kind: ConstraintKindThrottle,
				Throttle: &ThrottleConstraint{
					Scope:             enums.ThrottleScopeAccount,
					KeyExpressionHash: "throttle_expr",
					EvaluatedKeyHash:  "throttle_eval",
				},
			},
			expected:    "550e8400-e29b-41d4-a716-446655440001:t:a:throttle_expr:throttle_eval",
			description: "account throttle with custom key should include both hashes",
		},
		{
			name: "throttle env scope with custom key",
			constraint: ConstraintItem{
				Kind: ConstraintKindThrottle,
				Throttle: &ThrottleConstraint{
					Scope:             enums.ThrottleScopeEnv,
					KeyExpressionHash: "throttle_env_expr",
					EvaluatedKeyHash:  "throttle_env_eval",
				},
			},
			expected:    "550e8400-e29b-41d4-a716-446655440001:t:e:throttle_env_expr:throttle_env_eval",
			description: "env throttle with custom key should include both hashes",
		},
		{
			name: "throttle function scope with custom key",
			constraint: ConstraintItem{
				Kind: ConstraintKindThrottle,
				Throttle: &ThrottleConstraint{
					Scope:             enums.ThrottleScopeFn,
					KeyExpressionHash: "throttle_fn_expr",
					EvaluatedKeyHash:  "throttle_fn_eval",
				},
			},
			expected:    "550e8400-e29b-41d4-a716-446655440001:t:f:throttle_fn_expr:throttle_fn_eval",
			description: "function throttle with custom key should include both hashes",
		},

		// Rate Limit Constraints - No Custom Key
		{
			name: "rate limit account scope no custom key",
			constraint: ConstraintItem{
				Kind: ConstraintKindRateLimit,
				RateLimit: &RateLimitConstraint{
					Scope: enums.RateLimitScopeAccount,
				},
			},
			expected:    "550e8400-e29b-41d4-a716-446655440001:r:a",
			description: "account rate limit without custom key should use account ID",
		},
		{
			name: "rate limit env scope no custom key",
			constraint: ConstraintItem{
				Kind: ConstraintKindRateLimit,
				RateLimit: &RateLimitConstraint{
					Scope: enums.RateLimitScopeEnv,
				},
			},
			expected:    "550e8400-e29b-41d4-a716-446655440001:r:e:550e8400-e29b-41d4-a716-446655440002",
			description: "env rate limit without custom key should include env ID",
		},
		{
			name: "rate limit function scope no custom key",
			constraint: ConstraintItem{
				Kind: ConstraintKindRateLimit,
				RateLimit: &RateLimitConstraint{
					Scope: enums.RateLimitScopeFn,
				},
			},
			expected:    "550e8400-e29b-41d4-a716-446655440001:r:f:550e8400-e29b-41d4-a716-446655440003",
			description: "function rate limit without custom key should include function ID",
		},

		// Rate Limit Constraints - With Custom Key
		{
			name: "rate limit account scope with custom key",
			constraint: ConstraintItem{
				Kind: ConstraintKindRateLimit,
				RateLimit: &RateLimitConstraint{
					Scope:             enums.RateLimitScopeAccount,
					KeyExpressionHash: "rl_expr",
					EvaluatedKeyHash:  "rl_eval",
				},
			},
			expected:    "550e8400-e29b-41d4-a716-446655440001:r:a:rl_expr:rl_eval",
			description: "account rate limit with custom key should include both hashes",
		},
		{
			name: "rate limit env scope with custom key",
			constraint: ConstraintItem{
				Kind: ConstraintKindRateLimit,
				RateLimit: &RateLimitConstraint{
					Scope:             enums.RateLimitScopeEnv,
					KeyExpressionHash: "rl_env_expr",
					EvaluatedKeyHash:  "rl_env_eval",
				},
			},
			expected:    "550e8400-e29b-41d4-a716-446655440001:r:e:rl_env_expr:rl_env_eval",
			description: "env rate limit with custom key should include both hashes",
		},
		{
			name: "rate limit function scope with custom key",
			constraint: ConstraintItem{
				Kind: ConstraintKindRateLimit,
				RateLimit: &RateLimitConstraint{
					Scope:             enums.RateLimitScopeFn,
					KeyExpressionHash: "rl_fn_expr",
					EvaluatedKeyHash:  "rl_fn_eval",
				},
			},
			expected:    "550e8400-e29b-41d4-a716-446655440001:r:f:rl_fn_expr:rl_fn_eval",
			description: "function rate limit with custom key should include both hashes",
		},

		// Edge Cases
		{
			name: "concurrency with nil constraint pointer",
			constraint: ConstraintItem{
				Kind:        ConstraintKindConcurrency,
				Concurrency: nil,
			},
			expected:    "",
			description: "nil concurrency pointer should return empty string",
		},
		{
			name: "throttle with nil constraint pointer",
			constraint: ConstraintItem{
				Kind:     ConstraintKindThrottle,
				Throttle: nil,
			},
			expected:    "",
			description: "nil throttle pointer should return empty string",
		},
		{
			name: "rate limit with nil constraint pointer",
			constraint: ConstraintItem{
				Kind:      ConstraintKindRateLimit,
				RateLimit: nil,
			},
			expected:    "",
			description: "nil rate limit pointer should return empty string",
		},
		{
			name: "unknown constraint kind",
			constraint: ConstraintItem{
				Kind: ConstraintKind("unknown"),
			},
			expected:    "",
			description: "unknown constraint kind should return empty string",
		},
		{
			name:        "zero value constraint item",
			constraint:  ConstraintItem{},
			expected:    "",
			description: "zero-value constraint item should return empty string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.constraint.CacheKey(accountID, envID, functionID)
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

func TestConstraintItem_CacheKey_AlwaysIncludesAccountID(t *testing.T) {
	accountID := uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	envID := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	functionID := uuid.MustParse("66666666-7777-8888-9999-000000000000")

	tests := []struct {
		name       string
		constraint ConstraintItem
	}{
		{
			name: "concurrency account scope",
			constraint: ConstraintItem{
				Kind:        ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{Scope: enums.ConcurrencyScopeAccount},
			},
		},
		{
			name: "concurrency env scope",
			constraint: ConstraintItem{
				Kind:        ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{Scope: enums.ConcurrencyScopeEnv},
			},
		},
		{
			name: "concurrency function scope",
			constraint: ConstraintItem{
				Kind:        ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{Scope: enums.ConcurrencyScopeFn},
			},
		},
		{
			name: "throttle account scope",
			constraint: ConstraintItem{
				Kind:     ConstraintKindThrottle,
				Throttle: &ThrottleConstraint{Scope: enums.ThrottleScopeAccount},
			},
		},
		{
			name: "throttle env scope",
			constraint: ConstraintItem{
				Kind:     ConstraintKindThrottle,
				Throttle: &ThrottleConstraint{Scope: enums.ThrottleScopeEnv},
			},
		},
		{
			name: "throttle function scope",
			constraint: ConstraintItem{
				Kind:     ConstraintKindThrottle,
				Throttle: &ThrottleConstraint{Scope: enums.ThrottleScopeFn},
			},
		},
		{
			name: "rate limit account scope",
			constraint: ConstraintItem{
				Kind:      ConstraintKindRateLimit,
				RateLimit: &RateLimitConstraint{Scope: enums.RateLimitScopeAccount},
			},
		},
		{
			name: "rate limit env scope",
			constraint: ConstraintItem{
				Kind:      ConstraintKindRateLimit,
				RateLimit: &RateLimitConstraint{Scope: enums.RateLimitScopeEnv},
			},
		},
		{
			name: "rate limit function scope",
			constraint: ConstraintItem{
				Kind:      ConstraintKindRateLimit,
				RateLimit: &RateLimitConstraint{Scope: enums.RateLimitScopeFn},
			},
		},
		{
			name: "concurrency with custom key",
			constraint: ConstraintItem{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Scope:             enums.ConcurrencyScopeAccount,
					KeyExpressionHash: "expr",
					EvaluatedKeyHash:  "eval",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := tt.constraint.CacheKey(accountID, envID, functionID)
			assert.NotEmpty(t, key, "cache key should not be empty")
			assert.Contains(t, key, accountID.String(), "cache key must always start with account ID")

			// Verify account ID is at the beginning
			assert.True(t, len(key) > 0 && key[:36] == accountID.String(),
				"cache key should start with account ID")
		})
	}
}

func TestConstraintItem_CacheKey_Uniqueness(t *testing.T) {
	accountID1 := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	accountID2 := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	envID1 := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	envID2 := uuid.MustParse("44444444-4444-4444-4444-444444444444")
	functionID1 := uuid.MustParse("55555555-5555-5555-5555-555555555555")
	functionID2 := uuid.MustParse("66666666-6666-6666-6666-666666666666")

	t.Run("different accounts produce different keys", func(t *testing.T) {
		constraint := ConstraintItem{
			Kind:        ConstraintKindConcurrency,
			Concurrency: &ConcurrencyConstraint{Scope: enums.ConcurrencyScopeAccount},
		}

		key1 := constraint.CacheKey(accountID1, envID1, functionID1)
		key2 := constraint.CacheKey(accountID2, envID1, functionID1)

		assert.NotEqual(t, key1, key2, "different account IDs should produce different cache keys")
	})

	t.Run("different envs produce different keys for env scope", func(t *testing.T) {
		constraint := ConstraintItem{
			Kind:        ConstraintKindConcurrency,
			Concurrency: &ConcurrencyConstraint{Scope: enums.ConcurrencyScopeEnv},
		}

		key1 := constraint.CacheKey(accountID1, envID1, functionID1)
		key2 := constraint.CacheKey(accountID1, envID2, functionID1)

		assert.NotEqual(t, key1, key2, "different env IDs should produce different cache keys for env scope")
	})

	t.Run("different functions produce different keys for function scope", func(t *testing.T) {
		constraint := ConstraintItem{
			Kind:        ConstraintKindConcurrency,
			Concurrency: &ConcurrencyConstraint{Scope: enums.ConcurrencyScopeFn},
		}

		key1 := constraint.CacheKey(accountID1, envID1, functionID1)
		key2 := constraint.CacheKey(accountID1, envID1, functionID2)

		assert.NotEqual(t, key1, key2, "different function IDs should produce different cache keys for function scope")
	})

	t.Run("different constraint kinds produce different keys", func(t *testing.T) {
		concurrency := ConstraintItem{
			Kind:        ConstraintKindConcurrency,
			Concurrency: &ConcurrencyConstraint{Scope: enums.ConcurrencyScopeAccount},
		}
		throttle := ConstraintItem{
			Kind:     ConstraintKindThrottle,
			Throttle: &ThrottleConstraint{Scope: enums.ThrottleScopeAccount},
		}
		rateLimit := ConstraintItem{
			Kind:      ConstraintKindRateLimit,
			RateLimit: &RateLimitConstraint{Scope: enums.RateLimitScopeAccount},
		}

		key1 := concurrency.CacheKey(accountID1, envID1, functionID1)
		key2 := throttle.CacheKey(accountID1, envID1, functionID1)
		key3 := rateLimit.CacheKey(accountID1, envID1, functionID1)

		assert.NotEqual(t, key1, key2, "concurrency and throttle should produce different keys")
		assert.NotEqual(t, key2, key3, "throttle and rate limit should produce different keys")
		assert.NotEqual(t, key1, key3, "concurrency and rate limit should produce different keys")
	})

	t.Run("different scopes produce different keys", func(t *testing.T) {
		accountScope := ConstraintItem{
			Kind:        ConstraintKindConcurrency,
			Concurrency: &ConcurrencyConstraint{Scope: enums.ConcurrencyScopeAccount},
		}
		envScope := ConstraintItem{
			Kind:        ConstraintKindConcurrency,
			Concurrency: &ConcurrencyConstraint{Scope: enums.ConcurrencyScopeEnv},
		}
		functionScope := ConstraintItem{
			Kind:        ConstraintKindConcurrency,
			Concurrency: &ConcurrencyConstraint{Scope: enums.ConcurrencyScopeFn},
		}

		key1 := accountScope.CacheKey(accountID1, envID1, functionID1)
		key2 := envScope.CacheKey(accountID1, envID1, functionID1)
		key3 := functionScope.CacheKey(accountID1, envID1, functionID1)

		assert.NotEqual(t, key1, key2, "account and env scope should produce different keys")
		assert.NotEqual(t, key2, key3, "env and function scope should produce different keys")
		assert.NotEqual(t, key1, key3, "account and function scope should produce different keys")
	})

	t.Run("custom keys produce different keys from non-custom", func(t *testing.T) {
		standard := ConstraintItem{
			Kind:        ConstraintKindConcurrency,
			Concurrency: &ConcurrencyConstraint{Scope: enums.ConcurrencyScopeAccount},
		}
		custom := ConstraintItem{
			Kind: ConstraintKindConcurrency,
			Concurrency: &ConcurrencyConstraint{
				Scope:             enums.ConcurrencyScopeAccount,
				KeyExpressionHash: "expr",
				EvaluatedKeyHash:  "eval",
			},
		}

		key1 := standard.CacheKey(accountID1, envID1, functionID1)
		key2 := custom.CacheKey(accountID1, envID1, functionID1)

		assert.NotEqual(t, key1, key2, "custom key constraints should produce different keys than standard")
	})

	t.Run("different custom keys produce different keys", func(t *testing.T) {
		custom1 := ConstraintItem{
			Kind: ConstraintKindConcurrency,
			Concurrency: &ConcurrencyConstraint{
				Scope:             enums.ConcurrencyScopeAccount,
				KeyExpressionHash: "expr1",
				EvaluatedKeyHash:  "eval1",
			},
		}
		custom2 := ConstraintItem{
			Kind: ConstraintKindConcurrency,
			Concurrency: &ConcurrencyConstraint{
				Scope:             enums.ConcurrencyScopeAccount,
				KeyExpressionHash: "expr2",
				EvaluatedKeyHash:  "eval2",
			},
		}

		key1 := custom1.CacheKey(accountID1, envID1, functionID1)
		key2 := custom2.CacheKey(accountID1, envID1, functionID1)

		assert.NotEqual(t, key1, key2, "different custom keys should produce different cache keys")
	})
}

func TestConstraintItem_CacheKey_Format(t *testing.T) {
	accountID := uuid.MustParse("12345678-1234-1234-1234-123456789012")
	envID := uuid.MustParse("abcdef00-abcd-abcd-abcd-abcdefabcdef")
	functionID := uuid.MustParse("fedcba00-fedc-fedc-fedc-fedcbafedcba")

	t.Run("concurrency keys have correct format", func(t *testing.T) {
		constraint := ConstraintItem{
			Kind:        ConstraintKindConcurrency,
			Concurrency: &ConcurrencyConstraint{Scope: enums.ConcurrencyScopeAccount},
		}

		key := constraint.CacheKey(accountID, envID, functionID)
		assert.Contains(t, key, ":c:", "concurrency key should contain :c: marker")
		assert.True(t, len(key) > 0, "key should not be empty")
	})

	t.Run("throttle keys have correct format", func(t *testing.T) {
		constraint := ConstraintItem{
			Kind:     ConstraintKindThrottle,
			Throttle: &ThrottleConstraint{Scope: enums.ThrottleScopeAccount},
		}

		key := constraint.CacheKey(accountID, envID, functionID)
		assert.Contains(t, key, ":t:", "throttle key should contain :t: marker")
		assert.True(t, len(key) > 0, "key should not be empty")
	})

	t.Run("rate limit keys have correct format", func(t *testing.T) {
		constraint := ConstraintItem{
			Kind:      ConstraintKindRateLimit,
			RateLimit: &RateLimitConstraint{Scope: enums.RateLimitScopeAccount},
		}

		key := constraint.CacheKey(accountID, envID, functionID)
		assert.Contains(t, key, ":r:", "rate limit key should contain :r: marker")
		assert.True(t, len(key) > 0, "key should not be empty")
	})

	t.Run("account scope keys have correct format", func(t *testing.T) {
		constraint := ConstraintItem{
			Kind:        ConstraintKindConcurrency,
			Concurrency: &ConcurrencyConstraint{Scope: enums.ConcurrencyScopeAccount},
		}

		key := constraint.CacheKey(accountID, envID, functionID)
		assert.Contains(t, key, ":a", "account scope key should contain :a marker")
	})

	t.Run("env scope keys have correct format", func(t *testing.T) {
		constraint := ConstraintItem{
			Kind:        ConstraintKindConcurrency,
			Concurrency: &ConcurrencyConstraint{Scope: enums.ConcurrencyScopeEnv},
		}

		key := constraint.CacheKey(accountID, envID, functionID)
		assert.Contains(t, key, ":e:", "env scope key should contain :e: marker")
		assert.Contains(t, key, envID.String(), "env scope key should contain env ID")
	})

	t.Run("function scope keys have correct format", func(t *testing.T) {
		constraint := ConstraintItem{
			Kind:        ConstraintKindConcurrency,
			Concurrency: &ConcurrencyConstraint{Scope: enums.ConcurrencyScopeFn},
		}

		key := constraint.CacheKey(accountID, envID, functionID)
		assert.Contains(t, key, ":f:", "function scope key should contain :f: marker")
		assert.Contains(t, key, functionID.String(), "function scope key should contain function ID")
	})

	t.Run("custom key format includes both hashes", func(t *testing.T) {
		constraint := ConstraintItem{
			Kind: ConstraintKindConcurrency,
			Concurrency: &ConcurrencyConstraint{
				Scope:             enums.ConcurrencyScopeAccount,
				KeyExpressionHash: "my_expr",
				EvaluatedKeyHash:  "my_eval",
			},
		}

		key := constraint.CacheKey(accountID, envID, functionID)
		assert.Contains(t, key, "my_expr", "custom key should contain expression hash")
		assert.Contains(t, key, "my_eval", "custom key should contain evaluated hash")

		// Check that expr comes before eval
		exprIdx := strings.Index(key, "my_expr")
		evalIdx := strings.Index(key, "my_eval")
		assert.True(t, exprIdx < evalIdx, "expression hash should come before evaluated hash")
	})
}
