package pauses

import (
	"context"
	"crypto/rand"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

// A run that ends or is cancelled with an open step.subscribe subscription
// leaves nothing behind: finalizing the run deletes the pause of its subscription.
func TestFinalizingARunRemovesItsSubscriptionPauses(t *testing.T) {
	for name, remove := range map[string]func(context.Context, Manager, *redis_state.PauseStore, ulid.ULID, uuid.UUID) error{
		"through the pause manager": func(ctx context.Context, m Manager, _ *redis_state.PauseStore, run ulid.ULID, ws uuid.UUID) error {
			return m.DeletePausesForRun(ctx, run, ws)
		},
		"through the pause store the run state calls": func(ctx context.Context, _ Manager, s *redis_state.PauseStore, run ulid.ULID, ws uuid.UUID) error {
			return s.DeletePausesForRun(ctx, run, ws)
		},
	} {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			r := miniredis.RunT(t)
			rc, err := rueidis.NewClient(rueidis.ClientOption{
				InitAddress:  []string{r.Addr()},
				DisableCache: true,
			})
			require.NoError(t, err)
			defer rc.Close()

			store := redis_state.NewPauseStore(redis_state.NewUnshardedClient(rc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey))
			m := NewManager(StateBufferer(store), nil)

			ws := uuid.New()
			run := ulid.MustNew(ulid.Now(), rand.Reader)
			identifier := state.PauseIdentifier{RunID: run, FunctionID: uuid.New(), AccountID: uuid.New()}

			messageEvent := "agent/message"
			subscription := state.Pause{
				ID:          uuid.New(),
				WorkspaceID: ws,
				Expires:     state.Time(time.Now().Add(time.Hour)),
				CreatedAt:   time.Now(),
				Identifier:  identifier,
				Event:       &messageEvent,
				Subscribe:   &state.PauseSubscribe{Key: "inbox"},
			}
			subscriptionIndex := Index{WorkspaceID: ws, EventName: messageEvent}

			_, err = m.Write(ctx, subscriptionIndex, &subscription)
			require.NoError(t, err)

			ids, err := m.PauseIDsForRun(ctx, run)
			require.NoError(t, err)
			require.ElementsMatch(t, []uuid.UUID{subscription.ID}, ids)

			require.NoError(t, remove(ctx, m, store, run, ws))

			_, err = m.PauseByID(ctx, subscriptionIndex, subscription.ID)
			require.ErrorIs(t, err, state.ErrPauseNotFound)

			ids, err = m.PauseIDsForRun(ctx, run)
			require.NoError(t, err)
			require.Empty(t, ids)

		})
	}
}
