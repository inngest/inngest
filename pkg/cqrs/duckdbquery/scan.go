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

// asString type-asserts a scanned column value as a string — used by
// uuidColumn/ulidColumn, whose columns always come back that way (both the
// jsonlines and quack transports render UUID/ULID columns as their
// canonical string form — see pkg/db/duckdb/rows.go and quack_protocol.go).
func asString(v any, col string) (string, error) {
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("duckdbquery: expected string for column %q, got %T (%v)", col, v, v)
	}
	return s, nil
}

// uuidColumn parses a mandatory UUID column's scanned value.
func uuidColumn(v any, col string) (uuid.UUID, error) {
	s, err := asString(v, col)
	if err != nil {
		return uuid.UUID{}, err
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("duckdbquery: parsing column %q as UUID: %w", col, err)
	}
	return id, nil
}

// nullableUUIDColumn returns nil, rather than erroring, for a SQL NULL
// column — used for optional columns like events.source_id.
func nullableUUIDColumn(v any, col string) (*uuid.UUID, error) {
	if v == nil {
		return nil, nil
	}
	id, err := uuidColumn(v, col)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// ulidColumn parses a mandatory ULID column's scanned value.
func ulidColumn(v any, col string) (ulid.ULID, error) {
	s, err := asString(v, col)
	if err != nil {
		return ulid.ULID{}, err
	}
	id, err := ulid.Parse(s)
	if err != nil {
		return ulid.ULID{}, fmt.Errorf("duckdbquery: parsing column %q as ULID: %w", col, err)
	}
	return id, nil
}

// asMap requires the column's already-decoded JSON value (the driver
// auto-decodes JSON-typed columns into Go values) to be a JSON object, or
// SQL/JSON null — used for columns like events.event_data. A SQL NULL, or
// json.Marshal(nil) having written the literal JSON "null" (e.g. an event
// with no Data payload at all — see OnEventReceived), both decode to a Go
// nil here and mean "no data", so they map to an empty object rather than
// erroring; any other non-object shape (an array, a string, a number)
// indicates real data corruption worth surfacing rather than silently
// masking.
func asMap(v any, col string) (map[string]any, error) {
	if v == nil {
		return map[string]any{}, nil
	}
	m, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("duckdbquery: expected object for column %q, got %T (%v)", col, v, v)
	}
	return m, nil
}

// asJSON re-marshals a JSON column's already-decoded Go value (the driver
// auto-decodes JSON-typed columns into map[string]any/[]any/etc.) back into
// raw bytes, so callers can treat it exactly like a TEXT/BLOB column read
// from any other backend. Returns nil, nil for a SQL NULL column.
func asJSON(v any, col string) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("duckdbquery: re-marshaling column %q: %w", col, err)
	}
	return b, nil
}

// asTimestamp converts a scanned column value into a time.Time, deferring
// the actual transport-specific conversion to duckdb.AsTimestamp.
func asTimestamp(v any, col string) (time.Time, error) {
	ts, err := duckdb.AsTimestamp(v)
	if err != nil {
		return time.Time{}, fmt.Errorf("duckdbquery: parsing column %q: %w", col, err)
	}
	return ts, nil
}

// asNullableTimestamp returns the zero time.Time (not an error) when the
// column is SQL NULL — matching cqrs.TraceRun's convention of a zero-value
// StartedAt/EndedAt meaning "not yet set" (see runs_v2.go's
// `if r.StartedAt.UnixMilli() > 0`).
func asNullableTimestamp(v any, col string) (time.Time, error) {
	if v == nil {
		return time.Time{}, nil
	}
	return asTimestamp(v, col)
}

// scanCount scans a COUNT(*) AS c-shaped single-row, single-column result.
func scanCount(row *sql.Row) (int64, error) {
	var raw any
	if err := row.Scan(&raw); err != nil {
		return 0, fmt.Errorf("duckdbquery: scanning count: %w", err)
	}
	return duckdb.AsInt64(raw)
}
