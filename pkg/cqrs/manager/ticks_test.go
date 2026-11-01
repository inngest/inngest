package manager

import (
	"strconv"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/stretchr/testify/require"
)

func tickRow(id string, parent string, attrs map[string]any) testSpanRow {
	return testSpanRow{
		dynamicSpanID: id,
		parentSpanID:  parent,
		fragments: []map[string]any{{
			"span_id":    "otel-" + id,
			"name":       meta.SpanNameTick,
			"attributes": attrs,
		}},
	}
}

func millis(t time.Time) string {
	return strconv.FormatInt(t.UnixMilli(), 10)
}

func childNamed(span *cqrs.OtelSpan, id string) *cqrs.OtelSpan {
	for _, child := range span.Children {
		if child.SpanID == id {
			return child
		}
	}

	return nil
}

// Ticks never become rows: they ride on the row they belong to, and a read's
// tick says when its message arrived as well as when it was read.
func TestTicksRollOntoTheirParentRow(t *testing.T) {
	eachEncoding(t, func(t *testing.T, enc attrEncoding) {
		arrived := testStart.Add(time.Second)
		read := testStart.Add(2 * time.Second)

		root := buildRoot(t, enc, []testSpanRow{
			runRow(),
			stepRow("inbox", nil),
			stepRow("read", nil),
			tickRow("arrival", "inbox", map[string]any{
				meta.Attrs.TickID.Key():      "m1",
				meta.Attrs.TickSources.Key(): []string{"p1"},
				meta.Attrs.TickAt.Key():      millis(arrived),
			}),
			tickRow("took", "read", map[string]any{
				meta.Attrs.TickID.Key():     "m1",
				meta.Attrs.TickReadAt.Key(): millis(read),
			}),
		})

		require.Len(t, root.Children, 2, "ticks are not rows")

		inbox := childNamed(root, "inbox")
		require.NotNil(t, inbox)
		require.Len(t, inbox.Ticks, 1)
		require.Equal(t, "m1", inbox.Ticks[0].ID)
		require.Equal(t, []string{"p1"}, inbox.Ticks[0].Sources)
		require.Equal(t, arrived.UnixMilli(), inbox.Ticks[0].At.UnixMilli())
		require.Nil(t, inbox.Ticks[0].ReadAt)

		took := childNamed(root, "read")
		require.NotNil(t, took)
		require.Len(t, took.Ticks, 1)
		require.Equal(t, arrived.UnixMilli(), took.Ticks[0].At.UnixMilli(), "a read's tick has the arrival time")
		require.NotNil(t, took.Ticks[0].ReadAt)
		require.Equal(t, read.UnixMilli(), took.Ticks[0].ReadAt.UnixMilli())
	})
}

// A read that is still waiting when its run has ended (it lost a race) is
// drawn ended; before the run ends it is waiting.
func TestWaitingReadIsEndedWithItsRun(t *testing.T) {
	waiting := map[string]any{
		meta.Attrs.StepOp.Key():        enums.OpcodeSubscribeWait.String(),
		meta.Attrs.DynamicStatus.Key(): enums.StepStatusWaiting.String(),
	}
	ended := map[string]any{meta.Attrs.DynamicStatus.Key(): enums.StepStatusCompleted.String()}

	eachEncoding(t, func(t *testing.T, enc attrEncoding) {
		running := runRow()
		root := buildRoot(t, enc, []testSpanRow{running, stepRow("read", waiting)})
		require.Equal(t, enums.StepStatusWaiting, childNamed(root, "read").Status)

		finished := runRow()
		finished.fragments[0]["attributes"] = ended
		root = buildRoot(t, enc, []testSpanRow{finished, stepRow("read", waiting)})
		require.True(t, childNamed(root, "read").Status.IsEnded(), "status %s", childNamed(root, "read").Status)
	})
}

// An inbox that is never closed is drawn ended with its run; while the run is
// going it is running.
func TestOpenInboxRowIsEndedWithItsRun(t *testing.T) {
	open := map[string]any{
		meta.Attrs.StepOp.Key():        enums.OpcodeSubscribe.String(),
		meta.Attrs.DynamicStatus.Key(): enums.StepStatusRunning.String(),
	}
	ended := map[string]any{
		meta.Attrs.DynamicStatus.Key(): enums.StepStatusCompleted.String(),
		meta.Attrs.EndedAt.Key():       millis(testStart.Add(time.Minute)),
	}

	eachEncoding(t, func(t *testing.T, enc attrEncoding) {
		root := buildRoot(t, enc, []testSpanRow{runRow(), stepRow("inbox", open)})
		require.Equal(t, enums.StepStatusRunning, childNamed(root, "inbox").Status)

		finished := runRow()
		finished.fragments[0]["attributes"] = ended
		root = buildRoot(t, enc, []testSpanRow{finished, stepRow("inbox", open)})

		row := childNamed(root, "inbox")
		require.Equal(t, enums.StepStatusCompleted, row.Status)
		require.NotNil(t, row.Attributes.EndedAt, "it ends when the run does")
		require.Equal(t, testStart.Add(time.Minute).UnixMilli(), row.Attributes.EndedAt.UnixMilli())
	})
}
