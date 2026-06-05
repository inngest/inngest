package tracing

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution/queue"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFinalizedStepDynamicSeed(t *testing.T) {
	t.Run("seed is the step ID bytes", func(t *testing.T) {
		stepID := "my-step-id"
		seed := FinalizedStepDynamicSeed(stepID)
		assert.Equal(t, []byte(stepID), seed)
	})

	t.Run("same step ID always produces the same seed", func(t *testing.T) {
		stepID := "stable-step"
		assert.Equal(t, FinalizedStepDynamicSeed(stepID), FinalizedStepDynamicSeed(stepID))
	})

	t.Run("different step IDs produce different seeds", func(t *testing.T) {
		assert.NotEqual(t, FinalizedStepDynamicSeed("step-a"), FinalizedStepDynamicSeed("step-b"))
	})
}

func TestRetryStepDynamicSeed(t *testing.T) {
	t.Run("seed encodes step ID and attempt", func(t *testing.T) {
		stepID := "my-step"
		attempt := 2
		seed := RetryStepDynamicSeed(stepID, attempt)
		assert.Equal(t, fmt.Appendf(nil, "%s:%d", stepID, attempt), seed)
	})

	t.Run("same step and attempt produces the same seed", func(t *testing.T) {
		assert.Equal(t, RetryStepDynamicSeed("step-x", 1), RetryStepDynamicSeed("step-x", 1))
	})

	t.Run("different attempts produce different seeds for the same step", func(t *testing.T) {
		assert.NotEqual(t, RetryStepDynamicSeed("step-x", 0), RetryStepDynamicSeed("step-x", 1))
		assert.NotEqual(t, RetryStepDynamicSeed("step-x", 1), RetryStepDynamicSeed("step-x", 2))
	})

	t.Run("different steps produce different seeds at the same attempt", func(t *testing.T) {
		assert.NotEqual(t, RetryStepDynamicSeed("step-a", 0), RetryStepDynamicSeed("step-b", 0))
	})
}

func TestFinalizedVsRetryStepSeedsDoNotCollide(t *testing.T) {
	// The finalized seed is based purely on step ID; the retry seed uses "stepID:attempt".
	// These must not collide so that finalized spans have stable IDs while retry
	// spans remain distinct per attempt.
	stepID := "my-step"
	for attempt := range 5 {
		finalized := FinalizedStepDynamicSeed(stepID)
		retry := RetryStepDynamicSeed(stepID, attempt)
		assert.NotEqual(t, finalized, retry,
			"finalized and retry seeds must differ (attempt=%d)", attempt)
	}
}

func TestFinalizedStepSpanRefFromMetadataAndStepID(t *testing.T) {
	runID := ulid.Make()
	md := &statev2.Metadata{
		ID: statev2.ID{
			RunID:      runID,
			FunctionID: uuid.New(),
		},
	}
	stepID := "test-step"

	t.Run("nil metadata returns nil", func(t *testing.T) {
		ref := FinalizedStepSpanRefFromMetadataAndStepID(nil, stepID)
		assert.Nil(t, ref)
	})

	t.Run("ref contains non-empty fields", func(t *testing.T) {
		ref := FinalizedStepSpanRefFromMetadataAndStepID(md, stepID)
		require.NotNil(t, ref)
		assert.NotEmpty(t, ref.DynamicSpanID)
		assert.NotEmpty(t, ref.DynamicSpanTraceParent)
		assert.NotEmpty(t, ref.TraceParent)
	})

	t.Run("dynamic_span_id is derived purely from step ID", func(t *testing.T) {
		// Two calls with different run IDs but the same step ID must produce
		// the same DynamicSpanID — this is the core invariant introduced by
		// EXE-1891: finalized step span IDs are keyed only on step ID.
		otherRunID := ulid.Make()
		otherMD := &statev2.Metadata{
			ID: statev2.ID{RunID: otherRunID},
		}

		ref1 := FinalizedStepSpanRefFromMetadataAndStepID(md, stepID)
		ref2 := FinalizedStepSpanRefFromMetadataAndStepID(otherMD, stepID)
		require.NotNil(t, ref1)
		require.NotNil(t, ref2)

		assert.Equal(t, ref1.DynamicSpanID, ref2.DynamicSpanID,
			"DynamicSpanID must be the same across runs when step ID is identical")
	})

	t.Run("dynamic_span_id matches DeterministicSpanConfig of the finalized seed", func(t *testing.T) {
		expected := DeterministicSpanConfig(FinalizedStepDynamicSeed(stepID)).SpanID.String()
		ref := FinalizedStepSpanRefFromMetadataAndStepID(md, stepID)
		require.NotNil(t, ref)
		assert.Equal(t, expected, ref.DynamicSpanID)
	})

	t.Run("DynamicSpanTraceParent uses the run trace and run span as parent", func(t *testing.T) {
		runCfg := DeterministicSpanConfig(md.ID.RunID[:])
		ref := FinalizedStepSpanRefFromMetadataAndStepID(md, stepID)
		require.NotNil(t, ref)

		expected := fmt.Sprintf("00-%s-%s-00", runCfg.TraceID.String(), runCfg.SpanID.String())
		assert.Equal(t, expected, ref.DynamicSpanTraceParent)
	})

	t.Run("TraceParent uses the run trace and the step span ID", func(t *testing.T) {
		runCfg := DeterministicSpanConfig(md.ID.RunID[:])
		stepSpanID := DeterministicSpanConfig(FinalizedStepDynamicSeed(stepID)).SpanID.String()
		ref := FinalizedStepSpanRefFromMetadataAndStepID(md, stepID)
		require.NotNil(t, ref)

		expected := fmt.Sprintf("00-%s-%s-00", runCfg.TraceID.String(), stepSpanID)
		assert.Equal(t, expected, ref.TraceParent)
	})

	t.Run("different steps produce different refs", func(t *testing.T) {
		ref1 := FinalizedStepSpanRefFromMetadataAndStepID(md, "step-one")
		ref2 := FinalizedStepSpanRefFromMetadataAndStepID(md, "step-two")
		require.NotNil(t, ref1)
		require.NotNil(t, ref2)
		assert.NotEqual(t, ref1.DynamicSpanID, ref2.DynamicSpanID)
	})
}

func TestRetryStepSpanRefFromMetadataAndStepID(t *testing.T) {
	runID := ulid.Make()
	md := &statev2.Metadata{
		ID: statev2.ID{RunID: runID},
	}
	stepID := "retry-step"

	t.Run("nil metadata returns nil", func(t *testing.T) {
		ref := RetryStepSpanRefFromMetadataAndStepID(nil, stepID, 0)
		assert.Nil(t, ref)
	})

	t.Run("ref contains non-empty fields", func(t *testing.T) {
		ref := RetryStepSpanRefFromMetadataAndStepID(md, stepID, 0)
		require.NotNil(t, ref)
		assert.NotEmpty(t, ref.DynamicSpanID)
		assert.NotEmpty(t, ref.DynamicSpanTraceParent)
		assert.NotEmpty(t, ref.TraceParent)
	})

	t.Run("dynamic_span_id matches DeterministicSpanConfig of the retry seed", func(t *testing.T) {
		for attempt := range 3 {
			expected := DeterministicSpanConfig(RetryStepDynamicSeed(stepID, attempt)).SpanID.String()
			ref := RetryStepSpanRefFromMetadataAndStepID(md, stepID, attempt)
			require.NotNil(t, ref)
			assert.Equal(t, expected, ref.DynamicSpanID, "attempt=%d", attempt)
		}
	})

	t.Run("different attempts produce different DynamicSpanIDs", func(t *testing.T) {
		ref0 := RetryStepSpanRefFromMetadataAndStepID(md, stepID, 0)
		ref1 := RetryStepSpanRefFromMetadataAndStepID(md, stepID, 1)
		ref2 := RetryStepSpanRefFromMetadataAndStepID(md, stepID, 2)
		require.NotNil(t, ref0)
		require.NotNil(t, ref1)
		require.NotNil(t, ref2)
		assert.NotEqual(t, ref0.DynamicSpanID, ref1.DynamicSpanID)
		assert.NotEqual(t, ref1.DynamicSpanID, ref2.DynamicSpanID)
		assert.NotEqual(t, ref0.DynamicSpanID, ref2.DynamicSpanID)
	})

	t.Run("retry ref differs from finalized ref for same step", func(t *testing.T) {
		retry := RetryStepSpanRefFromMetadataAndStepID(md, stepID, 0)
		finalized := FinalizedStepSpanRefFromMetadataAndStepID(md, stepID)
		require.NotNil(t, retry)
		require.NotNil(t, finalized)
		assert.NotEqual(t, retry.DynamicSpanID, finalized.DynamicSpanID)
	})
}

func TestNonStepDynamicSeed(t *testing.T) {
	t.Run("seed encodes nonstep prefix, group ID, and attempt", func(t *testing.T) {
		item := queue.Item{GroupID: "grp-123", Attempt: 2}
		seed := NonStepDynamicSeed(item)
		assert.Equal(t, []byte("nonstep:grp-123:2"), seed)
	})

	t.Run("same item always produces the same seed", func(t *testing.T) {
		item := queue.Item{GroupID: "grp-abc", Attempt: 0}
		assert.Equal(t, NonStepDynamicSeed(item), NonStepDynamicSeed(item))
	})

	t.Run("different group IDs produce different seeds", func(t *testing.T) {
		a := queue.Item{GroupID: "grp-a", Attempt: 0}
		b := queue.Item{GroupID: "grp-b", Attempt: 0}
		assert.NotEqual(t, NonStepDynamicSeed(a), NonStepDynamicSeed(b))
	})

	t.Run("different attempts produce different seeds", func(t *testing.T) {
		a := queue.Item{GroupID: "grp-x", Attempt: 0}
		b := queue.Item{GroupID: "grp-x", Attempt: 1}
		assert.NotEqual(t, NonStepDynamicSeed(a), NonStepDynamicSeed(b))
	})
}
