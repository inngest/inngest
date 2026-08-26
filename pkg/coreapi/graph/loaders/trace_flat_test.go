package loader

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestConvertFlatSpanToGQLMapsBasicFields(t *testing.T) {
	runID := ulid.MustNew(ulid.Now(), nil)
	span := &cqrs.OtelSpan{
		RawOtelSpan: cqrs.RawOtelSpan{
			Name:      "executor.step",
			SpanID:    "span-1",
			TraceID:   "trace-1",
			StartTime: time.Now(),
			EndTime:   time.Now().Add(time.Second),
		},
		RunID:      runID,
		AppID:      uuid.New(),
		FunctionID: uuid.New(),
		Attributes: &meta.ExtractedValues{},
	}

	gql, err := convertFlatSpanToGQL(context.Background(), span)
	require.NoError(t, err)
	require.Equal(t, "span-1", gql.SpanID)
	require.Equal(t, runID, gql.RunID)
	require.True(t, gql.IsRoot)
	require.Empty(t, gql.ChildrenSpans)
}

func TestConvertFlatSpanToGQLRecursesIntoChildren(t *testing.T) {
	child := &cqrs.OtelSpan{
		RawOtelSpan: cqrs.RawOtelSpan{Name: "executor.step", SpanID: "child"},
		Attributes:  &meta.ExtractedValues{},
	}
	root := &cqrs.OtelSpan{
		RawOtelSpan: cqrs.RawOtelSpan{Name: "executor.run", SpanID: "root"},
		Attributes:  &meta.ExtractedValues{},
		Children:    []*cqrs.OtelSpan{child},
	}

	gql, err := convertFlatSpanToGQL(context.Background(), root)
	require.NoError(t, err)
	require.Len(t, gql.ChildrenSpans, 1)
	require.Equal(t, "child", gql.ChildrenSpans[0].SpanID)
}

// TestConvertFlatSpanToGQLSetsDurationForEndedSpan proves Duration is
// computed from Attributes.StartedAt/EndedAt (via GetStartedAtTime/
// GetEndedAtTime) whenever the span's status is one RunTraceEnded
// recognizes — mirroring convertRunSpanToGQL's own computation, which this
// converter previously had no equivalent for, leaving every DuckDB-backed
// completed step reading as "still running" per the schema's own duration
// doc comment.
func TestConvertFlatSpanToGQLSetsDurationForEndedSpan(t *testing.T) {
	completed := enums.StepStatusCompleted
	started := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ended := started.Add(1500 * time.Millisecond)
	span := &cqrs.OtelSpan{
		RawOtelSpan: cqrs.RawOtelSpan{Name: "executor.step", SpanID: "span-1"},
		Attributes: &meta.ExtractedValues{
			DynamicStatus: &completed,
			StartedAt:     &started,
			EndedAt:       &ended,
		},
	}

	gql, err := convertFlatSpanToGQL(context.Background(), span)
	require.NoError(t, err)
	require.NotNil(t, gql.Duration)
	require.Equal(t, 1500, *gql.Duration)
}

// TestConvertFlatSpanToGQLLeavesDurationNilForRunningSpan proves a
// still-running span (no DynamicStatus set here, defaulting to Running —
// see convertFlatSpanToGQL's own default) never gets a Duration, matching
// the schema's "if null, it's still running" contract.
func TestConvertFlatSpanToGQLLeavesDurationNilForRunningSpan(t *testing.T) {
	started := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	span := &cqrs.OtelSpan{
		RawOtelSpan: cqrs.RawOtelSpan{Name: "executor.step", SpanID: "span-1"},
		Attributes: &meta.ExtractedValues{
			StartedAt: &started,
		},
	}

	gql, err := convertFlatSpanToGQL(context.Background(), span)
	require.NoError(t, err)
	require.Nil(t, gql.Duration)
}

func TestConvertFlatSpanToGQLMapsRunStepInfo(t *testing.T) {
	stepOp := enums.OpcodeStepRun
	stepRunType := "Step"
	span := &cqrs.OtelSpan{
		RawOtelSpan: cqrs.RawOtelSpan{Name: "executor.step", SpanID: "span-1"},
		Attributes: &meta.ExtractedValues{
			StepOp:      &stepOp,
			StepRunType: &stepRunType,
		},
	}

	gql, err := convertFlatSpanToGQL(context.Background(), span)
	require.NoError(t, err)
	require.NotNil(t, gql.StepOp)
	require.Equal(t, models.StepOpRun, *gql.StepOp)
	info, ok := gql.StepInfo.(*models.RunStepInfo)
	require.True(t, ok)
	require.Equal(t, &stepRunType, info.Type)
	require.Equal(t, stepRunType, gql.StepType, "StepType prefers StepRunType when set")
}

// TestConvertFlatSpanToGQLDerivesStepTypeFromStepOpWhenNoStepRunType proves
// the fallback branch convertRunSpanToGQL also has: StepType derives from
// StepOp's own string form when StepRunType isn't set (e.g. a sleep step,
// which has no "run type" concept at all).
func TestConvertFlatSpanToGQLDerivesStepTypeFromStepOpWhenNoStepRunType(t *testing.T) {
	stepOp := enums.OpcodeSleep
	span := &cqrs.OtelSpan{
		RawOtelSpan: cqrs.RawOtelSpan{Name: "executor.step", SpanID: "span-1"},
		Attributes:  &meta.ExtractedValues{StepOp: &stepOp},
	}

	gql, err := convertFlatSpanToGQL(context.Background(), span)
	require.NoError(t, err)
	require.Equal(t, "SLEEP", gql.StepType)
}

// TestConvertFlatSpanToGQLLeavesStepTypeEmptyWithNoStepOpOrRunType proves a
// span with neither StepOp nor StepRunType (e.g. executor.run itself) gets
// no StepType — required since it's a non-nullable String! in the schema
// and must default to "", not be left as some sentinel/error value.
func TestConvertFlatSpanToGQLLeavesStepTypeEmptyWithNoStepOpOrRunType(t *testing.T) {
	span := &cqrs.OtelSpan{
		RawOtelSpan: cqrs.RawOtelSpan{Name: "executor.run", SpanID: "root"},
		Attributes:  &meta.ExtractedValues{},
	}

	gql, err := convertFlatSpanToGQL(context.Background(), span)
	require.NoError(t, err)
	require.Empty(t, gql.StepType)
}
