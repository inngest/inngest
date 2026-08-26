package duckdbquery

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/db/duckdb"
	"github.com/oklog/ulid/v2"
)

// scanNamedRows drains rows into a slice of column-name-keyed maps, using
// duckdb.ScanRowByName for every row — required because this driver's
// reported column order is not guaranteed to match the SELECT list (see
// pkg/db/duckdb/store.go's ScanRowByName doc comment).
func scanNamedRows(rows *sql.Rows) ([]map[string]any, error) {
	var out []map[string]any
	for rows.Next() {
		row, err := duckdb.ScanRowByName(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func stringField(row map[string]any, key string) (string, error) {
	v, ok := row[key].(string)
	if !ok {
		return "", fmt.Errorf("duckdbquery: expected string for column %q, got %T (%v)", key, row[key], row[key])
	}
	return v, nil
}

func uuidField(row map[string]any, key string) (uuid.UUID, error) {
	s, err := stringField(row, key)
	if err != nil {
		return uuid.UUID{}, err
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("duckdbquery: parsing column %q as UUID: %w", key, err)
	}
	return id, nil
}

func ulidField(row map[string]any, key string) (ulid.ULID, error) {
	s, err := stringField(row, key)
	if err != nil {
		return ulid.ULID{}, err
	}
	id, err := ulid.Parse(s)
	if err != nil {
		return ulid.ULID{}, fmt.Errorf("duckdbquery: parsing column %q as ULID: %w", key, err)
	}
	return id, nil
}

// nullableUUIDField returns nil when the column is SQL NULL rather than
// erroring — used for optional columns like events.source_id.
func nullableUUIDField(row map[string]any, key string) (*uuid.UUID, error) {
	if row[key] == nil {
		return nil, nil
	}
	id, err := uuidField(row, key)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func timeField(row map[string]any, key string) (time.Time, error) {
	return duckdb.AsTimestamp(row[key])
}

// nullableTimeField returns the zero time.Time (not an error) when the
// column is SQL NULL — matching cqrs.TraceRun's convention of a zero-value
// StartedAt/EndedAt meaning "not yet set" (see runs_v2.go's
// `if r.StartedAt.UnixMilli() > 0`).
func nullableTimeField(row map[string]any, key string) (time.Time, error) {
	if row[key] == nil {
		return time.Time{}, nil
	}
	return duckdb.AsTimestamp(row[key])
}

// jsonField re-marshals a JSON column's already-decoded Go value (the
// driver auto-decodes JSON-typed columns into map[string]any/[]any/etc.)
// back into raw bytes, so callers can treat it exactly like a TEXT/BLOB
// column read from any other backend. Returns nil, nil for a SQL NULL
// column.
func jsonField(row map[string]any, key string) ([]byte, error) {
	v := row[key]
	if v == nil {
		return nil, nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("duckdbquery: re-marshaling column %q: %w", key, err)
	}
	return b, nil
}
