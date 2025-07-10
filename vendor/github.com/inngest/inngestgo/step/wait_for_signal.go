package step

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngestgo/internal/sdkrequest"
	"github.com/xhit/go-str2duration/v2"
)

// ErrSignalNotReceived is returned when a WaitForSignal call times out.  It indicates that a
// matching signal was not received before the timeout.
var ErrSignalNotReceived = fmt.Errorf("signal not received")

type SignalConfict string

const (
	// SignalConflictFail fails the run if another signal currently exists.  This is the default behaviour.
	SignalConflictFail SignalConfict = "fail"
	// SignalConflictReplace replaces an existing signal if the signal currently exists.  Any run
	// waiting for the previous signal will fail.
	SignalConflictReplace SignalConfict = "replace"
)

type WaitForSignalOpts struct {
	// Name represents the optional step name.
	Name string `json:"name"`
	// Signal is the signal to wait for.  This is a string unique to your environment
	// which will resume this particular function run.  If this signal already exists,
	// the step will error.
	//
	// For resuming multiple runs from a signal, use WaitForEvent.  Generally speaking,
	// WaitForEvent fulfils WaitForSignal with fan out and an improved DX.
	Signal string `json:"signal"`
	// Timeout is how long to wait.  We must always timebound event lsiteners.
	Timeout time.Duration `json:"timeout"`

	// OnConflict
	OnConflict SignalConfict `json:"onConflict"`
}

// rawSignalResult is the raw result stored in step state.  We always embed step output
// within a Data field, allowing us to store metadata for steps in the future.
type rawSignalResult[T any] struct {
	Data SignalResult[T] `json:"data"`
}

type SignalResult[T any] struct {
	Signal string `json:"signal"`
	Data   T      `json:"data"`
}

func WaitForSignal[T any](ctx context.Context, stepID string, opts WaitForSignalOpts) (SignalResult[T], error) {
	mgr := preflight(ctx)

	args := map[string]any{
		"signal":     opts.Signal,
		"timeout":    str2duration.String(opts.Timeout),
		"conflict": SignalConflictFail,
	}
	if opts.Name == "" {
		opts.Name = stepID
	}
	if opts.OnConflict != "" {
		args["conflict"] = opts.OnConflict
	}

	op := mgr.NewOp(enums.OpcodeWaitForSignal, stepID, args)

	// Check if this exists already.
	if val, ok := mgr.Step(ctx, op); ok {
		var output rawSignalResult[T]
		if val == nil || bytes.Equal(val, []byte{0x6e, 0x75, 0x6c, 0x6c}) {
			return output.Data, ErrSignalNotReceived
		}
		if err := json.Unmarshal(val, &output); err != nil {
			mgr.SetErr(fmt.Errorf("error unmarshalling wait for signal value in '%s': %w", opts.Signal, err))
			panic(ControlHijack{})
		}
		return output.Data, nil
	}

	mgr.AppendOp(sdkrequest.GeneratorOpcode{
		ID:          op.MustHash(),
		Op:          op.Op,
		Name:        opts.Name,
		DisplayName: &opts.Name,
		Opts:        op.Opts,
	})
	panic(ControlHijack{})
}
