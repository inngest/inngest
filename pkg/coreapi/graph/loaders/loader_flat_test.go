package loader

import (
	"context"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/stretchr/testify/require"
)

type flatSpanManagerStub struct{ cqrs.Manager }

func (flatSpanManagerStub) FlatSpans() bool { return true }

func TestNewLoadersSelectsFlatConverterForFlatSpanSource(t *testing.T) {
	loaders := NewLoaders(LoaderParams{DB: flatSpanManagerStub{}})
	require.NotNil(t, loaders.RunTraceLoader)
}

// discoverySpanRoot builds a minimal run span with a single
// step-discovery child — convertRunSpanToGQL (the rollup-aware converter)
// omits discovery spans from ChildrenSpans, while convertFlatSpanToGQL
// (trace_flat.go) carries no such logic, so the two converters disagree on
// this shape. Used below to prove ConvertRunSpanFor's dispatch actually
// selects between them rather than always picking one.
func discoverySpanRoot() *cqrs.OtelSpan {
	queuedAt := time.Date(2026, 4, 9, 12, 0, 0, 0, time.UTC)
	discovery := &cqrs.OtelSpan{
		RawOtelSpan: cqrs.RawOtelSpan{
			Name:      meta.SpanNameStepDiscovery,
			SpanID:    "discovery-span",
			StartTime: queuedAt,
		},
		Attributes: &meta.ExtractedValues{QueuedAt: &queuedAt},
	}
	return &cqrs.OtelSpan{
		RawOtelSpan: cqrs.RawOtelSpan{
			Name:      meta.SpanNameRun,
			SpanID:    "run-span",
			StartTime: queuedAt,
		},
		Attributes: &meta.ExtractedValues{QueuedAt: &queuedAt},
		Children:   []*cqrs.OtelSpan{discovery},
	}
}

func TestConvertRunSpanForUsesRollupConverterByDefault(t *testing.T) {
	result, err := ConvertRunSpanFor(context.Background(), nil, discoverySpanRoot())
	require.NoError(t, err)
	require.Empty(t, result.ChildrenSpans, "rollup-aware converter must omit the discovery child")
}

func TestConvertRunSpanForUsesFlatConverterForFlatSpanSource(t *testing.T) {
	result, err := ConvertRunSpanFor(context.Background(), flatSpanManagerStub{}, discoverySpanRoot())
	require.NoError(t, err)
	require.Len(t, result.ChildrenSpans, 1, "flat converter must keep the discovery child")
	require.Equal(t, "discovery-span", result.ChildrenSpans[0].SpanID)
}
