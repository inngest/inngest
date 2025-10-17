package step

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngestgo/errors"
	"github.com/inngest/inngestgo/internal"
	"github.com/inngest/inngestgo/internal/middleware"
	"github.com/inngest/inngestgo/internal/sdkrequest"
	"github.com/inngest/inngestgo/pkg/interval"
)

type RunOpts struct {
	// ID represents the optional step name.
	ID string
	// Name represents the optional step name.
	Name string
}

// response represents the basic response format for all steps.  if the step errored,
// we expect to find the error in our Error field.
type response struct {
	Data  json.RawMessage `json:"data"`
	Error json.RawMessage `json:"error"`
}

// StepRun runs any code reliably, with retries, returning the resulting data.  If this
// fails the function stops.
func Run[T any](
	ctx context.Context,
	id string,
	f func(ctx context.Context) (T, error),
) (T, error) {
	targetID := getTargetStepID(ctx)
	mgr := preflight(ctx, enums.OpcodeStepRun)
	op := mgr.NewOp(enums.OpcodeStepRun, id)
	hashedID := op.MustHash()

	if mgr == nil {
		// If there's no manager, execute the function directly.
		return f(ctx)
	}

	if val, ok := mgr.Step(ctx, op); ok {
		return loadExistingStep(id, mgr, val, f)
	}

	if targetID != nil && *targetID != hashedID {
		// Don't report this step since targeting is happening and it isn't
		// targeted
		panic(sdkrequest.ControlHijack{})
	}

	planParallel := targetID == nil && sdkrequest.IsParallel(ctx)
	planBeforeRun := targetID == nil && mgr.Request().CallCtx.DisableImmediateExecution
	if planParallel || planBeforeRun {
		plannedOp := sdkrequest.GeneratorOpcode{
			ID:   hashedID,
			Op:   enums.OpcodeStepPlanned,
			Name: id,
		}
		mgr.AppendOp(ctx, plannedOp)
		panic(sdkrequest.ControlHijack{})
	}

	mw := internal.MiddlewareFromContext(ctx)

	// We're about to run a step callback, which is "new code".
	mw.BeforeExecution(ctx, mgr.CallContext())
	pre := time.Now()
	result, err := f(setWithinStep(ctx))
	post := time.Now()
	mw.AfterExecution(ctx, mgr.CallContext(), result, err)

	out := &middleware.TransformableOutput{
		Result: result,
		Error:  err,
	}
	mw.TransformOutput(ctx, mgr.CallContext(), out)

	mutated := out.Result
	err = out.Error
	if err != nil {
		// If tihs is a StepFailure already, fail fast.
		if errors.IsStepError(err) {
			mgr.SetErr(fmt.Errorf("unhandled step error: %s", err))
			panic(sdkrequest.ControlHijack{})
		}

		marshalled, _ := json.Marshal(mutated)

		// Implement per-step errors.
		mgr.SetErr(err)
		mgr.AppendOp(ctx, sdkrequest.GeneratorOpcode{
			ID:   hashedID,
			Op:   enums.OpcodeStepError,
			Name: id,
			Error: &sdkrequest.UserError{
				Name:    "Step failed",
				Message: err.Error(),
				Data:    marshalled,
			},
			Timing: interval.New(pre, post),
		})

		// API functions: return the error without panic
		return result, err
	}

	byt, err := json.Marshal(mutated)
	if err != nil {
		mgr.SetErr(fmt.Errorf("unable to marshal run respone for '%s': %w", id, err))
	}

	// Depending on the manager's step mode, this will either return control to the handler
	// to prevent function execution or checkpoint the step immediately.
	mgr.AppendOp(ctx, sdkrequest.GeneratorOpcode{
		ID:     hashedID,
		Op:     enums.OpcodeStepRun,
		Name:   id,
		Data:   byt,
		Timing: interval.New(pre, post),
	})

	return result, nil
}

func loadExistingStep[T any](
	id string,
	mgr sdkrequest.InvocationManager,
	existing json.RawMessage,
	f func(ctx context.Context) (T, error),
) (T, error) {
	// Create a new empty type T in v
	ft := reflect.TypeOf(f)
	v := reflect.New(ft.Out(0)).Interface()

	// This step has already ran as we have state for it. Unmarshal the JSON into type T
	unwrapped := response{}
	if err := json.Unmarshal(existing, &unwrapped); err == nil {
		// Check for step errors first.
		if len(unwrapped.Error) > 0 {
			err := errors.StepError{}
			if err := json.Unmarshal(unwrapped.Error, &err); err != nil {
				mgr.SetErr(fmt.Errorf("error unmarshalling error for step '%s': %w", id, err))
				panic(sdkrequest.ControlHijack{})
			}

			// See if we have any data for multiple returns in the error type.
			if err := json.Unmarshal(err.Data, v); err != nil {
				mgr.SetErr(fmt.Errorf("error unmarshalling state for step '%s': %w", id, err))
				panic(sdkrequest.ControlHijack{})
			}

			val, _ := reflect.ValueOf(v).Elem().Interface().(T)
			return val, err
		}
		// If there's an error, assume that val is already of type T without wrapping
		// in the 'data' object as per the SDK spec.  Here, if this succeeds we can be
		// sure that we're wrapping the data in a compliant way.
		if len(unwrapped.Data) > 0 {
			existing = unwrapped.Data
		}
	}

	// Grab the data as the step type.
	if err := json.Unmarshal(existing, v); err != nil {
		mgr.SetErr(fmt.Errorf("error unmarshalling state for step '%s': %w", id, err))
		panic(sdkrequest.ControlHijack{})
	}

	val, _ := reflect.ValueOf(v).Elem().Interface().(T)
	return val, nil
}
