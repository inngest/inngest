package step

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngestgo/errors"
	"github.com/inngest/inngestgo/internal"
	"github.com/inngest/inngestgo/internal/middleware"
	"github.com/inngest/inngestgo/internal/sdkrequest"
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
	mgr := preflight(ctx)
	op := mgr.NewOp(enums.OpcodeStepRun, id, nil)
	hashedID := op.MustHash()

	if val, ok := mgr.Step(ctx, op); ok {
		// Create a new empty type T in v
		ft := reflect.TypeOf(f)
		v := reflect.New(ft.Out(0)).Interface()

		// This step has already ran as we have state for it. Unmarshal the JSON into type T
		unwrapped := response{}
		if err := json.Unmarshal(val, &unwrapped); err == nil {
			// Check for step errors first.
			if len(unwrapped.Error) > 0 {
				err := errors.StepError{}
				if err := json.Unmarshal(unwrapped.Error, &err); err != nil {
					mgr.SetErr(fmt.Errorf("error unmarshalling error for step '%s': %w", id, err))
					panic(ControlHijack{})
				}

				// See if we have any data for multiple returns in the error type.
				if err := json.Unmarshal(err.Data, v); err != nil {
					mgr.SetErr(fmt.Errorf("error unmarshalling state for step '%s': %w", id, err))
					panic(ControlHijack{})
				}

				val, _ := reflect.ValueOf(v).Elem().Interface().(T)
				return val, err
			}
			// If there's an error, assume that val is already of type T without wrapping
			// in the 'data' object as per the SDK spec.  Here, if this succeeds we can be
			// sure that we're wrapping the data in a compliant way.
			if len(unwrapped.Data) > 0 {
				val = unwrapped.Data
			}
		}

		// Grab the data as the step type.
		if err := json.Unmarshal(val, v); err != nil {
			mgr.SetErr(fmt.Errorf("error unmarshalling state for step '%s': %w", id, err))
			panic(ControlHijack{})
		}
		val, _ := reflect.ValueOf(v).Elem().Interface().(T)
		return val, nil
	}

	if targetID != nil && *targetID != hashedID {
		panic(ControlHijack{})
	}

	planParallel := targetID == nil && isParallel(ctx)
	planBeforeRun := targetID == nil && mgr.Request().CallCtx.DisableImmediateExecution
	if planParallel || planBeforeRun {
		mgr.AppendOp(sdkrequest.GeneratorOpcode{
			ID:   hashedID,
			Op:   enums.OpcodeStepPlanned,
			Name: id,
		})
		panic(ControlHijack{})
	}

	// We're calling a function, so always cancel the context afterwards so that no
	// other tools run.
	defer mgr.Cancel()

	mw, ok := internal.MiddlewareManagerFromContext(ctx)
	if !ok {
		mgr.SetErr(fmt.Errorf("no middleware manager found in context"))
		panic(ControlHijack{})
	}

	// We're about to run a step callback, which is "new code".
	mw.BeforeExecution(ctx, mgr.MiddlewareCallCtx())
	result, err := f(setWithinStep(ctx))

	mw.AfterExecution(ctx, mgr.MiddlewareCallCtx(), result, err)
	out := &middleware.TransformableOutput{
		Result: result,
		Error:  err,
	}
	mw.TransformOutput(ctx, mgr.MiddlewareCallCtx(), out)

	mutated := out.Result
	err = out.Error
	if err != nil {
		// If tihs is a StepFailure already, fail fast.
		if errors.IsStepError(err) {
			mgr.SetErr(fmt.Errorf("unhandled step error: %s", err))
			panic(ControlHijack{})
		}

		result, _ := json.Marshal(mutated)

		// Implement per-step errors.
		mgr.AppendOp(sdkrequest.GeneratorOpcode{
			ID:   hashedID,
			Op:   enums.OpcodeStepError,
			Name: id,
			Error: &sdkrequest.UserError{
				Name:    "Step failed",
				Message: err.Error(),
				Data:    result,
			},
		})
		mgr.SetErr(err)
		panic(ControlHijack{})
	}

	byt, err := json.Marshal(mutated)
	if err != nil {
		mgr.SetErr(fmt.Errorf("unable to marshal run respone for '%s': %w", id, err))
	}
	mgr.AppendOp(sdkrequest.GeneratorOpcode{
		ID:   hashedID,
		Op:   enums.OpcodeStepRun,
		Name: id,
		Data: byt,
	})
	panic(ControlHijack{})
}
