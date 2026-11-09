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
func isWaitForEventOutput[V any](o map[string]V) bool {
	_, name := o["name"]
	_, data := o["data"]
	_, ts := o["ts"]
	return name && data && ts
}

// SplitSpanOutputEnvelope splits a stored span output into its data and
// error halves, the same way UnwrapSpanOutputEnvelope reads it: output is
// either a step.waitForEvent-shaped value (data is output as is,
// waitForEvent is true), an {"error": ...} envelope (errVal is the error
// value, data the "data" value if the envelope also has one), a
// {"data": ...} envelope (data is the data value), or anything else, IE
// invalid JSON, a non-object or an object w/ neither key (data is output as
// is). errVal is nil unless output is an error envelope, and both are nil
// for an empty output.
//
// data and errVal are the envelope values' stored bytes rather than a
// re-marshaled decoded value, so key order and number text (IE 1.10, ints
// past 2^53) come back as the SDK sent them.
func SplitSpanOutputEnvelope(output []byte) (data, errVal []byte, waitForEvent bool) {
	if len(output) == 0 {
		return nil, nil, false
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(output, &m); err != nil || m == nil {
		return output, nil, false
	}
	if isWaitForEventOutput(m) {
		return output, nil, true
	}
	errV, isErr := m["error"]
	dataV, hasData := m["data"]
	if !isErr && !hasData {
		return output, nil, false
	}
	if isErr {
		errVal = errV
	}
	if hasData {
		data = dataV
	}
	return data, errVal, false
}

// UnwrapSpanOutputEnvelope applies one candidate row's raw output/input
// bytes onto so, mirroring the "preview" span exporter's envelope shape
// (pkg/tracing/tracer_sqlc.go's dbExporter writes this shape): see
// SplitSpanOutputEnvelope for how output is read. An error sets
// so.IsError and so.Data to the error value, else so.Data is set to the data
// value. done reports whether the caller should stop considering further
// candidate rows: true exactly when this row was wait-for-event-shaped,
// matching the early-return every existing caller already relies on.
//
// The lake (pkg/duckdb) splits output at write time instead, so its readers
// use ApplySpanOutputColumns.
func UnwrapSpanOutputEnvelope(so *SpanOutput, output, input []byte) (done bool) {
	data, errVal, waitForEvent := SplitSpanOutputEnvelope(output)
	ApplySpanOutputColumns(so, data, errVal, input)
	return waitForEvent
}

// ApplySpanOutputColumns applies one row's already split output (see
// SplitSpanOutputEnvelope) and input onto so: an error sets so.IsError and
// so.Data to the error value, else a data value sets so.Data. A nil value
// leaves so as is.
func ApplySpanOutputColumns(so *SpanOutput, data, errVal, input []byte) {
	if len(input) > 0 {
		so.Input = input
	}
	switch {
	case errVal != nil:
		so.IsError = true
		so.Data = errVal
	case data != nil:
		so.Data = data
	}
}

// SpanOutputEnvelope re-wraps an already split output (see
// SplitSpanOutputEnvelope) as the envelope an unsplit output column holds:
// {"error": errVal} for an error, else {"data": data}, nil when both are
// nil. The values' bytes are kept as is. A data value that wasn't
// enveloped when it was split (a waitForEvent-shaped or unrecognized
// output) comes back wrapped, which is fine for a run's output: the
// executor always envelopes it (state.DriverResponse.GetTraceFunctionOutput).
func SpanOutputEnvelope(data, errVal []byte) []byte {
	key, v := "data", data
	if errVal != nil {
		key, v = "error", errVal
	}
	if v == nil {
		return nil
	}
	out := make([]byte, 0, len(key)+len(v)+5)
	out = append(out, `{"`...)
	out = append(out, key...)
	out = append(out, `":`...)
	out = append(out, v...)
	return append(out, '}')
}
