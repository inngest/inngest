package state

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/consts"
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

func TestDeferAddOpts_Validate(t *testing.T) {
	t.Run("rejects empty FnSlug", func(t *testing.T) {
		opts := DeferAddOpts{Input: json.RawMessage(`{}`)}
		require.Error(t, opts.Validate())
	})

	t.Run("rejects empty Input", func(t *testing.T) {
		opts := DeferAddOpts{FnSlug: "fn"}
		require.Error(t, opts.Validate())
	})

	t.Run("accepts valid opts", func(t *testing.T) {
		opts := DeferAddOpts{FnSlug: "fn", Input: json.RawMessage(`{"k":"v"}`)}
		require.NoError(t, opts.Validate())
	})

	t.Run("rejects Input larger than MaxStepInputSize", func(t *testing.T) {
		// Build an Input one byte over the limit. Padding is filler — the
		// validator only cares about byte length.
		oversized := bytes.Repeat([]byte("x"), consts.MaxStepInputSize+1)
		opts := DeferAddOpts{FnSlug: "fn", Input: json.RawMessage(oversized)}
		err := opts.Validate()
		require.ErrorIs(t, err, ErrStepInputTooLarge,
			"oversized defer Input must be rejected at validation, not silently stored in Redis")
	})

	t.Run("accepts Input at exactly MaxStepInputSize", func(t *testing.T) {
		// Boundary: equal-to-limit must pass; only strictly greater fails.
		atLimit := bytes.Repeat([]byte("x"), consts.MaxStepInputSize)
		opts := DeferAddOpts{FnSlug: "fn", Input: json.RawMessage(atLimit)}
		require.NoError(t, opts.Validate())
	})
}
