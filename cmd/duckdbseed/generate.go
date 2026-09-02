package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/oklog/ulid/v2"
)

// GenerateRuns synthesizes cfg.RunCount runs (each with a span tree and
// linked trigger events) sampled from tmpl, using rng for all randomness so
// output is fully deterministic given the same rng seed and inputs.
func GenerateRuns(rng *rand.Rand, tmpl Templates, cfg GenerateConfig) ([]GeneratedRun, error) {
	if len(tmpl.Tenants) == 0 {
		return nil, fmt.Errorf("duckdbseed: templates have no tenants to generate runs for")
	}

	out := make([]GeneratedRun, 0, cfg.RunCount)
	for i := 0; i < cfg.RunCount; i++ {
		out = append(out, generateRun(rng, tmpl, cfg))
	}
	return out, nil
}

func generateRun(rng *rand.Rand, tmpl Templates, cfg GenerateConfig) GeneratedRun {
	tenant := tmpl.Tenants[rng.Intn(len(tmpl.Tenants))]

	// queuedAt is the one random offset (its own placement within the
	// window) that every other timestamp on this run -- its own
	// started_at/ended_at, its spans, its triggering events -- shifts by;
	// see SpanTemplate/EventTemplate's doc comments.
	queuedAt := randomTimeInWindow(rng, cfg)

	trace := defaultTraceTemplate
	if len(tmpl.Traces) > 0 {
		trace = tmpl.Traces[rng.Intn(len(tmpl.Traces))]
	}

	events := generateTriggerEvents(rng, tmpl, tenant, queuedAt)
	eventIDs := make([]string, len(events))
	for i, e := range events {
		eventIDs[i] = e.InternalID
	}

	var startedAt, endedAt *time.Time
	if trace.StartedOffset != nil {
		t := queuedAt.Add(*trace.StartedOffset)
		startedAt = &t
	}
	if trace.EndedOffset != nil {
		t := queuedAt.Add(*trace.EndedOffset)
		endedAt = &t
	}

	run := RunRow{
		AccountID:  tenant.AccountID,
		EnvID:      tenant.EnvID,
		RunID:      newULID(rng, queuedAt),
		QueuedAt:   queuedAt,
		StartedAt:  startedAt,
		EndedAt:    endedAt,
		AppID:      tenant.AppID,
		FunctionID: tenant.FunctionID,
		Status:     pickString(rng, tmpl.Statuses, "Completed"),
		Inputs:     pickString(rng, tmpl.Inputs, `{}`),
		Output:     pickString(rng, tmpl.Outputs, `{}`),
		EventIDs:   eventIDs,
	}

	spans := generateSpanTree(trace, run)
	metadata := generateMetadata(rng, tmpl, run, spans)

	return GeneratedRun{Run: run, Spans: spans, Events: events, Metadata: metadata}
}

// defaultTraceTemplate is replayed when tmpl has no sampled traces to draw
// from (a source database with runs but no span data yet).
var defaultTraceTemplate = TraceTemplate{
	TraceID: "seeded-trace",
	Spans: []SpanTemplate{
		{SpanID: "seeded-root", Name: "function", Attributes: `{}`, Output: `{}`, Input: `{}`, StartOffset: 100 * time.Millisecond, EndOffset: time.Second},
		{SpanID: "seeded-step", ParentSpanID: strPtr("seeded-root"), Name: "step", Attributes: `{}`, Output: `{}`, Input: `{}`, StartOffset: 100 * time.Millisecond, EndOffset: time.Second},
	},
	StartedOffset: durationPtr(100 * time.Millisecond),
	EndedOffset:   durationPtr(time.Second),
}

// generateSpanTree replays trace onto run: same span names/attributes/
// output/input as the sampled trace, verbatim, just given start/end times
// computed by adding each template span's StartOffset/EndOffset to the
// new run's own QueuedAt — the same anchor trace.StartedOffset/EndedOffset
// (already applied to run.StartedAt/EndedAt by generateRun) and each
// event's Offset use, so every timestamp on this generated run shifts by
// the one random offset that placed its QueuedAt, never independently.
//
// TraceID/SpanID/ParentSpanID are copied completely unchanged from trace
// (see SpanTemplate's doc comment for why that's safe and sufficient to
// preserve any real nesting, however deep, without this function
// resolving or rebuilding a tree itself) -- no IDs are generated here at
// all, which is also why this takes no *rand.Rand.
func generateSpanTree(trace TraceTemplate, run RunRow) []SpanRow {
	anchor := run.QueuedAt

	spans := make([]SpanRow, len(trace.Spans))
	for i, s := range trace.Spans {
		spans[i] = SpanRow{
			AccountID:    run.AccountID,
			EnvID:        run.EnvID,
			RunID:        run.RunID,
			RunQueuedAt:  run.QueuedAt,
			AppID:        run.AppID,
			FunctionID:   run.FunctionID,
			Name:         s.Name,
			StartTime:    anchor.Add(s.StartOffset),
			EndTime:      anchor.Add(s.EndOffset),
			TraceID:      trace.TraceID,
			SpanID:       s.SpanID,
			ParentSpanID: s.ParentSpanID,
			Attributes:   s.Attributes,
			Output:       s.Output,
			Input:        s.Input,
		}
	}
	return spans
}

// generateMetadata replays one randomly picked MetadataProfile from
// tmpl.MetadataProfiles verbatim: each item's Scope/StepID/StepIndex/
// StepAttempt/Kind/IsUser/Values are copied unchanged, and it attaches to
// whichever of spans shares its exact SpanID (see MetadataTemplateItem's
// doc comment for why that match works without recomputing any
// root/step position here) -- skipping any item whose SpanID isn't one of
// spans' own, which only happens when the trace and metadata profile
// picked for this run were sampled from two different source runs (see
// Templates' doc comment). Returns nil when tmpl has no profiles to draw
// from (a source with runs but no metadata at all is a real, faithfully
// replicated shape, not an error).
func generateMetadata(rng *rand.Rand, tmpl Templates, run RunRow, spans []SpanRow) []MetadataRow {
	if len(tmpl.MetadataProfiles) == 0 {
		return nil
	}
	profile := tmpl.MetadataProfiles[rng.Intn(len(tmpl.MetadataProfiles))]
	if len(profile.Items) == 0 {
		return nil
	}

	rows := make([]MetadataRow, 0, len(profile.Items))
	for _, item := range profile.Items {
		if !hasSpanID(spans, item.SpanID) {
			continue
		}
		values := item.Values
		if values == "" {
			values = "{}"
		}
		rows = append(rows, MetadataRow{
			AccountID:   run.AccountID,
			EnvID:       run.EnvID,
			RunID:       run.RunID,
			RunQueuedAt: run.QueuedAt,
			AppID:       run.AppID,
			FunctionID:  run.FunctionID,
			SpanID:      item.SpanID,
			CreatedAt:   run.QueuedAt.Add(item.Offset),
			Scope:       item.Scope,
			StepID:      item.StepID,
			StepIndex:   item.StepIndex,
			StepAttempt: item.StepAttempt,
			Kind:        item.Kind,
			IsUser:      item.IsUser,
			Values:      values,
		})
	}
	if len(rows) == 0 {
		return nil
	}
	return rows
}

// hasSpanID reports whether spanID matches one of spans' own SpanIDs. A
// plain linear scan rather than a map: a run's span tree is typically tiny
// (single digits to a few dozen), where building a map fresh on every
// single generated run costs more -- a heap allocation per run, purely to
// avoid string comparisons on that same handful of spans -- than it saves.
func hasSpanID(spans []SpanRow, spanID string) bool {
	for _, s := range spans {
		if s.SpanID == spanID {
			return true
		}
	}
	return false
}

// defaultEventProfile is replayed when tmpl has no sampled event profiles
// to draw from (a source database with runs but no event data yet).
var defaultEventProfile = []EventTemplate{{Name: "app/seeded.event", Data: `{}`, Offset: -200 * time.Millisecond}}

// generateTriggerEvents replays one randomly picked event profile from
// tmpl.EventProfiles onto the run: same event name/data as the sampled
// run's triggering events, verbatim, just given fresh internal/event IDs
// and received_at computed by adding each item's Offset (relative to
// queued_at, see EventTemplate's doc comment) to queuedAt -- the same
// single random offset (queuedAt's own placement in the window) that
// shifts every other timestamp on this generated run.
func generateTriggerEvents(rng *rand.Rand, tmpl Templates, tenant Tenant, queuedAt time.Time) []EventRow {
	profile := defaultEventProfile
	if len(tmpl.EventProfiles) > 0 {
		profile = tmpl.EventProfiles[rng.Intn(len(tmpl.EventProfiles))]
	}

	events := make([]EventRow, len(profile))
	for i, item := range profile {
		receivedAt := queuedAt.Add(item.Offset)
		id := newULID(rng, receivedAt)

		events[i] = EventRow{
			AccountID:  tenant.AccountID,
			EnvID:      tenant.EnvID,
			InternalID: id,
			ReceivedAt: receivedAt,
			Source:     "duckdbseed",
			EventID:    id,
			EventName:  item.Name,
			EventData:  item.Data,
			EventV:     "1",
			EventTS:    receivedAt,
			EventMeta:  `{}`,
		}
	}
	return events
}

func randomTimeInWindow(rng *rand.Rand, cfg GenerateConfig) time.Time {
	if cfg.Window <= 0 {
		return cfg.Now
	}
	offset := time.Duration(rng.Int63n(int64(cfg.Window)))
	return cfg.Now.Add(-offset)
}

func pickString(rng *rand.Rand, options []string, fallback string) string {
	if len(options) == 0 {
		return fallback
	}
	return options[rng.Intn(len(options))]
}

func newULID(rng *rand.Rand, t time.Time) string {
	return ulid.MustNew(ulid.Timestamp(t), rng).String()
}
