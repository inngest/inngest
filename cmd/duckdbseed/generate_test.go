package main

import (
	"math/rand"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func testTemplates() Templates {
	return Templates{
		Tenants: []Tenant{
			{
				AccountID:  uuid.New(),
				EnvID:      uuid.New(),
				AppID:      uuid.New(),
				FunctionID: uuid.New(),
			},
		},
		Statuses: []string{"Completed", "Failed"},
		Inputs:   []string{`{"event":{"name":"app/test.event"}}`},
		Outputs:  []string{`{"data":"ok"}`},
		Traces: []TraceTemplate{
			{
				TraceID: "trace-1",
				Spans: []SpanTemplate{
					{
						SpanID:      "root",
						Name:        "my-function",
						Attributes:  `{"sys.step.name":"my-function"}`,
						Output:      `{"data":"ok"}`,
						Input:       `{"event":{"name":"app/test.event"}}`,
						StartOffset: 100 * time.Millisecond,
						EndOffset:   time.Second,
					},
					{
						SpanID:       "step-1",
						ParentSpanID: strPtr("root"),
						Name:         "my-function-step-1",
						Attributes:   `{"sys.step.name":"my-function-step-1"}`,
						Output:       `{"data":"ok"}`,
						Input:        `{}`,
						StartOffset:  100 * time.Millisecond,
						EndOffset:    400 * time.Millisecond,
					},
					{
						SpanID:       "step-2",
						ParentSpanID: strPtr("root"),
						Name:         "my-function-step-2",
						Attributes:   `{"sys.step.name":"my-function-step-2"}`,
						Output:       `{"data":"ok"}`,
						Input:        `{}`,
						StartOffset:  400 * time.Millisecond,
						EndOffset:    time.Second,
					},
				},
				StartedOffset: durationPtr(100 * time.Millisecond),
				EndedOffset:   durationPtr(time.Second),
			},
		},
		EventProfiles: [][]EventTemplate{
			{{Name: "app/test.event", Data: `{"k":"v"}`, Offset: -200 * time.Millisecond}},
		},
	}
}

func testConfig(runCount int) GenerateConfig {
	return GenerateConfig{
		RunCount: runCount,
		Window:   24 * time.Hour,
		Now:      time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC),
	}
}

func TestGenerateRunsProducesRequestedCountWithUniqueRunIDs(t *testing.T) {
	tmpl := testTemplates()
	rng := rand.New(rand.NewSource(1))

	generated, err := GenerateRuns(rng, tmpl, testConfig(5))
	require.NoError(t, err)
	require.Len(t, generated, 5)

	seen := map[string]bool{}
	for _, g := range generated {
		require.False(t, seen[g.Run.RunID], "run IDs must be unique")
		seen[g.Run.RunID] = true
	}
}

func TestGenerateRunsUsesATenantFromTemplates(t *testing.T) {
	tmpl := testTemplates()
	rng := rand.New(rand.NewSource(1))

	generated, err := GenerateRuns(rng, tmpl, testConfig(1))
	require.NoError(t, err)
	require.Len(t, generated, 1)

	run := generated[0].Run
	want := tmpl.Tenants[0]
	require.Equal(t, want.AccountID, run.AccountID)
	require.Equal(t, want.EnvID, run.EnvID)
	require.Equal(t, want.AppID, run.AppID)
	require.Equal(t, want.FunctionID, run.FunctionID)
}

func TestGenerateRunsKeepsQueuedAtWithinWindow(t *testing.T) {
	tmpl := testTemplates()
	rng := rand.New(rand.NewSource(1))
	cfg := testConfig(20)

	generated, err := GenerateRuns(rng, tmpl, cfg)
	require.NoError(t, err)

	earliest := cfg.Now.Add(-cfg.Window)
	for _, g := range generated {
		require.False(t, g.Run.QueuedAt.Before(earliest), "queued_at must not be before the window start")
		require.False(t, g.Run.QueuedAt.After(cfg.Now), "queued_at must not be after now")
	}
}

func TestGenerateRunsIsDeterministicGivenTheSameSeed(t *testing.T) {
	tmpl := testTemplates()
	cfg := testConfig(10)

	first, err := GenerateRuns(rand.New(rand.NewSource(42)), tmpl, cfg)
	require.NoError(t, err)
	second, err := GenerateRuns(rand.New(rand.NewSource(42)), tmpl, cfg)
	require.NoError(t, err)

	require.Equal(t, first, second)
}

func TestGenerateRunsErrorsWithNoTenants(t *testing.T) {
	tmpl := testTemplates()
	tmpl.Tenants = nil
	rng := rand.New(rand.NewSource(1))

	_, err := GenerateRuns(rng, tmpl, testConfig(1))
	require.Error(t, err)
}

func TestGenerateRunsBuildsASpanTreeWithARootAndChildSpans(t *testing.T) {
	tmpl := testTemplates()
	rng := rand.New(rand.NewSource(7))

	generated, err := GenerateRuns(rng, tmpl, testConfig(1))
	require.NoError(t, err)

	spans := generated[0].Spans
	require.GreaterOrEqual(t, len(spans), 2, "expect a root span plus at least one child step span")

	var roots []SpanRow
	for _, s := range spans {
		require.Equal(t, generated[0].Run.RunID, s.RunID)
		if s.ParentSpanID == nil {
			roots = append(roots, s)
		} else {
			require.Equal(t, roots[0].SpanID, *s.ParentSpanID, "child spans must chain to the root span")
		}
		require.False(t, s.StartTime.After(s.EndTime))
	}
	require.Len(t, roots, 1, "exactly one root span per run")
}

// TestGenerateSpanTreePreservesRealNestingNotJustRootChildren proves the
// fix for a real bug: a nested span (e.g. a userland/extended-trace span
// nested under another userland span, not directly under the run root)
// was being reparented to the root on replay, flattening the sampled
// run's real tree shape. Here step-1 has its own child ("extended-trace",
// three levels deep from the root) and step-2 does not; the replayed
// spans must keep that exact shape with freshly generated IDs, not chain
// everything to the root.
func TestGenerateSpanTreePreservesRealNestingNotJustRootChildren(t *testing.T) {
	trace := TraceTemplate{
		TraceID: "trace-1",
		Spans: []SpanTemplate{
			{SpanID: "root", EndOffset: time.Second},
			{SpanID: "step-1", ParentSpanID: strPtr("root"), StartOffset: 100 * time.Millisecond, EndOffset: 500 * time.Millisecond},
			{SpanID: "step-1-extended-trace", ParentSpanID: strPtr("step-1"), StartOffset: 200 * time.Millisecond, EndOffset: 300 * time.Millisecond},
			{SpanID: "step-2", ParentSpanID: strPtr("root"), StartOffset: 500 * time.Millisecond, EndOffset: time.Second},
		},
	}
	run := RunRow{RunID: "run-1", QueuedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}

	spans := generateSpanTree(trace, run)
	require.Len(t, spans, 4)

	bySpanID := map[string]SpanRow{}
	for _, s := range spans {
		bySpanID[s.SpanID] = s
	}

	require.Nil(t, bySpanID["root"].ParentSpanID)
	require.NotNil(t, bySpanID["step-1"].ParentSpanID)
	require.Equal(t, "root", *bySpanID["step-1"].ParentSpanID)
	require.NotNil(t, bySpanID["step-2"].ParentSpanID)
	require.Equal(t, "root", *bySpanID["step-2"].ParentSpanID)

	require.NotNil(t, bySpanID["step-1-extended-trace"].ParentSpanID)
	require.Equal(t, "step-1", *bySpanID["step-1-extended-trace"].ParentSpanID,
		"a span nested under a non-root span must keep that real parent, not be reparented to the root")
}

// TestGenerateRunsShiftsEveryTimestampByASingleRandomOffset proves the
// contract underlying generateRun/generateSpanTree/generateTriggerEvents:
// the run's own started_at/ended_at, every span's start/end time, and
// every triggering event's received_at are each queued_at plus a
// template-supplied, sample-derived offset -- never independent jitter --
// so the only randomness in a generated run's timing is where its own
// queued_at lands in the window.
func TestGenerateRunsShiftsEveryTimestampByASingleRandomOffset(t *testing.T) {
	tmpl := testTemplates()
	trace := tmpl.Traces[0]
	events := tmpl.EventProfiles[0]
	rng := rand.New(rand.NewSource(9))

	generated, err := GenerateRuns(rng, tmpl, testConfig(1))
	require.NoError(t, err)

	run := generated[0].Run
	require.NotNil(t, run.StartedAt)
	require.NotNil(t, run.EndedAt)
	require.Equal(t, run.QueuedAt.Add(*trace.StartedOffset), *run.StartedAt)
	require.Equal(t, run.QueuedAt.Add(*trace.EndedOffset), *run.EndedAt)

	for i, s := range generated[0].Spans {
		want := trace.Spans[i]
		require.Equal(t, run.QueuedAt.Add(want.StartOffset), s.StartTime)
		require.Equal(t, run.QueuedAt.Add(want.EndOffset), s.EndTime)
	}

	for i, e := range generated[0].Events {
		require.Equal(t, run.QueuedAt.Add(events[i].Offset), e.ReceivedAt)
	}
}

func TestGenerateRunsLinksEventsToTheRunsEventIDs(t *testing.T) {
	tmpl := testTemplates()
	rng := rand.New(rand.NewSource(3))

	generated, err := GenerateRuns(rng, tmpl, testConfig(1))
	require.NoError(t, err)

	run := generated[0].Run
	events := generated[0].Events
	require.NotEmpty(t, run.EventIDs, "generated runs should be triggered by at least one event")
	require.Len(t, events, len(run.EventIDs))

	byInternalID := map[string]EventRow{}
	for _, e := range events {
		byInternalID[e.InternalID] = e
	}
	for _, id := range run.EventIDs {
		e, ok := byInternalID[id]
		require.True(t, ok, "every run.EventIDs entry must have a matching generated event row")
		require.Equal(t, run.AccountID, e.AccountID)
		require.Equal(t, run.EnvID, e.EnvID)
	}
}
