package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/db/duckdb"
)

// DefaultTemplates returns a small, self-contained set of synthetic
// templates used when a source database has no real data to sample —
// GenerateRuns always has something usable to work from.
func DefaultTemplates() Templates {
	return Templates{
		Tenants: []Tenant{
			{
				AccountID:  uuid.New(),
				EnvID:      uuid.New(),
				AppID:      uuid.New(),
				FunctionID: uuid.New(),
			},
		},
		Statuses:   []string{"Queued", "Running", "Completed", "Completed", "Failed", "Cancelled"},
		SpanNames:  []string{"seeded-function"},
		EventNames: []string{"app/seeded.event"},
		Inputs:     []string{`{"event":{"name":"app/seeded.event","data":{}}}`},
		Outputs:    []string{`{"data":"ok"}`},
		Attributes: []string{`{"sys.step.name":"seeded-function"}`},
		EventData:  []string{`{}`},
	}
}

// BuildTemplates samples up to limit rows per table from db (see
// SampleTemplates) and falls back to DefaultTemplates when the source has
// no runs at all, so callers never need to special-case an empty database.
// sampled reports which path was taken, so callers can log it.
func BuildTemplates(ctx context.Context, db *sql.DB, limit int) (tmpl Templates, sampled bool, err error) {
	tmpl, err = SampleTemplates(ctx, db, limit)
	if err != nil {
		return Templates{}, false, err
	}
	if len(tmpl.Tenants) == 0 {
		return DefaultTemplates(), false, nil
	}
	return tmpl, true, nil
}

// SampleTemplates reads up to limit rows from each of
// inngest.runs/run_trace_spans/events and returns the distinct tenant
// tuples, statuses, names, and JSON payload shapes observed — a template
// GenerateRuns samples from to produce data that looks like it. Returns a
// zero-value Templates (no error) when the source tables are empty.
//
// Rows are scanned by column name (duckdb.ScanRowByName), not position:
// pkg/db/duckdb's driver builds each row from a Go map, so the column order
// it reports is not guaranteed to match the SELECT list.
func SampleTemplates(ctx context.Context, db *sql.DB, limit int) (Templates, error) {
	var tmpl Templates

	tenants, statuses, inputs, outputs, err := sampleRuns(ctx, db, limit)
	if err != nil {
		return Templates{}, err
	}
	tmpl.Tenants, tmpl.Statuses, tmpl.Inputs, tmpl.Outputs = tenants, statuses, inputs, outputs

	spanNames, attributes, err := sampleSpans(ctx, db, limit)
	if err != nil {
		return Templates{}, err
	}
	tmpl.SpanNames, tmpl.Attributes = spanNames, attributes

	eventNames, eventData, err := sampleEvents(ctx, db, limit)
	if err != nil {
		return Templates{}, err
	}
	tmpl.EventNames, tmpl.EventData = eventNames, eventData

	return tmpl, nil
}

func sampleRuns(ctx context.Context, db *sql.DB, limit int) (tenants []Tenant, statuses, inputs, outputs []string, err error) {
	query := fmt.Sprintf("SELECT account_id, env_id, app_id, function_id, status, inputs, output FROM %s.runs LIMIT ?;", duckdb.DuckLakeAlias)
	rows, err := db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("duckdbseed: sampling runs: %w", err)
	}
	defer rows.Close()

	seenTenants := map[Tenant]bool{}
	for rows.Next() {
		row, err := duckdb.ScanRowByName(rows)
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("duckdbseed: scanning sampled run: %w", err)
		}

		tenant, err := tenantFromRow(row)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		if !seenTenants[tenant] {
			seenTenants[tenant] = true
			tenants = append(tenants, tenant)
		}

		statuses = append(statuses, fmt.Sprint(row["status"]))
		if v := jsonText(row["inputs"]); v != "" {
			inputs = append(inputs, v)
		}
		if v := jsonText(row["output"]); v != "" {
			outputs = append(outputs, v)
		}
	}
	return tenants, statuses, inputs, outputs, rows.Err()
}

func sampleSpans(ctx context.Context, db *sql.DB, limit int) (names, attributes []string, err error) {
	query := fmt.Sprintf("SELECT name, attributes FROM %s.run_trace_spans LIMIT ?;", duckdb.DuckLakeAlias)
	rows, err := db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, nil, fmt.Errorf("duckdbseed: sampling spans: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		row, err := duckdb.ScanRowByName(rows)
		if err != nil {
			return nil, nil, fmt.Errorf("duckdbseed: scanning sampled span: %w", err)
		}
		names = append(names, fmt.Sprint(row["name"]))
		if v := jsonText(row["attributes"]); v != "" {
			attributes = append(attributes, v)
		}
	}
	return names, attributes, rows.Err()
}

func sampleEvents(ctx context.Context, db *sql.DB, limit int) (names, data []string, err error) {
	query := fmt.Sprintf("SELECT event_name, event_data FROM %s.events LIMIT ?;", duckdb.DuckLakeAlias)
	rows, err := db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, nil, fmt.Errorf("duckdbseed: sampling events: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		row, err := duckdb.ScanRowByName(rows)
		if err != nil {
			return nil, nil, fmt.Errorf("duckdbseed: scanning sampled event: %w", err)
		}
		names = append(names, fmt.Sprint(row["event_name"]))
		if v := jsonText(row["event_data"]); v != "" {
			data = append(data, v)
		}
	}
	return names, data, rows.Err()
}

func tenantFromRow(row map[string]any) (Tenant, error) {
	accountID, err := uuidField(row, "account_id")
	if err != nil {
		return Tenant{}, err
	}
	envID, err := uuidField(row, "env_id")
	if err != nil {
		return Tenant{}, err
	}
	appID, err := uuidField(row, "app_id")
	if err != nil {
		return Tenant{}, err
	}
	functionID, err := uuidField(row, "function_id")
	if err != nil {
		return Tenant{}, err
	}
	return Tenant{AccountID: accountID, EnvID: envID, AppID: appID, FunctionID: functionID}, nil
}

// uuidField parses a UUID column's value — pkg/db/duckdb's driver (both the
// jsonlines and quack transports) renders a UUID column as its canonical
// string form, not raw bytes.
func uuidField(row map[string]any, key string) (uuid.UUID, error) {
	s, ok := row[key].(string)
	if !ok {
		return uuid.UUID{}, fmt.Errorf("duckdbseed: expected string for column %q, got %T", key, row[key])
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("duckdbseed: parsing column %q as UUID: %w", key, err)
	}
	return id, nil
}

// jsonText re-marshals a JSON column's already-decoded Go value (the
// duckdb driver auto-decodes JSON-typed columns into
// map[string]any/[]any/etc.) back into text, so it can be reused verbatim
// as a template value. Returns "" for SQL NULL or a marshal failure.
func jsonText(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}
