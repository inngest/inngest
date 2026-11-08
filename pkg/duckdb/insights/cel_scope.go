package insights

import (
	"fmt"
	"strings"

	sq "github.com/doug-martin/goqu/v9"
	"github.com/google/cel-go/common/operators"
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
// fragment in list_filter(json_transform(inputs::JSON, '["JSON"]'), lambda x:
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
	Add([]string{"event", "id"}, celEventStringFieldHandler(celNullableSQL("(x::JSON->>?)", "$.id"))).
	Add([]string{"event", "name"}, celEventStringFieldHandler(celNullableSQL("(x::JSON->>?)", "$.name"))).
	Add([]string{"event", "v"}, celEventVersionHandler(celNullableSQL("(x::JSON->>?)", "$.v"))).
	Add([]string{"event", "ts"}, celEventTimestampHandler(celNullableSQL("TRY_CAST((x::JSON->>?) AS DOUBLE)", "$.ts"))).
	AddWild([]string{"event", "data"}, celEventDataHandler(celWildJSONFieldHandler("x", "event.data", "data"))).
	AddWild([]string{"output"}, celWildJSONFieldHandler("output", "output", "data")).
	AddWild([]string{"error"}, celWildJSONFieldHandler("output", "error", "error"))

// runsInputsMatchSQL is the inputs-array match CELOutputFilters wraps an
// event.* fragment in itself, for an event.* part of a conjunct that also
// references output.*/error.* and so can't be handed to CELEventFilters'
// caller (see celExprsToSQL). It must stay equivalent to pkg/duckdb/
// query's eventCELArrayMatchClause, which applies the same match to
// CELEventFilters' result.
const runsInputsMatchSQL = `len(list_filter(json_transform(inputs::JSON, '["JSON"]'), lambda x: ?)) > 0`

// eventsTableCELScope backs CELEventTableFilters (used by
// pkg/cqrs/duckdbquery.GetEventsByExpressions to search inngest.events
// directly — one row per event, not an array of them). Unlike
// runsInputsCELScope, event.id/name/v/ts target inngest.events' own typed
// columns directly rather than a JSON path into a lambda parameter — only
// event.data.* needs JSON extraction, from event_data.
var eventsTableCELScope = trie.New[[]string, string, celFieldHandler]().
	Add([]string{"event", "id"}, celEventStringFieldHandler(celColumnSQL("event_id"))).
	Add([]string{"event", "name"}, celEventStringFieldHandler(celColumnSQL("event_name"))).
	Add([]string{"event", "v"}, celEventVersionHandler(celColumnSQL("event_v"))).
	Add([]string{"event", "ts"}, celEventTimestampHandler(celColumnSQL("epoch_ms(event_ts)"))).
	AddWild([]string{"event", "data"}, celEventDataHandler(celWildJSONFieldHandler("event_data", "event.data", "")))

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

// The handlers below cover event.id/name/v/ts/data, which (unlike any
// other JSON path) pkg/event.Event.Map() never exposes as "missing": the Go
// re-check (see celFieldConverter) sees "" for a missing/null id or name, 0
// for a missing/null ts, and an empty map for missing data, while a v that's
// missing, null, or "" is left out of the map (so it compares like null).
// Each handler's SQL has to account for that or it'd drop rows the Go side
// matches, e.g. event_v != 'x' for an unversioned event (stored as '') that
// event.v != "x" matches.

// celEventFieldSQL is the SQL for one of those fields' raw values.
type celEventFieldSQL struct {
	expr string
	args []any
	// nullable is set when expr is SQL NULL for a missing/null value (a
	// JSON path), rather than a NOT NULL column already holding the zero
	// value.
	nullable bool
}

func celNullableSQL(expr string, args ...any) celEventFieldSQL {
	return celEventFieldSQL{expr: expr, args: args, nullable: true}
}

func celColumnSQL(column string) celEventFieldSQL {
	return celEventFieldSQL{expr: column}
}

// orZero returns f's value with a missing/null value replaced by zero (a
// SQL literal), plus its args.
func (f celEventFieldSQL) orZero(zero string) (string, []any) {
	if !f.nullable {
		return f.expr, f.args
	}
	return fmt.Sprintf("COALESCE(%s, %s)", f.expr, zero), f.args
}

// celEventStringFieldHandler handles event.id/event.name. Only a string
// literal is constrained: Go compares any other literal against a string
// field as a type mismatch, which is a non-match, so leaving it
// unconstrained is still a superset.
func celEventStringFieldHandler(f celEventFieldSQL) celFieldHandler {
	return func(_ string, literal any, op string) ([]sq.Expression, error) {
		v, ok := literal.(string)
		if !ok {
			return []sq.Expression{}, nil
		}
		expr, args := f.orZero("''")
		return handleStringOp(sq.L(expr, args...), v, op)
	}
}

// celEventVersionHandler handles event.v, whose Go value is missing (null)
// rather than "" for an unversioned event.
func celEventVersionHandler(f celEventFieldSQL) celFieldHandler {
	v, args := f.orZero("''")
	return func(_ string, literal any, op string) ([]sq.Expression, error) {
		switch lit := literal.(type) {
		case nil:
			switch op {
			case operators.Equals:
				return []sq.Expression{sq.L(v+" = ''", args...)}, nil
			case operators.NotEquals:
				return []sq.Expression{sq.L(v+" != ''", args...)}, nil
			}
		case string:
			switch op {
			case operators.Equals:
				// == "" never matches in Go, so this is just a superset then.
				return []sq.Expression{sq.L(v, args...).Eq(lit)}, nil
			case operators.NotEquals:
				// A missing v != "x" holds too.
				both := append(append(append([]any{}, args...), args...), lit)
				return []sq.Expression{sq.L(fmt.Sprintf("(%s = '' OR %s != ?)", v, v), both...)}, nil
			}
		}
		return []sq.Expression{}, nil
	}
}

// celEventTimestampHandler handles event.ts (epoch milliseconds). f must
// already be numeric.
func celEventTimestampHandler(f celEventFieldSQL) celFieldHandler {
	return func(_ string, literal any, op string) ([]sq.Expression, error) {
		switch literal.(type) {
		case int64, float64:
			expr, args := f.orZero("0")
			return handleNumericOp(sq.L(expr, args...), literal, op)
		}
		return []sq.Expression{}, nil
	}
}

// celEventDataHandler wraps event.data's JSON handler: a bare event.data
// is always a map in Go, so event.data == null never holds and
// event.data != null always does, whatever the stored JSON says.
func celEventDataHandler(h celFieldHandler) celFieldHandler {
	return func(ident string, literal any, op string) ([]sq.Expression, error) {
		if ident == "event.data" && literal == nil {
			return []sq.Expression{}, nil
		}
		return h(ident, literal, op)
	}
}
