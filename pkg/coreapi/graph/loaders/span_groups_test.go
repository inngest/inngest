package loader

import (
	"context"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var spanGroupsBase = time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)

// groupedStep is a run-level step span queued at `at` seconds that ran for
// `secs` seconds, or is still running if status isn't an ended status.
func groupedStep(id string, at, secs int, status enums.StepStatus, path ...meta.SpanPathElement) *cqrs.OtelSpan {
	queued := spanGroupsBase.Add(time.Duration(at) * time.Second)
	attrs := &meta.ExtractedValues{
		StepID:        new(id),
		StepName:      new(id),
		DynamicStatus: &status,
		QueuedAt:      &queued,
		StartedAt:     &queued,
	}
	if status != enums.StepStatusRunning {
		attrs.EndedAt = new(queued.Add(time.Duration(secs) * time.Second))
	}
	if len(path) > 0 {
		attrs.StepSpanPath = &path
	}

	return &cqrs.OtelSpan{
		RawOtelSpan: cqrs.RawOtelSpan{Name: meta.SpanNameStep, SpanID: id},
		Attributes:  attrs,
	}
}

func convertGroupedRun(t *testing.T, children ...*cqrs.OtelSpan) *models.RunTraceSpan {
	t.Helper()

	run := &cqrs.OtelSpan{
		RawOtelSpan: cqrs.RawOtelSpan{Name: meta.SpanNameRun, SpanID: "run"},
		Attributes:  &meta.ExtractedValues{QueuedAt: &spanGroupsBase},
		Children:    children,
	}

	result, err := ConvertRunSpan(context.Background(), run)
	require.NoError(t, err)
	return result
}

func childNames(span *models.RunTraceSpan) []string {
	names := make([]string, len(span.ChildrenSpans))
	for i, child := range span.ChildrenSpans {
		names[i] = child.Name
	}
	return names
}

func TestGroupBySpanPath(t *testing.T) {
	completed := enums.StepStatusCompleted
	failed := enums.StepStatusFailed
	running := enums.StepStatusRunning

	agent := meta.SpanPathElement{ID: "agent", Name: "Research agent", Kind: "agent"}
	search := meta.SpanPathElement{ID: "search", Name: "search tool"}

	t.Run("runs without span paths are unchanged", func(t *testing.T) {
		result := convertGroupedRun(t,
			groupedStep("b", 2, 1, completed),
			groupedStep("a", 0, 1, completed),
		)

		assert.Equal(t, []string{"b", "a"}, childNames(result))
		for _, child := range result.ChildrenSpans {
			assert.NotEqual(t, SpanGroupStepType, child.StepType)
			assert.Nil(t, child.ParentSpanID)
		}
	})

	t.Run("nests steps by path, sorted by queue time", func(t *testing.T) {
		result := convertGroupedRun(t,
			groupedStep("plan", 0, 1, completed, agent),
			groupedStep("retry-query", 2, 1, completed, agent, search),
			groupedStep("query", 1, 1, completed, agent, search),
			groupedStep("after", 3, 1, completed),
		)

		require.Equal(t, []string{"Research agent", "after"}, childNames(result))

		group := result.ChildrenSpans[0]
		assert.Equal(t, SpanGroupStepType, group.StepType)
		assert.Equal(t, spanGroupID([]meta.SpanPathElement{agent}), group.SpanID)
		assert.Regexp(t, `^span:[0-9a-f]{16}$`, group.SpanID)
		assert.Equal(t, "run", *group.ParentSpanID)
		assert.Nil(t, group.StepID)
		assert.Nil(t, group.StepOp)
		assert.Nil(t, group.OutputID)
		assert.Equal(t, "agent", *group.GroupKind)
		require.Equal(t, []string{"plan", "search tool"}, childNames(group))

		sub := group.ChildrenSpans[1]
		assert.Equal(t, SpanGroupStepType, sub.StepType)
		assert.Nil(t, sub.GroupKind)
		assert.Nil(t, sub.ChildrenSpans[0].GroupKind)
		assert.Equal(t, group.SpanID, *sub.ParentSpanID)
		assert.NotEqual(t, group.SpanID, sub.SpanID)
		assert.Equal(t, []string{"query", "retry-query"}, childNames(sub))
		assert.Equal(t, sub.SpanID, *sub.ChildrenSpans[0].ParentSpanID)
	})

	t.Run("the same path re-enters the same group", func(t *testing.T) {
		bg := meta.SpanPathElement{ID: "server", Name: "dev server"}
		result := convertGroupedRun(t,
			groupedStep("start", 0, 1, completed, bg),
			groupedStep("test", 1, 5, completed),
			groupedStep("kill", 6, 1, completed, bg),
		)

		require.Equal(t, []string{"dev server", "test"}, childNames(result))
		assert.Equal(t, []string{"start", "kill"}, childNames(result.ChildrenSpans[0]))
	})

	t.Run("a group with a different parent is a different group", func(t *testing.T) {
		result := convertGroupedRun(t,
			groupedStep("a", 0, 1, completed, search),
			groupedStep("b", 1, 1, completed, agent, search),
		)

		require.Equal(t, []string{"search tool", "Research agent"}, childNames(result))
		assert.NotEqual(t, result.ChildrenSpans[0].SpanID, result.ChildrenSpans[1].ChildrenSpans[0].SpanID)
	})

	t.Run("timing spans the children and status is the last to end", func(t *testing.T) {
		result := convertGroupedRun(t,
			groupedStep("attempt-1", 0, 2, failed, agent),
			groupedStep("attempt-2", 3, 4, completed, agent),
		)

		group := result.ChildrenSpans[0]
		assert.Equal(t, models.RunTraceSpanStatusCompleted, group.Status)
		assert.Equal(t, spanGroupsBase, group.QueuedAt)
		assert.Equal(t, spanGroupsBase, *group.StartedAt)
		assert.Equal(t, spanGroupsBase.Add(7*time.Second), *group.EndedAt)
		assert.Equal(t, 7000, *group.Duration)

		result = convertGroupedRun(t,
			groupedStep("long", 0, 9, completed, agent),
			groupedStep("short", 1, 1, failed, agent),
		)
		assert.Equal(t, models.RunTraceSpanStatusCompleted, result.ChildrenSpans[0].Status)

		result = convertGroupedRun(t,
			groupedStep("ok", 0, 1, completed, agent),
			groupedStep("broken", 1, 1, failed, agent),
		)
		assert.Equal(t, models.RunTraceSpanStatusFailed, result.ChildrenSpans[0].Status)
	})

	t.Run("a group with a running child is running", func(t *testing.T) {
		result := convertGroupedRun(t,
			groupedStep("done", 0, 1, completed, agent),
			groupedStep("busy", 2, 0, running, agent, search),
		)

		outer := result.ChildrenSpans[0]
		inner := outer.ChildrenSpans[1]
		assert.Equal(t, spanGroupsBase, outer.QueuedAt)
		assert.Equal(t, spanGroupsBase.Add(2*time.Second), inner.QueuedAt)
		for _, group := range []*models.RunTraceSpan{outer, inner} {
			assert.Equal(t, models.RunTraceSpanStatusRunning, group.Status)
			assert.Equal(t, group.QueuedAt, *group.StartedAt)
			assert.Nil(t, group.EndedAt)
			assert.Nil(t, group.Duration)
		}
	})

	t.Run("an attempt that failed before the SDK answered joins the group of its step", func(t *testing.T) {
		stepless := func(id string, at int) *cqrs.OtelSpan {
			span := groupedStep(id, at, 1, failed)
			span.Name = meta.SpanNameNonStep
			span.Attributes.StepID = nil
			span.Attributes.GroupID = new("g")
			span.OutputID = new(id)
			return span
		}
		step := groupedStep("work", 2, 1, completed, agent)
		step.Attributes.GroupID = new("g")

		// The function's own error after the step shares its group ID too, but
		// isn't the step's, so it stays on the run.
		result := convertGroupedRun(t, stepless("lost", 0), step, stepless("final", 4))

		require.Equal(t, []string{"Research agent", "final"}, childNames(result))
		assert.Equal(t, []string{"lost", "work"}, childNames(result.ChildrenSpans[0]))
	})

	t.Run("reads the path from a step's execution", func(t *testing.T) {
		step := groupedStep("query", 0, 1, completed)
		execPath := []meta.SpanPathElement{agent}
		step.Children = []*cqrs.OtelSpan{{
			RawOtelSpan: cqrs.RawOtelSpan{Name: meta.SpanNameExecution, SpanID: "exec"},
			Attributes: &meta.ExtractedValues{
				DynamicStatus: &completed,
				StepID:        new("query"),
				StepSpanPath:  &execPath,
			},
		}}

		result := convertGroupedRun(t, step)
		require.Equal(t, []string{"Research agent"}, childNames(result))
		assert.Equal(t, []string{"query"}, childNames(result.ChildrenSpans[0]))
	})
}
