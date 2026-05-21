package redis_state

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestScavengeStaleRuns_ZeroOutstanding(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	clock := clockwork.NewFakeClock()
	_, shard := newQueue(t, rc, osqueue.WithClock(clock))
	ctx := context.Background()

	accountID := uuid.New()
	wsID := uuid.New()
	fnID := uuid.New()
	appID := uuid.New()

	// Run started 10 minutes ago → older than StaleRunThreshold (5 min).
	startTime := clock.Now().Add(-10 * time.Minute)
	runID := ulid.MustNew(uint64(startTime.UnixMilli()), rand.Reader)

	info := osqueue.StaleRunInfo{
		RunID:       runID,
		FunctionID:  fnID,
		AccountID:   accountID,
		WorkspaceID: wsID,
		AppID:       appID,
	}
	err = shard.(osqueue.StaleRunScavenger).TrackActiveRun(ctx, info, startTime)
	require.NoError(t, err)

	// No queue items → should be detected as stale.
	stale, err := shard.(osqueue.StaleRunScavenger).ScavengeStaleRuns(ctx, consts.StaleRunThreshold)
	require.NoError(t, err)
	require.Len(t, stale, 1)
	require.Equal(t, runID, stale[0].RunID)
}

func TestScavengeStaleRuns_NonInvokeOutstanding(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	clock := clockwork.NewFakeClock()
	_, shard := newQueue(t, rc, osqueue.WithClock(clock))
	ctx := context.Background()

	accountID := uuid.New()
	wsID := uuid.New()
	fnID := uuid.New()
	appID := uuid.New()

	// Run started 20 minutes ago.
	startTime := clock.Now().Add(-20 * time.Minute)
	runID := ulid.MustNew(uint64(startTime.UnixMilli()), rand.Reader)

	info := osqueue.StaleRunInfo{
		RunID:       runID,
		FunctionID:  fnID,
		AccountID:   accountID,
		WorkspaceID: wsID,
		AppID:       appID,
	}
	err = shard.(osqueue.StaleRunScavenger).TrackActiveRun(ctx, info, startTime)
	require.NoError(t, err)

	// Enqueue a non-pause (KindEdge) item so the run has outstanding items.
	qi := osqueue.QueueItem{
		FunctionID:  fnID,
		WorkspaceID: wsID,
		Data: osqueue.Item{
			Kind: osqueue.KindEdge,
			Identifier: state.Identifier{
				RunID:       runID,
				AccountID:   accountID,
				WorkspaceID: wsID,
				WorkflowID:  fnID,
				AppID:       appID,
			},
		},
	}
	_, err = shard.EnqueueItem(ctx, qi, clock.Now(), osqueue.EnqueueOpts{})
	require.NoError(t, err)

	// Run has non-invoke outstanding items → should NOT be detected as stale.
	stale, err := shard.(osqueue.StaleRunScavenger).ScavengeStaleRuns(ctx, consts.StaleRunThreshold)
	require.NoError(t, err)
	require.Len(t, stale, 0)
}

func TestScavengeStaleRuns_StuckInvokeTimeout(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	clock := clockwork.NewFakeClock()
	_, shard := newQueue(t, rc, osqueue.WithClock(clock))
	ctx := context.Background()

	accountID := uuid.New()
	wsID := uuid.New()
	fnID := uuid.New()
	appID := uuid.New()

	// Run started 2 hours ago → older than StaleInvokeRecoveryThreshold (1 hour).
	startTime := clock.Now().Add(-2 * time.Hour)
	runID := ulid.MustNew(uint64(startTime.UnixMilli()), rand.Reader)

	info := osqueue.StaleRunInfo{
		RunID:       runID,
		FunctionID:  fnID,
		AccountID:   accountID,
		WorkspaceID: wsID,
		AppID:       appID,
	}
	err = shard.(osqueue.StaleRunScavenger).TrackActiveRun(ctx, info, startTime)
	require.NoError(t, err)

	// Enqueue a KindPause item with an InvokeCorrelationID to simulate an
	// invoke timeout job whose function.finished event was lost.
	correlationID := fmt.Sprintf("%s.some-step-hash", runID.String())
	pauseID := uuid.New()
	pausePayload := osqueue.PayloadPauseTimeout{
		PauseID: pauseID,
		Pause: state.Pause{
			ID:                  pauseID,
			WorkspaceID:         wsID,
			InvokeCorrelationID: &correlationID,
			Expires:             state.Time(clock.Now().Add(365 * 24 * time.Hour)), // 1-year timeout
		},
	}
	payloadBytes, err := json.Marshal(pausePayload)
	require.NoError(t, err)

	qi := osqueue.QueueItem{
		FunctionID:  fnID,
		WorkspaceID: wsID,
		Data: osqueue.Item{
			Kind:    osqueue.KindPause,
			Payload: json.RawMessage(payloadBytes),
			Identifier: state.Identifier{
				RunID:       runID,
				AccountID:   accountID,
				WorkspaceID: wsID,
				WorkflowID:  fnID,
				AppID:       appID,
			},
		},
	}
	_, err = shard.EnqueueItem(ctx, qi, clock.Now().Add(365*24*time.Hour), osqueue.EnqueueOpts{})
	require.NoError(t, err)

	// Run has only invoke timeout items and is older than StaleInvokeRecoveryThreshold
	// → should be detected as stale.
	stale, err := shard.(osqueue.StaleRunScavenger).ScavengeStaleRuns(ctx, consts.StaleRunThreshold)
	require.NoError(t, err)
	require.Len(t, stale, 1)
	require.Equal(t, runID, stale[0].RunID)
}

func TestScavengeStaleRuns_InvokeTimeoutTooYoung(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	clock := clockwork.NewFakeClock()
	_, shard := newQueue(t, rc, osqueue.WithClock(clock))
	ctx := context.Background()

	accountID := uuid.New()
	wsID := uuid.New()
	fnID := uuid.New()
	appID := uuid.New()

	// Run started 30 minutes ago — older than StaleRunThreshold (5 min)
	// but younger than StaleInvokeRecoveryThreshold (1 hour).
	startTime := clock.Now().Add(-30 * time.Minute)
	runID := ulid.MustNew(uint64(startTime.UnixMilli()), rand.Reader)

	info := osqueue.StaleRunInfo{
		RunID:       runID,
		FunctionID:  fnID,
		AccountID:   accountID,
		WorkspaceID: wsID,
		AppID:       appID,
	}
	err = shard.(osqueue.StaleRunScavenger).TrackActiveRun(ctx, info, startTime)
	require.NoError(t, err)

	// Enqueue invoke timeout job.
	correlationID := fmt.Sprintf("%s.some-step-hash", runID.String())
	pauseID := uuid.New()
	pausePayload := osqueue.PayloadPauseTimeout{
		PauseID: pauseID,
		Pause: state.Pause{
			ID:                  pauseID,
			WorkspaceID:         wsID,
			InvokeCorrelationID: &correlationID,
		},
	}
	payloadBytes, err := json.Marshal(pausePayload)
	require.NoError(t, err)

	qi := osqueue.QueueItem{
		FunctionID:  fnID,
		WorkspaceID: wsID,
		Data: osqueue.Item{
			Kind:    osqueue.KindPause,
			Payload: json.RawMessage(payloadBytes),
			Identifier: state.Identifier{
				RunID:       runID,
				AccountID:   accountID,
				WorkspaceID: wsID,
				WorkflowID:  fnID,
				AppID:       appID,
			},
		},
	}
	_, err = shard.EnqueueItem(ctx, qi, clock.Now().Add(365*24*time.Hour), osqueue.EnqueueOpts{})
	require.NoError(t, err)

	// Run is younger than StaleInvokeRecoveryThreshold → should NOT be stale.
	stale, err := shard.(osqueue.StaleRunScavenger).ScavengeStaleRuns(ctx, consts.StaleRunThreshold)
	require.NoError(t, err)
	require.Len(t, stale, 0)
}

func TestScavengeStaleRuns_MixedItems(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	clock := clockwork.NewFakeClock()
	_, shard := newQueue(t, rc, osqueue.WithClock(clock))
	ctx := context.Background()

	accountID := uuid.New()
	wsID := uuid.New()
	fnID := uuid.New()
	appID := uuid.New()

	// Run started 2 hours ago.
	startTime := clock.Now().Add(-2 * time.Hour)
	runID := ulid.MustNew(uint64(startTime.UnixMilli()), rand.Reader)

	info := osqueue.StaleRunInfo{
		RunID:       runID,
		FunctionID:  fnID,
		AccountID:   accountID,
		WorkspaceID: wsID,
		AppID:       appID,
	}
	err = shard.(osqueue.StaleRunScavenger).TrackActiveRun(ctx, info, startTime)
	require.NoError(t, err)

	ident := state.Identifier{
		RunID:       runID,
		AccountID:   accountID,
		WorkspaceID: wsID,
		WorkflowID:  fnID,
		AppID:       appID,
	}

	// Enqueue an invoke timeout job.
	correlationID := fmt.Sprintf("%s.step-hash", runID.String())
	pauseID := uuid.New()
	pausePayload := osqueue.PayloadPauseTimeout{
		PauseID: pauseID,
		Pause: state.Pause{
			ID:                  pauseID,
			WorkspaceID:         wsID,
			InvokeCorrelationID: &correlationID,
		},
	}
	payloadBytes, err := json.Marshal(pausePayload)
	require.NoError(t, err)

	_, err = shard.EnqueueItem(ctx, osqueue.QueueItem{
		FunctionID:  fnID,
		WorkspaceID: wsID,
		Data: osqueue.Item{
			Kind:       osqueue.KindPause,
			Payload:    json.RawMessage(payloadBytes),
			Identifier: ident,
		},
	}, clock.Now().Add(365*24*time.Hour), osqueue.EnqueueOpts{})
	require.NoError(t, err)

	// Also enqueue a non-invoke item (KindEdge).
	_, err = shard.EnqueueItem(ctx, osqueue.QueueItem{
		FunctionID:  fnID,
		WorkspaceID: wsID,
		Data: osqueue.Item{
			Kind:       osqueue.KindEdge,
			Identifier: ident,
		},
	}, clock.Now(), osqueue.EnqueueOpts{})
	require.NoError(t, err)

	// Mixed items (invoke + non-invoke) → should NOT be detected as stale.
	stale, err := shard.(osqueue.StaleRunScavenger).ScavengeStaleRuns(ctx, consts.StaleRunThreshold)
	require.NoError(t, err)
	require.Len(t, stale, 0)
}

func TestScavengeStaleRuns_WaitForEventTimeout(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	clock := clockwork.NewFakeClock()
	_, shard := newQueue(t, rc, osqueue.WithClock(clock))
	ctx := context.Background()

	accountID := uuid.New()
	wsID := uuid.New()
	fnID := uuid.New()
	appID := uuid.New()

	// Run started 2 hours ago.
	startTime := clock.Now().Add(-2 * time.Hour)
	runID := ulid.MustNew(uint64(startTime.UnixMilli()), rand.Reader)

	info := osqueue.StaleRunInfo{
		RunID:       runID,
		FunctionID:  fnID,
		AccountID:   accountID,
		WorkspaceID: wsID,
		AppID:       appID,
	}
	err = shard.(osqueue.StaleRunScavenger).TrackActiveRun(ctx, info, startTime)
	require.NoError(t, err)

	// Enqueue a KindPause item WITHOUT InvokeCorrelationID (waitForEvent timeout).
	pauseID := uuid.New()
	eventName := "app/some.event"
	pausePayload := osqueue.PayloadPauseTimeout{
		PauseID: pauseID,
		Pause: state.Pause{
			ID:          pauseID,
			WorkspaceID: wsID,
			Event:       &eventName,
			// No InvokeCorrelationID — this is a waitForEvent, not an invoke.
		},
	}
	payloadBytes, err := json.Marshal(pausePayload)
	require.NoError(t, err)

	qi := osqueue.QueueItem{
		FunctionID:  fnID,
		WorkspaceID: wsID,
		Data: osqueue.Item{
			Kind:    osqueue.KindPause,
			Payload: json.RawMessage(payloadBytes),
			Identifier: state.Identifier{
				RunID:       runID,
				AccountID:   accountID,
				WorkspaceID: wsID,
				WorkflowID:  fnID,
				AppID:       appID,
			},
		},
	}
	_, err = shard.EnqueueItem(ctx, qi, clock.Now().Add(1*time.Hour), osqueue.EnqueueOpts{})
	require.NoError(t, err)

	// KindPause without InvokeCorrelationID → should NOT be detected as stale.
	stale, err := shard.(osqueue.StaleRunScavenger).ScavengeStaleRuns(ctx, consts.StaleRunThreshold)
	require.NoError(t, err)
	require.Len(t, stale, 0)
}
