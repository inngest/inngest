package step

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	sdkerrors "github.com/inngest/inngestgo/errors"
	"github.com/inngest/inngestgo/internal/sdkrequest"
	"github.com/xhit/go-str2duration/v2"
)

type InvokeOpts struct {
	// ID is the ID of the function to invoke, including the client ID prefix.
	FunctionId string
	// Data is the data to pass to the invoked function.
	Data map[string]any
	// User is the user data to pass to the invoked function.
	User any
	// Timeout is an optional duration specifying when the invoked function will be
	// considered timed out
	Timeout time.Duration
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
	if opts.Timeout > 0 {
		args["timeout"] = str2duration.String(opts.Timeout)
	}

	op := mgr.NewOp(enums.OpcodeInvokeFunction, id, args)
	if val, ok := mgr.Step(ctx, op); ok {
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

			return output, sdkerrors.NoRetryError(fmt.Errorf("%s", errObj.Message))
		}

		mgr.SetErr(fmt.Errorf("error parsing invoke value for '%s'; unknown shape", opts.FunctionId))
		panic(ControlHijack{})
	}

	mgr.AppendOp(sdkrequest.GeneratorOpcode{
		ID:   op.MustHash(),
		Op:   op.Op,
		Name: id,
		Opts: op.Opts,
	})
	panic(ControlHijack{})
}
