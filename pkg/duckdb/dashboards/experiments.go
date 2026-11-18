// experiments.go holds the Experiments domain's registry entries, the OSS
// counterpart of Cloud's pkg/applogic/experiments. It reads the experiment
// assignments and scores Inngest records as inngest.run_metadata, w/ fixed
// SQL and bound args: scores come back as rows (or a JSON list per variant)
// rather than Cloud's per-key positional columns, so nothing about a score
// name ever reaches the SQL text.
//
// An experiment's assignment and its scores are Inngest metadata on the same
// span: group.experiment() writes kind experiment on its selection step, the
// executor stamps it on every step inside the variant callback, and a run
// scored against an experiment (scoreExperiment, POST .../scores w/ an
// experiment) gets it on the run itself. A score counts toward a variant
// only through that shared (run_id, span_id), so a run in two experiments
// can't count one experiment's scores toward the other. Kinds are stored w/o
// their inngest. prefix: experiment, score.<name>, or the legacy score kind
// keyed by name.
//
// Unlike Cloud, which reads assignments from the steps table and scores from
// steps and runs separately and merges them, both shapes are spans here: a
// variant's runs are every run w/ a span assigned to it (the selection step
// included, so a variant whose callback runs no step.run still counts).
package dashboards

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	KeyExperiments = "experiments"
	KeyExperiment  = "experiment"
)

// defaultExperimentWindow is the window ListExperiments/GetExperiment read
// w/o a time range, the same default Sessions uses (Cloud defaults to the
// account's history entitlement, which OSS doesn't have).
const defaultExperimentWindow = defaultSessionWindow

// experimentsSQL lists experiments, one row per (name, function), most
// recently seen first. Each span's latest experiment emission wins, and
// selection_strategy/available_variants are only on the selection step
// (NULL elsewhere). A run's function never changes across its lifecycle
// rows, so any row will do, and queued_at is run_queued_at, so the same
// bounds prune both tables. Every NULL-able arg is typed so `? IS NULL`
// binds.
const experimentsSQL = `
WITH latest AS (
  SELECT run_id, run_queued_at, "values"::JSON AS v
  FROM inngest.run_metadata
  WHERE account_id = ?::UUID AND env_id = ?::UUID
    AND run_queued_at >= ?::TIMESTAMPTZ AND run_queued_at < ?::TIMESTAMPTZ
    AND NOT is_user AND kind = 'experiment'
  QUALIFY ROW_NUMBER() OVER (PARTITION BY run_id, span_id ORDER BY created_at DESC, "values" DESC) = 1
),
spans AS (
  SELECT
    run_id,
    run_queued_at,
    v->>'name' AS name,
    v->>'variant' AS variant,
    NULLIF(v->>'selection_strategy', '') AS selection_strategy,
    CASE WHEN json_type(v->'available_variants') = 'ARRAY'
      THEN json_transform(v->'available_variants', '["VARCHAR"]') END AS available_variants
  FROM latest
  WHERE COALESCE(v->>'name', '') <> '' AND COALESCE(v->>'variant', '') <> ''
),
functions AS (
  SELECT DISTINCT ON (run_id) run_id, function_id, function_slug, app_id, app_name
  FROM inngest.runs
  WHERE account_id = ?::UUID AND env_id = ?::UUID
    AND queued_at >= ?::TIMESTAMPTZ AND queued_at < ?::TIMESTAMPTZ
    AND (?::UUID IS NULL OR function_id = ?::UUID)
    AND run_id IN (SELECT run_id FROM spans)
)
SELECT
  s.name,
  f.function_id,
  max(f.function_slug) AS function_slug,
  max(f.app_id) AS app_id,
  max(f.app_name) AS app_name,
  arg_max(s.selection_strategy, s.run_queued_at) FILTER (WHERE s.selection_strategy IS NOT NULL) AS selection_strategy,
  arg_max(s.available_variants, s.run_queued_at) FILTER (WHERE s.available_variants IS NOT NULL) AS available_variants,
  list_sort(list_distinct(list(s.variant))) AS observed_variants,
  count(DISTINCT s.run_id) AS total_runs,
  min(s.run_queued_at) AS first_seen,
  max(s.run_queued_at) AS last_seen
FROM spans s
JOIN functions f USING (run_id)
GROUP BY s.name, f.function_id
HAVING ?::TIMESTAMPTZ IS NULL
  OR max(s.run_queued_at) < ?::TIMESTAMPTZ
  OR (max(s.run_queued_at) = ?::TIMESTAMPTZ AND (f.function_id < ?::UUID OR (f.function_id = ?::UUID AND s.name < ?::VARCHAR)))
ORDER BY last_seen DESC, f.function_id DESC, s.name DESC
LIMIT ?;`

// experimentSQL is one experiment on one function, one row per variant: run
// counts, first and last seen, the selection step's strategy and weights w/
// when they were recorded (strategy_at, so the latest across variants
// wins), and the scores on the variant's spans as a JSON list of {key, avg,
// min, max}. A bool scores as 1/0 and anything else non-numeric is
// dropped, and a name in both score shapes on one span takes the per-name
// kind.
const experimentSQL = `
WITH latest AS (
  SELECT run_id, run_queued_at, span_id, kind, "values"::JSON AS v
  FROM inngest.run_metadata
  WHERE account_id = ?::UUID AND env_id = ?::UUID
    AND run_queued_at >= ?::TIMESTAMPTZ AND run_queued_at < ?::TIMESTAMPTZ
    AND NOT is_user
    AND (kind = 'experiment' OR kind = 'score' OR starts_with(kind, 'score.'))
  QUALIFY ROW_NUMBER() OVER (PARTITION BY run_id, span_id, kind ORDER BY created_at DESC, "values" DESC) = 1
),
spans AS (
  SELECT
    run_id,
    run_queued_at,
    span_id,
    v->>'variant' AS variant,
    NULLIF(v->>'selection_strategy', '') AS selection_strategy,
    CASE WHEN json_type(v->'variant_weights') = 'OBJECT'
      THEN json_transform(v->'variant_weights', '"MAP(VARCHAR, DOUBLE)"') END AS variant_weights
  FROM latest
  WHERE kind = 'experiment' AND v->>'name' = ?::VARCHAR AND COALESCE(v->>'variant', '') <> ''
    AND run_id IN (
      SELECT run_id FROM inngest.runs
      WHERE account_id = ?::UUID AND env_id = ?::UUID
        AND queued_at >= ?::TIMESTAMPTZ AND queued_at < ?::TIMESTAMPTZ
        AND function_id = ?::UUID
    )
),
raw_scores AS (
  SELECT run_id, span_id, substr(kind, 7) AS key, v->'value' AS value, 0 AS precedence
  FROM latest
  WHERE starts_with(kind, 'score.')
  UNION ALL
  SELECT run_id, span_id, e.score.key AS key, e.score.value->'value' AS value, 1 AS precedence
  FROM latest, unnest(map_entries(json_transform(v, '"MAP(VARCHAR, JSON)"'))) AS e(score)
  WHERE kind = 'score' AND json_type(v) = 'OBJECT'
),
scores AS (
  SELECT
    run_id,
    span_id,
    key,
    CASE json_type(value) WHEN 'BOOLEAN' THEN value::BOOLEAN::DOUBLE ELSE TRY_CAST(value AS DOUBLE) END AS value
  FROM raw_scores
  QUALIFY ROW_NUMBER() OVER (PARTITION BY run_id, span_id, key ORDER BY precedence) = 1
),
metrics AS (
  SELECT s.variant, sc.key, avg(sc.value) AS avg, min(sc.value) AS min, max(sc.value) AS max
  FROM spans s
  JOIN scores sc USING (run_id, span_id)
  WHERE sc.value IS NOT NULL
  GROUP BY s.variant, sc.key
),
variant_metrics AS (
  SELECT variant, to_json(list({'key': key, 'avg': avg, 'min': min, 'max': max} ORDER BY key))::VARCHAR AS metrics
  FROM metrics
  GROUP BY variant
),
variants AS (
  SELECT
    variant,
    count(DISTINCT run_id) AS run_count,
    min(run_queued_at) AS first_seen,
    max(run_queued_at) AS last_seen,
    arg_max(selection_strategy, run_queued_at) FILTER (WHERE selection_strategy IS NOT NULL) AS selection_strategy,
    to_json(arg_max(variant_weights, run_queued_at) FILTER (WHERE selection_strategy IS NOT NULL))::VARCHAR AS variant_weights,
    max(run_queued_at) FILTER (WHERE selection_strategy IS NOT NULL) AS strategy_at
  FROM spans
  GROUP BY variant
)
SELECT v.*, m.metrics
FROM variants v
LEFT JOIN variant_metrics m USING (variant)
ORDER BY v.variant;`

func tenantArgs(scope Scope) []any {
	return []any{scope.AccountID.String(), scope.EnvID.String(), scope.From, scope.To}
}

// nullableUUID binds uuid.Nil as NULL.
func nullableUUID(id uuid.UUID) any {
	if id == uuid.Nil {
		return nil
	}
	return id.String()
}

func buildExperimentsSQL(scope Scope) (string, []any, error) {
	args := tenantArgs(scope)
	args = append(args, tenantArgs(scope)...)
	fn := nullableUUID(scope.FunctionID)
	args = append(args, fn, fn)

	cur, err := decodeExperimentCursor(scope.Cursor)
	if err != nil {
		return "", nil, err
	}
	if cur == nil {
		args = append(args, nil, nil, nil, nil, nil, nil)
	} else {
		fnID := cur.functionID.String()
		args = append(args, cur.lastSeen, cur.lastSeen, cur.lastSeen, fnID, fnID, cur.name)
	}

	args = append(args, resolveLimit(scope.Limit)+1)
	return experimentsSQL, args, nil
}

func buildExperimentSQL(scope Scope) (string, []any, error) {
	if scope.FunctionID == uuid.Nil || scope.ExperimentName == "" {
		return "", nil, fmt.Errorf("function id and experiment name are required")
	}
	args := tenantArgs(scope)
	args = append(args, scope.ExperimentName)
	args = append(args, tenantArgs(scope)...)
	args = append(args, scope.FunctionID.String())
	return experimentSQL, args, nil
}

// experimentCursor is the keyset ListExperiments pages by: (last_seen,
// function_id, name), all descending.
type experimentCursor struct {
	lastSeen   time.Time
	functionID uuid.UUID
	name       string
}

// A cursor's ID is "<function uuid>:<name>". The UUID's text is fixed
// width, so a name w/ colons still splits cleanly.
const experimentCursorField = "last_seen"

func encodeExperimentCursor(functionID uuid.UUID, name string, lastSeen time.Time) (string, error) {
	return encodeSessionCursor(functionID.String()+":"+name, experimentCursorField, lastSeen)
}

func decodeExperimentCursor(cursorStr string) (*experimentCursor, error) {
	cur, err := decodeSessionCursor(cursorStr, experimentCursorField)
	if err != nil || cur == nil {
		return nil, err
	}
	fnID, name, ok := strings.Cut(cur.id, ":")
	if !ok {
		return nil, invalidArgumentf("invalid cursor: not a cursor for this listing")
	}
	functionID, err := uuid.Parse(fnID)
	if err != nil {
		return nil, invalidArgumentf("invalid cursor: not a cursor for this listing")
	}
	// The cursor stores ms, so the keyset compares at ms too: last_seen is
	// run_queued_at, a ULID time, which is ms already.
	return &experimentCursor{lastSeen: cur.at, functionID: functionID, name: name}, nil
}

// ResolveExperimentTimeRange defaults a zero until to now and a zero from to
// defaultExperimentWindow before until, like ResolveSessionTimeRange.
func ResolveExperimentTimeRange(from, until time.Time) (time.Time, time.Time, error) {
	if until.IsZero() {
		until = time.Now().UTC()
	}
	if from.IsZero() {
		from = until.Add(-defaultExperimentWindow)
	}
	if !from.Before(until) {
		return time.Time{}, time.Time{}, invalidArgumentf(
			"invalid time range: from (%s) must be before until (%s)",
			from.Format(time.RFC3339), until.Format(time.RFC3339),
		)
	}
	return from, until, nil
}

// Experiment is one experiment observed on one function.
type Experiment struct {
	Name              string
	Function          FunctionRef
	FunctionID        uuid.UUID
	SelectionStrategy string
	// Variants are the selection step's declared variants in order, then
	// any other variant runs were assigned to.
	Variants  []string
	TotalRuns int
	FirstSeen time.Time
	LastSeen  time.Time
	Cursor    string
}

// ExperimentsOpts are ListExperiments' caller-provided parameters.
// FunctionID, if set, scopes the listing to one function.
type ExperimentsOpts struct {
	FunctionID uuid.UUID
	From       time.Time
	Until      time.Time
	Cursor     string
	Limit      int
}

// ExperimentsResult is ListExperiments' output.
type ExperimentsResult struct {
	Experiments []Experiment
	HasMore     bool
}

// ListExperiments lists the experiments observed in (accountID, envID), one
// per (name, function), most recently seen first.
func ListExperiments(ctx context.Context, db *sql.DB, accountID, envID uuid.UUID, opts ExperimentsOpts) (*ExperimentsResult, error) {
	from, until, err := ResolveExperimentTimeRange(opts.From, opts.Until)
	if err != nil {
		return nil, fmt.Errorf("dashboard: %w", err)
	}
	scope := Scope{
		AccountID: accountID, EnvID: envID,
		From: from, To: until,
		FunctionID: opts.FunctionID,
		Cursor:     opts.Cursor, Limit: opts.Limit,
	}
	result, err := Get(ctx, db, scope, KeyExperiments)
	if err != nil {
		return nil, fmt.Errorf("dashboard: listing experiments: %w", err)
	}

	limit := resolveLimit(opts.Limit)
	idx := columnIndex(result.Columns)
	out := make([]Experiment, 0, len(result.Rows))
	for _, row := range result.Rows {
		exp, err := parseExperimentRow(row, idx)
		if err != nil {
			return nil, fmt.Errorf("dashboard: %w", err)
		}
		out = append(out, exp)
	}

	hasMore := len(out) > limit
	if hasMore {
		out = out[:limit]
	}
	return &ExperimentsResult{Experiments: out, HasMore: hasMore}, nil
}

func parseExperimentRow(row []any, idx map[string]int) (Experiment, error) {
	name, err := asString(col(row, idx, "name"), "name")
	if err != nil {
		return Experiment{}, err
	}
	functionID, err := asUUID(col(row, idx, "function_id"), "function_id")
	if err != nil {
		return Experiment{}, err
	}
	slug, err := asString(col(row, idx, "function_slug"), "function_slug")
	if err != nil {
		return Experiment{}, err
	}
	appID, err := asUUID(col(row, idx, "app_id"), "app_id")
	if err != nil {
		return Experiment{}, err
	}
	appName, err := asString(col(row, idx, "app_name"), "app_name")
	if err != nil {
		return Experiment{}, err
	}
	strategy, err := asNullableString(col(row, idx, "selection_strategy"))
	if err != nil {
		return Experiment{}, err
	}
	declared, err := asStrings(col(row, idx, "available_variants"), "available_variants")
	if err != nil {
		return Experiment{}, err
	}
	observed, err := asStrings(col(row, idx, "observed_variants"), "observed_variants")
	if err != nil {
		return Experiment{}, err
	}
	totalRuns, err := asInt64(col(row, idx, "total_runs"), "total_runs")
	if err != nil {
		return Experiment{}, err
	}
	firstSeen, err := asTime(col(row, idx, "first_seen"), "first_seen")
	if err != nil {
		return Experiment{}, err
	}
	lastSeen, err := asTime(col(row, idx, "last_seen"), "last_seen")
	if err != nil {
		return Experiment{}, err
	}
	cursor, err := encodeExperimentCursor(functionID, name, lastSeen)
	if err != nil {
		return Experiment{}, fmt.Errorf("encoding experiment cursor: %w", err)
	}

	return Experiment{
		Name:              name,
		Function:          FunctionRef{Slug: slug, AppID: appID, AppName: appName},
		FunctionID:        functionID,
		SelectionStrategy: strategy,
		Variants:          mergeVariants(declared, observed),
		TotalRuns:         int(totalRuns),
		FirstSeen:         firstSeen,
		LastSeen:          lastSeen,
		Cursor:            cursor,
	}, nil
}

// mergeVariants is declared in order, then each observed variant declared
// doesn't have (observed is sorted).
func mergeVariants(declared, observed []string) []string {
	out := make([]string, 0, len(declared)+len(observed))
	for _, v := range declared {
		if !slices.Contains(out, v) {
			out = append(out, v)
		}
	}
	for _, v := range observed {
		if !slices.Contains(out, v) {
			out = append(out, v)
		}
	}
	return out
}

// ExperimentMetric is one score key's aggregate over a variant's spans.
type ExperimentMetric struct {
	Key           string
	Avg, Min, Max float64
}

// ExperimentVariant is one variant's run count and metrics.
type ExperimentVariant struct {
	Name     string
	RunCount int
	Metrics  []ExperimentMetric
}

// ExperimentWeight is a variant's weight in a weighted selection.
type ExperimentWeight struct {
	Variant string
	Weight  float64
}

// ExperimentDetail is one experiment's per-variant aggregates. Variants is
// empty when the experiment wasn't seen on the function in the window.
type ExperimentDetail struct {
	Name              string
	SelectionStrategy string
	Variants          []ExperimentVariant
	Weights           []ExperimentWeight
	FirstSeen         time.Time
	LastSeen          time.Time
}

// ExperimentOpts are GetExperiment's caller-provided parameters.
type ExperimentOpts struct {
	FunctionID uuid.UUID
	Name       string
	From       time.Time
	Until      time.Time
}

// GetExperiment aggregates one experiment on one function per variant: run
// counts and each score key's avg/min/max.
func GetExperiment(ctx context.Context, db *sql.DB, accountID, envID uuid.UUID, opts ExperimentOpts) (*ExperimentDetail, error) {
	if opts.FunctionID == uuid.Nil || opts.Name == "" {
		return nil, invalidArgumentf("function id and experiment name are required")
	}
	from, until, err := ResolveExperimentTimeRange(opts.From, opts.Until)
	if err != nil {
		return nil, fmt.Errorf("dashboard: %w", err)
	}
	scope := Scope{
		AccountID: accountID, EnvID: envID,
		From: from, To: until,
		FunctionID: opts.FunctionID, ExperimentName: opts.Name,
	}

	result, err := Get(ctx, db, scope, KeyExperiment)
	if err != nil {
		return nil, fmt.Errorf("dashboard: fetching experiment: %w", err)
	}
	detail := &ExperimentDetail{Name: opts.Name}
	var strategyAt time.Time
	idx := columnIndex(result.Columns)
	for _, row := range result.Rows {
		variant := ExperimentVariant{}
		if variant.Name, err = asString(col(row, idx, "variant"), "variant"); err != nil {
			return nil, fmt.Errorf("dashboard: %w", err)
		}
		runCount, err := asInt64(col(row, idx, "run_count"), "run_count")
		if err != nil {
			return nil, fmt.Errorf("dashboard: %w", err)
		}
		variant.RunCount = int(runCount)
		if variant.Metrics, err = asMetrics(col(row, idx, "metrics")); err != nil {
			return nil, fmt.Errorf("dashboard: %w", err)
		}
		detail.Variants = append(detail.Variants, variant)

		firstSeen, err := asTime(col(row, idx, "first_seen"), "first_seen")
		if err != nil {
			return nil, fmt.Errorf("dashboard: %w", err)
		}
		lastSeen, err := asTime(col(row, idx, "last_seen"), "last_seen")
		if err != nil {
			return nil, fmt.Errorf("dashboard: %w", err)
		}
		if detail.FirstSeen.IsZero() || firstSeen.Before(detail.FirstSeen) {
			detail.FirstSeen = firstSeen
		}
		if lastSeen.After(detail.LastSeen) {
			detail.LastSeen = lastSeen
		}

		at, err := asNullableTime(col(row, idx, "strategy_at"))
		if err != nil {
			return nil, fmt.Errorf("dashboard: %w", err)
		}
		if !at.IsZero() && at.After(strategyAt) {
			strategyAt = at
			if detail.SelectionStrategy, err = asNullableString(col(row, idx, "selection_strategy")); err != nil {
				return nil, fmt.Errorf("dashboard: %w", err)
			}
			if detail.Weights, err = asWeights(col(row, idx, "variant_weights")); err != nil {
				return nil, fmt.Errorf("dashboard: %w", err)
			}
		}
	}
	return detail, nil
}
