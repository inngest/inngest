package v3

import "github.com/inngest/inngest/pkg/tracing/meta"

// Span names for this package's TracerProvider — see pkg/execution/dualwrite,
// its only caller. Some are genuinely new (no equivalent in the real
// production tracing pipeline, pkg/tracing/meta); others deliberately alias
// the same constant meta defines, for spans whose identity (name, and often
// Seed too) is meant to match the real span it mirrors exactly.
const (
	// SpanNameStep and SpanNameExecution alias meta's own constants: the
	// spans built under these names (OnStepRunFinished/OnStepGatewayRequestFinished/
	// OnSleep for SpanNameStep; OnStepFinished for SpanNameExecution) use the
	// exact same name — and for SpanNameStep, the exact same Seed — as their
	// real production counterparts, so a reader can correlate the two by
	// name/span_id.
	SpanNameStep      = meta.SpanNameStep
	SpanNameExecution = meta.SpanNameExecution

	// SpanNameRunQueued and SpanNameRunStarted mark the moments
	// OnFunctionScheduled/OnFunctionStarted fire — the real production
	// pipeline has no standalone spans for these; it instead extends the
	// run's own root span in place (an EXTEND row), which this
	// insert-only package doesn't do — see SpanExporter's doc comment.
	SpanNameRunQueued  = "executor.run.queued"
	SpanNameRunStarted = "executor.run.started"

	// SpanNameError and SpanNameFinal replace the real production
	// pipeline's single meta.SpanNameNonStep name (used for both outcomes,
	// distinguished only by a status attribute) with two distinct names,
	// split by whether OnFunctionFinished's resp carried an error.
	SpanNameError = "executor.error"
	SpanNameFinal = "executor.final"

	// SpanNameStepPauseStarted marks the moment a pause (wait-for-event,
	// wait-for-signal, invoke) begins — see createPauseStartedSpan. No real
	// production equivalent: the real pipeline only records the pause's
	// resolution, not its start, as a span.
	SpanNameStepPauseStarted = "executor.step.pause_started"

	// SpanNameStepPlanned marks the moment OnStepScheduled fires — a
	// distinct span kind (and identity: a random span_id, never
	// FinalizedStepDynamicSeed) from SpanNameStep, since this package only
	// ever inserts and reusing the eventual finished step span's identity
	// here would collide with it once that real span is inserted.
	SpanNameStepPlanned = "executor.step.planned"

	// SpanNameExtendedTrace marks a userland (extended-trace) span written by
	// OnExtendedTraceSpan (pkg/api/apiv1/traces.go's commitSpan, via OTLP
	// ingestion).
	SpanNameExtendedTrace = "sdk.extended_trace"
)
