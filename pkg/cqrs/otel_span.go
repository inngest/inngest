package cqrs

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/inngest/inngest/pkg/tracing/meta"
)

// ApplyExtractedSpanAttributes parses span's raw attribute map via
// meta.ExtractTypedValues and promotes the values every span-tree builder
// needs onto the OtelSpan struct itself: status, identity overrides
// (AppID/FunctionID/RunID/DebugRunID/DebugSessionID), timing overrides
// (StartTime/EndTime), and the dropped flag. This is identical regardless
// of how the row was assembled — both pkg/cqrs/manager (dynamic,
// fragment-merged spans) and pkg/cqrs/duckdbquery (flat spans) call it
// immediately after building a span's RawOtelSpan.
func ApplyExtractedSpanAttributes(ctx context.Context, span *OtelSpan) error {
	extracted, err := meta.ExtractTypedValues(ctx, span.RawOtelSpan.Attributes)
	if err != nil {
		return fmt.Errorf("error extracting typed values from span attributes: %w", err)
	}
	span.Attributes = extracted

	if span.Attributes.DynamicStatus != nil {
		span.Status = *span.Attributes.DynamicStatus
	}
	if span.Attributes.AppID != nil {
		span.AppID = *span.Attributes.AppID
	}
	if span.Attributes.FunctionID != nil {
		span.FunctionID = *span.Attributes.FunctionID
	}
	if span.Attributes.RunID != nil {
		span.RunID = *span.Attributes.RunID
	}
	if span.Attributes.DebugRunID != nil {
		span.DebugRunID = *span.Attributes.DebugRunID
	}
	if span.Attributes.DebugSessionID != nil {
		span.DebugSessionID = *span.Attributes.DebugSessionID
	}
	if span.Attributes.StartedAt != nil {
		span.StartTime = *span.Attributes.StartedAt
	}
	if span.Attributes.EndedAt != nil {
		span.EndTime = *span.Attributes.EndedAt
	}
	if span.Attributes.DropSpan != nil && *span.Attributes.DropSpan {
		span.MarkedAsDropped = true
	}
	return nil
}

// isWaitForEventOutput reports whether o is shaped like a
// step.waitForEvent output ({name, data, ts}), which — unlike every other
// step output — is not wrapped in a {"data": ...}/{"error": ...} envelope
// and must be returned as-is.
func isWaitForEventOutput(o map[string]any) bool {
	_, name := o["name"]
	_, data := o["data"]
	_, ts := o["ts"]
	return name && data && ts
}

// UnwrapSpanOutputEnvelope applies one candidate row's raw output/input
// bytes onto so, mirroring the "preview" span exporter's envelope shape
// (pkg/execution/dualwrite/tracing.go and pkg/tracing/tracer_sqlc.go's
// dbExporter both write this shape): output is either a
// step.waitForEvent-shaped value (returned as-is, so.Data left set to the
// raw bytes), an {"error": ...} envelope (so.IsError=true, so.Data set to
// the error value), an {"data": ...} envelope (so.Data set to the success
// value), or an unrecognized shape (so.Data left as the raw bytes).
// done reports whether the caller should stop considering further
// candidate rows — true exactly when this row was wait-for-event-shaped,
// matching the early-return every existing caller already relies on.
func UnwrapSpanOutputEnvelope(so *SpanOutput, output, input []byte) (done bool) {
	if len(input) > 0 {
		so.Input = input
	}
	if len(output) == 0 {
		return false
	}

	var m map[string]any
	so.Data = output
	if err := json.Unmarshal(so.Data, &m); err != nil || m == nil {
		return false
	}
	if isWaitForEventOutput(m) {
		return true
	}
	if errData, ok := m["error"]; ok {
		so.IsError = true
		so.Data, _ = json.Marshal(errData)
	} else if successData, ok := m["data"]; ok {
		so.Data, _ = json.Marshal(successData)
	}
	return false
}
