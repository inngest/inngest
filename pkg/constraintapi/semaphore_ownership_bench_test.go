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
func BenchmarkManualSemaphoreOwnership(b *testing.B) {
	addr := os.Getenv("SEMAPHORE_TEST_REDIS_ADDR")
	if addr == "" {
		b.Skip("set SEMAPHORE_TEST_REDIS_ADDR to a disposable localhost engine")
	}
	host, _, err := net.SplitHostPort(addr)
	require.NoError(b, err)
	require.Equal(b, "127.0.0.1", host)
	for _, size := range []struct{ jobs, limits int }{{1, 1}, {20, 3}} {
		for _, tracked := range []bool{false, true} {
			b.Run(fmt.Sprintf("jobs=%d/limits=%d/tracked=%t", size.jobs, size.limits, tracked), func(b *testing.B) {
				ctx := context.Background()
				client, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{addr}, DisableCache: true})
				require.NoError(b, err)
				defer client.Close()
				cm, err := NewRedisCapacityManager(WithClient(client), WithShardName("benchmark"))
				require.NoError(b, err)
				cm.manualSemaphoreOwnership = tracked
				account, env, fn := uuid.New(), uuid.New(), uuid.New()
				req := &CapacityAcquireRequest{
					AccountID: account, EnvID: env, FunctionID: fn, IdempotencyKey: "benchmark", Amount: size.jobs,
					LeaseRunIDs: map[string]ulid.ULID{}, Configuration: ConstraintConfig{FunctionVersion: 1},
					CurrentTime: time.Now(), Duration: time.Minute, MaximumLifetime: time.Hour,
					Source: LeaseSource{Service: ServiceExecutor, Location: CallerLocationItemLease},
				}
				for j := range size.jobs {
					key := fmt.Sprint(j)
					req.LeaseIdempotencyKeys = append(req.LeaseIdempotencyKeys, key)
					req.LeaseRunIDs[key] = ulid.Make()
				}
				fixedKeys := []string{cm.keyAccountLeases(account), cm.keyConstraintCheckIdempotency(account, req.IdempotencyKey)}
				for _, key := range req.LeaseIdempotencyKeys {
					fixedKeys = append(fixedKeys, cm.keyConstraintCheckIdempotency(account, key))
				}
				var capKeys []string
				for i := range size.limits {
					sem := &SemaphoreConstraint{ID: fmt.Sprintf("fnkey:bench-%d", i), Weight: 1, Release: SemaphoreReleaseManual}
					req.Constraints = append(req.Constraints, ConstraintItem{Kind: ConstraintKindSemaphore, Semaphore: sem})
					req.Configuration.Semaphores = append(req.Configuration.Semaphores, Semaphore{ID: sem.ID, Weight: 1, Release: sem.Release})
					require.NoError(b, client.Do(ctx, client.B().Set().Key(sem.CapacityKey(account)).Value(fmt.Sprint(size.jobs)).Build()).Error())
					capKeys = append(capKeys, sem.CapacityKey(account))
					fixedKeys = append(fixedKeys, sem.UsageKey(account), sem.UsageKey(account)+":ownership")
					for _, run := range req.LeaseRunIDs {
						owner := sem.UsageKey(account) + ":owner:" + run.String()
						fixedKeys = append(fixedKeys, owner, owner+":claim")
					}
				}
				_, _, _, fingerprint, err := buildRequestStateWithOwnership(req, tracked)
				require.NoError(b, err)
				fixedKeys = append(fixedKeys, cm.keyOperationIdempotency(account, "acq", req.IdempotencyKey+"-"+fingerprint))
				cleanup := func(res *CapacityAcquireResponse) {
					keys := append([]string{}, fixedKeys...)
					keys = append(keys, cm.keyRequestState(account, res.RequestID))
					for _, lease := range res.Leases {
						keys = append(keys, cm.keyLeaseDetails(account, lease.LeaseID))
					}
					require.NoError(b, client.Do(ctx, client.B().Del().Key(keys...).Build()).Error())
					require.NoError(b, client.Do(ctx, client.B().Zrem().Key(cm.keyScavengerShard()).Member(account.String()).Build()).Error())
				}
				// Load the Lua script before collecting commandstats. Each timed
				// call must grant real leases, not hit dedupe or exhausted capacity.
				warm, internalErr := cm.Acquire(ctx, req)
				require.NoError(b, internalErr)
				require.Len(b, warm.Leases, size.jobs)
				cleanup(warm)
				calls, usec := benchmarkEvalStats(b, client)
				b.ResetTimer()
				for range b.N {
					res, err := cm.Acquire(ctx, req)
					b.StopTimer()
					require.NoError(b, err)
					require.Len(b, res.Leases, size.jobs)
					require.False(b, res.OperationIdempotencyHit)
					cleanup(res)
					b.StartTimer()
				}
				b.StopTimer()
				endCalls, endUsec := benchmarkEvalStats(b, client)
				require.EqualValues(b, b.N, endCalls-calls, "use a dedicated engine with no other Lua callers")
				b.ReportMetric(float64(endUsec-usec)/float64(b.N), "redis-us/op")
				require.NoError(b, client.Do(ctx, client.B().Del().Key(capKeys...).Build()).Error())
			})
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
