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
// uuidColumn/ulidColumn, since both transports render UUID/ULID columns as
// their canonical string form.
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

// asMap requires the column's already-decoded JSON value to be an object,
// or SQL/JSON null — used for columns like events.event_data. Both a SQL
// NULL and a stored JSON "null" decode to a Go nil and mean "no data", so
// they map to an empty object; any other non-object shape is real
// corruption worth surfacing.
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

// asJSON re-marshals a JSON column's already-decoded Go value back into raw
// bytes, so callers can treat it like a TEXT/BLOB column from any other
// backend. Returns nil, nil for a SQL NULL column.
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

// asNullableTimestamp returns the zero time.Time (not an error) for a SQL
// NULL column — matching cqrs.TraceRun's convention that a zero
// StartedAt/EndedAt means "not yet set".
func asNullableTimestamp(v any, col string) (time.Time, error) {
	if v == nil {
		return time.Time{}, nil
	}
	return asTimestamp(v, col)
}

// asNullableBool returns false (not an error) for a SQL NULL column. Its
// only caller, is_deferred, is BOOLEAN NOT NULL DEFAULT FALSE so this
// branch shouldn't be reachable in practice, but costs nothing to keep as a
// defensive default rather than erroring on an unexpected NULL.
func asNullableBool(v any, col string) (bool, error) {
	if v == nil {
		return false, nil
	}
	b, ok := v.(bool)
	if !ok {
		return false, fmt.Errorf("duckdbquery: expected bool for column %q, got %T (%v)", col, v, v)
	}
	return b, nil
}

// scanCount scans a COUNT(*) AS c-shaped single-row, single-column result.
func scanCount(row *sql.Row) (int64, error) {
	var raw any
	if err := row.Scan(&raw); err != nil {
		return 0, fmt.Errorf("duckdbquery: scanning count: %w", err)
	}
	return duckdb.AsInt64(raw)
}
