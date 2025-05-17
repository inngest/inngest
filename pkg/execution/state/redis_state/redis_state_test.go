package redis_state

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/testharness"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestNewRunMetadata(t *testing.T) {
	byt, _ := json.Marshal(state.Identifier{})

	tests := []struct {
		name          string
		data          map[string]string
		expectedVal   *runMetadata
		expectedError error
	}{
		{
			name: "should return value if data is valid",
			data: map[string]string{
				"status":   "1",
				"version":  "1",
				"debugger": "false",
				"id":       string(byt),
			},
			expectedVal: &runMetadata{
				Status:   enums.RunStatusCompleted,
				Version:  1,
				Debugger: false,
			},
			expectedError: nil,
		},
		{
			name:          "should error with missing identifier",
			data:          map[string]string{},
			expectedError: state.ErrRunNotFound,
		},
		{
			name: "missing ID should return err run not found",
			data: map[string]string{
				"status": "hello",
			},
			expectedError: state.ErrRunNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runMeta, err := newRunMetadata(test.data)
			require.Equal(t, test.expectedError, err)
			require.Equal(t, test.expectedVal, runMeta)
		})
	}
}

func TestIdempotencyCheck(t *testing.T) {
	ctx := context.Background()

	r, rc := initRedis(t)
	defer rc.Close()

	unshardedClient := NewUnshardedClient(rc, StateDefaultKey, QueueDefaultKey)
	shardedClient := NewShardedClient(ShardedClientOpts{
		UnshardedClient:        unshardedClient,
		FunctionRunStateClient: rc,
		BatchClient:            rc,
		StateDefaultKey:        StateDefaultKey,
		QueueDefaultKey:        QueueDefaultKey,
		FnRunIsSharded:         AlwaysShardOnRun,
	})

	acctID, wsID, appID, fnID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	runID := ulid.MustNew(ulid.Now(), rand.Reader)

	runState := shardedClient.fnRunState
	ftc, shared := runState.Client(ctx, acctID, runID)
	require.True(t, shared)

	mgr := shardedMgr{s: shardedClient}

	t.Run("with idempotency key defined in identifier", func(t *testing.T) {
		id := state.Identifier{
			AccountID:   acctID,
			WorkspaceID: wsID,
			AppID:       appID,
			WorkflowID:  fnID,
			RunID:       runID,
			Key:         "yolo",
		}
		key := runState.kg.Idempotency(ctx, shared, id)

		t.Run("returns nil if no idempotency key is available", func(t *testing.T) {
			r.FlushAll()

			runID, err := mgr.idempotencyCheck(ctx, ftc, key, id)
			require.NoError(t, err)
			require.Nil(t, runID)
		})

		t.Run("returns state if idempotency is already there", func(t *testing.T) {
			r.FlushAll()

			created, err := mgr.New(ctx, state.Input{
				Identifier:     id,
				EventBatchData: []map[string]any{},
				Steps:          []state.MemoizedStep{},
				StepInputs:     []state.MemoizedStep{},
				Context: map[string]any{
					"hello": "world",
				},
			})
			require.NoError(t, err)

			runID, err := mgr.idempotencyCheck(ctx, ftc, key, id)
			require.NoError(t, err)
			require.NotNil(t, runID)

			require.Equal(t, created.Identifier().RunID, *runID)
		})

		t.Run("returns invalid identifier error if previous value is not a ULID", func(t *testing.T) {
			r.FlushAll()
			require.NoError(t, r.Set(key, ""))

			runID, err := mgr.idempotencyCheck(ctx, ftc, key, id)
			require.Nil(t, runID)
			require.ErrorIs(t, err, state.ErrInvalidIdentifier)
		})
	})

	t.Run("with idempotency key not defined in identifier", func(t *testing.T) {
		id := state.Identifier{
			AccountID:   acctID,
			WorkspaceID: wsID,
			AppID:       appID,
			WorkflowID:  fnID,
			RunID:       runID,
		}
		key := runState.kg.Idempotency(ctx, shared, id)

		t.Run("returns nil if no idempotency key is available", func(t *testing.T) {
			r.FlushAll()

			st, err := mgr.idempotencyCheck(ctx, ftc, key, id)
			require.NoError(t, err)
			require.Nil(t, st)
		})

		t.Run("returns nil if runID is different", func(t *testing.T) {
			r.FlushAll()

			_, err := mgr.New(ctx, state.Input{
				Identifier:     id,
				EventBatchData: []map[string]any{},
			})
			require.NoError(t, err)

			diffID := id // copy
			diffID.RunID = ulid.MustNew(ulid.Now(), rand.Reader)
			diffKey := runState.kg.Idempotency(ctx, shared, diffID)
			runID, err := mgr.idempotencyCheck(ctx, ftc, diffKey, diffID)
			require.NoError(t, err)
			require.Nil(t, runID)
		})

		t.Run("returns invalid identifier error if previous value is not a ULID", func(t *testing.T) {
			r.FlushAll()
			require.NoError(t, r.Set(key, ""))

			runID, err := mgr.idempotencyCheck(ctx, ftc, key, id)
			require.Nil(t, runID)
			require.ErrorIs(t, err, state.ErrInvalidIdentifier)
		})
	})
}

func TestStateHarness(t *testing.T) {
	r, rc := initRedis(t)
	defer rc.Close()

	unshardedClient := NewUnshardedClient(rc, StateDefaultKey, QueueDefaultKey)
	shardedClient := NewShardedClient(ShardedClientOpts{
		UnshardedClient:        unshardedClient,
		FunctionRunStateClient: rc,
		BatchClient:            rc,
		StateDefaultKey:        StateDefaultKey,
		QueueDefaultKey:        QueueDefaultKey,
		FnRunIsSharded:         AlwaysShardOnRun,
	})

	sm, err := New(
		context.Background(),
		WithUnshardedClient(unshardedClient),
		WithShardedClient(shardedClient),
	)
	require.NoError(t, err)

	create := func() (state.Manager, func()) {
		return sm, func() {
			r.FlushAll()
		}
	}

	testharness.CheckState(t, create)
}

func TestScanIter(t *testing.T) {
	ctx := context.Background()
	redis := miniredis.RunT(t)
	r, err := rueidis.NewClient(rueidis.ClientOption{
		// InitAddress: []string{"127.0.0.1:6379"},
		InitAddress:  []string{redis.Addr()},
		Password:     "",
		DisableCache: true,
	})
	require.NoError(t, err)
	defer r.Close()

	entries := 50_000
	key := "test-scan"
	for i := 0; i < entries; i++ {
		cmd := r.B().Hset().Key(key).FieldValue().
			FieldValue(strconv.Itoa(i), strconv.Itoa(i)).
			Build()
		require.NoError(t, r.Do(ctx, cmd).Error())
	}
	fmt.Println("loaded keys")

	// Create a new scanIter, which iterates through all items.
	si := &scanIter{r: r}
	err = si.init(ctx, key, 1000)
	require.NoError(t, err)

	listed := 0

	for n := 0; n < entries*2; n++ {
		if !si.Next(ctx) {
			break
		}
		listed++
	}

	require.Equal(t, context.Canceled, si.Error())
	require.Equal(t, entries, listed)
}

func TestBufIter(t *testing.T) {
	ctx := context.Background()
	redis := miniredis.RunT(t)
	r, err := rueidis.NewClient(rueidis.ClientOption{
		// InitAddress: []string{"127.0.0.1:6379"},
		InitAddress:  []string{redis.Addr()},
		Password:     "",
		DisableCache: true,
	})
	require.NoError(t, err)
	defer r.Close()

	entries := 10_000
	key := "test-bufiter"
	for i := 0; i < entries; i++ {
		cmd := r.B().Hset().Key(key).FieldValue().FieldValue(strconv.Itoa(i), "{}").Build()
		require.NoError(t, r.Do(ctx, cmd).Error())
	}
	fmt.Println("loaded keys")

	// Create a new scanIter, which iterates through all items.
	bi := &bufIter{r: r}
	err = bi.init(ctx, key)
	require.NoError(t, err)

	listed := 0
	for n := 0; n < entries*2; n++ {
		if !bi.Next(ctx) {
			break
		}
		listed++
	}

	require.Equal(t, context.Canceled, bi.Error())
	require.Equal(t, entries, listed)
}

func BenchmarkNew(b *testing.B) {
	r := miniredis.RunT(b)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(b, err)
	defer rc.Close()

	statePrefix := "state"
	unshardedClient := NewUnshardedClient(rc, statePrefix, QueueDefaultKey)
	shardedClient := NewShardedClient(ShardedClientOpts{
		UnshardedClient:        unshardedClient,
		FunctionRunStateClient: rc,
		BatchClient:            rc,
		StateDefaultKey:        statePrefix,
		QueueDefaultKey:        QueueDefaultKey,
		FnRunIsSharded:         AlwaysShardOnRun,
	})

	sm, err := New(
		context.Background(),
		WithUnshardedClient(unshardedClient),
		WithShardedClient(shardedClient),
	)
	require.NoError(b, err)

	id := state.Identifier{
		WorkflowID: uuid.New(),
	}
	evt := event.Event{
		Name: "test-event",
		Data: map[string]any{
			"title": "They don't think it be like it is, but it do",
			"data": map[string]any{
				"float": 3.14132,
			},
		},
		User: map[string]any{
			"external_id": "1",
		},
		Version: "1985-01-01",
	}.Map()
	init := state.Input{
		Identifier:     id,
		EventBatchData: []map[string]any{evt},
	}

	ctx := context.Background()
	for n := 0; n < b.N; n++ {
		init.Identifier.RunID = ulid.MustNew(ulid.Now(), rand.Reader)
		_, err := sm.New(ctx, init)
		require.NoError(b, err)
	}

}
