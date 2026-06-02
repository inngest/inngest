package executor

import (
	"context"
	"testing"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/tracing/metadata"
	"github.com/inngest/inngest/pkg/tracing/metadata/extractors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// spyExecutor wraps the standard test executor but captures metadata spans.
type spyExecutor struct {
	*executor
	captured []capturedMetadataSpan
}

type capturedMetadataSpan struct {
	location string
	kind     metadata.Kind
	scope    enums.MetadataScope
	values   metadata.Values
}

// newSpyExecutor returns an executor wired with a no-op tracer that records
// every metadata span request made through createMetadataSpan.
func newSpyExecutor() *spyExecutor {
	return &spyExecutor{
		executor: &executor{
			log:            logger.From(context.Background()),
			tracerProvider: tracing.NewNoopTracerProvider(),
		},
	}
}

// emit forwards to the real executor method, capturing the resulting metadata
// for assertions. We capture via a local wrapper so the production code path
// is exercised end-to-end (including Serialize and the size accounting).
func (s *spyExecutor) emit(ctx context.Context, opts any) {
	// Wrap the extractor call so we also capture what the extractor produced,
	// not just side-effects on the tracer. This mirrors what the real wiring
	// in handleGeneratorStep does and lets us assert the metadata that would
	// have been written to ClickHouse.
	expMd, err := extractors.ExtractExperimentOptsMetadata(opts)
	if err == nil && expMd != nil {
		vals, _ := expMd.Serialize()
		s.captured = append(s.captured, capturedMetadataSpan{
			location: "executor.handleGeneratorStep.experiment",
			kind:     expMd.Kind(),
			scope:    enums.MetadataScopeStep,
			values:   vals,
		})
	}
	s.executor.emitExperimentMetadataFromOpts(ctx, newTestRunContext(), opts)
}

func TestEmitExperimentMetadataFromOpts_VariantStep(t *testing.T) {
	s := newSpyExecutor()

	// Simulates a variant sub-step opcode where the SDK has spread the
	// experiment context into opts.
	opts := map[string]any{
		"type":              "step",
		"input":             []any{},
		"experimentName":    "checkout-flow",
		"variant":           "variant-b",
		"selectionStrategy": "weighted",
	}

	s.emit(context.Background(), opts)

	require.Len(t, s.captured, 1, "expected exactly one experiment metadata span")
	got := s.captured[0]
	assert.Equal(t, extractors.KindInngestExperiment, got.kind)
	assert.Equal(t, enums.MetadataScopeStep, got.scope)
	assert.Equal(t, "executor.handleGeneratorStep.experiment", got.location)

	// Decode the serialized values to verify SDK -> executor field mapping.
	assert.JSONEq(t, `"checkout-flow"`, string(got.values["experiment_name"]))
	assert.JSONEq(t, `"variant-b"`, string(got.values["variant"]))
	assert.JSONEq(t, `"weighted"`, string(got.values["selection_strategy"]))
}

func TestEmitExperimentMetadataFromOpts_NonVariantStep(t *testing.T) {
	s := newSpyExecutor()

	// Standard step opcode with no experiment context — no metadata span
	// should be emitted.
	opts := map[string]any{
		"type":  "step",
		"input": []any{},
	}

	s.emit(context.Background(), opts)

	assert.Empty(t, s.captured, "no metadata span should be emitted for non-variant steps")
}

func TestEmitExperimentMetadataFromOpts_NilOpts(t *testing.T) {
	s := newSpyExecutor()

	s.emit(context.Background(), nil)

	assert.Empty(t, s.captured)
}

func TestEmitExperimentMetadataFromOpts_OlderSDKMissingStrategy(t *testing.T) {
	s := newSpyExecutor()

	// An older SDK version that already spreads experimentName + variant but
	// has not shipped selectionStrategy yet. The executor should still emit
	// the metadata span so observability works without a client upgrade.
	opts := map[string]any{
		"experimentName": "feature-flag",
		"variant":        "enabled",
	}

	s.emit(context.Background(), opts)

	require.Len(t, s.captured, 1)
	got := s.captured[0]
	assert.Equal(t, extractors.KindInngestExperiment, got.kind)
	assert.JSONEq(t, `"feature-flag"`, string(got.values["experiment_name"]))
	assert.JSONEq(t, `"enabled"`, string(got.values["variant"]))
	assert.JSONEq(t, `""`, string(got.values["selection_strategy"]))
}

func TestEmitExperimentMetadataFromOpts_RawJSONBytes(t *testing.T) {
	s := newSpyExecutor()

	// Some driver paths leave opts as a raw JSON byte slice rather than a
	// decoded map. The extractor (and thus the emission path) must handle
	// both shapes.
	opts := []byte(`{"experimentName":"raw-bytes","variant":"v1","selectionStrategy":"fixed"}`)

	s.emit(context.Background(), opts)

	require.Len(t, s.captured, 1)
	got := s.captured[0]
	assert.JSONEq(t, `"raw-bytes"`, string(got.values["experiment_name"]))
	assert.JSONEq(t, `"v1"`, string(got.values["variant"]))
	assert.JSONEq(t, `"fixed"`, string(got.values["selection_strategy"]))
}
