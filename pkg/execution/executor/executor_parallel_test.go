package executor

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/execution/state"
)

const (
	ms   = time.Millisecond
	ms10 = time.Millisecond * 10
)

func TestParallelism(t *testing.T) {

	tests := []struct {
		name  string
		steps []Promise
	}{
		{
			name: "basic",
			steps: []Promise{
				Await(step{op: opStep("a")}),
			},
		},
		{
			name: "sequence",
			steps: []Promise{
				Await(step{op: opStep("a")}),
				Await(step{op: opStep("b")}),
			},
		},
		{
			name: "sequence (then)",
			steps: []Promise{
				Await(step{op: opStep("a")}),
			},
		},
	}
	_ = tests
}

func TestParallelismDSL(t *testing.T) {
	t.Run("It works with a basic all", func(t *testing.T) {
		chain := All(
			step{op: opStep("a")},
		)
		_ = chain

		//require.Equal(t, 1, len(ops))
		//require.Equal(t, 1, len(ops[0]))
		//require.Equal(t, opStep("a"), s[0][0])
	})
}

func opcodes(items []step) [][]state.GeneratorOpcode {
	for _, block := range items {
		// Iterate through
		_ = block
	}
	return nil
}

type mode string

const (
	modeAll  mode = "all"
	modeRace mode = "race"
)

// stack is a type alias for faster typing
type stack driver.FunctionStack

type Promise interface {
	Then(p ...Promise)
}

// await awaits on a single step, informing our DSL that we must invoke
// this step to continue.
//
// This results in a single step call.
func Await(s step) Promise {
	return &then{
		mode:    modeAll,
		promise: []step{s},
	}
}

// all runs all steps then continues.
func All(s ...step) Promise {
	return &then{
		mode:    modeAll,
		promise: s,
	}
}

type then struct {
	mode mode
	// promise are the steps that must execute for onResolve steps
	// to continue.
	promise []step
	// onResolve are the steps calle when this the is resolved
	onResolve *then
}

func (t *then) Then(p ...Promise) {
}

type step struct {
	op            state.GeneratorOpcode
	delay         time.Duration
	expectedStack stack
}

func opStep(id string) state.GeneratorOpcode {
	byt, _ := json.Marshal(id)
	return state.GeneratorOpcode{
		Op:   enums.OpcodeStep,
		ID:   id,
		Name: id,
		Data: byt,
	}
}

func opSleep(id string, t time.Duration) state.GeneratorOpcode {
	return state.GeneratorOpcode{
		Op:   enums.OpcodeSleep,
		ID:   id,
		Name: t.String(),
	}
}
