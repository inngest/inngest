package constraintapi

import (
	"context"
	"fmt"
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
	}

	type testCase struct {
		name string

		amount      int
		config      ConstraintConfig
		constraints []ConstraintItem
		mi          MigrationIdentifier

		beforeAcquire func(t *testing.T, deps *deps)

		afterAcquire func(t *testing.T, deps *deps, resp *CapacityAcquireResponse)

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
					cm.keyOperationIdempotency(cm.queueStateKeyPrefix, accountID, "acq", "acquire"),
					cm.keyConstraintCheckIdempotency(cm.queueStateKeyPrefix, accountID, "acquire"),
					cm.keyConstraintCheckIdempotency(cm.queueStateKeyPrefix, accountID, "item0"),
					cm.keyRequestState(cm.queueStateKeyPrefix, accountID, "acquire"),
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
					cm.keyOperationIdempotency(cm.queueStateKeyPrefix, accountID, "acq", "acquire"),
					cm.keyOperationIdempotency(cm.queueStateKeyPrefix, accountID, "ext", "extend"),
					cm.keyConstraintCheckIdempotency(cm.queueStateKeyPrefix, accountID, "acquire"),
					cm.keyConstraintCheckIdempotency(cm.queueStateKeyPrefix, accountID, "item0"),
					cm.keyRequestState(cm.queueStateKeyPrefix, accountID, "acquire"),
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
					cm.keyOperationIdempotency(cm.queueStateKeyPrefix, accountID, "acq", "acquire"),
					cm.keyOperationIdempotency(cm.queueStateKeyPrefix, accountID, "ext", "extend"),
					cm.keyOperationIdempotency(cm.queueStateKeyPrefix, accountID, "rel", "release"),
					cm.keyConstraintCheckIdempotency(cm.queueStateKeyPrefix, accountID, "acquire"),
					cm.keyConstraintCheckIdempotency(cm.queueStateKeyPrefix, accountID, "item0"),
				}, keys)
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
						Location:          LeaseLocationPartitionLease,
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
				require.Len(t, keys, 8)
				require.Subset(t, []string{
					cm.keyScavengerShard(cm.queueStateKeyPrefix, 0),
					cm.keyAccountLeases(cm.queueStateKeyPrefix, accountID),
					cm.keyOperationIdempotency(cm.queueStateKeyPrefix, accountID, "acq", "acquire"),
					cm.keyConstraintCheckIdempotency(cm.queueStateKeyPrefix, accountID, "acquire"),
					cm.keyConstraintCheckIdempotency(cm.queueStateKeyPrefix, accountID, "item0"),
					cm.keyRequestState(cm.queueStateKeyPrefix, accountID, "acquire"),
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
					cm.keyOperationIdempotency(cm.queueStateKeyPrefix, accountID, "acq", "acquire"),
					cm.keyOperationIdempotency(cm.queueStateKeyPrefix, accountID, "ext", "extend"),
					cm.keyConstraintCheckIdempotency(cm.queueStateKeyPrefix, accountID, "acquire"),
					cm.keyConstraintCheckIdempotency(cm.queueStateKeyPrefix, accountID, "item0"),
					cm.keyRequestState(cm.queueStateKeyPrefix, accountID, "acquire"),
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
					cm.keyOperationIdempotency(cm.queueStateKeyPrefix, accountID, "acq", "acquire"),
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
					cm.keyOperationIdempotency(cm.queueStateKeyPrefix, accountID, "acq", "acquire"),
					cm.keyConstraintCheckIdempotency(cm.queueStateKeyPrefix, accountID, "acquire"),
					cm.keyConstraintCheckIdempotency(cm.queueStateKeyPrefix, accountID, "item0"),
					cm.keyRequestState(cm.queueStateKeyPrefix, accountID, "acquire"),
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
					cm.keyOperationIdempotency(cm.queueStateKeyPrefix, accountID, "acq", "acquire"),
					cm.keyOperationIdempotency(cm.queueStateKeyPrefix, accountID, "ext", "extend"),
					cm.keyConstraintCheckIdempotency(cm.queueStateKeyPrefix, accountID, "acquire"),
					cm.keyConstraintCheckIdempotency(cm.queueStateKeyPrefix, accountID, "item0"),
					cm.keyRequestState(cm.queueStateKeyPrefix, accountID, "acquire"),
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
					cm.keyOperationIdempotency(cm.queueStateKeyPrefix, accountID, "acq", "acquire"),
					cm.keyOperationIdempotency(cm.queueStateKeyPrefix, accountID, "ext", "extend"),
					cm.keyOperationIdempotency(cm.queueStateKeyPrefix, accountID, "rel", "release"),
					cm.keyConstraintCheckIdempotency(cm.queueStateKeyPrefix, accountID, "acquire"),
					cm.keyConstraintCheckIdempotency(cm.queueStateKeyPrefix, accountID, "item0"),
				}, keys)
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

			clock := clockwork.NewFakeClock()

			cm, err := NewRedisCapacityManager(
				WithRateLimitClient(rc),
				WithQueueShards(map[string]rueidis.Client{
					"test": rc,
				}),
				WithClock(clock),
				WithNumScavengerShards(1),
				WithQueueStateKeyPrefix("q:v1"),
				WithRateLimitKeyPrefix("rl"),
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

			leaseIdempotencyKeys := make([]string, test.amount)
			for i := range test.amount {
				leaseIdempotencyKeys[i] = fmt.Sprintf("item%d", i)
			}

			acquireResp, err := cm.Acquire(ctx, &CapacityAcquireRequest{
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
					Location:          LeaseLocationItemLease,
					RunProcessingMode: RunProcessingModeBackground,
				},
			})
			require.NoError(t, err)

			if test.afterAcquire != nil {
				test.afterAcquire(t, deps, acquireResp)
			}

			require.Len(t, acquireResp.Leases, test.expectedLeaseAmount)

			if test.expectedLeaseAmount == 0 {
				return
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
