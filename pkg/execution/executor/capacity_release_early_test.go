package executor

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/exechttp"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// Opcodes that leave the run waiting rather than executing hand back their
// concurrency capacity once their state is durable, not after the queue's
// final dequeue. step.invoke releases only once the invocation is published,
// so a failed publish (which is retried) keeps it.
func TestHandlersReleaseCapacityOnceRunIsWaiting(t *testing.T) {
	publishErr := errors.New("publish failed")
	for _, tc := range []struct {
		name        string
		gen         state.GeneratorOpcode
		publish     error
		run         func(e *executor, rc *mockRunContext, gen state.GeneratorOpcode) error
		disabled    bool
		wantErr     bool
		wantRelease int
	}{
		{
			name: "sleep",
			gen:  state.GeneratorOpcode{Op: enums.OpcodeSleep, ID: "step-sleep", Opts: []byte(`{"duration":"1h"}`)},
			run: func(e *executor, rc *mockRunContext, gen state.GeneratorOpcode) error {
				return e.handleGeneratorSleep(context.Background(), rc, gen, queue.PayloadEdge{Edge: inngest.Edge{Incoming: "step"}}, OpcodeGroup{})
			},
			wantRelease: 1,
		},
		{
			name: "wait for signal",
			gen:  state.GeneratorOpcode{Op: enums.OpcodeWaitForSignal, ID: "step-signal", Opts: []byte(`{"signal":"sig","timeout":"1h"}`)},
			run: func(e *executor, rc *mockRunContext, gen state.GeneratorOpcode) error {
				return e.handleGeneratorWaitForSignal(context.Background(), rc, gen, queue.PayloadEdge{Edge: inngest.Edge{Incoming: "step"}}, OpcodeGroup{})
			},
			wantRelease: 1,
		},
		{
			name: "invoke",
			gen:  state.GeneratorOpcode{Op: enums.OpcodeInvokeFunction, ID: "step-invoke", Opts: []byte(`{"function_id":"app-fn","payload":{"name":"some/event","data":{}}}`)},
			run: func(e *executor, rc *mockRunContext, gen state.GeneratorOpcode) error {
				return e.handleGeneratorInvokeFunction(context.Background(), rc, gen, queue.PayloadEdge{Edge: inngest.Edge{Incoming: "step"}}, OpcodeGroup{})
			},
			wantRelease: 1,
		},
		{
			name:     "sleep with early release disabled",
			gen:      state.GeneratorOpcode{Op: enums.OpcodeSleep, ID: "step-sleep", Opts: []byte(`{"duration":"1h"}`)},
			disabled: true,
			run: func(e *executor, rc *mockRunContext, gen state.GeneratorOpcode) error {
				return e.handleGeneratorSleep(context.Background(), rc, gen, queue.PayloadEdge{Edge: inngest.Edge{Incoming: "step"}}, OpcodeGroup{})
			},
			wantRelease: 0,
		},
		{
			name:    "invoke whose publish fails",
			gen:     state.GeneratorOpcode{Op: enums.OpcodeInvokeFunction, ID: "step-invoke", Opts: []byte(`{"function_id":"app-fn","payload":{"name":"some/event","data":{}}}`)},
			publish: publishErr,
			run: func(e *executor, rc *mockRunContext, gen state.GeneratorOpcode) error {
				return e.handleGeneratorInvokeFunction(context.Background(), rc, gen, queue.PayloadEdge{Edge: inngest.Edge{Incoming: "step"}}, OpcodeGroup{})
			},
			wantErr:     true,
			wantRelease: 0,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := &executor{
				log:            logger.From(context.Background()),
				clock:          clockwork.NewFakeClock(),
				tracerProvider: tracing.NewNoopTracerProvider(),
				pm:             &stubPauseManager{},
				queue:          &stubQueue{},
				handleInvokeEvent: func(context.Context, event.TrackedEvent) error {
					return tc.publish
				},
			}
			if !tc.disabled {
				e.earlyCapacityRelease = allowEarlyCapacityRelease
			}
			rc := &mockRunContext{
				md: sv2.Metadata{
					ID:     sv2.ID{RunID: ulid.MustNew(ulid.Now(), nil), FunctionID: uuid.New()},
					Config: *sv2.InitConfig(&sv2.Config{}),
				},
			}
			err := tc.run(e, rc, tc.gen)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tc.wantRelease, rc.released)
		})
	}
}

func allowEarlyCapacityRelease(context.Context, uuid.UUID) bool { return true }

// blockingHTTPClient holds a fetch open until unblock is closed.
type blockingHTTPClient struct {
	started chan struct{}
	unblock chan struct{}
}

func (b *blockingHTTPClient) DoRequest(_ context.Context, _ exechttp.SerializableRequest) (*exechttp.Response, error) {
	close(b.started)
	<-b.unblock
	return &exechttp.Response{StatusCode: 200, Body: []byte(`{}`)}, nil
}

// Opcodes of one SDK response run concurrently and share the queue item's
// capacity lease, so an opcode finishing beside a fetch still in flight must
// not hand the capacity back. It is released once, after the whole response.
// The step output, step failure and waitForEvent releases predate the early
// release switch and so are held even with it off.
func TestReleaseBesideFetchHoldsCapacityUntilFetchFinishes(t *testing.T) {
	edge := queue.PayloadEdge{Edge: inngest.Edge{Incoming: "step"}}
	for _, tc := range []struct {
		name     string
		op       *state.GeneratorOpcode
		disabled bool
		run      func(e *executor, rc *mockRunContext, gen state.GeneratorOpcode, group OpcodeGroup) error
	}{
		{
			name: "sleep",
			op:   &state.GeneratorOpcode{Op: enums.OpcodeSleep, ID: "step-sleep", Opts: []byte(`{"duration":"1h"}`)},
			run: func(e *executor, rc *mockRunContext, gen state.GeneratorOpcode, group OpcodeGroup) error {
				return e.handleGeneratorSleep(context.Background(), rc, gen, edge, group)
			},
		},
		{
			name: "wait for signal",
			op:   &state.GeneratorOpcode{Op: enums.OpcodeWaitForSignal, ID: "step-signal", Opts: []byte(`{"signal":"sig","timeout":"1h"}`)},
			run: func(e *executor, rc *mockRunContext, gen state.GeneratorOpcode, group OpcodeGroup) error {
				return e.handleGeneratorWaitForSignal(context.Background(), rc, gen, edge, group)
			},
		},
		{
			name: "invoke",
			op:   &state.GeneratorOpcode{Op: enums.OpcodeInvokeFunction, ID: "step-invoke", Opts: []byte(`{"function_id":"app-fn","payload":{"name":"some/event","data":{}}}`)},
			run: func(e *executor, rc *mockRunContext, gen state.GeneratorOpcode, group OpcodeGroup) error {
				return e.handleGeneratorInvokeFunction(context.Background(), rc, gen, edge, group)
			},
		},
		{
			name:     "step output",
			op:       &state.GeneratorOpcode{Op: enums.OpcodeStepRun, ID: "step-run", Data: []byte(`null`)},
			disabled: true,
			run: func(e *executor, rc *mockRunContext, gen state.GeneratorOpcode, group OpcodeGroup) error {
				return e.handleGeneratorStep(context.Background(), rc, gen, edge, group)
			},
		},
		{
			name:     "step failed",
			op:       &state.GeneratorOpcode{Op: enums.OpcodeStepFailed, ID: "step-failed", Error: &state.UserError{Name: "Error", Message: "boom"}},
			disabled: true,
			run: func(e *executor, rc *mockRunContext, gen state.GeneratorOpcode, group OpcodeGroup) error {
				return e.handleStepFailed(context.Background(), rc, gen, edge, group)
			},
		},
		{
			name:     "wait for event",
			op:       &state.GeneratorOpcode{Op: enums.OpcodeWaitForEvent, ID: "step-wait", Opts: []byte(`{"event":"some/event","timeout":"1h"}`)},
			disabled: true,
			run: func(e *executor, rc *mockRunContext, gen state.GeneratorOpcode, group OpcodeGroup) error {
				return e.handleGeneratorWaitForEvent(context.Background(), rc, gen, edge, group)
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			client := &blockingHTTPClient{started: make(chan struct{}), unblock: make(chan struct{})}
			e := &executor{
				log:            logger.From(ctx),
				clock:          clockwork.NewFakeClock(),
				tracerProvider: tracing.NewNoopTracerProvider(),
				smv2:           &stubRunServiceMD{saveStepHasPending: true},
				pm:             &stubPauseManager{},
				queue:          &stubQueue{},
				handleInvokeEvent: func(context.Context, event.TrackedEvent) error {
					return nil
				},
			}
			if !tc.disabled {
				e.earlyCapacityRelease = allowEarlyCapacityRelease
			}
			rc := &mockRunContext{
				md: sv2.Metadata{
					ID:     sv2.ID{RunID: ulid.MustNew(ulid.Now(), nil), FunctionID: uuid.New()},
					Config: *sv2.InitConfig(&sv2.Config{}),
				},
				httpClient: client,
			}
			fetch := &state.GeneratorOpcode{Op: enums.OpcodeGateway, ID: "step-fetch", Opts: []byte(`{"url":"https://example.com","method":"GET"}`)}
			capacity := newResponseCapacity([]*state.GeneratorOpcode{tc.op, fetch})
			group := OpcodeGroup{capacity: capacity}

			fetched := make(chan error, 1)
			go func() { fetched <- e.handleGeneratorGateway(ctx, rc, *fetch, edge, group) }()
			<-client.started

			require.NoError(t, tc.run(e, rc, *tc.op, group))
			require.Equal(t, 0, rc.released, "released while the fetch was still running")

			close(client.unblock)
			select {
			case err := <-fetched:
				require.NoError(t, err)
			case <-time.After(5 * time.Second):
				t.Fatal("fetch did not finish")
			}
			require.Equal(t, 0, rc.released)

			capacity.releaseDeferred(ctx, rc)
			require.Equal(t, 1, rc.released)
		})
	}
}
