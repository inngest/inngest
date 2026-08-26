package duckdbquery

import (
	"context"
	"fmt"
	"strings"

	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/db/duckdb"
	tracingv3 "github.com/inngest/inngest/pkg/tracing/v3"
	"github.com/oklog/ulid/v2"
)

const spanColumns = "span_id, trace_id, parent_span_id, start_time, end_time, name, attributes, run_id, app_id, function_id, output, input"

// GetSpansByRunID builds the run's span tree from inngest.run_trace_spans.
// Unlike pkg/cqrs/manager's dynamic/fragment-merged model, this table is
// flat — one physical row is one logical span — so there is no
// dynamic_span_id fragment-merge step: every row is scanned once, and
// children are attached to their parent by span_id/parent_span_id, in two
// passes (build every span first, then link) so a child's parent — however
// it happens to be ordered by start_time, which can tie for point-in-time
// spans — is always already present in the lookup map before it's needed.
func (m *Manager) GetSpansByRunID(ctx context.Context, runID ulid.ULID) (*cqrs.OtelSpan, error) {
	query := fmt.Sprintf(
		"SELECT %s FROM %s.run_trace_spans WHERE run_id = ? ORDER BY start_time ASC;",
		spanColumns, duckdb.DuckLakeAlias,
	)
	rows, err := m.db.QueryContext(ctx, query, runID.String())
	if err != nil {
		return nil, fmt.Errorf("duckdbquery: querying spans by run ID: %w", err)
	}
	defer rows.Close()

	named, err := scanNamedRows(rows)
	if err != nil {
		return nil, fmt.Errorf("duckdbquery: scanning span rows: %w", err)
	}
	if len(named) == 0 {
		return nil, fmt.Errorf("duckdbquery: no spans found for run %s", runID)
	}

	byID := make(map[string]*cqrs.OtelSpan, len(named))
	parentOf := make(map[string]string, len(named))
	order := make([]string, 0, len(named))

	for _, row := range named {
		span, parentSpanID, err := spanFromRow(ctx, row)
		if err != nil {
			return nil, err
		}
		byID[span.SpanID] = span
		order = append(order, span.SpanID)
		if parentSpanID != "" {
			parentOf[span.SpanID] = parentSpanID
		}
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
			// hasn't landed — e.g. a still-running function's
			// executor.run.queued/executor.run.started span, whose true
			// parent (executor.run) isn't written until OnFunctionFinished
			// (see listener.go). Kept as a fallback root candidate, not
			// promoted outright: a genuinely rootless span always wins.
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
	// genuineRoot is true when the span's own parent_span_id was empty
	// outright (e.g. "executor.run", the run's real root, written once at
	// OnFunctionFinished) rather than merely unresolved.
	genuineRoot bool
}

// selectRootSpan picks the run's root from every span with no resolvable
// parent. A genuinely rootless span always wins when present. Before a run
// finishes, none exists yet, so without this the tree would be mis-rooted by
// whichever orphaned executor.run.queued/executor.run.started span the query
// happened to return first — both point-in-time-tie on start_time (queuedAt)
// while queued, so scan order there is otherwise arbitrary. Deterministically
// prefer executor.run.started (the more advanced, more representative point
// in a still-running function's lifecycle) over executor.run.queued.
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

// GetSpanOutput backs Query.RunTraceSpanOutputByID's preview path
// (runs_v2.go's `id.Preview` branch). output/input are stored directly on
// the row by the dual-write span exporter, so this is a direct
// (run_id, span_id) lookup — no separate output-span indirection to chase.
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

	named, err := scanNamedRows(rows)
	if err != nil {
		return nil, fmt.Errorf("duckdbquery: scanning span output rows: %w", err)
	}

	so := &cqrs.SpanOutput{}
	for _, row := range named {
		output, err := jsonField(row, "output")
		if err != nil {
			return nil, err
		}
		input, err := jsonField(row, "input")
		if err != nil {
			return nil, err
		}
		if cqrs.UnwrapSpanOutputEnvelope(so, output, input) {
			break
		}
	}
	return so, nil
}

func spanFromRow(ctx context.Context, row map[string]any) (span *cqrs.OtelSpan, parentSpanID string, err error) {
	spanID, err := stringField(row, "span_id")
	if err != nil {
		return nil, "", err
	}
	traceID, err := stringField(row, "trace_id")
	if err != nil {
		return nil, "", err
	}
	name, err := stringField(row, "name")
	if err != nil {
		return nil, "", err
	}
	startTime, err := timeField(row, "start_time")
	if err != nil {
		return nil, "", err
	}
	endTime, err := timeField(row, "end_time")
	if err != nil {
		return nil, "", err
	}
	runID, err := ulidField(row, "run_id")
	if err != nil {
		return nil, "", err
	}
	appID, err := uuidField(row, "app_id")
	if err != nil {
		return nil, "", err
	}
	functionID, err := uuidField(row, "function_id")
	if err != nil {
		return nil, "", err
	}

	attrs, _ := row["attributes"].(map[string]any)
	if attrs == nil {
		attrs = map[string]any{}
	}

	if s, ok := row["parent_span_id"].(string); ok {
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

	// A span's own row carries its output/input directly (see GetSpanOutput's
	// doc comment) — no separate output-span indirection to resolve, unlike
	// the rollup path's fragment-merged model. So "appropriate" here mirrors
	// pkg/cqrs/manager/cqrs.go's own rule ("if this span has finished, set a
	// preliminary output ID") applied to the span's own ID for both output
	// and input: set OutputID whenever either column is non-NULL.
	output, err := jsonField(row, "output")
	if err != nil {
		return nil, "", err
	}
	input, err := jsonField(row, "input")
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

	return newSpan, parentSpanID, nil
}
