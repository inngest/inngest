package executor

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/batch"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	telemetrytrace "github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

type appNameBatchManager struct {
	batch.BatchManager
	appendResult *batch.BatchAppendResult
	items        []batch.BatchItem
	scheduled    *batch.ScheduleBatchOpts
}

func (m *appNameBatchManager) Append(context.Context, batch.BatchItem, inngest.Function) (*batch.BatchAppendResult, error) {
	return m.appendResult, nil
}

func (m *appNameBatchManager) RetrieveItems(context.Context, uuid.UUID, ulid.ULID) ([]batch.BatchItem, error) {
	return m.items, nil
}

func (m *appNameBatchManager) ScheduleExecution(_ context.Context, opts batch.ScheduleBatchOpts) error {
	m.scheduled = &opts
	return nil
}

func (m *appNameBatchManager) DeleteKeys(context.Context, uuid.UUID, ulid.ULID) error {
	return nil
}

func (m *appNameBatchManager) StartExecution(context.Context, uuid.UUID, ulid.ULID, string) (string, error) {
	return enums.BatchStatusReady.String(), nil
}

type appNameCQRSManager struct {
	cqrs.Manager
	fn inngest.Function
}

func (m *appNameCQRSManager) Functions(context.Context) ([]inngest.Function, error) {
	return []inngest.Function{m.fn}, nil
}

type appNameExecutor struct {
	execution.Executor
	payload batch.ScheduleBatchPayload
}

func (e *appNameExecutor) RetrieveAndScheduleBatch(_ context.Context, _ inngest.Function, payload batch.ScheduleBatchPayload, _ *execution.BatchExecOpts) error {
	e.payload = payload
	return nil
}

func TestAppendAndScheduleBatchCarriesAppName(t *testing.T) {
	batchID := ulid.Make()
	mgr := &appNameBatchManager{
		appendResult: &batch.BatchAppendResult{
			Status:          enums.BatchNew,
			BatchID:         batchID.String(),
			BatchPointerKey: "batch-pointer",
		},
	}
	e := &executor{batcher: mgr}
	fn := inngest.Function{
		ID: uuid.New(),
		EventBatch: &inngest.EventBatchConfig{
			MaxSize: 10,
			Timeout: "1m",
		},
	}

	err := e.AppendAndScheduleBatch(context.Background(), fn, batch.BatchItem{
		AccountID:       uuid.New(),
		WorkspaceID:     uuid.New(),
		AppID:           uuid.New(),
		AppName:         "customer-facing-app",
		FunctionID:      fn.ID,
		FunctionVersion: 1,
		EventID:         ulid.Make(),
		Event:           event.Event{Name: "test/batch"},
	}, nil)
	require.NoError(t, err)
	require.NotNil(t, mgr.scheduled)
	require.Equal(t, "customer-facing-app", mgr.scheduled.AppName)
}

func TestHandleScheduledBatchPreservesPayloadAppName(t *testing.T) {
	fn := inngest.Function{ID: uuid.New(), FunctionVersion: 2}
	payload := batch.ScheduleBatchPayload{
		BatchID:      ulid.Make(),
		BatchPointer: "batch-pointer",
		AppName:      "customer-facing-app",
		FunctionID:   fn.ID,
	}
	encoded, err := json.Marshal(batch.ScheduleBatchOpts{ScheduleBatchPayload: payload})
	require.NoError(t, err)
	recorder := &appNameExecutor{}
	s := &svc{
		batcher: &appNameBatchManager{},
		data:    &appNameCQRSManager{fn: fn},
		exec:    recorder,
	}

	err = s.handleScheduledBatch(context.Background(), queue.Item{
		Payload: json.RawMessage(encoded),
		Identifier: state.Identifier{
			AccountID:   uuid.New(),
			WorkspaceID: uuid.New(),
			AppID:       uuid.New(),
			WorkflowID:  fn.ID,
		},
	})
	require.NoError(t, err)
	require.Equal(t, "customer-facing-app", recorder.payload.AppName)
}

func TestRetrieveAndScheduleBatchAddsAppNameToRunSpan(t *testing.T) {
	eventID := ulid.Make()
	mgr := &appNameBatchManager{
		items: []batch.BatchItem{{
			EventID: eventID,
			Event: event.Event{
				ID:        eventID.String(),
				Name:      "test/batch",
				Timestamp: time.Now().UnixMilli(),
			},
		}},
	}
	rec := newRecordingTracerProvider()
	e := &executor{
		batcher:           mgr,
		log:               logger.From(context.Background()),
		tracerProvider:    rec,
		conditionalTracer: telemetrytrace.NoopConditionalTracer(),
	}
	pausedAt := time.Now().Add(-time.Minute)
	fn := inngest.Function{ID: uuid.New(), FunctionVersion: 1, Name: "Batch Function"}

	err := e.RetrieveAndScheduleBatch(context.Background(), fn, batch.ScheduleBatchPayload{
		BatchID:     ulid.Make(),
		AccountID:   uuid.New(),
		WorkspaceID: uuid.New(),
		AppID:       uuid.New(),
		AppName:     "customer-facing-app",
		FunctionID:  fn.ID,
	}, &execution.BatchExecOpts{FunctionPausedAt: &pausedAt})
	require.NoError(t, err)

	var runSpan *createSpanCall
	for _, call := range rec.createCalls {
		if call.name == meta.SpanNameRun {
			runSpan = call
			break
		}
	}
	require.NotNil(t, runSpan)
	appName, ok := runSpan.opts.Attributes.Get(meta.Attrs.AppName.Key()).(*string)
	require.True(t, ok)
	require.Equal(t, "customer-facing-app", *appName)
}
