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

	queuedAt := randomTimeInWindow(rng, cfg)
	events := generateTriggerEvents(rng, tmpl, tenant, queuedAt)
	eventIDs := make([]string, len(events))
	for i, e := range events {
		eventIDs[i] = e.InternalID
	}

	status := pickString(rng, tmpl.Statuses, "Completed")
	startedAt := queuedAt.Add(time.Duration(rng.Intn(1000)) * time.Millisecond)
	endedAt := startedAt.Add(time.Duration(50+rng.Intn(5000)) * time.Millisecond)

	run := RunRow{
		AccountID:  tenant.AccountID,
		EnvID:      tenant.EnvID,
		RunID:      newULID(rng, queuedAt),
		QueuedAt:   queuedAt,
		StartedAt:  &startedAt,
		EndedAt:    &endedAt,
		AppID:      tenant.AppID,
		FunctionID: tenant.FunctionID,
		Status:     status,
		Inputs:     pickString(rng, tmpl.Inputs, `{}`),
		Output:     pickString(rng, tmpl.Outputs, `{}`),
		EventIDs:   eventIDs,
	}

	spans := generateSpanTree(rng, tmpl, run)

	return GeneratedRun{Run: run, Spans: spans, Events: events}
}

// generateSpanTree builds one root span covering the run's full duration
// plus a handful of child step spans nested inside it and chained to it via
// ParentSpanID, mirroring the shape a real function run produces.
func generateSpanTree(rng *rand.Rand, tmpl Templates, run RunRow) []SpanRow {
	traceID := newULID(rng, run.QueuedAt)
	rootStart, rootEnd := *run.StartedAt, *run.EndedAt

	root := SpanRow{
		AccountID:   run.AccountID,
		EnvID:       run.EnvID,
		RunID:       run.RunID,
		RunQueuedAt: run.QueuedAt,
		AppID:       run.AppID,
		FunctionID:  run.FunctionID,
		Name:        pickString(rng, tmpl.SpanNames, "function"),
		StartTime:   rootStart,
		EndTime:     rootEnd,
		TraceID:     traceID,
		SpanID:      newULID(rng, rootStart),
		Attributes:  pickString(rng, tmpl.Attributes, `{}`),
		Output:      run.Output,
		Input:       run.Inputs,
	}

	spans := []SpanRow{root}

	childCount := 1 + rng.Intn(3)
	span := rootEnd.Sub(rootStart)
	for i := range childCount {
		childStart := rootStart.Add(time.Duration(i) * span / time.Duration(childCount))
		childEnd := rootStart.Add(time.Duration(i+1) * span / time.Duration(childCount))
		parentID := root.SpanID

		spans = append(spans, SpanRow{
			AccountID:    run.AccountID,
			EnvID:        run.EnvID,
			RunID:        run.RunID,
			RunQueuedAt:  run.QueuedAt,
			AppID:        run.AppID,
			FunctionID:   run.FunctionID,
			Name:         fmt.Sprintf("%s-step-%d", pickString(rng, tmpl.SpanNames, "step"), i+1),
			StartTime:    childStart,
			EndTime:      childEnd,
			TraceID:      traceID,
			SpanID:       newULID(rng, childStart),
			ParentSpanID: &parentID,
			Attributes:   pickString(rng, tmpl.Attributes, `{}`),
			Output:       pickString(rng, tmpl.Outputs, `{}`),
			Input:        pickString(rng, tmpl.Inputs, `{}`),
		})
	}

	return spans
}

// generateTriggerEvents synthesizes the event(s) that triggered a run,
// received shortly before queuedAt.
func generateTriggerEvents(rng *rand.Rand, tmpl Templates, tenant Tenant, queuedAt time.Time) []EventRow {
	receivedAt := queuedAt.Add(-time.Duration(rng.Intn(500)) * time.Millisecond)
	id := newULID(rng, receivedAt)

	return []EventRow{{
		AccountID:  tenant.AccountID,
		EnvID:      tenant.EnvID,
		InternalID: id,
		ReceivedAt: receivedAt,
		Source:     "duckdbseed",
		EventID:    id,
		EventName:  pickString(rng, tmpl.EventNames, "app/seeded.event"),
		EventData:  pickString(rng, tmpl.EventData, `{}`),
		EventV:     "1",
		EventTS:    receivedAt,
		EventMeta:  `{}`,
	}}
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
