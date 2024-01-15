package step

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngestgo/errors"
)

type InvokeOpts struct {
	// ID is the ID of the function to invoke, including the client ID prefix.
	FunctionId string
	// Data is the data to pass to the invoked function.
	Data map[string]any
	// User is the user data to pass to the invoked function.
	User any
}

// Invoke another Inngest function using its ID. Returns the value returned from
// that function.
//
// If the invoked function can't be found or otherwise errors, the step will
// fail and the function will stop with a `NoRetryError`.
func Invoke[T any](ctx context.Context, id string, opts InvokeOpts) (T, error) {
	mgr := preflight(ctx)
	args := map[string]any{
		"function_id": opts.FunctionId,
		"payload": map[string]any{
			"data": opts.Data,
			"user": opts.User,
		},
	}

	op := mgr.NewOp(enums.OpcodeInvokeFunction, id, args)
	if val, ok := mgr.Step(op); ok {
		var output T
		var valMap map[string]json.RawMessage
		if err := json.Unmarshal(val, &valMap); err != nil {
			mgr.SetErr(fmt.Errorf("error unmarshalling invoke value for '%s': %w", opts.FunctionId, err))
			panic(ControlHijack{})
		}

		if data, ok := valMap["data"]; ok {
			if err := json.Unmarshal(data, &output); err != nil {
				mgr.SetErr(fmt.Errorf("error unmarshalling invoke data for '%s': %w", opts.FunctionId, err))
				panic(ControlHijack{})
			}
			return output, nil
		}

		// Handled in this single tool until we want to make broader changes
		// to add per-step errors everywhere.
		if errorVal, ok := valMap["error"]; ok {
			var errObj struct {
				Message string `json:"message"`
			}
			if err := json.Unmarshal(errorVal, &errObj); err != nil {
				mgr.SetErr(fmt.Errorf("error unmarshalling invoke error for '%s': %w", opts.FunctionId, err))
				panic(ControlHijack{})
			}

			errMsg := "invoking function failed"
			if errObj.Message != "" {
				errMsg += "; " + errObj.Message
			}

			customErr := errors.NoRetryError(fmt.Errorf(errMsg))
			mgr.SetErr(customErr)
			panic(ControlHijack{})
		}

		mgr.SetErr(fmt.Errorf("error parsing invoke value for '%s'; unknown shape", opts.FunctionId))
		panic(ControlHijack{})
	}
	mgr.AppendOp(state.GeneratorOpcode{
		ID:   op.MustHash(),
		Op:   op.Op,
		Name: id,
		Opts: op.Opts,
	})
	panic(ControlHijack{})
}
