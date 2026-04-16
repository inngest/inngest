package state

import (
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWaitForEventOpts_Expires(t *testing.T) {
	t.Run("accepts duration", func(t *testing.T) {
		opts := WaitForEventOpts{Timeout: "3d"}
		now := time.Now()
		got, err := opts.Expires()
		require.NoError(t, err)
		assert.WithinDuration(t, now.Add(3*24*time.Hour), got, time.Second)
	})

	t.Run("empty timeout returns now", func(t *testing.T) {
		opts := WaitForEventOpts{Timeout: ""}
		got, err := opts.Expires()
		require.NoError(t, err)
		assert.WithinDuration(t, time.Now(), got, time.Second)
	})
}

func TestSignalOpts_Expires(t *testing.T) {
	t.Run("accepts duration", func(t *testing.T) {
		opts := SignalOpts{Timeout: "3d"}
		now := time.Now()
		got, err := opts.Expires()
		require.NoError(t, err)
		assert.WithinDuration(t, now.Add(3*24*time.Hour), got, time.Second)
	})

	t.Run("accepts RFC 3339", func(t *testing.T) {
		opts := SignalOpts{Timeout: "2030-01-02T03:04:05Z"}
		want, err := time.Parse(time.RFC3339, "2030-01-02T03:04:05Z")
		require.NoError(t, err)
		got, err := opts.Expires()
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("empty defaults to ~1 year", func(t *testing.T) {
		opts := SignalOpts{}
		now := time.Now()
		got, err := opts.Expires()
		require.NoError(t, err)
		assert.WithinDuration(t, now.AddDate(1, 0, 0), got, time.Second)
	})
}

func TestInvokeFunctionOpts_Expires(t *testing.T) {
	t.Run("accepts duration", func(t *testing.T) {
		opts := InvokeFunctionOpts{Timeout: "3d"}
		now := time.Now()
		got, err := opts.Expires()
		require.NoError(t, err)
		assert.WithinDuration(t, now.Add(3*24*time.Hour), got, time.Second)
	})

	t.Run("accepts RFC 3339", func(t *testing.T) {
		opts := InvokeFunctionOpts{Timeout: "2030-01-02T03:04:05Z"}
		want, err := time.Parse(time.RFC3339, "2030-01-02T03:04:05Z")
		require.NoError(t, err)
		got, err := opts.Expires()
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("empty defaults to ~1 year", func(t *testing.T) {
		opts := InvokeFunctionOpts{}
		now := time.Now()
		got, err := opts.Expires()
		require.NoError(t, err)
		assert.WithinDuration(t, now.AddDate(1, 0, 0), got, time.Second)
	})
}

// TestDeferAddOpts pins down the first real behavior of OpcodeDeferAdd:
// the SDK emits a GeneratorOpcode whose Opts field carries the target
// companion ID and the user input passed to `step.defer(...)`, and the
// executor needs a typed accessor to read them.
//
// This mirrors the pattern used by every other opcode with options —
// see InvokeFunctionOpts(), WaitForEventOpts(), SignalOpts() in opcode.go.
//
// To make this pass:
//  1. Add a DeferAddOpts struct with `companion_id` and `input` JSON fields.
//  2. Add an UnmarshalAny method on it (copy the shape of InvokeFunctionOpts).
//  3. Add a DeferAddOpts() method on GeneratorOpcode that parses g.Opts.
func TestDeferAddOpts(t *testing.T) {
	g := GeneratorOpcode{
		Op: enums.OpcodeDeferAdd,
		ID: "deferred-step",
		Opts: map[string]any{
			"companion_id": "score",
			"input":        map[string]any{"user_id": "u_123"},
		},
	}

	opts, err := g.DeferAddOpts()
	require.NoError(t, err)
	require.Equal(t, "score", opts.CompanionID)
	require.JSONEq(t, `{"user_id":"u_123"}`, string(opts.Input))
}

func TestDeferCancelOpts(t *testing.T) {
	g := GeneratorOpcode{
		Op: enums.OpcodeDeferCancel,
		ID: "deferred-step",
		Opts: map[string]any{
			"companion_id": "score",
		},
	}

	opts, err := g.DeferCancelOpts()
	require.NoError(t, err)
	require.Equal(t, "score", opts.CompanionID)
}
