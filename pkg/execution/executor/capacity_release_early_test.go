package executor

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
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
