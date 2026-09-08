package duckdbquery

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/db/duckdb"
	tracingv3 "github.com/inngest/inngest/pkg/tracing/v3"
	"github.com/oklog/ulid/v2"
)

const spanColumns = "span_id, trace_id, parent_span_id, start_time, end_time, name, attributes, run_id, app_id, function_id, output, input"

// spanColumnsPrefixed is spanColumns with each column prefixed "s." for use
// inside GetSpansByRunID's latest_metadata CTE, which joins run_trace_spans
// (aliased s) against run_metadata (aliased m) and needs the two
// disambiguated.
const spanColumnsPrefixed = "s.span_id, s.trace_id, s.parent_span_id, s.start_time, s.end_time, s.name, s.attributes, s.run_id, s.app_id, s.function_id, s.output, s.input"

// GetSpansByRunID builds the run's span tree from inngest.run_trace_spans.
// This table is flat — one physical row is one logical span — so there's
// no fragment-merge step: every row is scanned once in two passes (build,
// then link) so a child's parent is always already in the lookup map by
// the time it's needed.
//
// Metadata (inngest.run_metadata) is aggregated in SQL rather than a
// second round trip: the latest_metadata CTE LEFT JOINs it and collapses
// multiple emissions of the same (span_id, kind) to the latest by
// created_at via QUALIFY, then the outer query GROUPs BY every span column
// and aggregates metadata into one LIST(STRUCT(...)) column per span.
func (m *Manager) GetSpansByRunID(ctx context.Context, runID ulid.ULID) (*cqrs.OtelSpan, error) {
	query := fmt.Sprintf(
		`WITH latest_metadata AS (
		   SELECT %s, m.scope, m.kind, m.values, m.created_at
		   FROM %s.run_trace_spans s
		   LEFT JOIN %s.run_metadata m
		    ON m.account_id = s.account_id AND m.env_id = s.env_id
		    AND m.run_id = s.run_id AND m.span_id = s.span_id
		   WHERE s.run_id = ?
		   QUALIFY m.kind IS NULL OR ROW_NUMBER() OVER (
		     PARTITION BY m.span_id, m.kind ORDER BY m.created_at DESC
		   ) = 1
		 )
		 SELECT
		   %s,
		   list({'scope': scope, 'kind': kind, 'values': values, 'created_at': created_at})
		     FILTER (WHERE kind IS NOT NULL) AS metadata
		 FROM latest_metadata
		 GROUP BY %s
		 ORDER BY start_time ASC;`,
		spanColumnsPrefixed, duckdb.DuckLakeAlias, duckdb.DuckLakeAlias,
		spanColumns, spanColumns,
	)
	rows, err := m.db.QueryContext(ctx, query, runID.String())
	if err != nil {
		return nil, fmt.Errorf("duckdbquery: querying spans by run ID: %w", err)
	}
	defer rows.Close()

	byID := make(map[string]*cqrs.OtelSpan)
	parentOf := make(map[string]string)
	var order []string

	for rows.Next() {
		span, parentSpanID, err := scanSpan(ctx, rows)
		if err != nil {
			return nil, fmt.Errorf("duckdbquery: scanning span row: %w", err)
		}
		byID[span.SpanID] = span
		order = append(order, span.SpanID)
		if parentSpanID != "" {
			parentOf[span.SpanID] = parentSpanID
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("duckdbquery: reading span rows: %w", err)
	}
	if len(order) == 0 {
		return nil, fmt.Errorf("duckdbquery: no spans found for run %s", runID)
	}

	var candidates []rootCandidate
	for _, id := range order {
		span := byID[id]
		if parentID, ok := parentOf[id]; ok {
			if parent, ok := byID[parentID]; ok {
				parent.Children = append(parent.Children, span)
				continue
			}
			// Orphaned, not rootless: parent_span_id is set but that row
			// hasn't landed yet (e.g. a still-running function's true root
			// span isn't written until OnFunctionFinished). Kept as a
			// fallback candidate — a genuinely rootless span always wins.
			candidates = append(candidates, rootCandidate{span: span})
			continue
		}
		candidates = append(candidates, rootCandidate{span: span, genuineRoot: true})
	}

	root := selectRootSpan(candidates)
	if root == nil {
		return nil, fmt.Errorf("duckdbquery: no root span found for run %s", runID)
	}
	return root, nil
}

type rootCandidate struct {
	span *cqrs.OtelSpan
	// genuineRoot is true when parent_span_id was empty outright, not
	// merely unresolved.
	genuineRoot bool
}

// selectRootSpan picks the run's root from every span with no resolvable
// parent. A genuinely rootless span always wins. Before a run finishes,
// none exists yet, so the tree would otherwise be mis-rooted by whichever
// orphaned span the query happened to scan first; deterministically prefer
// executor.run.started over executor.run.queued instead.
func selectRootSpan(candidates []rootCandidate) *cqrs.OtelSpan {
	var started, queued, first *cqrs.OtelSpan
	for _, c := range candidates {
		if c.genuineRoot {
			return c.span
		}
		if first == nil {
			first = c.span
		}
		switch c.span.Name {
		case tracingv3.SpanNameRunStarted:
			if started == nil {
				started = c.span
			}
		case tracingv3.SpanNameRunQueued:
			if queued == nil {
				queued = c.span
			}
		}
	}
	if started != nil {
		return started
	}
	if queued != nil {
		return queued
	}
	return first
}

// GetSpanOutput backs Query.RunTraceSpanOutputByID's preview path.
// output/input are stored directly on the row, so this is a direct
// (run_id, span_id) lookup — no separate output-span indirection.
func (m *Manager) GetSpanOutput(ctx context.Context, id cqrs.SpanIdentifier) (*cqrs.SpanOutput, error) {
	if id.SpanID == "" {
		return nil, fmt.Errorf("span ID is required to retrieve output")
	}
	if id.RunID == "" {
		return nil, fmt.Errorf("run ID is required to retrieve span output")
	}

	ids := []string{id.SpanID}
	if id.InputSpanID != nil && *id.InputSpanID != "" {
		ids = append(ids, *id.InputSpanID)
	}

	placeholders := make([]string, len(ids))
	args := make([]any, 0, len(ids)+1)
	args = append(args, id.RunID)
	for i, sid := range ids {
		placeholders[i] = "?"
		args = append(args, sid)
	}

	query := fmt.Sprintf(
		"SELECT output, input FROM %s.run_trace_spans WHERE run_id = ? AND span_id IN (%s);",
		duckdb.DuckLakeAlias, strings.Join(placeholders, ", "),
	)
	rows, err := m.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("duckdbquery: querying span output: %w", err)
	}
	defer rows.Close()

	so := &cqrs.SpanOutput{}
	for rows.Next() {
		var rawOutput, rawInput any
		if err := rows.Scan(&rawOutput, &rawInput); err != nil {
			return nil, fmt.Errorf("duckdbquery: scanning span output row: %w", err)
		}
		output, err := asJSON(rawOutput, "output")
		if err != nil {
			return nil, err
		}
		input, err := asJSON(rawInput, "input")
		if err != nil {
			return nil, err
		}
		if cqrs.UnwrapSpanOutputEnvelope(so, output, input) {
			break
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("duckdbquery: reading span output rows: %w", err)
	}
	return so, nil
}

// scanSpan scans one row of a spanColumns-plus-metadata-shaped SELECT (see
// GetSpansByRunID's query) into a *cqrs.OtelSpan. Destination order must
// match spanColumns exactly, followed by the aggregated metadata column.
func scanSpan(ctx context.Context, rows *sql.Rows) (span *cqrs.OtelSpan, parentSpanID string, err error) {
	var (
		spanID, traceID                   string
		rawParentSpanID                   any
		rawStartTime, rawEndTime          any
		name                              string
		rawAttributes                     any
		rawRunID, rawAppID, rawFunctionID any
		rawOutput, rawInput               any
		rawMetadata                       any
	)
	if err := rows.Scan(
		&spanID, &traceID, &rawParentSpanID, &rawStartTime, &rawEndTime, &name,
		&rawAttributes, &rawRunID, &rawAppID, &rawFunctionID, &rawOutput, &rawInput,
		&rawMetadata,
	); err != nil {
		return nil, "", err
	}

	startTime, err := asTimestamp(rawStartTime, "start_time")
	if err != nil {
		return nil, "", err
	}
	endTime, err := asTimestamp(rawEndTime, "end_time")
	if err != nil {
		return nil, "", err
	}
	runID, err := ulidColumn(rawRunID, "run_id")
	if err != nil {
		return nil, "", err
	}
	appID, err := uuidColumn(rawAppID, "app_id")
	if err != nil {
		return nil, "", err
	}
	functionID, err := uuidColumn(rawFunctionID, "function_id")
	if err != nil {
		return nil, "", err
	}

	attrs, _ := rawAttributes.(map[string]any)
	if attrs == nil {
		attrs = map[string]any{}
	}

	if s, ok := rawParentSpanID.(string); ok {
		parentSpanID = s
	}

	newSpan := &cqrs.OtelSpan{
		RawOtelSpan: cqrs.RawOtelSpan{
			Name:       name,
			SpanID:     spanID,
			TraceID:    traceID,
			StartTime:  startTime,
			EndTime:    endTime,
			Attributes: attrs,
		},
		RunID:      runID,
		AppID:      appID,
		FunctionID: functionID,
	}
	if parentSpanID != "" {
		newSpan.ParentSpanID = &parentSpanID
	}

	if err := cqrs.ApplyExtractedSpanAttributes(ctx, newSpan); err != nil {
		return nil, "", err
	}

	// Set OutputID whenever either column is non-NULL — a span's own row
	// carries its output/input directly, so this is always its own ID.
	output, err := asJSON(rawOutput, "output")
	if err != nil {
		return nil, "", err
	}
	input, err := asJSON(rawInput, "input")
	if err != nil {
		return nil, "", err
	}
	if len(output) > 0 || len(input) > 0 {
		preview := true
		id := cqrs.SpanIdentifier{RunID: runID.String(), SpanID: spanID, Preview: &preview}
		encoded, err := id.Encode()
		if err != nil {
			return nil, "", fmt.Errorf("duckdbquery: encoding span output ID: %w", err)
		}
		newSpan.OutputID = &encoded
	}

	newSpan.Metadata, err = scanSpanMetadata(rawMetadata)
	if err != nil {
		return nil, "", err
	}

	return newSpan, parentSpanID, nil
}

// scanSpanMetadata decodes GetSpansByRunID's aggregated
// LIST(STRUCT(scope, kind, values, created_at)) metadata column.
func scanSpanMetadata(raw any) ([]*cqrs.SpanMetadata, error) {
	return nil, nil // XXX: Implemented in PR that'll get merged into this
}
