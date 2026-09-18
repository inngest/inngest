package insights

import (
	"fmt"
	"strings"

	sq "github.com/doug-martin/goqu/v9"
	"github.com/inngest/inngest/pkg/duckdb/insights/trie"
)

// celFieldHandler converts one CEL predicate — already resolved to a
// specific field registration in a celFieldScope — into SQL. ident is the
// predicate's full original CEL field path (e.g. "event.data.foo.bar"); a
// wildcard handler derives whatever suffix it needs itself via
// strings.TrimPrefix against the fixed prefix it was registered under (see
// celWildFieldHandler).
type celFieldHandler func(ident string, literal any, op string) ([]sq.Expression, error)

// celFieldScope maps CEL field paths to the handler that converts a
// predicate on that field into SQL, via trie-based registration — adding a
// new field is a registration call, not an edit to a switch statement.
// This package registers two: one per query shape a CEL search can target
// (runsInputsCELScope, eventsTableCELScope below).
type celFieldScope = trie.Trie[[]string, string, celFieldHandler]

// runsInputsCELScope backs CELEventFilters/CELOutputFilters (used by
// pkg/cqrs/duckdbquery.GetTraceRuns).
//
// event.* (including event.data.*) predicates are converted against "x", a
// lambda parameter the caller supplies by wrapping the resulting SQL
// fragment in list_filter(json_transform(inputs, '["JSON"]'), lambda x:
// <fragment>) — see eventCELArrayMatchClause in runs.go.
// inngest.runs.inputs is a JSON array of the run's triggering event(s)
// (plural for a batch trigger), each JSON-encoded like pkg/event.Event's
// own json tags, so x is one whole triggering event and every event.*
// field is just another JSON path into it — no join to inngest.events
// needed.
//
// output.*/error.* target the "output" column, which holds the same
// {"data":T}/{"error":T} envelope DriverResponse.GetTraceFunctionOutput
// always produces.
var runsInputsCELScope = trie.New[[]string, string, celFieldHandler]().
	Add([]string{"event", "id"}, celJSONFieldHandler("x", "id")).
	Add([]string{"event", "name"}, celJSONFieldHandler("x", "name")).
	Add([]string{"event", "v"}, celJSONFieldHandler("x", "v")).
	Add([]string{"event", "ts"}, celJSONFieldHandler("x", "ts")).
	AddWild([]string{"event", "data"}, celWildJSONFieldHandler("x", "event.data", "data")).
	AddWild([]string{"output"}, celWildJSONFieldHandler("output", "output", "data")).
	AddWild([]string{"error"}, celWildJSONFieldHandler("output", "error", "error"))

// eventsTableCELScope backs CELEventTableFilters (used by
// pkg/cqrs/duckdbquery.GetEventsByExpressions to search inngest.events
// directly — one row per event, not an array of them). Unlike
// runsInputsCELScope, event.id/name/v/ts target inngest.events' own typed
// columns directly rather than a JSON path into a lambda parameter — only
// event.data.* needs JSON extraction, from event_data.
var eventsTableCELScope = trie.New[[]string, string, celFieldHandler]().
	Add([]string{"event", "id"}, celColumnEqHandler("event_id")).
	Add([]string{"event", "name"}, celColumnEqHandler("event_name")).
	Add([]string{"event", "v"}, celColumnEqHandler("event_v")).
	Add([]string{"event", "ts"}, celColumnTimestampHandler("event_ts")).
	AddWild([]string{"event", "data"}, celWildJSONFieldHandler("event_data", "event.data", ""))

// celJSONFieldHandler builds a handler for a field registered via Add — it
// always targets the same fixed JSON path off base, ignoring ident (which
// can only ever be the one path it was registered under).
func celJSONFieldHandler(base, fixedPath string) celFieldHandler {
	return func(_ string, literal any, op string) ([]sq.Expression, error) {
		return handleJSONFilter(base, fixedPath, literal, op)
	}
}

// celWildJSONFieldHandler builds a handler for a field registered via
// AddWild: it strips celPrefix (the CEL field path it was registered under,
// e.g. "event.data") off ident to recover whatever deeper path the caller
// actually referenced (e.g. "foo.bar" from "event.data.foo.bar"), then
// targets base's fixedJSONPrefix joined with that suffix — or, when
// fixedJSONPrefix is "" (base is itself the JSON value the CEL field names,
// as with eventsTableCELScope's event_data column), the bare suffix.
func celWildJSONFieldHandler(base, celPrefix, fixedJSONPrefix string) celFieldHandler {
	return func(ident string, literal any, op string) ([]sq.Expression, error) {
		suffix := strings.TrimPrefix(ident, celPrefix)
		suffix = strings.TrimPrefix(suffix, ".")

		fieldPath := suffix
		switch {
		case fixedJSONPrefix != "" && suffix != "":
			fieldPath = fixedJSONPrefix + "." + suffix
		case fixedJSONPrefix != "":
			fieldPath = fixedJSONPrefix
		}
		return handleJSONFilter(base, fieldPath, literal, op)
	}
}

// celColumnEqHandler builds a handler for a field registered via Add that
// targets a plain typed column directly (no JSON extraction) — only ==/!=
// are supported, matching pkg/run.EventFieldConverter's own event.id/name/v
// contract for the equivalent SQLite/Postgres column search.
func celColumnEqHandler(column string) celFieldHandler {
	return func(ident string, literal any, op string) ([]sq.Expression, error) {
		v, ok := literal.(string)
		if !ok {
			return nil, fmt.Errorf("expects %q to be a string: %v", ident, literal)
		}
		return handleStringOp(column, v, op)
	}
}

// celColumnTimestampHandler builds a handler for a field registered via Add
// that targets a TIMESTAMP_MS column directly. DuckDB has no BIGINT->
// TIMESTAMP_MS cast, so the column (not the int64-milliseconds CEL literal)
// is converted, via epoch_ms, rather than the other way around.
func celColumnTimestampHandler(column string) celFieldHandler {
	return func(ident string, literal any, op string) ([]sq.Expression, error) {
		ts, ok := literal.(int64)
		if !ok {
			return nil, fmt.Errorf("expects %q to be an integer: %v", ident, literal)
		}
		return handleNumericOp(fmt.Sprintf("epoch_ms(%s)", column), ts, op)
	}
}
