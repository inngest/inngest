package constraintapi

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

// Use a dedicated, disposable localhost Redis/Valkey with no other Lua callers.
// ns/op includes Go and the round trip; redis-us/op is server-side EVALSHA CPU.
// A warm legacy denial is served in-process and explicitly reports zero EVALs.
func BenchmarkManualSemaphoreOwnership(b *testing.B) {
	addr := os.Getenv("SEMAPHORE_TEST_REDIS_ADDR")
	if addr == "" {
		b.Skip("set SEMAPHORE_TEST_REDIS_ADDR to a disposable localhost engine")
	}
	host, _, err := net.SplitHostPort(addr)
	require.NoError(b, err)
	require.Equal(b, "127.0.0.1", host)
	for _, limits := range []int{1, 3} {
		for _, tracked := range []bool{false, true} {
			for _, operation := range []string{"acquire", "denied-cache", "retry", "finalize"} {
				if operation == "retry" && !tracked {
					continue // Legacy retries cannot reuse a whole-run permit.
				}
				if operation == "finalize" && limits != 1 {
					continue // The legacy release dedupe key is shared across a run's limits.
				}
				b.Run(fmt.Sprintf("%s/limits=%d/tracked=%t", operation, limits, tracked), func(b *testing.B) {
					ctx := context.Background()
					client, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{addr}, DisableCache: true})
					require.NoError(b, err)
					defer client.Close()
					cm, err := NewRedisCapacityManager(WithClient(client), WithShardName("benchmark"),
						WithManualSemaphoreOwnership(func(uuid.UUID) bool { return tracked }))
					require.NoError(b, err)
					account, env, fn := uuid.New(), uuid.New(), uuid.New()
					run := ulid.Make()
					req := &CapacityAcquireRequest{
						AccountID: account, EnvID: env, FunctionID: fn, IdempotencyKey: "benchmark", Amount: 1,
						LeaseIdempotencyKeys: []string{"start"}, LeaseRunIDs: map[string]ulid.ULID{"start": run},
						Configuration: ConstraintConfig{FunctionVersion: 1},
						CurrentTime:   time.Now(), Duration: time.Minute, MaximumLifetime: time.Hour,
						Source: LeaseSource{Service: ServiceExecutor, Location: CallerLocationItemLease},
					}
					fixedKeys := []string{cm.keyAccountLeases(account), cm.keyConstraintCheckIdempotency(account, "start"),
						cm.keyOperationIdempotency(account, "rel", "benchmark-release"), semaphoreIdempotencyKey(account, "rel", run.String())}
					var capKeys []string
					for i := range limits {
						sem := &SemaphoreConstraint{ID: fmt.Sprintf("fnkey:bench-%d", i), Weight: 1, Release: SemaphoreReleaseManual}
						req.Constraints = append(req.Constraints, ConstraintItem{Kind: ConstraintKindSemaphore, Semaphore: sem})
						req.Configuration.Semaphores = append(req.Configuration.Semaphores, Semaphore{ID: sem.ID, Weight: 1, Release: sem.Release})
						require.NoError(b, client.Do(ctx, client.B().Set().Key(sem.CapacityKey(account)).Value("1").Build()).Error())
						capKeys = append(capKeys, sem.CapacityKey(account))
						fixedKeys = append(fixedKeys, sem.UsageKey(account), sem.UsageKey(account)+":ownership", sem.UsageKey(account)+":owner:"+run.String(),
							cm.keyConstraintCache(account, env, fn, req.Constraints[len(req.Constraints)-1]))
					}
					attempt := *req
					attempt.IdempotencyKey = "benchmark-attempt"
					if operation == "denied-cache" {
						attempt.LeaseRunIDs = map[string]ulid.ULID{"start": ulid.Make()}
					}
					for _, request := range []*CapacityAcquireRequest{req, &attempt} {
						_, _, _, fingerprint, err := buildRequestState(request, tracked)
						require.NoError(b, err)
						fixedKeys = append(fixedKeys, cm.keyConstraintCheckIdempotency(account, request.IdempotencyKey),
							cm.keyOperationIdempotency(account, "acq", request.IdempotencyKey+"-"+fingerprint))
					}
					cleanup := func(responses ...*CapacityAcquireResponse) {
						keys := append([]string{}, fixedKeys...)
						for _, res := range responses {
							if res == nil {
								continue
							}
							keys = append(keys, cm.keyRequestState(account, res.RequestID))
							for _, lease := range res.Leases {
								keys = append(keys, cm.keyLeaseDetails(account, lease.LeaseID))
							}
						}
						require.NoError(b, client.Do(ctx, client.B().Del().Key(keys...).Build()).Error())
						require.NoError(b, client.Do(ctx, client.B().Zrem().Key(cm.keyScavengerShard()).Member(account.String()).Build()).Error())
					}
					var manager CapacityManager = cm
					var resident *CapacityAcquireResponse
					if operation == "denied-cache" {
						cm.enableAcquireCache = enableAllCache()
						cm.acquireCacheTTL = func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (time.Duration, time.Duration) {
							return time.Hour, time.Hour
						}
						cache := NewConstraintCache(WithConstraintCacheManager(cm),
							WithConstraintCacheEnable(func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (bool, time.Duration, time.Duration) {
								return true, time.Hour, time.Hour
							}),
							WithConstraintCacheShouldCache(func(ConstraintItem) bool { return !tracked }))
						defer cache.cache.Stop()
						manager = cache
						resident, err = cm.Acquire(ctx, req)
						require.NoError(b, err)
						require.Len(b, resident.Leases, 1)
					}
					sm := NewRedisSemaphoreManager(client)
					var evals, usec int64
					b.ResetTimer()
					b.StopTimer()
					// The untimed first iteration warms scripts and the denial cache.
					for i := -1; i < b.N; i++ {
						var first, res *CapacityAcquireResponse
						if operation == "retry" || operation == "finalize" {
							first, err = cm.Acquire(ctx, req)
							require.NoError(b, err)
							require.Len(b, first.Leases, 1)
							if operation == "retry" {
								_, err = cm.Release(ctx, &CapacityReleaseRequest{AccountID: account, LeaseID: first.Leases[0].LeaseID,
									IdempotencyKey: "benchmark-release", LeaseIssuedAt: time.Now(), Source: req.Source})
								require.NoError(b, err)
							}
						}
						var calls, cpu int64
						if i >= 0 {
							calls, cpu = benchmarkEvalStats(b, client)
							b.StartTimer()
						}
						if operation == "finalize" {
							for _, ci := range req.Constraints {
								err = sm.ReleaseSemaphore(ctx, account, ci.Semaphore.ID, "", run.String(), 1)
								if err != nil {
									break
								}
							}
						} else if operation == "acquire" {
							res, err = manager.Acquire(ctx, req)
						} else {
							res, err = manager.Acquire(ctx, &attempt)
						}
						b.StopTimer()
						require.NoError(b, err)
						wantCalls := int64(1)
						if operation == "finalize" {
							wantCalls = int64(limits)
						} else if operation == "denied-cache" {
							require.Empty(b, res.Leases)
							if !tracked && i >= 0 {
								wantCalls = 0
							}
						} else {
							require.Len(b, res.Leases, 1)
							require.False(b, res.OperationIdempotencyHit, "measure actual admission, not a dedupe hit")
						}
						if i >= 0 {
							endCalls, endCPU := benchmarkEvalStats(b, client)
							require.Equal(b, wantCalls, endCalls-calls, "use a dedicated engine with no other Lua callers")
							evals += endCalls - calls
							usec += endCPU - cpu
						}
						if operation == "finalize" {
							_, usage, err := sm.GetCapacity(ctx, account, req.Constraints[0].Semaphore.ID, "")
							require.NoError(b, err)
							require.Zero(b, usage, "measure a real finalization, not a dedupe hit")
						}
						if operation != "denied-cache" {
							cleanup(first, res)
						}
					}
					b.ReportMetric(float64(evals)/float64(b.N), "redis-evals/op")
					b.ReportMetric(float64(usec)/float64(b.N), "redis-us/op")
					cleanup(resident)
					require.NoError(b, client.Do(ctx, client.B().Del().Key(capKeys...).Build()).Error())
				})
			}
		}
	}
}

func benchmarkEvalStats(b *testing.B, client rueidis.Client) (calls, usec int64) {
	b.Helper()
	info, err := client.Do(context.Background(), client.B().Info().Section("commandstats").Build()).ToString()
	require.NoError(b, err)
	for _, line := range strings.Split(info, "\n") {
		if strings.HasPrefix(line, "cmdstat_evalsha:") {
			n, err := fmt.Sscanf(line, "cmdstat_evalsha:calls=%d,usec=%d,", &calls, &usec)
			require.NoError(b, err)
			require.Equal(b, 2, n)
			return calls, usec
		}
	}
	b.Fatal("Redis did not expose EVALSHA commandstats")
	return
}
