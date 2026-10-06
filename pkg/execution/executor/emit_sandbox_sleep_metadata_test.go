package executor

import (
	"context"
	"testing"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/tracing/metadata"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmitSandboxSleepMetadataFromOpts(t *testing.T) {
	rec := &metadataEntryRecorder{}
	e := newTestExecutor()
	e.syncLifecycles = []execution.SyncLifecycleListener{rec}
	rc := newTestRunContext()
	rc.md.ID.RunID = ulid.Make()

	// A step.sleep between a CI command's polls, as inngest-js sends it inside
	// withSandboxStatement().
	gen := &state.GeneratorOpcode{
		ID: "hashed-sleep",
		Op: enums.OpcodeSleep,
		Opts: map[string]any{
			"duration": "1s",
			"sandboxStatement": map[string]any{
				"statement":      "commands.run",
				"statement_id":   "hashed-command",
				"statement_name": "test",
				"sandbox_id":     "22222222-2222-4222-8222-222222222222",
				"sandbox_name":   "ci-box",
			},
		},
	}

	e.emitSandboxSleepMetadataFromOpts(context.Background(), rc, gen)

	require.Len(t, rec.entries, 1)
	entry := rec.entries[0]
	assert.Equal(t, metadata.KindInngestSandbox, entry.Kind)
	assert.Equal(t, enums.MetadataScopeStep, entry.Scope)
	assert.Equal(t, "hashed-sleep", entry.StepHashedID)

	// It hangs off the sleep's own span, not a finalized step span.
	assert.Equal(t, tracing.SleepStepSpanRef(rc.md.ID.RunID, gen.ID).DynamicSpanID, entry.Parent.DynamicSpanID)

	assert.JSONEq(t, `"sleep"`, string(entry.Values["action"]))
	assert.JSONEq(t, `"internal"`, string(entry.Values["role"]))
	assert.JSONEq(t, `"commands.run"`, string(entry.Values["statement"]))
	assert.JSONEq(t, `"hashed-command"`, string(entry.Values["statement_id"]))
	assert.JSONEq(t, `"test"`, string(entry.Values["statement_name"]))
	assert.JSONEq(t, `"ci-box"`, string(entry.Values["sandbox_name"]))
}

func TestEmitSandboxSleepMetadataFromOpts_PlainSleep(t *testing.T) {
	rec := &metadataEntryRecorder{}
	e := newTestExecutor()
	e.syncLifecycles = []execution.SyncLifecycleListener{rec}

	e.emitSandboxSleepMetadataFromOpts(context.Background(), newTestRunContext(), &state.GeneratorOpcode{
		ID:   "hashed-sleep",
		Op:   enums.OpcodeSleep,
		Opts: map[string]any{"duration": "1s"},
	})

	assert.Empty(t, rec.entries)
}
