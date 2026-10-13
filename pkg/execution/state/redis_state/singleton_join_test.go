package redis_state

import (
	"context"
	"crypto/rand"
	"fmt"
	"sync"
	"testing"
	"time"

	"encoding/json"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func newJoinShard(t *testing.T) (*miniredis.Miniredis, RedisQueueShard) {
	t.Helper()

	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	t.Cleanup(rc.Close)

	return r, shardFromClient("default", rc)
}

func TestSingletonJoin(t *testing.T) {
	ctx := context.Background()
	scope := osqueue.Scope{}
	grace := 10 * time.Minute

	t.Run("a waiter registered before completion is returned by completion", func(t *testing.T) {
		r, shard := newJoinShard(t)
		kg := shard.Client().kg
		run := ulid.MustNew(ulid.Now(), rand.Reader)

		payload, err := shard.SingletonJoin(ctx, scope, run, "a.step", time.Hour)
		require.NoError(t, err)
		require.Nil(t, payload, "not complete yet, so the waiter is registered")
		require.True(t, r.Exists(kg.SingletonJoinKey(run.String())))

		members, err := shard.SingletonJoinComplete(ctx, scope, run, []byte(`{"result":1}`), grace)
		require.NoError(t, err)
		require.Equal(t, []string{"a.step"}, members)
	})

	t.Run("a waiter arriving after completion gets the payload and is not registered", func(t *testing.T) {
		r, shard := newJoinShard(t)
		kg := shard.Client().kg
		run := ulid.MustNew(ulid.Now(), rand.Reader)

		_, err := shard.SingletonJoinComplete(ctx, scope, run, []byte(`{"result":1}`), grace)
		require.NoError(t, err)

		payload, err := shard.SingletonJoin(ctx, scope, run, "late.step", time.Hour)
		require.NoError(t, err)
		require.JSONEq(t, `{"result":1}`, string(payload))

		members, err := r.SMembers(kg.SingletonJoinKey(run.String()))
		if err == nil {
			require.NotContains(t, members, "late.step")
		}
	})

	t.Run("completion is idempotent and keeps returning the same waiters", func(t *testing.T) {
		_, shard := newJoinShard(t)
		run := ulid.MustNew(ulid.Now(), rand.Reader)

		for i := range 3 {
			_, err := shard.SingletonJoin(ctx, scope, run, fmt.Sprintf("w%d.step", i), time.Hour)
			require.NoError(t, err)
		}

		first, err := shard.SingletonJoinComplete(ctx, scope, run, []byte(`{}`), grace)
		require.NoError(t, err)
		second, err := shard.SingletonJoinComplete(ctx, scope, run, []byte(`{}`), grace)
		require.NoError(t, err)

		require.ElementsMatch(t, []string{"w0.step", "w1.step", "w2.step"}, first)
		require.ElementsMatch(t, first, second)
	})

	t.Run("registering the same waiter twice is a no-op", func(t *testing.T) {
		_, shard := newJoinShard(t)
		run := ulid.MustNew(ulid.Now(), rand.Reader)

		for range 5 {
			_, err := shard.SingletonJoin(ctx, scope, run, "same.step", time.Hour)
			require.NoError(t, err)
		}

		members, err := shard.SingletonJoinComplete(ctx, scope, run, []byte(`{}`), grace)
		require.NoError(t, err)
		require.Equal(t, []string{"same.step"}, members)
	})

	t.Run("runs are isolated from each other", func(t *testing.T) {
		_, shard := newJoinShard(t)
		a := ulid.MustNew(ulid.Now(), rand.Reader)
		b := ulid.MustNew(ulid.Now(), rand.Reader)

		_, err := shard.SingletonJoin(ctx, scope, a, "for-a.step", time.Hour)
		require.NoError(t, err)

		members, err := shard.SingletonJoinComplete(ctx, scope, b, []byte(`{}`), grace)
		require.NoError(t, err)
		require.Empty(t, members)

		payload, err := shard.SingletonJoin(ctx, scope, a, "also-a.step", time.Hour)
		require.NoError(t, err)
		require.Nil(t, payload, "completing b must not complete a")
	})

	// No orphaned registrations: every key has a TTL and disappears without
	// anyone cleaning it up, e.g. when the active run never finalizes.
	t.Run("keys always expire", func(t *testing.T) {
		r, shard := newJoinShard(t)
		kg := shard.Client().kg
		run := ulid.MustNew(ulid.Now(), rand.Reader)
		setKey, doneKey := kg.SingletonJoinKey(run.String()), kg.SingletonJoinDoneKey(run.String())

		// A run that never finalizes: the waiter set expires by TTL alone.
		_, err := shard.SingletonJoin(ctx, scope, run, "orphan.step", time.Hour)
		require.NoError(t, err)
		require.Equal(t, time.Hour, r.TTL(setKey))
		r.FastForward(time.Hour + time.Second)
		require.False(t, r.Exists(setKey))

		// A run that does finalize: both keys expire after the grace period.
		_, err = shard.SingletonJoin(ctx, scope, run, "w.step", time.Hour)
		require.NoError(t, err)
		_, err = shard.SingletonJoinComplete(ctx, scope, run, []byte(`{}`), grace)
		require.NoError(t, err)
		require.Equal(t, grace, r.TTL(setKey))
		require.Equal(t, grace, r.TTL(doneKey))
		r.FastForward(grace + time.Second)
		require.False(t, r.Exists(setKey))
		require.False(t, r.Exists(doneKey))
	})

	t.Run("a short-lived waiter never shortens a long-lived one", func(t *testing.T) {
		r, shard := newJoinShard(t)
		kg := shard.Client().kg
		run := ulid.MustNew(ulid.Now(), rand.Reader)

		_, err := shard.SingletonJoin(ctx, scope, run, "long.step", 24*time.Hour)
		require.NoError(t, err)
		_, err = shard.SingletonJoin(ctx, scope, run, "short.step", time.Minute)
		require.NoError(t, err)

		require.Equal(t, 24*time.Hour, r.TTL(kg.SingletonJoinKey(run.String())))
	})
}

// TestSingletonJoinInterleavings hammers Join against Complete. The invariant
// that makes the feature hang-free: every waiter is EITHER returned by Complete
// OR handed the completion payload by Join. Never neither (a hang), never both
// (a double delivery).
func TestSingletonJoinInterleavings(t *testing.T) {
	ctx := context.Background()
	scope := osqueue.Scope{}

	const (
		rounds  = 25
		waiters = 64
	)

	_, shard := newJoinShard(t)

	var byMembers, byPayload int

	for round := range rounds {
		run := ulid.MustNew(ulid.Now(), rand.Reader)

		var (
			start    = make(chan struct{})
			wg       sync.WaitGroup
			mu       sync.Mutex
			resolved = map[string]bool{} // joined via payload
		)

		for i := range waiters {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start

				id := fmt.Sprintf("r%d-w%d.step", round, i)
				payload, err := shard.SingletonJoin(ctx, scope, run, id, time.Hour)
				require.NoError(t, err)
				if payload != nil {
					mu.Lock()
					resolved[id] = true
					mu.Unlock()
				}
			}()
		}

		var members []string
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start

			// Land the completion somewhere in the middle of the joins.
			time.Sleep(time.Duration(round%5) * 200 * time.Microsecond)

			var err error
			members, err = shard.SingletonJoinComplete(ctx, scope, run, []byte(`{"result":"ok"}`), 10*time.Minute)
			require.NoError(t, err)
		}()

		close(start)
		wg.Wait()

		// Waiters that joined after completion saw the payload...
		seen := map[string]bool{}
		for _, m := range members {
			require.False(t, resolved[m], "%s was both returned by Complete and handed the payload", m)
			seen[m] = true
		}
		for id := range resolved {
			seen[id] = true
		}

		// ...and together with those returned by completion, that is everyone.
		require.Len(t, seen, waiters, "round %d: a waiter was neither returned nor resolved (would hang)", round)

		byMembers += len(members)
		byPayload += len(resolved)

		// A final completion (e.g. a retried finalize) adds nobody new and
		// does not lose anybody.
		again, err := shard.SingletonJoinComplete(ctx, scope, run, []byte(`{"result":"ok"}`), 10*time.Minute)
		require.NoError(t, err)
		require.ElementsMatch(t, members, again)
	}

	t.Logf("resolved via Complete: %d, via payload: %d", byMembers, byPayload)
}

// TestSingletonJoinFlagSurvivesStateRoundTrip proves the join flag set at
// schedule time is what finalization reads back from the real state store; if
// it were lost, joiners would never be resolved.
func TestSingletonJoinFlagSurvivesStateRoundTrip(t *testing.T) {
	ctx := context.Background()
	svc, _ := mustV2Service(t, ctx)

	build := func(join bool) statev2.Metadata {
		cfg := statev2.InitConfig(&statev2.Config{})
		cfg.SetSingletonJoin(join)

		return statev2.Metadata{
			ID: statev2.ID{
				RunID:      ulid.MustNew(ulid.Now(), rand.Reader),
				FunctionID: uuid.New(),
				Tenant:     statev2.Tenant{AccountID: uuid.New(), EnvID: uuid.New(), AppID: uuid.New()},
			},
			Config: *cfg,
		}
	}

	for _, join := range []bool{true, false} {
		md := build(join)
		_, err := svc.Create(ctx, statev2.CreateState{
			Metadata: md,
			Events:   []json.RawMessage{json.RawMessage(`{"name":"test/e","data":{}}`)},
		})
		require.NoError(t, err)

		loaded, err := svc.LoadMetadata(ctx, md.ID)
		require.NoError(t, err)
		require.Equal(t, join, loaded.Config.SingletonJoin())
	}
}
