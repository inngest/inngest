package tracing

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/inngest/inngest/pkg/tracing/metadata"
	"github.com/inngest/inngest/pkg/tracing/metadata/extractors"
	"github.com/stretchr/testify/require"
)

// mockStructured implements metadata.Structured with configurable behavior.
type mockStructured struct {
	kind         metadata.Kind
	values       metadata.Values
	serializeErr error
}

func (m *mockStructured) Kind() metadata.Kind      { return m.kind }
func (m *mockStructured) Op() enums.MetadataOpcode { return enums.MetadataOpcodeMerge }
func (m *mockStructured) Serialize() (metadata.Values, error) {
	if m.serializeErr != nil {
		return nil, m.serializeErr
	}
	return m.values, nil
}

// makeValues creates a Values map with a single key whose total size (key + value) equals targetSize.
func makeValues(targetSize int) metadata.Values {
	if targetSize <= 0 {
		return metadata.Values{}
	}
	key := "k"
	valSize := targetSize - len(key)
	if valSize < 0 {
		return metadata.Values{key[:targetSize]: json.RawMessage{}}
	}
	return metadata.Values{
		key: json.RawMessage(strings.Repeat("x", valSize)),
	}
}

func TestCreateMetadataSpan_SpanExactlyAtLimit(t *testing.T) {
	tp := NewNoopTracerProvider()

	md := &mockStructured{
		kind:   "test.kind",
		values: makeValues(consts.MaxMetadataSpanSize),
	}

	ref, err := CreateMetadataSpan(
		context.Background(), tp, &meta.SpanReference{},
		"test.location", "test", nil, md, enums.MetadataScopeStep,
	)
	require.NoError(t, err)
	require.NotNil(t, ref)
}

func TestCreateMetadataSpan_SpanOverLimit(t *testing.T) {
	md := &mockStructured{
		kind:   "test.kind",
		values: makeValues(consts.MaxMetadataSpanSize + 1),
	}

	ref, err := CreateMetadataSpan(
		context.Background(), nil, nil,
		"test.location", "test", nil, md, enums.MetadataScopeStep,
	)
	require.True(t, errors.Is(err, metadata.ErrMetadataSpanTooLarge))
	require.Nil(t, ref)
}

func TestCreateMetadataSpan_EmptyValues(t *testing.T) {
	tp := NewNoopTracerProvider()

	md := &mockStructured{
		kind:   "test.kind",
		values: metadata.Values{},
	}

	ref, err := CreateMetadataSpan(
		context.Background(), tp, &meta.SpanReference{},
		"test.location", "test", nil, md, enums.MetadataScopeStep,
	)
	require.NoError(t, err)
	require.NotNil(t, ref)
}

// captureTracerProvider records CreateSpan options for attribute assertions.
type captureTracerProvider struct {
	TracerProvider
	spans []*CreateSpanOptions
}

func (p *captureTracerProvider) CreateSpan(ctx context.Context, name string, opts *CreateSpanOptions) (*meta.SpanReference, error) {
	p.spans = append(p.spans, opts)
	return &meta.SpanReference{}, nil
}

func TestCreateMetadataSpanFromValues_EnrichesAIValues(t *testing.T) {
	tp := &captureTracerProvider{}

	values := metadata.Values{
		"input_tokens":  json.RawMessage(`100`),
		"output_tokens": json.RawMessage(`50`),
		"request_model": json.RawMessage(`"gpt-4o"`),
	}

	ref, err := CreateMetadataSpanFromValues(
		context.Background(), tp, &meta.SpanReference{},
		"test.location", "test", nil,
		extractors.KindInngestAI, enums.MetadataOpcodeMerge, values, enums.MetadataScopeStep,
	)
	require.NoError(t, err)
	require.NotNil(t, ref)

	require.Len(t, tp.spans, 1)
	got, ok := tp.spans[0].Attributes.Get(meta.Attrs.Metadata.Key()).(*metadata.Values)
	require.True(t, ok, "expected metadata values attribute on the span")
	require.Equal(t, json.RawMessage(`100`), (*got)["input_tokens"])
	require.Equal(t, json.RawMessage(`150`), (*got)["total_tokens"])
	require.Equal(t, json.RawMessage(`0.00075`), (*got)["estimated_cost"])
	_, hasLatency := (*got)["latency_ms"]
	require.False(t, hasLatency, "no latency should be derived")
}

func TestCreateMetadataSpanFromValues_DeleteOpNotEnriched(t *testing.T) {
	tp := &captureTracerProvider{}

	values := metadata.Values{
		"input_tokens":  json.RawMessage(`100`),
		"output_tokens": json.RawMessage(`50`),
		"request_model": json.RawMessage(`"gpt-4o"`),
	}

	ref, err := CreateMetadataSpanFromValues(
		context.Background(), tp, &meta.SpanReference{},
		"test.location", "test", nil,
		extractors.KindInngestAI, enums.MetadataOpcodeDelete, values, enums.MetadataScopeStep,
	)
	require.NoError(t, err)
	require.NotNil(t, ref)

	require.Len(t, tp.spans, 1)
	got, ok := tp.spans[0].Attributes.Get(meta.Attrs.Metadata.Key()).(*metadata.Values)
	require.True(t, ok, "expected metadata values attribute on the span")
	require.Equal(t, values, *got, "delete op values must not be enriched")
}

func TestCreateMetadataSpanFromValues_CumulativeLimitExceeded(t *testing.T) {
	spanSize := 50000
	stateMd := &statev2.Metadata{
		Metrics: statev2.RunMetrics{
			MetadataSize:       consts.MaxRunMetadataSize - spanSize + 1, // just over with new span
			MetadataSizeLoaded: consts.MaxRunMetadataSize - spanSize + 1,
		},
	}

	values := makeValues(spanSize)
	ref, err := CreateMetadataSpanFromValues(
		context.Background(), nil, nil,
		"test.location", "test", stateMd,
		"test.kind", enums.MetadataOpcodeMerge, values, enums.MetadataScopeStep,
	)
	require.ErrorIs(t, err, metadata.ErrRunMetadataSizeExceeded)
	require.Nil(t, ref)
	// In-memory counter should NOT have been incremented
	require.Equal(t, consts.MaxRunMetadataSize-spanSize+1, stateMd.Metrics.MetadataSize)
}

func TestCreateMetadataSpanFromValues_CumulativeLimitAccepted(t *testing.T) {
	tp := NewNoopTracerProvider()

	previousSize := consts.MaxRunMetadataSize - 50000
	stateMd := &statev2.Metadata{
		Metrics: statev2.RunMetrics{
			MetadataSize:       previousSize,
			MetadataSizeLoaded: previousSize,
		},
	}

	spanSize := 40000 // fits within remaining budget
	values := makeValues(spanSize)
	ref, err := CreateMetadataSpanFromValues(
		context.Background(), tp, &meta.SpanReference{},
		"test.location", "test", stateMd,
		"test.kind", enums.MetadataOpcodeMerge, values, enums.MetadataScopeStep,
	)
	require.NoError(t, err)
	require.NotNil(t, ref)
	// In-memory counter should be incremented
	require.Equal(t, previousSize+spanSize, stateMd.Metrics.MetadataSize)
}

func TestCreateMetadataSpanFromValues_CumulativeIncrementAcrossMultipleSpans(t *testing.T) {
	tp := NewNoopTracerProvider()

	// Start near the cumulative limit so a second small span pushes over it
	initialSize := consts.MaxRunMetadataSize - 50000
	stateMd := &statev2.Metadata{
		Metrics: statev2.RunMetrics{
			MetadataSize:       initialSize,
			MetadataSizeLoaded: initialSize,
		},
	}

	spanSize := 40000 // fits within remaining 50000 budget
	values := makeValues(spanSize)

	// First span — accepted
	ref, err := CreateMetadataSpanFromValues(
		context.Background(), tp, &meta.SpanReference{},
		"test.location", "test", stateMd,
		"test.kind", enums.MetadataOpcodeMerge, values, enums.MetadataScopeStep,
	)
	require.NoError(t, err)
	require.NotNil(t, ref)
	require.Equal(t, initialSize+spanSize, stateMd.Metrics.MetadataSize)

	// Second span of same size pushes over the cumulative limit (only 10000 remaining)
	ref, err = CreateMetadataSpanFromValues(
		context.Background(), tp, &meta.SpanReference{},
		"test.location", "test", stateMd,
		"test.kind2", enums.MetadataOpcodeMerge, values, enums.MetadataScopeStep,
	)
	require.ErrorIs(t, err, metadata.ErrRunMetadataSizeExceeded)
	require.Nil(t, ref)
	// Counter should still reflect only the first span
	require.Equal(t, initialSize+spanSize, stateMd.Metrics.MetadataSize)

	// Delta should be the first span only
	delta := stateMd.Metrics.MetadataSize - stateMd.Metrics.MetadataSizeLoaded
	require.Equal(t, spanSize, delta)
}
