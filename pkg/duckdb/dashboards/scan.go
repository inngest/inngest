package dashboards

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/db/duckdb"
)

// asString type-asserts a scanned column value as a string.
func asString(v any, col string) (string, error) {
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("expected string for column %q, got %T (%v)", col, v, v)
	}
	return s, nil
}

// asNullableString returns "" (not an error) for a SQL NULL column.
func asNullableString(v any) (string, error) {
	if v == nil {
		return "", nil
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("expected string, got %T (%v)", v, v)
	}
	return s, nil
}

// asUUID parses a mandatory UUID column's scanned value -- both a
// top-level UUID column and one nested inside a struct_pack'd value
// decode as a plain Go string (confirmed empirically; see sessions.go's
// doc comment).
func asUUID(v any, col string) (uuid.UUID, error) {
	s, err := asString(v, col)
	if err != nil {
		return uuid.UUID{}, err
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("parsing column %q as UUID: %w", col, err)
	}
	return id, nil
}

// asTime converts a scanned column value into a time.Time, deferring the
// actual transport-specific conversion to duckdb.AsTimestamp -- the same
// helper pkg/cqrs/duckdbquery's own scan.go uses.
func asTime(v any, col string) (time.Time, error) {
	ts, err := duckdb.AsTimestamp(v)
	if err != nil {
		return time.Time{}, fmt.Errorf("parsing column %q: %w", col, err)
	}
	return ts, nil
}

// asNullableTime returns the zero time.Time (not an error) for a SQL NULL
// column -- matching cqrs.TraceRun's own convention that a zero
// StartedAt/EndedAt means "not yet set".
func asNullableTime(v any) (time.Time, error) {
	if v == nil {
		return time.Time{}, nil
	}
	return asTime(v, "")
}

// asInt64 converts a scanned COUNT(...)-shaped column value to int64,
// deferring to duckdb.AsInt64 for the actual transport-specific
// conversion.
func asInt64(v any, col string) (int64, error) {
	n, err := duckdb.AsInt64(v)
	if err != nil {
		return 0, fmt.Errorf("parsing column %q: %w", col, err)
	}
	return n, nil
}

// asFunctionRefs decodes a `list_distinct(list(struct_pack(function_slug,
// app_id, app_name)))`-shaped column -- a LIST(STRUCT(...)) decodes as
// []any of map[string]any (confirmed empirically), one entry per distinct
// (function, app) pair. A NULL column (no matching rows at all, shouldn't
// occur since every session group here is built from at least one run)
// decodes to an empty slice rather than erroring.
func asFunctionRefs(v any) ([]FunctionRef, error) {
	if v == nil {
		return nil, nil
	}
	items, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("expected list for column %q, got %T (%v)", "functions", v, v)
	}
	out := make([]FunctionRef, len(items))
	for i, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("expected struct for functions[%d], got %T (%v)", i, item, item)
		}
		slug, err := asString(m["function_slug"], "functions[].function_slug")
		if err != nil {
			return nil, err
		}
		appID, err := asUUID(m["app_id"], "functions[].app_id")
		if err != nil {
			return nil, err
		}
		appName, err := asString(m["app_name"], "functions[].app_name")
		if err != nil {
			return nil, err
		}
		out[i] = FunctionRef{Slug: slug, AppID: appID, AppName: appName}
	}
	return out, nil
}
