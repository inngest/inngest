// Package shapes builds the set of inngestgo functions the harness exercises.
// Each shape isolates one kind of queue/executor pressure: step count,
// sleep-queue pressure, fanout, retries, etc.
package shapes

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/inngest/inngest/loadtest/internal/config"
	"github.com/inngest/inngest/loadtest/internal/telemetry"
	"github.com/inngest/inngestgo"
	"github.com/inngest/inngestgo/group"
	"github.com/inngest/inngestgo/step"
)

// Payload is the event data shape every harness event carries. The
// correlation ID lets the harness join the fired event to the fn_start frame
// emitted once the SDK begins executing.
type Payload struct {
	Shape         string `json:"shape"`
	CorrelationID string `json:"corr"`
	SentAt        int64  `json:"sentAt"`
}

// EventName returns the canonical event name for a given shape on a given app.
// Each app registers its own event names so apps don't cross-fire each other.
func EventName(appID string, shape config.Shape) string {
	return fmt.Sprintf("loadtest/%s.%s", appID, shape)
}

// retryCounter tracks "have we already failed once for this run id?" so the
// retry-forced shape produces exactly one retry and then succeeds.
var retryCounter sync.Map // runID -> *int32

// Register creates and registers all requested shapes on the given client.
// The telemetry client is used inside each handler to emit per-step frames.
func Register(c inngestgo.Client, tc *telemetry.Client, appID string, shapes []config.Shape) error {
	for _, s := range shapes {
		if err := registerOne(c, tc, appID, s); err != nil {
			return fmt.Errorf("register %s: %w", s, err)
		}
	}
	return nil
}

func registerOne(c inngestgo.Client, tc *telemetry.Client, appID string, s config.Shape) error {
	slug := string(s)
	trigger := inngestgo.EventTrigger(EventName(appID, s), nil)
	opts := inngestgo.FunctionOpts{ID: slug, Name: slug}

	switch s {
	case config.ShapeNoop:
		_, err := inngestgo.CreateFunction(c, opts, trigger, handlerNoop(tc, slug))
		return err
	case config.ShapeSteps3:
		_, err := inngestgo.CreateFunction(c, opts, trigger, handlerSteps(tc, slug, 3))
		return err
	case config.ShapeSteps10:
		_, err := inngestgo.CreateFunction(c, opts, trigger, handlerSteps(tc, slug, 10))
		return err
	case config.ShapeSleep1s:
		_, err := inngestgo.CreateFunction(c, opts, trigger, handlerSleep(tc, slug, time.Second))
		return err
	case config.ShapeFanout5:
		_, err := inngestgo.CreateFunction(c, opts, trigger, handlerFanout(tc, slug, 5))
		return err
	case config.ShapeRetryForced:
		_, err := inngestgo.CreateFunction(c, opts, trigger, handlerRetry(tc, slug))
		return err
	default:
		return fmt.Errorf("unknown shape %q", s)
	}
}

func emitFnStart(tc *telemetry.Client, runID, corr, fn string) {
	tc.EmitWithCorr(telemetry.PhaseFnStart, runID, corr, fn, "", 0)
}

func emitFn(tc *telemetry.Client, phase telemetry.Phase, runID, fn string) {
	tc.Emit(phase, runID, fn, "", 0)
}

func emitStep(tc *telemetry.Client, phase telemetry.Phase, runID, fn, step string, attempt int) {
	tc.Emit(phase, runID, fn, step, attempt)
}

func handlerNoop(tc *telemetry.Client, slug string) inngestgo.SDKFunction[Payload] {
	return func(ctx context.Context, in inngestgo.Input[Payload]) (any, error) {
		rid := in.InputCtx.RunID
		emitFnStart(tc, rid, in.Event.Data.CorrelationID, slug)
		emitFn(tc, telemetry.PhaseFnEnd, rid, slug)
		return "ok", nil
	}
}

func handlerSteps(tc *telemetry.Client, slug string, n int) inngestgo.SDKFunction[Payload] {
	return func(ctx context.Context, in inngestgo.Input[Payload]) (any, error) {
		rid := in.InputCtx.RunID
		emitFnStart(tc, rid, in.Event.Data.CorrelationID, slug)
		for i := 0; i < n; i++ {
			stepID := fmt.Sprintf("s%d", i)
			emitStep(tc, telemetry.PhaseStepStart, rid, slug, stepID, in.InputCtx.Attempt)
			_, err := step.Run(ctx, stepID, func(ctx context.Context) (int, error) {
				return i, nil
			})
			emitStep(tc, telemetry.PhaseStepEnd, rid, slug, stepID, in.InputCtx.Attempt)
			if err != nil {
				return nil, err
			}
		}
		emitFn(tc, telemetry.PhaseFnEnd, rid, slug)
		return n, nil
	}
}

func handlerSleep(tc *telemetry.Client, slug string, d time.Duration) inngestgo.SDKFunction[Payload] {
	return func(ctx context.Context, in inngestgo.Input[Payload]) (any, error) {
		rid := in.InputCtx.RunID
		emitFnStart(tc, rid, in.Event.Data.CorrelationID, slug)
		emitStep(tc, telemetry.PhaseStepStart, rid, slug, "sleep", in.InputCtx.Attempt)
		step.Sleep(ctx, "sleep", d)
		emitStep(tc, telemetry.PhaseStepEnd, rid, slug, "sleep", in.InputCtx.Attempt)
		emitStep(tc, telemetry.PhaseStepStart, rid, slug, "after", in.InputCtx.Attempt)
		_, err := step.Run(ctx, "after", func(ctx context.Context) (string, error) { return "done", nil })
		emitStep(tc, telemetry.PhaseStepEnd, rid, slug, "after", in.InputCtx.Attempt)
		if err != nil {
			return nil, err
		}
		emitFn(tc, telemetry.PhaseFnEnd, rid, slug)
		return "ok", nil
	}
}

func handlerFanout(tc *telemetry.Client, slug string, n int) inngestgo.SDKFunction[Payload] {
	return func(ctx context.Context, in inngestgo.Input[Payload]) (any, error) {
		rid := in.InputCtx.RunID
		emitFnStart(tc, rid, in.Event.Data.CorrelationID, slug)
		fns := make([]func(ctx context.Context) (any, error), n)
		for i := 0; i < n; i++ {
			i := i
			stepID := fmt.Sprintf("p%d", i)
			fns[i] = func(ctx context.Context) (any, error) {
				emitStep(tc, telemetry.PhaseStepStart, rid, slug, stepID, in.InputCtx.Attempt)
				v, err := step.Run(ctx, stepID, func(ctx context.Context) (int, error) { return i, nil })
				emitStep(tc, telemetry.PhaseStepEnd, rid, slug, stepID, in.InputCtx.Attempt)
				return v, err
			}
		}
		res := group.Parallel(ctx, fns...)
		if res.AnyError() != nil {
			return nil, res.AnyError()
		}
		emitFn(tc, telemetry.PhaseFnEnd, rid, slug)
		return n, nil
	}
}

func handlerRetry(tc *telemetry.Client, slug string) inngestgo.SDKFunction[Payload] {
	return func(ctx context.Context, in inngestgo.Input[Payload]) (any, error) {
		rid := in.InputCtx.RunID
		emitFnStart(tc, rid, in.Event.Data.CorrelationID, slug)
		emitStep(tc, telemetry.PhaseStepStart, rid, slug, "flaky", in.InputCtx.Attempt)
		v, err := step.Run(ctx, "flaky", func(ctx context.Context) (string, error) {
			cntAny, _ := retryCounter.LoadOrStore(rid, new(int32))
			cnt := cntAny.(*int32)
			n := atomic.AddInt32(cnt, 1)
			if n == 1 {
				return "", fmt.Errorf("forced-retry")
			}
			retryCounter.Delete(rid)
			return "ok", nil
		})
		emitStep(tc, telemetry.PhaseStepEnd, rid, slug, "flaky", in.InputCtx.Attempt)
		if err != nil {
			return nil, err
		}
		emitFn(tc, telemetry.PhaseFnEnd, rid, slug)
		return v, nil
	}
}

// All returns the full set of supported shapes, in stable order. Useful for
// default configurations.
func All() []config.Shape {
	return []config.Shape{
		config.ShapeNoop,
		config.ShapeSteps3,
		config.ShapeSteps10,
		config.ShapeSleep1s,
		config.ShapeFanout5,
		config.ShapeRetryForced,
	}
}
