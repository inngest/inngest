package executor

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestSubscriptionStepIDs(t *testing.T) {
	sum := sha1.Sum([]byte("inbox:3"))
	require.Equal(t, hex.EncodeToString(sum[:]), subscriptionStepID("inbox", 3))

	ack := sha1.Sum([]byte("inbox:ack:3"))
	require.Equal(t, hex.EncodeToString(ack[:]), subscriptionAckID("inbox", 3))

	require.NotEqual(t, subscriptionStepID("inbox", 1), subscriptionStepID("inbox", 2))
	require.NotEqual(t, subscriptionStepID("a", 1), subscriptionStepID("b", 1))
}

func TestFinalizeEventSeed(t *testing.T) {
	evt := event.Event{ID: "msg-1", Name: "agent/message", Timestamp: 1700000000000}

	first, err := finalizeEventSeed(evt).ToULID()
	require.NoError(t, err)
	second, err := finalizeEventSeed(evt).ToULID()
	require.NoError(t, err)
	require.Equal(t, first, second, "the same event ID gives the same internal ID")

	evt.ID = "msg-2"
	other, err := finalizeEventSeed(evt).ToULID()
	require.NoError(t, err)
	require.NotEqual(t, first, other)

	require.Nil(t, finalizeEventSeed(event.Event{ID: "x", Name: "inngest/function.finished", Timestamp: 1}))
	require.Nil(t, finalizeEventSeed(event.Event{Name: "agent/message", Timestamp: 1}))
}

func TestWaiterParallelMode(t *testing.T) {
	for _, mode := range []enums.ParallelMode{enums.ParallelModeNone, enums.ParallelModeWait, enums.ParallelModeRace} {
		require.Equal(t, mode, waiterParallelMode([]byte(fmt.Sprintf(`{"data":%d}`, int(mode)))))
	}
	require.Equal(t, enums.ParallelModeNone, waiterParallelMode([]byte(`not json`)))
}

// A subscription with several streams has one wait per stream pending at the
// executor. A message delivered to one stream wakes the run only when the wait
// was recorded as a race; otherwise the other pending wait would hold the
// discovery back and the run would sleep until some other event woke it.
func TestResumeSubscriberWakesRaceWaitWhileOthersPending(t *testing.T) {
	cases := []struct {
		name    string
		mode    enums.ParallelMode
		enqueue int
	}{
		{name: "race wakes the run", mode: enums.ParallelModeRace, enqueue: 1},
		{name: "a plain wait stays asleep while another is pending", mode: enums.ParallelModeNone, enqueue: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runID := ulid.MustNew(ulid.Now(), nil)
			wsID, fnID, aID := uuid.New(), uuid.New(), uuid.New()

			md := sv2.Metadata{
				ID: sv2.ID{
					RunID:      runID,
					FunctionID: fnID,
					Tenant:     sv2.Tenant{EnvID: wsID, AccountID: aID},
				},
				Config: *sv2.InitConfig(&sv2.Config{}),
			}

			q := &stubQueue{}
			e := &executor{
				smv2: &stubRunServiceMD{md: md},
				pm: &stubPauseMgr{consumeResult: state.ConsumePauseResult{
					DidConsume:      true,
					HasPendingSteps: true,
				}},
				queue:          q,
				log:            logger.From(context.Background()),
				tracerProvider: tracing.NewNoopTracerProvider(),
			}

			event := "agent/message"
			pause := state.Pause{
				ID:          uuid.New(),
				WorkspaceID: wsID,
				Event:       &event,
				Subscribe:   &state.PauseSubscribe{Key: "inbox:1"},
				Identifier: state.PauseIdentifier{
					RunID:      runID,
					FunctionID: fnID,
					AccountID:  aID,
				},
			}

			err := e.resumeSubscriber(context.Background(), &pause, subscriptionStepID("inbox:1", 1), []byte(`{"data":{"name":"x"}}`), tc.mode)
			require.NoError(t, err)
			require.Len(t, q.enqueued, tc.enqueue)
		})
	}
}
