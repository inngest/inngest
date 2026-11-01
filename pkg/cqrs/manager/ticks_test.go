package manager

import (
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/inngest/inngest/pkg/tracing/metadata"
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

// A message is a child span of the row it belongs to: it is drawn dimmed in
// the same group as its parent, an arrival spans no time, and a read's message
// runs from its arrival to its read. A read row lists the ids it took.
func TestTicksBecomeDimmedChildSpans(t *testing.T) {
	eachEncoding(t, func(t *testing.T, enc attrEncoding) {
		arrived := testStart.Add(time.Second)
		read := testStart.Add(2 * time.Second)
		group := []meta.SpanPathElement{{ID: "next", Name: "next", Kind: "tag", Origin: "inngest@1.0.0"}}
		path, err := json.Marshal(group)
		require.NoError(t, err)

		root := buildRoot(t, enc, []testSpanRow{
			runRow(),
			stepRow("inbox", map[string]any{
				meta.Attrs.StepSpanPath.Key(): string(path),
				meta.Attrs.StepOrigin.Key():   "inngest@1.0.0",
			}),
			stepRow("read", map[string]any{
				meta.Attrs.StepSpanPath.Key(): string(path),
				meta.Attrs.StepOrigin.Key():   "inngest@1.0.0",
			}),
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

		require.Len(t, root.Children, 4, "a message is a sibling of its row, so it lands in its group")

		inbox := childNamed(root, "inbox")
		require.NotNil(t, inbox)
		require.Empty(t, inbox.Children, "under the row it would be taken for an attempt")

		arrival := childNamed(root, "arrival")
		require.NotNil(t, arrival)
		require.Equal(t, "m1", *arrival.Attributes.StepName)
		require.Equal(t, group, *arrival.Attributes.StepSpanPath, "it is in its parent's group")
		require.Equal(t, "inngest@1.0.0", *arrival.Attributes.StepOrigin, "it is dimmed")
		require.Equal(t, arrived.UnixMilli(), arrival.Attributes.StartedAt.UnixMilli())
		require.Equal(t, arrived.UnixMilli(), arrival.Attributes.EndedAt.UnixMilli())
		require.True(t, arrival.Status.IsEnded())

		took := childNamed(root, "read")
		require.NotNil(t, took)
		require.Empty(t, took.Children)

		message := childNamed(root, "took")
		require.NotNil(t, message)
		require.Equal(t, arrived.UnixMilli(), message.Attributes.StartedAt.UnixMilli(), "arrived")
		require.Equal(t, read.UnixMilli(), message.Attributes.EndedAt.UnixMilli(), "read")

		// The arrival is also a tick on the row of the inbox, which the trace
		// draws as a circle on its bar; a read's tick has the arrival time too.
		require.Len(t, inbox.Ticks, 1)
		require.Equal(t, "m1", inbox.Ticks[0].ID)
		require.Equal(t, []string{"p1"}, inbox.Ticks[0].Sources)
		require.Equal(t, arrived.UnixMilli(), inbox.Ticks[0].At.UnixMilli())
		require.NotNil(t, inbox.Ticks[0].ReadAt, "an arrival says when its message was read")
		require.Equal(t, read.UnixMilli(), inbox.Ticks[0].ReadAt.UnixMilli())

		require.Len(t, took.Ticks, 1)
		require.Equal(t, arrived.UnixMilli(), took.Ticks[0].At.UnixMilli())
		require.NotNil(t, took.Ticks[0].ReadAt)
		require.Equal(t, read.UnixMilli(), took.Ticks[0].ReadAt.UnixMilli())

		require.Len(t, took.Metadata, 1)
		require.Equal(t, metadata.Kind("inngest.inbox"), took.Metadata[0].Kind)
		require.JSONEq(t, `["m1"]`, string(took.Metadata[0].Values["messages"]))
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
