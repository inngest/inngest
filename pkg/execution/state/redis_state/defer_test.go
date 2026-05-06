package redis_state

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/logger"
)

func newDeferTestRunService(t *testing.T) (statev2.RunService, statev2.ID) {
	t.Helper()
	ctx := logger.WithStdlib(context.Background(), logger.VoidLogger())

	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{mr.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)

	unshardedClient := NewUnshardedClient(rc, StateDefaultKey, QueueDefaultKey)
	shardedClient := NewShardedClient(ShardedClientOpts{
		UnshardedClient:        unshardedClient,
		FunctionRunStateClient: rc,
		BatchClient:            rc,
		StateDefaultKey:        StateDefaultKey,
		QueueDefaultKey:        QueueDefaultKey,
		FnRunIsSharded:         AlwaysShardOnRun,
	})
	pauseStore := NewPauseStore(unshardedClient)

	mgr, err := New(ctx,
		WithShardedClient(shardedClient),
		WithPauseDeleter(pauseStore),
	)
	require.NoError(t, err)
	v2svc := MustRunServiceV2(mgr)

	id := statev2.ID{
		RunID:      ulid.MustNew(ulid.Now(), rand.Reader),
		FunctionID: uuid.New(),
		Tenant: statev2.Tenant{
			AccountID: uuid.New(),
			EnvID:     uuid.New(),
			AppID:     uuid.New(),
		},
	}
	eventID := ulid.MustNew(ulid.Now(), rand.Reader)
	_, err = v2svc.Create(ctx, statev2.CreateState{
		Metadata: statev2.Metadata{
			ID: id,
			Config: *statev2.InitConfig(&statev2.Config{
				EventIDs: []ulid.ULID{eventID},
			}),
		},
		Events: []json.RawMessage{[]byte(`{"name":"test.event"}`)},
	})
	require.NoError(t, err)

	return v2svc, id
}

func TestSaveDefer(t *testing.T) {
	ctx := context.Background()

	t.Run("new", func(t *testing.T) {
		v2svc, id := newDeferTestRunService(t)

		d := statev2.Defer{
			FnSlug:         "onDefer-score",
			HashedID:       "hash-step-1",
			ScheduleStatus: enums.DeferStatusAfterRun,
			Input:          json.RawMessage(`{"user_id":"u_123"}`),
		}
		require.NoError(t, v2svc.SaveDefer(ctx, id, d))

		defers, err := v2svc.LoadDefers(ctx, id)
		require.NoError(t, err)
		require.Equal(t, d, defers[d.HashedID])
	})

	t.Run("idempotent", func(t *testing.T) {
		v2svc, id := newDeferTestRunService(t)

		first := statev2.Defer{
			FnSlug:         "onDefer-score",
			HashedID:       "hash-step-1",
			ScheduleStatus: enums.DeferStatusAfterRun,
			Input:          json.RawMessage(`{"user_id":"u_old"}`),
		}
		require.NoError(t, v2svc.SaveDefer(ctx, id, first))

		// SDK retransmit with a different payload (e.g. user-code
		// non-determinism). Insert-only semantics: the original wins,
		// retransmit is a silent no-op.
		second := first
		second.Input = json.RawMessage(`{"user_id":"u_new"}`)
		require.NoError(t, v2svc.SaveDefer(ctx, id, second))

		defers, err := v2svc.LoadDefers(ctx, id)
		require.NoError(t, err)
		require.JSONEq(t, `{"user_id":"u_old"}`, string(defers[first.HashedID].Input))
	})

	t.Run("cannot reverse abort", func(t *testing.T) {
		// Once a defer is aborted it stays aborted
		v2svc, id := newDeferTestRunService(t)

		original := statev2.Defer{
			FnSlug:         "onDefer-score",
			HashedID:       "hash-step-1",
			ScheduleStatus: enums.DeferStatusAfterRun,
			Input:          json.RawMessage(`{"user_id":"u_123"}`),
		}
		require.NoError(t, v2svc.SaveDefer(ctx, id, original))
		require.NoError(t, v2svc.SetDeferStatus(ctx, id, original.HashedID, enums.DeferStatusAborted))

		// SDK retransmits the original DeferAdd. Must not error, must stay
		// aborted.
		require.NoError(t, v2svc.SaveDefer(ctx, id, original))

		defers, err := v2svc.LoadDefers(ctx, id)
		require.NoError(t, err)
		require.Len(t, defers, 1)

		got := defers[original.HashedID]
		require.Equal(t, enums.DeferStatusAborted, got.ScheduleStatus)
		require.Equal(t, original.FnSlug, got.FnSlug)
		require.Empty(t, got.Input, "Input released on cancel; retransmit must not re-attach it")
	})

	t.Run("rejects new hashed IDs once per-run count limit reached", func(t *testing.T) {
		r := require.New(t)
		v2svc, id := newDeferTestRunService(t)

		for i := 0; i < consts.MaxDefersPerRun; i++ {
			r.NoError(v2svc.SaveDefer(ctx, id, statev2.Defer{
				FnSlug:         "onDefer-score",
				HashedID:       fmt.Sprintf("hash-step-%d", i),
				ScheduleStatus: enums.DeferStatusAfterRun,
				Input:          json.RawMessage(`{}`),
			}))
		}

		// Reject since it'd exceed the limit
		err := v2svc.SaveDefer(ctx, id, statev2.Defer{
			FnSlug:         "onDefer-score",
			HashedID:       "hash-step-overflow",
			ScheduleStatus: enums.DeferStatusAfterRun,
			Input:          json.RawMessage(`{}`),
		})
		r.ErrorIs(err, statev2.ErrDeferLimitExceeded)

		// Idempotency for existing hashed ID
		r.NoError(v2svc.SaveDefer(ctx, id, statev2.Defer{
			FnSlug:         "onDefer-score",
			HashedID:       "hash-step-0",
			ScheduleStatus: enums.DeferStatusAfterRun,
			Input:          json.RawMessage(`{"updated":true}`),
		}))

		defers, err := v2svc.LoadDefers(ctx, id)
		r.NoError(err)
		r.Len(defers, consts.MaxDefersPerRun)
		r.JSONEq(`{}`, string(defers["hash-step-0"].Input), "first writer wins")
		_, ok := defers["hash-step-overflow"]
		r.False(ok)
	})

	t.Run("aggregate cap writes rejected sentinel", func(t *testing.T) {
		r := require.New(t)
		v2svc, id := newDeferTestRunService(t)

		// 3MB accepted, then a 2MB defer overflows the 4MB cap.
		bigInput := make([]byte, 3*1024*1024)
		for i := range bigInput {
			bigInput[i] = 'x'
		}
		r.NoError(v2svc.SaveDefer(ctx, id, statev2.Defer{
			FnSlug:         "onDefer-score",
			HashedID:       "hash-accepted",
			ScheduleStatus: enums.DeferStatusAfterRun,
			Input:          json.RawMessage(`"` + string(bigInput) + `"`),
		}))

		// Reject since it'd exceed the aggregate cap
		overflowInput := make([]byte, 2*1024*1024)
		for i := range overflowInput {
			overflowInput[i] = 'y'
		}
		err := v2svc.SaveDefer(ctx, id, statev2.Defer{
			FnSlug:         "onDefer-score",
			HashedID:       "hash-rejected",
			ScheduleStatus: enums.DeferStatusAfterRun,
			Input:          json.RawMessage(`"` + string(overflowInput) + `"`),
		})
		r.ErrorIs(err, statev2.ErrDeferInputAggregateExceeded)

		defers, err := v2svc.LoadDefers(ctx, id)
		r.NoError(err)
		r.Len(defers, 2)

		rejected, ok := defers["hash-rejected"]
		r.True(ok)
		r.Equal(enums.DeferStatusRejected, rejected.ScheduleStatus)
		r.Empty(rejected.Input)
		r.Equal("onDefer-score", rejected.FnSlug)
		r.Equal(enums.DeferStatusAfterRun, defers["hash-accepted"].ScheduleStatus)
	})

	t.Run("rejected is sticky across retransmits", func(t *testing.T) {
		r := require.New(t)
		v2svc, id := newDeferTestRunService(t)

		// 5MB defer overflows on the first write.
		overflow := make([]byte, 5*1024*1024)
		for i := range overflow {
			overflow[i] = 'z'
		}
		op := statev2.Defer{
			FnSlug:         "onDefer-score",
			HashedID:       "hash-rejected",
			ScheduleStatus: enums.DeferStatusAfterRun,
			Input:          json.RawMessage(`"` + string(overflow) + `"`),
		}
		r.ErrorIs(v2svc.SaveDefer(ctx, id, op), statev2.ErrDeferInputAggregateExceeded)

		// Retransmit hits the terminal-sticky check and no-ops.
		r.NoError(v2svc.SaveDefer(ctx, id, op))

		defers, err := v2svc.LoadDefers(ctx, id)
		r.NoError(err)
		r.Len(defers, 1)
		r.Equal(enums.DeferStatusRejected, defers["hash-rejected"].ScheduleStatus)
		r.Empty(defers["hash-rejected"].Input)
	})

	t.Run("retransmit does not bump aggregate counter", func(t *testing.T) {
		r := require.New(t)
		v2svc, id := newDeferTestRunService(t)

		// 3MB once, then retransmit twice. Each retransmit is a no-op
		// (insert-only) so the budget stays at the original 3MB+quotes.
		big := make([]byte, 3*1024*1024)
		for i := range big {
			big[i] = 'a'
		}
		first := statev2.Defer{
			FnSlug:         "onDefer-score",
			HashedID:       "hash-first",
			ScheduleStatus: enums.DeferStatusAfterRun,
			Input:          json.RawMessage(`"` + string(big) + `"`),
		}
		r.NoError(v2svc.SaveDefer(ctx, id, first))
		r.NoError(v2svc.SaveDefer(ctx, id, first))
		r.NoError(v2svc.SaveDefer(ctx, id, first))

		// Second defer fills the remaining budget. First Input on the wire
		// is `"<3MB>"` = 3MB+2 bytes; remaining is 1MB-2 bytes total, minus
		// 2 quote bytes for the second JSON wrapper = 1MB-4 inner. Would
		// fail if any retransmit had bumped the counter.
		small := make([]byte, 1024*1024-4)
		for i := range small {
			small[i] = 'b'
		}
		second := statev2.Defer{
			FnSlug:         "onDefer-score",
			HashedID:       "hash-second",
			ScheduleStatus: enums.DeferStatusAfterRun,
			Input:          json.RawMessage(`"` + string(small) + `"`),
		}
		r.NoError(v2svc.SaveDefer(ctx, id, second))

		defers, err := v2svc.LoadDefers(ctx, id)
		r.NoError(err)
		r.Len(defers, 2)
		r.Equal(enums.DeferStatusAfterRun, defers["hash-first"].ScheduleStatus)
		r.Equal(enums.DeferStatusAfterRun, defers["hash-second"].ScheduleStatus)
	})

	t.Run("does not affect run state_size", func(t *testing.T) {
		// CRITICAL: Defer input cannot affect the run state size budget, since
		// doing so would mean defers could fail the run.

		r := require.New(t)
		v2svc, id := newDeferTestRunService(t)

		mdBefore, err := v2svc.LoadMetadata(ctx, id)
		r.NoError(err)

		r.NoError(v2svc.SaveDefer(ctx, id, statev2.Defer{
			FnSlug:         "onDefer-score",
			HashedID:       "hash-step-1",
			ScheduleStatus: enums.DeferStatusAfterRun,
			Input:          json.RawMessage(`{"user_id":"u_123"}`),
		}))

		mdAfter, err := v2svc.LoadMetadata(ctx, id)
		r.NoError(err)
		r.Equal(mdBefore.Metrics.StateSize, mdAfter.Metrics.StateSize,
			"defer storage is on a separate budget from run state")
	})
}

func TestSetDeferStatus(t *testing.T) {
	ctx := context.Background()

	t.Run("errors on missing hashedID", func(t *testing.T) {
		v2svc, id := newDeferTestRunService(t)
		err := v2svc.SetDeferStatus(ctx, id, "missing-hashed-id", enums.DeferStatusAborted)
		require.Error(t, err)
	})

	t.Run("preserves meta fields across abort", func(t *testing.T) {
		r := require.New(t)
		v2svc, id := newDeferTestRunService(t)

		original := statev2.Defer{
			FnSlug:         "onDefer-score",
			HashedID:       "hash-step-1",
			ScheduleStatus: enums.DeferStatusAfterRun,
			Input:          json.RawMessage(`{"user":{"id":"u_123"}}`),
		}
		r.NoError(v2svc.SaveDefer(ctx, id, original))
		r.NoError(v2svc.SetDeferStatus(ctx, id, original.HashedID, enums.DeferStatusAborted))

		defers, err := v2svc.LoadDefers(ctx, id)
		r.NoError(err)
		r.Len(defers, 1)

		got := defers[original.HashedID]
		r.Equal(original.FnSlug, got.FnSlug)
		r.Equal(original.HashedID, got.HashedID)
		r.Equal(enums.DeferStatusAborted, got.ScheduleStatus)
		r.Empty(got.Input)
	})

	t.Run("meta survives cjson edge-case inputs", func(t *testing.T) {
		cases := []struct {
			name  string
			input string
		}{
			{"empty object", `{}`},
			{"nested empty object", `{"user_id":"u_123","options":{}}`},
			// 2^53 + 1 — first integer that can't be represented exactly
			// as a float64. cjson rounds it down to 2^53 if it ever decodes.
			{"integer above 2^53", `{"external_id":9007199254740993,"requested_at_ns":1735689600000000000}`},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				r := require.New(t)
				v2svc, id := newDeferTestRunService(t)

				d := statev2.Defer{
					FnSlug:         "onDefer-score",
					HashedID:       "hash-step-1",
					ScheduleStatus: enums.DeferStatusAfterRun,
					Input:          json.RawMessage(tc.input),
				}
				r.NoError(v2svc.SaveDefer(ctx, id, d))
				r.NoError(v2svc.SetDeferStatus(ctx, id, d.HashedID, enums.DeferStatusAborted))

				defers, err := v2svc.LoadDefers(ctx, id)
				r.NoError(err)
				got := defers[d.HashedID]
				r.Equal(d.FnSlug, got.FnSlug)
				r.Equal(d.HashedID, got.HashedID)
				r.Equal(enums.DeferStatusAborted, got.ScheduleStatus)
				r.Empty(got.Input)
			})
		}
	})

	t.Run("abort releases input budget", func(t *testing.T) {
		r := require.New(t)
		v2svc, id := newDeferTestRunService(t)

		big := make([]byte, 3*1024*1024)
		for i := range big {
			big[i] = 'q'
		}
		first := statev2.Defer{
			FnSlug:         "onDefer-score",
			HashedID:       "hash-aborted",
			ScheduleStatus: enums.DeferStatusAfterRun,
			Input:          json.RawMessage(`"` + string(big) + `"`),
		}
		r.NoError(v2svc.SaveDefer(ctx, id, first))

		// Pre-abort: a 2MB second defer overflows the budget.
		mid := make([]byte, 2*1024*1024)
		for i := range mid {
			mid[i] = 'r'
		}
		second := statev2.Defer{
			FnSlug:         "onDefer-score",
			HashedID:       "hash-second",
			ScheduleStatus: enums.DeferStatusAfterRun,
			Input:          json.RawMessage(`"` + string(mid) + `"`),
		}
		r.ErrorIs(v2svc.SaveDefer(ctx, id, second), statev2.ErrDeferInputAggregateExceeded)

		// Abort the first defer; its 3MB are released.
		r.NoError(v2svc.SetDeferStatus(ctx, id, first.HashedID, enums.DeferStatusAborted))

		// Same 2MB defer now fits. Use a fresh hashedID since the previous
		// attempt left a Rejected sentinel that blocks retransmits.
		second.HashedID = "hash-second-retry"
		r.NoError(v2svc.SaveDefer(ctx, id, second))

		defers, err := v2svc.LoadDefers(ctx, id)
		r.NoError(err)
		got := defers["hash-aborted"]
		r.Equal(enums.DeferStatusAborted, got.ScheduleStatus)
		r.Empty(got.Input)
	})
}

func TestSaveRejectedDefer(t *testing.T) {
	ctx := context.Background()

	t.Run("idempotent against existing entry", func(t *testing.T) {
		r := require.New(t)
		v2svc, id := newDeferTestRunService(t)

		original := statev2.Defer{
			FnSlug:         "onDefer-score",
			HashedID:       "hash-step-1",
			ScheduleStatus: enums.DeferStatusAfterRun,
			Input:          json.RawMessage(`{"user_id":"u_123"}`),
		}
		r.NoError(v2svc.SaveDefer(ctx, id, original))

		// Late rejection signal must not downgrade the accepted defer.
		r.NoError(v2svc.SaveRejectedDefer(ctx, id, original.FnSlug, original.HashedID))

		defers, err := v2svc.LoadDefers(ctx, id)
		r.NoError(err)
		r.Len(defers, 1)
		r.Equal(enums.DeferStatusAfterRun, defers[original.HashedID].ScheduleStatus)
		r.JSONEq(string(original.Input), string(defers[original.HashedID].Input))
	})

	t.Run("writes new sentinel and blocks retransmits", func(t *testing.T) {
		r := require.New(t)
		v2svc, id := newDeferTestRunService(t)

		r.NoError(v2svc.SaveRejectedDefer(ctx, id, "onDefer-score", "hash-rejected"))

		defers, err := v2svc.LoadDefers(ctx, id)
		r.NoError(err)
		r.Len(defers, 1)
		got := defers["hash-rejected"]
		r.Equal(enums.DeferStatusRejected, got.ScheduleStatus)
		r.Equal("onDefer-score", got.FnSlug)
		r.Empty(got.Input)

		// Subsequent SaveDefer hits the terminal-sticky check.
		r.NoError(v2svc.SaveDefer(ctx, id, statev2.Defer{
			FnSlug:         "onDefer-score",
			HashedID:       "hash-rejected",
			ScheduleStatus: enums.DeferStatusAfterRun,
			Input:          json.RawMessage(`{"x":1}`),
		}))

		defers, err = v2svc.LoadDefers(ctx, id)
		r.NoError(err)
		r.Equal(enums.DeferStatusRejected, defers["hash-rejected"].ScheduleStatus)
		r.Empty(defers["hash-rejected"].Input)
	})
}
