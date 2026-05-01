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

// A retried SaveDefer must not undo an interleaved SetDeferStatus(Cancelled).
func TestSaveDefer_DoesNotResurrectCancelled(t *testing.T) {
	ctx := context.Background()
	v2svc, id := newDeferTestRunService(t)

	original := statev2.Defer{
		FnSlug:         "onDefer-score",
		HashedID:       "hash-step-1",
		ScheduleStatus: enums.DeferStatusAfterRun,
		Input:          json.RawMessage(`{"user_id":"u_123"}`),
	}

	require.NoError(t, v2svc.SaveDefer(ctx, id, original))
	require.NoError(t, v2svc.SetDeferStatus(ctx, id, original.HashedID, enums.DeferStatusCancelled))

	// T3: SDK retransmits the original DeferAdd. Must not error, must not resurrect.
	require.NoError(t, v2svc.SaveDefer(ctx, id, original))

	defers, err := v2svc.LoadDefers(ctx, id)
	require.NoError(t, err)
	require.Len(t, defers, 1)

	got := defers[original.HashedID]
	require.Equal(t, enums.DeferStatusCancelled, got.ScheduleStatus,
		"cancelled defer must not be resurrected by a retried DeferAdd")
	require.Equal(t, original.FnSlug, got.FnSlug, "FnSlug must be preserved across the no-op")
	require.JSONEq(t, string(original.Input), string(got.Input),
		"Input must be preserved across the no-op")
}

// Non-cancelled records get overwritten — the legitimate SDK retransmit path.
func TestSaveDefer_OverwritesAfterRun(t *testing.T) {
	ctx := context.Background()
	v2svc, id := newDeferTestRunService(t)

	first := statev2.Defer{
		FnSlug:         "onDefer-score",
		HashedID:       "hash-step-1",
		ScheduleStatus: enums.DeferStatusAfterRun,
		Input:          json.RawMessage(`{"user_id":"u_old"}`),
	}
	require.NoError(t, v2svc.SaveDefer(ctx, id, first))

	second := first
	second.Input = json.RawMessage(`{"user_id":"u_new"}`)
	require.NoError(t, v2svc.SaveDefer(ctx, id, second))

	defers, err := v2svc.LoadDefers(ctx, id)
	require.NoError(t, err)
	require.JSONEq(t, `{"user_id":"u_new"}`, string(defers[first.HashedID].Input),
		"non-cancelled defer should be overwritten by a subsequent SaveDefer")
}

// SetDeferStatus must reject hashedIDs with no existing defer; otherwise a
// stray DeferCancel could silently no-op against a missing target.
func TestSetDeferStatus_ErrorsOnMissing(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()
	v2svc, id := newDeferTestRunService(t)

	err := v2svc.SetDeferStatus(ctx, id, "missing-hashed-id", enums.DeferStatusCancelled)
	r.Error(err, "expected SetDeferStatus to error when defer is missing")
}

// SetDeferStatus must update only ScheduleStatus; FnSlug, HashedID, and Input
// all need to survive the status flip unchanged.
func TestSetDeferStatus_PreservesFields(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()
	v2svc, id := newDeferTestRunService(t)

	original := statev2.Defer{
		FnSlug:         "onDefer-score",
		HashedID:       "hash-step-1",
		ScheduleStatus: enums.DeferStatusAfterRun,
		Input:          json.RawMessage(`{"user":{"id":"u_123"}}`),
	}
	r.NoError(v2svc.SaveDefer(ctx, id, original))
	r.NoError(v2svc.SetDeferStatus(ctx, id, original.HashedID, enums.DeferStatusCancelled))

	defers, err := v2svc.LoadDefers(ctx, id)
	r.NoError(err)
	r.Len(defers, 1)

	expected := original
	expected.ScheduleStatus = enums.DeferStatusCancelled
	r.Equal(expected, defers[original.HashedID])
}

// Ensure that updating the defer status doesn't corrupt the input
func TestSetDeferStatus_InputRoundTripsByteForByte(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"empty object", `{}`},
		{"nested empty object", `{"user_id":"u_123","options":{}}`},
		// 9007199254740993 is 2^53 + 1, the first integer that can't be
		// represented exactly as a float64. cjson would round it down to 2^53.
		{"integer above 2^53", `{"external_id":9007199254740993,"requested_at_ns":1735689600000000000}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := require.New(t)
			ctx := context.Background()
			v2svc, id := newDeferTestRunService(t)

			d := statev2.Defer{
				FnSlug:         "onDefer-score",
				HashedID:       "hash-step-1",
				ScheduleStatus: enums.DeferStatusAfterRun,
				Input:          json.RawMessage(tc.input),
			}
			r.NoError(v2svc.SaveDefer(ctx, id, d))
			r.NoError(v2svc.SetDeferStatus(ctx, id, d.HashedID, enums.DeferStatusCancelled))

			defers, err := v2svc.LoadDefers(ctx, id)
			r.NoError(err)
			r.Equal(tc.input, string(defers[d.HashedID].Input))
		})
	}
}

// SaveDefer must reject new hashedIDs once the per-run limit is reached, but
// re-saves of an existing hashedID (legitimate SDK retransmits) must still go
// through.
func TestSaveDefer_LimitPerRun(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()
	v2svc, id := newDeferTestRunService(t)

	for i := 0; i < consts.MaxDefersPerRun; i++ {
		r.NoError(v2svc.SaveDefer(ctx, id, statev2.Defer{
			FnSlug:         "onDefer-score",
			HashedID:       fmt.Sprintf("hash-step-%d", i),
			ScheduleStatus: enums.DeferStatusAfterRun,
			Input:          json.RawMessage(`{}`),
		}))
	}

	// One past the limit: reject.
	err := v2svc.SaveDefer(ctx, id, statev2.Defer{
		FnSlug:         "onDefer-score",
		HashedID:       "hash-step-overflow",
		ScheduleStatus: enums.DeferStatusAfterRun,
		Input:          json.RawMessage(`{}`),
	})
	r.ErrorIs(err, statev2.ErrDeferLimitExceeded)

	// Update of an existing hashedID must still succeed at the cap.
	r.NoError(v2svc.SaveDefer(ctx, id, statev2.Defer{
		FnSlug:         "onDefer-score",
		HashedID:       "hash-step-0",
		ScheduleStatus: enums.DeferStatusAfterRun,
		Input:          json.RawMessage(`{"updated":true}`),
	}))

	defers, err := v2svc.LoadDefers(ctx, id)
	r.NoError(err)
	r.Len(defers, consts.MaxDefersPerRun)
	r.JSONEq(`{"updated":true}`, string(defers["hash-step-0"].Input))
	_, ok := defers["hash-step-overflow"]
	r.False(ok, "rejected defer must not appear in LoadDefers")
}

func TestSaveDefer_FirstWriteWhenMissing(t *testing.T) {
	ctx := context.Background()
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
}
