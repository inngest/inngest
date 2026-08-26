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
}
