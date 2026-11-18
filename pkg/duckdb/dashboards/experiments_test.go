package dashboards_test

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/duckdb/dashboards"
	"github.com/stretchr/testify/require"
)

type testMetadata struct {
	runID     string
	queuedAt  time.Time
	spanID    string
	scope     string
	kind      string
	isUser    bool
	values    any
	createdAt time.Time
}

// insertMetadata inserts one inngest.run_metadata emission. kind is stored
// w/o its inngest./userland. prefix, as the dual-write listener does.
func insertMetadata(t *testing.T, db *sql.DB, accountID, envID uuid.UUID, m testMetadata) {
	t.Helper()
	values, err := json.Marshal(m.values)
	require.NoError(t, err)
	if m.scope == "" {
		m.scope = "step"
	}
	if m.createdAt.IsZero() {
		m.createdAt = m.queuedAt.Add(time.Second)
	}
	_, err = db.ExecContext(t.Context(), `
INSERT INTO inngest.run_metadata
  (account_id, env_id, run_id, run_queued_at, span_id, scope, kind, is_user, "values", created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`,
		accountID.String(), envID.String(), m.runID, m.queuedAt, m.spanID, m.scope, m.kind, m.isUser, string(values), m.createdAt,
	)
	require.NoError(t, err)
}

type experimentFixture struct {
	db                  *sql.DB
	accountID, envID    uuid.UUID
	appID               uuid.UUID
	checkoutFn, otherFn uuid.UUID
	now                 time.Time
}

func (f experimentFixture) run(t *testing.T, runID string, fn uuid.UUID, slug string, queuedAt time.Time) {
	t.Helper()
	insertRun(t, f.db, testRun{
		runID: runID, accountID: f.accountID, envID: f.envID, appID: f.appID, appName: "app",
		functionID: fn, functionSlug: slug, status: "Completed", queuedAt: queuedAt,
	})
}

func (f experimentFixture) md(t *testing.T, m testMetadata) {
	t.Helper()
	insertMetadata(t, f.db, f.accountID, f.envID, m)
}

// seedExperiments seeds:
//   - checkout on checkoutFn: r1 -> a, r2 -> b via group.experiment (a
//     selection step w/ strategy + weights, and a stamped step w/ scores),
//     r3 -> b scored run-scoped (scoreExperiment).
//   - pricing on otherFn: r4 -> x, seen last.
func seedExperiments(t *testing.T) experimentFixture {
	db, cleanup := newTestDuckDB(t)
	t.Cleanup(cleanup)
	f := experimentFixture{
		db: db, accountID: uuid.New(), envID: uuid.New(), appID: uuid.New(),
		checkoutFn: uuid.New(), otherFn: uuid.New(),
		now: time.Now().UTC().Truncate(time.Millisecond),
	}

	selection := func(variant string) map[string]any {
		return map[string]any{
			"name": "checkout", "variant": variant, "selection_strategy": "weighted",
			"available_variants": []string{"a", "b", "c"},
			"variant_weights":    map[string]float64{"a": 0.5, "b": 0.3, "c": 0.2},
		}
	}
	stamp := func(variant string) map[string]any {
		return map[string]any{"name": "checkout", "variant": variant, "selection_strategy": "weighted"}
	}

	t1 := f.now.Add(-3 * time.Hour)
	f.run(t, "r1", f.checkoutFn, "app-checkout", t1)
	f.md(t, testMetadata{runID: "r1", queuedAt: t1, spanID: "r1-select", kind: "experiment", values: selection("a")})
	f.md(t, testMetadata{runID: "r1", queuedAt: t1, spanID: "r1-step", kind: "experiment", values: stamp("a")})
	f.md(t, testMetadata{runID: "r1", queuedAt: t1, spanID: "r1-step", kind: "score.latency", values: map[string]any{"value": 10}})
	// Only the latest emission of a kind on a span counts.
	f.md(t, testMetadata{runID: "r1", queuedAt: t1, spanID: "r1-step", kind: "score.latency", values: map[string]any{"value": 999}, createdAt: t1.Add(-time.Second)})
	// The legacy kind, w/ a bool and a name the per-name kind also has.
	f.md(t, testMetadata{runID: "r1", queuedAt: t1, spanID: "r1-step", kind: "score", values: map[string]any{
		"ok": map[string]any{"value": true}, "latency": map[string]any{"value": 12345},
	}})
	// A score on a span no experiment is on doesn't count.
	f.md(t, testMetadata{runID: "r1", queuedAt: t1, spanID: "r1-unrelated", kind: "score.latency", values: map[string]any{"value": 5000}})
	// Userland metadata is never Inngest's experiment.
	f.md(t, testMetadata{runID: "r1", queuedAt: t1, spanID: "r1-user", kind: "experiment", isUser: true, values: map[string]any{"name": "checkout", "variant": "z"}})

	t2 := f.now.Add(-2 * time.Hour)
	f.run(t, "r2", f.checkoutFn, "app-checkout", t2)
	f.md(t, testMetadata{runID: "r2", queuedAt: t2, spanID: "r2-select", kind: "experiment", values: selection("b")})
	f.md(t, testMetadata{runID: "r2", queuedAt: t2, spanID: "r2-step", kind: "experiment", values: stamp("b")})
	f.md(t, testMetadata{runID: "r2", queuedAt: t2, spanID: "r2-step", kind: "score.latency", values: map[string]any{"value": 30}})
	f.md(t, testMetadata{runID: "r2", queuedAt: t2, spanID: "r2-step", kind: "score.ok", values: map[string]any{"value": false}})

	t3 := f.now.Add(-90 * time.Minute)
	f.run(t, "r3", f.checkoutFn, "app-checkout", t3)
	f.md(t, testMetadata{runID: "r3", queuedAt: t3, spanID: "r3-run", scope: "run", kind: "experiment", values: map[string]any{"name": "checkout", "variant": "b"}})
	f.md(t, testMetadata{runID: "r3", queuedAt: t3, spanID: "r3-run", scope: "run", kind: "score.latency", values: map[string]any{"value": 50}})

	t4 := f.now.Add(-time.Hour)
	f.run(t, "r4", f.otherFn, "app-other", t4)
	f.md(t, testMetadata{runID: "r4", queuedAt: t4, spanID: "r4-run", scope: "run", kind: "experiment", values: map[string]any{"name": "pricing", "variant": "x"}})

	// Another env's experiment never surfaces.
	insertMetadata(t, db, f.accountID, uuid.New(), testMetadata{runID: "r5", queuedAt: t4, spanID: "r5", kind: "experiment", values: stamp("a")})

	return f
}

func TestListExperiments(t *testing.T) {
	f := seedExperiments(t)

	res, err := dashboards.ListExperiments(t.Context(), f.db, f.accountID, f.envID, dashboards.ExperimentsOpts{})
	require.NoError(t, err)
	require.False(t, res.HasMore)
	require.Len(t, res.Experiments, 2)

	pricing, checkout := res.Experiments[0], res.Experiments[1]
	require.Equal(t, "pricing", pricing.Name, "most recently seen first")
	require.Equal(t, f.otherFn, pricing.FunctionID)
	require.Equal(t, []string{"x"}, pricing.Variants)
	require.Empty(t, pricing.SelectionStrategy)
	require.Equal(t, 1, pricing.TotalRuns)

	require.Equal(t, "checkout", checkout.Name)
	require.Equal(t, f.checkoutFn, checkout.FunctionID)
	require.Equal(t, dashboards.FunctionRef{Slug: "app-checkout", AppID: f.appID, AppName: "app"}, checkout.Function)
	require.Equal(t, "weighted", checkout.SelectionStrategy)
	require.Equal(t, []string{"a", "b", "c"}, checkout.Variants, "declared variants in order")
	require.Equal(t, 3, checkout.TotalRuns)
	require.True(t, checkout.FirstSeen.Equal(f.now.Add(-3*time.Hour)), checkout.FirstSeen)
	require.True(t, checkout.LastSeen.Equal(f.now.Add(-90*time.Minute)), checkout.LastSeen)
}

func TestListExperiments_FunctionAndTimeRange(t *testing.T) {
	f := seedExperiments(t)

	res, err := dashboards.ListExperiments(t.Context(), f.db, f.accountID, f.envID, dashboards.ExperimentsOpts{FunctionID: f.checkoutFn})
	require.NoError(t, err)
	require.Len(t, res.Experiments, 1)
	require.Equal(t, "checkout", res.Experiments[0].Name)

	// Only r3 is in range.
	res, err = dashboards.ListExperiments(t.Context(), f.db, f.accountID, f.envID, dashboards.ExperimentsOpts{
		FunctionID: f.checkoutFn, From: f.now.Add(-100 * time.Minute), Until: f.now.Add(-80 * time.Minute),
	})
	require.NoError(t, err)
	require.Len(t, res.Experiments, 1)
	require.Equal(t, 1, res.Experiments[0].TotalRuns)
	require.Equal(t, []string{"b"}, res.Experiments[0].Variants, "no selection step in range, so only what was observed")

	_, err = dashboards.ListExperiments(t.Context(), f.db, f.accountID, f.envID, dashboards.ExperimentsOpts{From: f.now, Until: f.now.Add(-time.Hour)})
	var invalid *dashboards.InvalidArgumentError
	require.ErrorAs(t, err, &invalid)
}

func TestListExperiments_Pagination(t *testing.T) {
	f := seedExperiments(t)
	// A second experiment on the same function, seen at the same time as
	// pricing, so the page boundary falls on a last_seen tie.
	t4 := f.now.Add(-time.Hour)
	f.run(t, "r6", f.checkoutFn, "app-checkout", t4)
	f.md(t, testMetadata{runID: "r6", queuedAt: t4, spanID: "r6-run", scope: "run", kind: "experiment", values: map[string]any{"name": "copy", "variant": "long"}})

	full, err := dashboards.ListExperiments(t.Context(), f.db, f.accountID, f.envID, dashboards.ExperimentsOpts{})
	require.NoError(t, err)
	require.Len(t, full.Experiments, 3)

	var paged []string
	cursor := ""
	for range 5 {
		res, err := dashboards.ListExperiments(t.Context(), f.db, f.accountID, f.envID, dashboards.ExperimentsOpts{Cursor: cursor, Limit: 1})
		require.NoError(t, err)
		for _, e := range res.Experiments {
			paged = append(paged, e.Name)
		}
		if !res.HasMore {
			break
		}
		cursor = res.Experiments[len(res.Experiments)-1].Cursor
	}
	var want []string
	for _, e := range full.Experiments {
		want = append(want, e.Name)
	}
	require.Equal(t, want, paged)

	_, err = dashboards.ListExperiments(t.Context(), f.db, f.accountID, f.envID, dashboards.ExperimentsOpts{Cursor: "nope"})
	var invalid *dashboards.InvalidArgumentError
	require.ErrorAs(t, err, &invalid)
}

func TestGetExperiment(t *testing.T) {
	f := seedExperiments(t)

	d, err := dashboards.GetExperiment(t.Context(), f.db, f.accountID, f.envID, dashboards.ExperimentOpts{FunctionID: f.checkoutFn, Name: "checkout"})
	require.NoError(t, err)
	require.Equal(t, "checkout", d.Name)
	require.Equal(t, "weighted", d.SelectionStrategy)
	require.Equal(t, []dashboards.ExperimentWeight{{Variant: "a", Weight: 0.5}, {Variant: "b", Weight: 0.3}, {Variant: "c", Weight: 0.2}}, d.Weights)
	require.True(t, d.FirstSeen.Equal(f.now.Add(-3*time.Hour)), d.FirstSeen)
	require.True(t, d.LastSeen.Equal(f.now.Add(-90*time.Minute)), d.LastSeen)

	require.Equal(t, []dashboards.ExperimentVariant{
		{
			Name: "a", RunCount: 1,
			Metrics: []dashboards.ExperimentMetric{
				// The per-name kind wins over the legacy kind's latency.
				{Key: "latency", Avg: 10, Min: 10, Max: 10},
				{Key: "ok", Avg: 1, Min: 1, Max: 1},
			},
		},
		{
			Name: "b", RunCount: 2,
			Metrics: []dashboards.ExperimentMetric{
				{Key: "latency", Avg: 40, Min: 30, Max: 50},
				{Key: "ok", Avg: 0, Min: 0, Max: 0},
			},
		},
	}, d.Variants)
}

// TestGetExperiment_StrategyChange checks a switch to a strategy w/o
// weights doesn't keep the previous strategy's weights: both come from the
// latest selection.
func TestGetExperiment_StrategyChange(t *testing.T) {
	f := seedExperiments(t)
	t6 := f.now.Add(-30 * time.Minute)
	f.run(t, "r6", f.checkoutFn, "app-checkout", t6)
	f.md(t, testMetadata{runID: "r6", queuedAt: t6, spanID: "r6-select", kind: "experiment", values: map[string]any{
		"name": "checkout", "variant": "a", "selection_strategy": "fixed", "available_variants": []string{"a", "b", "c"},
	}})

	d, err := dashboards.GetExperiment(t.Context(), f.db, f.accountID, f.envID, dashboards.ExperimentOpts{FunctionID: f.checkoutFn, Name: "checkout"})
	require.NoError(t, err)
	require.Equal(t, "fixed", d.SelectionStrategy)
	require.Empty(t, d.Weights)
}

func TestGetExperiment_NotFound(t *testing.T) {
	f := seedExperiments(t)

	// pricing exists, but not on checkoutFn.
	d, err := dashboards.GetExperiment(t.Context(), f.db, f.accountID, f.envID, dashboards.ExperimentOpts{FunctionID: f.checkoutFn, Name: "pricing"})
	require.NoError(t, err)
	require.Empty(t, d.Variants)

	_, err = dashboards.GetExperiment(t.Context(), f.db, f.accountID, f.envID, dashboards.ExperimentOpts{Name: "checkout"})
	var invalid *dashboards.InvalidArgumentError
	require.ErrorAs(t, err, &invalid)
}
