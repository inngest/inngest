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
		Statuses:   []string{"Completed", "Failed"},
		SpanNames:  []string{"my-function"},
		EventNames: []string{"app/test.event"},
		Inputs:     []string{`{"event":{"name":"app/test.event"}}`},
		Outputs:    []string{`{"data":"ok"}`},
		Attributes: []string{`{"sys.step.name":"my-function"}`},
		EventData:  []string{`{"k":"v"}`},
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
