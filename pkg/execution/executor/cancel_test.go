package executor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestCancelForceLifecycleHookFinalizesWhenMetadataMissing(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
	}{
		{
			name: "metadata not found",
			err:  sv2.ErrMetadataNotFound,
		},
		{
			name: "run not found",
			err:  state.ErrRunNotFound,
		},
		{
			name: "wrapped run not found",
			err:  fmt.Errorf("load metadata: %w", state.ErrRunNotFound),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runID := ulid.Make()
			id := sv2.ID{
				RunID:      runID,
				FunctionID: uuid.New(),
				Tenant: sv2.Tenant{
					AccountID: uuid.New(),
					AppID:     uuid.New(),
					EnvID:     uuid.New(),
				},
			}

			runState := &missingMetadataRunService{err: tc.err}
			lifecycle := &recordingCancelLifecycle{
				cancelled: make(chan sv2.Metadata, 1),
			}
			e := &executor{
				log:            logger.VoidLogger(),
				smv2:           runState,
				shards:         missingShardRegistry{},
				tracerProvider: tracing.NewOtelTracerProvider(nil, time.Millisecond),
				lifecycles:     []execution.LifecycleListener{lifecycle},
			}

			err := e.Cancel(context.Background(), id, execution.CancelRequest{
				ForceLifecycleHook: true,
			})
			require.NoError(t, err)
			require.True(t, runState.deleted.Load())

			select {
			case md := <-lifecycle.cancelled:
				require.Equal(t, id, md.ID)
				require.Nil(t, md.Config.DebugRunID())
			case <-time.After(time.Second):
				t.Fatal("timed out waiting for cancellation lifecycle")
			}
		})
	}
}

type missingMetadataRunService struct {
	sv2.RunService
	deleted atomic.Bool
	err     error
}

func (m *missingMetadataRunService) LoadMetadata(context.Context, sv2.ID, ...sv2.LoadMetadataOption) (sv2.Metadata, error) {
	return sv2.Metadata{}, m.err
}

func (m *missingMetadataRunService) LoadEvents(context.Context, sv2.ID) ([]json.RawMessage, error) {
	return nil, state.ErrEventNotFound
}

func (m *missingMetadataRunService) LoadDefers(context.Context, sv2.ID) (map[string]sv2.Defer, error) {
	return nil, nil
}

func (m *missingMetadataRunService) Delete(context.Context, sv2.ID) error {
	m.deleted.Store(true)
	return nil
}

type missingShardRegistry struct {
	queue.ShardRegistry
}

func (missingShardRegistry) Resolve(context.Context, queue.Scope, *string) (queue.QueueShard, error) {
	return nil, errors.New("missing shard")
}

type recordingCancelLifecycle struct {
	execution.NoopLifecyceListener
	cancelled chan sv2.Metadata
}

func (r *recordingCancelLifecycle) OnFunctionCancelled(
	_ context.Context,
	md sv2.Metadata,
	_ execution.CancelRequest,
	_ []json.RawMessage,
) {
	r.cancelled <- md
}
