package responsecache

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/stretchr/testify/require"
)

func newTestCache(t *testing.T) *DiskCache {
	t.Helper()
	dir := t.TempDir()
	dc, err := NewDiskCache(DiskCacheOpt{
		Dir:             dir,
		TTL:             2 * time.Minute,
		CleanupInterval: time.Hour, // don't auto-clean in tests
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = dc.Close() })
	return dc
}

func TestDiskCache_Miss(t *testing.T) {
	dc := newTestCache(t)
	resp, err := dc.Get(context.Background(), "nonexistent")
	require.NoError(t, err)
	require.Nil(t, resp)
}

func TestDiskCache_RoundTrip(t *testing.T) {
	dc := newTestCache(t)
	ctx := context.Background()

	original := &state.DriverResponse{
		Step:           inngest.Step{ID: "step-1", Name: "my step"},
		Duration:       500 * time.Millisecond,
		RequestVersion: 1,
		Output:         json.RawMessage(`{"result":"ok"}`),
		OutputSize:     15,
		StatusCode:     200,
		SDK:            "go:v0.1.0",
	}

	require.NoError(t, dc.Set(ctx, "job-1:0", original))

	got, err := dc.Get(ctx, "job-1:0")
	require.NoError(t, err)
	require.NotNil(t, got)

	require.Equal(t, original.Step.ID, got.Step.ID)
	require.Equal(t, original.Step.Name, got.Step.Name)
	require.Equal(t, original.StatusCode, got.StatusCode)
	require.Equal(t, original.SDK, got.SDK)
	require.Equal(t, original.RequestVersion, got.RequestVersion)
	require.Equal(t, original.OutputSize, got.OutputSize)

	// Output round-trips as json.RawMessage.
	outBytes, err := json.Marshal(got.Output)
	require.NoError(t, err)
	require.JSONEq(t, `{"result":"ok"}`, string(outBytes))
}

func TestDiskCache_RoundTripWithError(t *testing.T) {
	dc := newTestCache(t)
	ctx := context.Background()

	errStr := "something went wrong"
	original := &state.DriverResponse{
		Step: inngest.Step{ID: "step-err"},
		Err:  &errStr,
		UserError: &state.UserError{
			Name:    "StepError",
			Message: "bad input",
		},
		NoRetry: true,
	}

	require.NoError(t, dc.Set(ctx, "job-err:1", original))

	got, err := dc.Get(ctx, "job-err:1")
	require.NoError(t, err)
	require.NotNil(t, got)
	require.NotNil(t, got.Err)
	require.Equal(t, errStr, *got.Err)
	require.NotNil(t, got.UserError)
	require.Equal(t, "StepError", got.UserError.Name)
	require.True(t, got.NoRetry)
}

func TestDiskCache_FinalPreserved(t *testing.T) {
	dc := newTestCache(t)
	ctx := context.Background()

	errStr := "fail"
	original := &state.DriverResponse{
		Step: inngest.Step{ID: "step-final"},
		Err:  &errStr,
	}
	original.SetFinal()
	require.True(t, original.Final())

	require.NoError(t, dc.Set(ctx, "job-final:0", original))

	got, err := dc.Get(ctx, "job-final:0")
	require.NoError(t, err)
	require.NotNil(t, got)
	require.True(t, got.Final())
	require.False(t, got.Retryable())
}

func TestDiskCache_OutputTypes(t *testing.T) {
	dc := newTestCache(t)
	ctx := context.Background()

	tests := []struct {
		name   string
		output any
	}{
		{"json.RawMessage", json.RawMessage(`{"key":"value"}`)},
		{"string", `{"key":"value"}`},
		{"[]byte", []byte(`{"key":"value"}`)},
		{"nil", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &state.DriverResponse{
				Step:   inngest.Step{ID: "step-" + tt.name},
				Output: tt.output,
			}

			key := "job-" + tt.name + ":0"
			require.NoError(t, dc.Set(ctx, key, resp))

			got, err := dc.Get(ctx, key)
			require.NoError(t, err)
			require.NotNil(t, got)

			if tt.output == nil {
				require.Nil(t, got.Output)
			} else {
				outBytes, err := json.Marshal(got.Output)
				require.NoError(t, err)
				require.JSONEq(t, `{"key":"value"}`, string(outBytes))
			}
		})
	}
}

func TestDiskCache_CorruptFile(t *testing.T) {
	dc := newTestCache(t)
	ctx := context.Background()

	// Write garbage to the expected path.
	path := dc.pathFor("bad-key")
	require.NoError(t, os.WriteFile(path, []byte("not json"), 0600))

	got, err := dc.Get(ctx, "bad-key")
	require.NoError(t, err)
	require.Nil(t, got)

	// The corrupt file should have been removed.
	_, statErr := os.Stat(path)
	require.True(t, os.IsNotExist(statErr))
}

func TestDiskCache_Cleanup(t *testing.T) {
	dir := t.TempDir()
	dc, err := NewDiskCache(DiskCacheOpt{
		Dir:             dir,
		TTL:             50 * time.Millisecond,
		CleanupInterval: time.Hour,
	})
	require.NoError(t, err)
	defer dc.Close()

	ctx := context.Background()
	require.NoError(t, dc.Set(ctx, "old:0", &state.DriverResponse{
		Step: inngest.Step{ID: "old"},
	}))

	// Wait for the entry to age past the TTL.
	time.Sleep(100 * time.Millisecond)

	dc.cleanup()

	got, err := dc.Get(ctx, "old:0")
	require.NoError(t, err)
	require.Nil(t, got, "expired entry should have been cleaned up")
}

func TestDiskCache_CleanupKeepsRecent(t *testing.T) {
	dir := t.TempDir()
	dc, err := NewDiskCache(DiskCacheOpt{
		Dir:             dir,
		TTL:             10 * time.Minute,
		CleanupInterval: time.Hour,
	})
	require.NoError(t, err)
	defer dc.Close()

	ctx := context.Background()
	require.NoError(t, dc.Set(ctx, "recent:0", &state.DriverResponse{
		Step: inngest.Step{ID: "recent"},
	}))

	dc.cleanup()

	got, err := dc.Get(ctx, "recent:0")
	require.NoError(t, err)
	require.NotNil(t, got, "recent entry should survive cleanup")
}

func TestDiskCache_GeneratorOpcodes(t *testing.T) {
	dc := newTestCache(t)
	ctx := context.Background()

	original := &state.DriverResponse{
		Step:       inngest.Step{ID: "step-gen"},
		StatusCode: 206,
		Generator: []*state.GeneratorOpcode{
			{
				ID:   "hashed-id-1",
				Name: "my-step",
				Data: json.RawMessage(`{"output":"data"}`),
			},
		},
	}

	require.NoError(t, dc.Set(ctx, "job-gen:0", original))

	got, err := dc.Get(ctx, "job-gen:0")
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Len(t, got.Generator, 1)
	require.Equal(t, "hashed-id-1", got.Generator[0].ID)
	require.Equal(t, "my-step", got.Generator[0].Name)
	require.JSONEq(t, `{"output":"data"}`, string(got.Generator[0].Data))
}

func TestDiskCache_DoesNotMutateOriginal(t *testing.T) {
	dc := newTestCache(t)
	ctx := context.Background()

	m := map[string]any{"key": "value"}
	original := &state.DriverResponse{
		Step:   inngest.Step{ID: "step-nomut"},
		Output: m,
	}

	require.NoError(t, dc.Set(ctx, "job-nomut:0", original))

	// The original Output should still be a map, not json.RawMessage.
	_, ok := original.Output.(map[string]any)
	require.True(t, ok, "Set must not mutate the caller's response")
}

func TestDiskCache_AtomicWrite(t *testing.T) {
	dc := newTestCache(t)
	ctx := context.Background()

	// Write initial value.
	require.NoError(t, dc.Set(ctx, "atomic:0", &state.DriverResponse{
		Step: inngest.Step{ID: "v1"},
	}))

	// Overwrite with new value.
	require.NoError(t, dc.Set(ctx, "atomic:0", &state.DriverResponse{
		Step: inngest.Step{ID: "v2"},
	}))

	got, err := dc.Get(ctx, "atomic:0")
	require.NoError(t, err)
	require.Equal(t, "v2", got.Step.ID)

	// No leftover .tmp files.
	entries, err := os.ReadDir(dc.dir)
	require.NoError(t, err)
	for _, e := range entries {
		require.False(t, filepath.Ext(e.Name()) == ".tmp", "temp file should not remain")
	}
}
