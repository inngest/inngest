package insights

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/inngest/inngest/pkg/duckdb/parser"
)

// FunctionSchema is one allowed function's exported metadata -- used by
// cmd/gen-insights-schema's JSON dump, the same way TableSchema
// (describe.go) is for tables/columns.
type FunctionSchema struct {
	Name        string
	Description string
	DocsURL     string
	// Signature is the function's call signature (e.g.
	// "concat_ws(separator, string, ...)") -- see functionInfo.signature
	// for how it's sourced. Always non-empty; never omit or fall back to
	// Name for this field.
	Signature string
}

// AllFunctionSchemas returns every allowed function's schema, sorted by
// name.
func AllFunctionSchemas() []FunctionSchema {
	names := slices.Sorted(maps.Keys(allowedFunctions))
	out := make([]FunctionSchema, len(names))
	for i, name := range names {
		info := allowedFunctions[name]
		out[i] = FunctionSchema{Name: name, Description: info.description, DocsURL: info.docsURL, Signature: info.signature}
	}
	return out
}

// functionInfo is one allowed function's metadata: a short description and
// a link to its official DuckDB documentation, both surfaced by
// cmd/gen-insights-schema's JSON dump (embedded into the SQL editor UI) and
// by DESCRIBE-adjacent tooling.
//
// docsURL points at the function's own anchor on its DuckDB docs page
// (e.g. ".../aggregates#sumarg"), extracted by hand from that page's real
// HTML (curl + grep for the doc site's own "<a href="#anchor"><code>
// name(...)" self-links -- not guessed, and not taken from an LLM's
// unverified summary of the page, which was tried first and produced
// fabricated anchors that don't exist). A small number of functions have no
// per-function anchor in DuckDB's own docs at all (verified by the same
// method) -- those fall back to the closest page or subsection instead of
// a fabricated deep link: MOD, NOW, TO_TIMESTAMP have no anchor on their
// page; JSON_ARRAY/JSON_OBJECT/JSON_QUOTE/JSON_MERGE_PATCH/TO_JSON share
// the "Creating JSON" page with no per-function anchors there either.
// Captured against DuckDB's docs as published when this was written --
// not re-verified on every build, so a doc site restructure could stale
// these without this package noticing.
// functionTypeResolver computes one allowed function's result ColumnType,
// result pathHints, and any non-fatal Diagnostics about a specific call
// (name, args, scope) -- functionInfo.returnType's own type, named so
// withArity and every returnType implementation (fixedType,
// preservesArgType, ...) share one spelling instead of repeating this
// three-return-value signature at each definition site.
type functionTypeResolver func(name string, args []parser.Expr, scope *tableScope) (ColumnType, []PathHint, []Diagnostic)

type functionInfo struct {
	description string
	docsURL     string
	// signature is the function's call signature (e.g.
	// "concat_ws(separator, string, ...)"), so the UI can show DuckDB's own
	// parameter names instead of just a bare function name. Sourced three
	// ways, in order of preference: (1) most entries are scraped from the
	// same per-function anchor docsURL points at (curl + grep for the doc
	// site's own "<a href="#anchor"><code>...</code>" self-link's full
	// element text, not just the function-name prefix docsURL's own
	// extraction stopped at); (2) a function whose docsURL falls back to a
	// page-only link (no per-function anchor) can still have a real
	// signature scraped from a summary table elsewhere on the same page --
	// APPROX_COUNT_DISTINCT, APPROX_QUANTILE, and every JSON_* function
	// (JSON_EXTRACT_PATH substitutes its own name into JSON_EXTRACT's
	// identical table-listed signature, per the docs' own text naming it
	// that alias) were found this way; (3) a handful of functions have no
	// signature anywhere on their page in any form -- SINH/COSH/TANH's are
	// constructed by analogy to their documented inverse forms
	// (ASINH/ACOSH/ATANH), and MOD/NOW/TO_TIMESTAMP/COUNT_IF/STRUCT_KEYS'
	// are constructed from this same struct's own description and
	// allowedFunctions' own withArity bounds below -- an explicit exception
	// to this package's "don't fabricate" rule elsewhere, approved for
	// exactly these eight. No entry is left empty.
	signature string
	// returnType computes this function's result ColumnType from its
	// call-site arguments and scope, and any Diagnostics worth surfacing
	// about that call. Most carry DiagnosticWarning/Info and just ride
	// along; a DiagnosticError-severity one is promoted by checkFunction
	// (validate.go) into a real *ValidationError that rejects the query --
	// used only when the argument count or a literal argument's type is one
	// DuckDB itself would unconditionally reject too (confirmed empirically
	// against a live duckdb, not guessed), so this never rejects a query
	// that would actually have run. Start/End are left zero-valued on any
	// Diagnostic a returnType produces; checkFunction fills them in from
	// the real call site's own FunctionExpr node, since returnType only
	// ever sees a function's resolved name and arguments, not the node
	// itself.
	//
	// Every entry wraps its actual type-computing function in
	// withArity(min, max, ...) -- every allowed function gets an
	// argument-count check, not just the handful with argument-dependent
	// logic below, so the wrapped function itself can assume args already
	// has a valid length. Most functions ignore their (now
	// length-validated) arguments and report a fixed bucket with no further
	// diagnostics (fixedType); a few are genuinely a function of their
	// arguments -- preservesArgType for "pick one of my arguments"
	// functions (MIN/MAX/COALESCE-like), preservesArgTypeAndHint for the
	// subset of that same family whose result also carries args[0]'s own
	// pathHints verbatim, listReturnType for LIST/ARRAY_AGG (whose result
	// pathHints describe the aggregated LIST's *elements*, one
	// array-wildcard segment deeper than args[0]'s own), ifReturnType for
	// IF's then-branch and condition-type check, listAggregateReturnType
	// for LIST_AGGREGATE/ARRAY_AGGREGATE's dynamic dispatch and name-type
	// check. This is the single source of truth for a function's return
	// type *and* its result pathHints -- functionReturnType (typecheck.go)
	// is just allowedFunctions[name].returnType(name, args, scope); there's
	// no separate switch, or separate hint-only lookup, to keep in sync.
	//
	// The []PathHint result is nil for the overwhelming majority of
	// entries (fixedType, plain preservesArgType, ifReturnType,
	// listAggregateReturnType) -- correct for anything that computes a
	// genuinely new value from its inputs rather than reporting one of
	// them back unmodified. It's only ever non-nil for a genuine
	// aggregate/window function that returns one of its input rows'
	// actual, unmodified values (preservesArgTypeAndHint: ANY_VALUE,
	// ARG_MAX/MIN, FIRST(_VALUE), LAG/LEAD, LAST(_VALUE), MAX/MIN, MODE,
	// NTH_VALUE, QUANTILE_DISC) or LIST/ARRAY_AGG (listReturnType).
	// Deliberately left plain (no hint) on every other preservesArgType
	// entry: BIT_AND/OR/XOR computes a new value, not one of its inputs;
	// GREATEST/LEAST/IFNULL pick between independently-sourced expressions
	// row-by-row rather than accumulating one column (whichever one wins
	// usually isn't semantically "the same column" a hint could describe);
	// MEDIAN/QUANTILE_CONT can interpolate a value that was never actually
	// present in the input for an even-sized/non-discrete distribution, so
	// their output no longer reliably means whatever the input hint
	// claimed.
	returnType functionTypeResolver
}

// fixedType returns a returnType function that ignores its arguments and
// always reports t with no pathHints or diagnostics -- the common case,
// for every function whose return type doesn't depend on its arguments.
func fixedType(t ColumnType) functionTypeResolver {
	return func(string, []parser.Expr, *tableScope) (ColumnType, []PathHint, []Diagnostic) { return t, nil, nil }
}

// tooFewArgsDiagnostic notes a call to name with fewer than min arguments
// -- DiagnosticError, promoted by checkFunction into an actual rejection:
// every function this is used for genuinely has no valid form below min
// arguments (confirmed against a live duckdb, e.g. "SELECT min();" ->
// Binder Error, "SELECT if(true, 1);" -> Parser Error: Wrong number of
// arguments to IF), so this never rejects a query DuckDB would accept.
func tooFewArgsDiagnostic(name string, min, got int) Diagnostic {
	return Diagnostic{
		Severity: DiagnosticError,
		Code:     "wrong-argument-count",
		Message:  fmt.Sprintf("%s expects at least %d argument(s), got %d", strings.ToUpper(name), min, got),
	}
}

// tooManyArgsDiagnostic notes a call to name with more than max arguments
// -- same DiagnosticError promotion as tooFewArgsDiagnostic, used only
// where max is a confirmed real ceiling (see withArity).
func tooManyArgsDiagnostic(name string, max, got int) Diagnostic {
	return Diagnostic{
		Severity: DiagnosticError,
		Code:     "wrong-argument-count",
		Message:  fmt.Sprintf("%s expects at most %d argument(s), got %d", strings.ToUpper(name), max, got),
	}
}

// withArity wraps a returnType function with an argument-count check
// against [min, max] -- max < 0 means unbounded (some overload has
// varargs) -- short-circuiting to a tooFewArgsDiagnostic/
// tooManyArgsDiagnostic instead of calling next when the count is out of
// range, so every wrapped function's own returnType body can assume its
// args slice already has a valid length.
//
// (min, max) is derived from DuckDB's own duckdb_functions() catalog: min
// is the fewest parameters of any of the function's overloads, max the
// most (or unbounded if any overload has varargs) -- confirmed against a
// live (exact pinned-version) duckdb for a broad spot-check sample across
// categories, so trusted for the rest rather than individually reverified
// one by one. One real exception found and corrected for: DuckDB's window
// functions (LAG/LEAD/etc.) declare trailing parameters with a default
// value ("offset BIGINT := 1") that duckdb_functions() has no column to
// distinguish from a required parameter, so the catalog's raw parameter
// count over-reports their true minimum (e.g. it reports LAG as requiring
// exactly 3 arguments, but "SELECT lag(x) OVER (...)" -- 1 argument --
// genuinely succeeds) -- every window function's (min, max) here is hand
// -verified against a live duckdb instead of catalog-derived for exactly
// this reason. IF/IFNULL/UNNEST aren't in the catalog under those names at
// all (special parser-level forms, not real catalog functions) and are
// hand-verified the same way.
func withArity(min, max int, next functionTypeResolver) functionTypeResolver {
	return func(name string, args []parser.Expr, scope *tableScope) (ColumnType, []PathHint, []Diagnostic) {
		if len(args) < min {
			return ColumnTypeUnknown, nil, []Diagnostic{tooFewArgsDiagnostic(name, min, len(args))}
		}
		if max >= 0 && len(args) > max {
			return ColumnTypeUnknown, nil, []Diagnostic{tooManyArgsDiagnostic(name, max, len(args))}
		}
		return next(name, args, scope)
	}
}

// wrongArgTypeDiagnostic notes arg (1-based, for the message only) of a
// call to name having the wrong type -- same DiagnosticError promotion as
// the argument-count diagnostics above, used only where the wrong type is
// confirmed unconditionally rejected by DuckDB itself, not guessed.
func wrongArgTypeDiagnostic(name string, arg int, want ColumnType, got ColumnType) Diagnostic {
	return Diagnostic{
		Severity: DiagnosticError,
		Code:     "wrong-argument-type",
		Message: fmt.Sprintf("%s's argument %d expects %s, got %s",
			strings.ToUpper(name), arg, want, got),
	}
}

// duckDBBoolLiterals is every VARCHAR value DuckDB's own implicit
// VARCHAR->BOOLEAN cast accepts (case-insensitively), confirmed against a
// live duckdb: true/false, t/f, yes/no, y/n, 1/0 all succeed; anything
// else (e.g. "on"/"off", "2", "") raises "Conversion Error: Could not
// convert string ... to BOOL". Only used to check a string *literal*
// passed where DuckDB requires a BOOLEAN (IF's condition) -- a column or
// other dynamic expression's actual runtime value can't be checked
// statically, so those are never rejected here. Not re-verified on every
// build, matching this package's other hand-captured DuckDB behavior
// (functions.go's docsURL comment) -- a future DuckDB version could widen
// or narrow this set without this package noticing.
var duckDBBoolLiterals = map[string]bool{
	"true": true, "false": true,
	"t": true, "f": true,
	"yes": true, "no": true,
	"y": true, "n": true,
	"1": true, "0": true,
}

// preservesArgType is the returnType function for "pick one of my
// arguments" functions (MIN, MAX, GREATEST, LEAST, ANY_VALUE, IFNULL,
// MEDIAN, MODE, QUANTILE_CONT/DISC, FIRST, LAST, ARG_MAX, ARG_MIN,
// BIT_AND/OR/XOR, LAG, LEAD, FIRST_VALUE, LAST_VALUE, NTH_VALUE) -- they
// report their representative argument's type (always args[0]) rather
// than converting it, matching what they actually do in DuckDB (confirmed
// empirically for MEDIAN/QUANTILE_*, which return the input's own type for
// ordinal types like TIMESTAMP, not always a DOUBLE). Assumes len(args) >=
// 1 -- every allowedFunctions entry using this wraps it in withArity with
// a real minimum of at least 1, so this never needs its own arg-count
// check.
func preservesArgType(_ string, args []parser.Expr, scope *tableScope) (ColumnType, []PathHint, []Diagnostic) {
	return inferType(args[0], scope), nil, nil
}

// preservesArgTypeAndHint extends preservesArgType with pathHint
// propagation, for the subset of that same "pick one of my arguments"
// family whose result is genuinely one of args[0]'s own actual,
// unmodified values (functionInfo's own doc comment names exactly which,
// and why the rest stay plain preservesArgType). Reports args[0]'s full
// pathHints -- root/whole-value hint included -- exactly as if args[0]
// itself were the SELECT list item in this position (exprPathHints,
// columnhints.go).
func preservesArgTypeAndHint(name string, args []parser.Expr, scope *tableScope) (ColumnType, []PathHint, []Diagnostic) {
	t, _, diags := preservesArgType(name, args, scope)
	return t, exprPathHints(args[0], scope), diags
}

// listReturnType is LIST()/ARRAY_AGG()'s returnType: always ColumnTypeJSON
// (a LIST value), but its result pathHints describe the aggregated LIST's
// *elements* -- not the LIST value itself, which carries no hint of its
// own -- carrying args[0]'s own hint/pathHints one array-wildcard segment
// deeper (wrapArrayElementPathHints, columnhints.go).
func listReturnType(_ string, args []parser.Expr, scope *tableScope) (ColumnType, []PathHint, []Diagnostic) {
	return ColumnTypeJSON, wrapArrayElementPathHints(exprPathHints(args[0], scope)), nil
}

// unnestReturnType is UNNEST(list)'s returnType: always ColumnTypeJSON,
// but its result pathHints describe the single element UNNEST extracts --
// args[0]'s own pathHints (exprPathHints, columnhints.go) with one
// leading wc segment stripped (projectPathHints, pathhint.go). This is
// the exact inverse of listReturnType's own "one wc segment prepended,"
// so UNNEST(list(x)) round-trips back to x's own pathHints, and both
// compose to arbitrary nesting depth (UNNEST(list(list(x))), a JSON
// sub-path access layered on top of either, ...) since each is just
// another []PathHint-to-[]PathHint transform over the same shape.
func unnestReturnType(_ string, args []parser.Expr, scope *tableScope) (ColumnType, []PathHint, []Diagnostic) {
	return ColumnTypeJSON, projectPathHints(exprPathHints(args[0], scope), []PathSegment{wc}), nil
}

// ifReturnType is IF's returnType: IF(condition, then, else) -- the "then"
// branch (args[1]) is the representative type, matching how caseExprType
// treats a CASE's branches. Assumes len(args) == 3 -- allowedFunctions'
// "if" entry wraps this in withArity(3, 3, ifReturnType), confirmed
// empirically to be IF's real (and only) valid arity ("SELECT if(true,
// 1);" -- 2 arguments -- errors the same "Wrong number of arguments to IF"
// as a 1-argument call).
//
// The condition (args[0]) isn't required to already be BOOLEAN-typed --
// confirmed empirically, DuckDB implicitly casts a NUMBER (truthy/falsy,
// C-style) or a NULL literal (like CASE, takes the else branch) with no
// error, so only a STRING *literal* whose own text doesn't parse as a
// DuckDB boolean (duckDBBoolLiterals) is rejected here: that's the one
// shape confirmed to always fail ("SELECT if('a', 1, 2);" -> Conversion
// Error). A column reference or other dynamic expression's actual runtime
// value can't be checked statically, so those are never rejected either,
// matching this package's "never reject on a guess" rule.
func ifReturnType(name string, args []parser.Expr, scope *tableScope) (ColumnType, []PathHint, []Diagnostic) {
	if lit, ok := args[0].(*parser.Literal); ok && lit.Kind == parser.LitString {
		if !duckDBBoolLiterals[strings.ToLower(lit.Text)] {
			return ColumnTypeUnknown, nil, []Diagnostic{wrongArgTypeDiagnostic(name, 1, ColumnTypeBoolean, ColumnTypeString)}
		}
	}
	return inferType(args[1], scope), nil, nil
}

// literalColumnType is a Literal's own ColumnType bucket, by Kind -- used
// only to name the wrong type in a diagnostic message (e.g.
// listAggregateReturnType below); inferType's *parser.Literal case is the
// general-purpose version of this same mapping.
func literalColumnType(kind parser.LiteralKind) ColumnType {
	switch kind {
	case parser.LitString:
		return ColumnTypeString
	case parser.LitNumber:
		return ColumnTypeNumber
	case parser.LitTrue, parser.LitFalse:
		return ColumnTypeBoolean
	default: // LitNull
		return ColumnTypeUnknown
	}
}

// listAggregateReturnType is LIST_AGGREGATE/ARRAY_AGGREGATE's returnType:
// list_aggregate(list, name) applies the aggregate named by args[1] (e.g.
// 'sum', 'string_agg', 'bit_and') to args[0] -- statically typeable only
// when that name is a literal string constant (by far the common case:
// written directly in the query, not built from an expression). Recurses
// into functionReturnType with the resolved name and args[0] alone, so
// e.g. list_aggregate(x, 'string_agg') reports the same type STRING_AGG(x)
// would, reusing that function's own returnType instead of duplicating it.
//
// A dynamic (non-literal) name is genuinely unknowable without asking
// DuckDB -- reported as ColumnTypeUnknown with no diagnostic, not guessed.
// A *literal* of the wrong kind (a NUMBER/BOOLEAN/NULL where DuckDB
// requires VARCHAR) is a different situation: confirmed empirically
// ("SELECT list_aggregate([1,2,3], 123);" -> Binder Error, no candidate
// function matches a non-VARCHAR second argument), so that case is
// rejected outright rather than reported as unknown. A literal string that
// doesn't match any known aggregate name (e.g. 'not_a_real_aggregate')
// stays unknown too, not rejected -- this package's own allowlist isn't
// necessarily DuckDB's complete aggregate list, so absence here doesn't
// prove DuckDB would also reject it. Assumes len(args) >= 2 --
// allowedFunctions' "list_aggregate"/"array_aggregate" entries wrap this in
// withArity(2, -1, listAggregateReturnType), confirmed empirically to be
// this function's real minimum (a 1-argument call is a Binder Error) with
// no real maximum (extra arguments are forwarded to the named aggregate,
// e.g. list_aggregate(x, 'string_agg', ',')).
func listAggregateReturnType(name string, args []parser.Expr, scope *tableScope) (ColumnType, []PathHint, []Diagnostic) {
	lit, ok := args[1].(*parser.Literal)
	if !ok {
		return ColumnTypeUnknown, nil, nil
	}
	if lit.Kind != parser.LitString {
		return ColumnTypeUnknown, nil, []Diagnostic{wrongArgTypeDiagnostic(name, 2, ColumnTypeString, literalColumnType(lit.Kind))}
	}
	return functionReturnType(strings.ToLower(lit.Text), args[:1], scope)
}

// allowedFunctions is the conservative starting function allowlist. Keys
// are upper-cased, dot-joined function names. No table-function or
// filesystem/catalog-touching surface (read_csv, ATTACH, etc.) is ever
// included here. Every entry's returnType is that function's own
// return-type rule -- if you add a function, give it one.
//
// Deliberately excludes COALESCE, NULLIF, TRIM, and SUBSTRING even though
// they're ordinary DuckDB functions: pkg/duckdb/parser doesn't implement
// their grammar production (DuckDB gives these their own special
// keyword-based syntax instead of routing through the generic function-call
// rule), so ParseString errors on them unconditionally — confirmed
// empirically. Including them here would be dead code. SUBSTR/LTRIM/RTRIM
// (this package's plain-function-call equivalents) are unaffected.
//
// Descriptions are DuckDB's own (from duckdb_functions().description against
// the vendored DuckDB version, or hand-written for the handful of entries
// that table leaves NULL -- IF/IFNULL/UNNEST aren't in the catalog at all
// under those names, and the JSON extension's functions don't populate
// description in the build this was captured against). Not auto-synced --
// if a re-vendored DuckDB changes a function's behavior, these won't notice.
var allowedFunctions map[string]functionInfo

// init populates allowedFunctions rather than a plain var initializer,
// because some entries' returnType (e.g. listAggregateReturnType) call
// functionReturnType (typecheck.go), which reads allowedFunctions itself
// -- a var initializer referencing allowedFunctions from inside its own
// literal is a cycle Go's initialization-order check rejects at compile
// time, even though the actual reference only ever runs after this map is
// fully built. Assigning it inside a function body sidesteps that static
// check; nothing calls a returnType func before main() begins, well after
// every init() has run.
func init() {
	allowedFunctions = map[string]functionInfo{
		"abs": {
			description: "Absolute value.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/numeric#absx",
			signature:   "abs(x)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"acos": {
			description: "Computes the arccosine of x.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/numeric#acosx",
			signature:   "acos(x)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"acosh": {
			description: "Computes the inverse hyperbolic cosine of x.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/numeric#acoshx",
			signature:   "acosh(x)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"age": {
			description: "Subtract arguments, resulting in the time difference between the two timestamps.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/timestamp#agetimestamp-timestamp",
			signature:   "age(timestamp, timestamp)",
			returnType:  withArity(1, 2, fixedType(ColumnTypeDatetime)),
		},
		"any_value": {
			description: "Returns the first non-NULL value from arg. This function is affected by ordering.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/aggregates#any_valuearg",
			signature:   "any_value(arg)",
			returnType:  withArity(1, 1, preservesArgTypeAndHint),
		},
		"approx_count_distinct": {
			description: "Computes the approximate count of distinct elements using HyperLogLog.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/aggregates#approximate-aggregates",
			signature:   "approx_count_distinct(x)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"approx_quantile": {
			description: "Computes the approximate quantile using T-Digest.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/aggregates#approximate-aggregates",
			signature:   "approx_quantile(x, pos)",
			returnType:  withArity(2, 2, fixedType(ColumnTypeNumber)),
		},
		"arg_max": {
			description: "Finds the row with the maximum val. Calculates the non-NULL arg expression at that row.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/aggregates#arg_maxarg-val",
			signature:   "arg_max(arg, val)",
			returnType:  withArity(2, 3, preservesArgTypeAndHint),
		},
		"arg_min": {
			description: "Finds the row with the minimum val. Calculates the non-NULL arg expression at that row.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/aggregates#arg_minarg-val",
			signature:   "arg_min(arg, val)",
			returnType:  withArity(2, 3, preservesArgTypeAndHint),
		},
		"array_agg": {
			description: "Returns a LIST containing all the values of a column.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/aggregates#listarg",
			signature:   "array_agg(arg)",
			returnType:  withArity(1, 1, listReturnType),
		},
		"array_aggregate": {
			description: "Executes the aggregate function function_name on the elements of list.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/list#list_aggregatelist-function_name-",
			signature:   "array_aggregate(list, function_name, ...)",
			returnType:  withArity(2, -1, listAggregateReturnType),
		},
		"array_contains": {
			description: "Returns true if the list contains the element.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/list#list_containslist-element",
			signature:   "array_contains(list, element)",
			returnType:  withArity(2, 2, fixedType(ColumnTypeBoolean)),
		},
		"array_length": {
			description: "Returns the length of the list.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/list#lengthlist",
			signature:   "array_length(list)",
			returnType:  withArity(1, 2, fixedType(ColumnTypeNumber)),
		},
		"array_position": {
			description: "Returns the index of the element if the list contains the element. If the element is not found, it returns NULL.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/list#list_positionlist-element",
			signature:   "array_position(list, element)",
			returnType:  withArity(2, 2, fixedType(ColumnTypeNumber)),
		},
		"array_to_string": {
			description: "Concatenates list/array elements using an optional delimiter.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/list#array_to_stringlist-delimiter",
			signature:   "array_to_string(list, delimiter)",
			returnType:  withArity(2, 2, fixedType(ColumnTypeString)),
		},
		"ascii": {
			description: "Returns an integer that represents the Unicode code point of the first character of the string.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/text#asciistring",
			signature:   "ascii(string)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"asin": {
			description: "Computes the arcsine of x.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/numeric#asinx",
			signature:   "asin(x)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"asinh": {
			description: "Computes the inverse hyperbolic sine of x.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/numeric#asinhx",
			signature:   "asinh(x)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"atan": {
			description: "Computes the arctangent of x.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/numeric#atanx",
			signature:   "atan(x)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"atan2": {
			description: "Computes the arctangent (y, x).",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/numeric#atan2y-x",
			signature:   "atan2(y, x)",
			returnType:  withArity(2, 2, fixedType(ColumnTypeNumber)),
		},
		"atanh": {
			description: "Computes the inverse hyperbolic tangent of x.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/numeric#atanhx",
			signature:   "atanh(x)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"avg": {
			description: "Calculates the average value for all tuples in x.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/aggregates#avgarg",
			signature:   "avg(arg)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"bit_and": {
			description: "Returns the bitwise AND of all bits in a given expression.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/aggregates#bit_andarg",
			signature:   "bit_and(arg)",
			returnType:  withArity(1, 1, preservesArgType),
		},
		"bit_or": {
			description: "Returns the bitwise OR of all bits in a given expression.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/aggregates#bit_orarg",
			signature:   "bit_or(arg)",
			returnType:  withArity(1, 1, preservesArgType),
		},
		"bit_xor": {
			description: "Returns the bitwise XOR of all bits in a given expression.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/aggregates#bit_xorarg",
			signature:   "bit_xor(arg)",
			returnType:  withArity(1, 1, preservesArgType),
		},
		"bool_and": {
			description: "Returns TRUE if every input value is TRUE, otherwise FALSE.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/aggregates#bool_andarg",
			signature:   "bool_and(arg)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeBoolean)),
		},
		"bool_or": {
			description: "Returns TRUE if any input value is TRUE, otherwise FALSE.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/aggregates#bool_orarg",
			signature:   "bool_or(arg)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeBoolean)),
		},
		"cbrt": {
			description: "Returns the cube root of x.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/numeric#cbrtx",
			signature:   "cbrt(x)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"ceil": {
			description: "Rounds the number up.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/numeric#ceilx",
			signature:   "ceil(x)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"ceiling": {
			description: "Rounds the number up.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/numeric#ceilingx",
			signature:   "ceiling(x)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"chr": {
			description: "Returns a character which is corresponding the ASCII code value or Unicode code point.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/text#chrcode_point",
			signature:   "chr(code_point)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeString)),
		},
		"concat": {
			description: "Concatenates multiple strings or lists. NULL inputs are skipped. See also operator ||.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/text#concatvalue-",
			signature:   "concat(value, ...)",
			returnType:  withArity(1, -1, fixedType(ColumnTypeString)),
		},
		"concat_ws": {
			description: "Concatenates multiple strings, separated by separator. NULL inputs are skipped.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/text#concat_wsseparator-string-",
			signature:   "concat_ws(separator, string, ...)",
			returnType:  withArity(2, -1, fixedType(ColumnTypeString)),
		},
		"contains": {
			description: "Returns true if search_string is found within string.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/text#containsstring-search_string",
			signature:   "contains(string, search_string)",
			returnType:  withArity(2, 2, fixedType(ColumnTypeBoolean)),
		},
		"corr": {
			description: "Returns the correlation coefficient for non-NULL pairs in a group.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/aggregates#corry-x",
			signature:   "corr(y, x)",
			returnType:  withArity(2, 2, fixedType(ColumnTypeNumber)),
		},
		"cos": {
			description: "Computes the cosine of x.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/numeric#cosx",
			signature:   "cos(x)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"cosh": {
			description: "Computes the hyperbolic cosine of x.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/numeric",
			signature:   "cosh(x)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"cot": {
			description: "Computes the cotangent of x.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/numeric#cotx",
			signature:   "cot(x)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"count": {
			description: "Returns the number of non-NULL values in arg.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/aggregates#count",
			signature:   "count()",
			returnType:  withArity(0, 1, fixedType(ColumnTypeNumber)),
		},
		"count_if": {
			description: "Counts the total number of TRUE values for a boolean column.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/aggregates",
			signature:   "count_if(arg)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"covar_pop": {
			description: "Returns the population covariance of input values.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/aggregates#covar_popy-x",
			signature:   "covar_pop(y, x)",
			returnType:  withArity(2, 2, fixedType(ColumnTypeNumber)),
		},
		"covar_samp": {
			description: "Returns the sample covariance for non-NULL pairs in a group.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/aggregates#covar_sampy-x",
			signature:   "covar_samp(y, x)",
			returnType:  withArity(2, 2, fixedType(ColumnTypeNumber)),
		},
		"cume_dist": {
			description: "The number of partition rows preceding or peer with current row / total partition rows.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/window_functions#cume_distorder-by-ordering",
			signature:   "cume_dist([ORDER BY ordering])",
			returnType:  withArity(0, 0, fixedType(ColumnTypeNumber)),
		},
		// current_date is DuckDB's own alias for today() ("no parentheses
		// necessary", per its own docs table) -- documented at today()'s
		// anchor, not one of its own. Only usable here via its own
		// current_date() call form: bare CURRENT_DATE (no parens) parses as
		// a plain Ident in pkg/duckdb/parser (confirmed empirically), not a
		// FunctionExpr, so it would be checked -- and rejected -- as an
		// unknown column instead of ever reaching this allowlist.
		"current_date": {
			description: "Current date (start of current transaction) in the local time zone (alias for today()).",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/date#today",
			signature:   "current_date()",
			returnType:  withArity(0, 0, fixedType(ColumnTypeDatetime)),
		},
		"date_add": {
			description: "Adds an interval to a date, time, or timestamp.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/date#date_adddate-interval",
			signature:   "date_add(date, interval)",
			returnType:  withArity(2, 2, fixedType(ColumnTypeDatetime)),
		},
		"date_diff": {
			description: "The number of partition boundaries between the timestamps.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/date#date_diffpart-startdate-enddate",
			signature:   "date_diff(part, startdate, enddate)",
			returnType:  withArity(3, 3, fixedType(ColumnTypeNumber)),
		},
		"date_part": {
			description: "Get subfield (equivalent to extract).",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/date#date_partpart-date",
			signature:   "date_part(part, date)",
			returnType:  withArity(2, 2, fixedType(ColumnTypeNumber)),
		},
		"date_sub": {
			description: "The number of complete partitions between the timestamps.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/date#date_subpart-startdate-enddate",
			signature:   "date_sub(part, startdate, enddate)",
			returnType:  withArity(3, 3, fixedType(ColumnTypeDatetime)),
		},
		"date_trunc": {
			description: "Truncate to specified precision.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/date#date_truncpart-date",
			signature:   "date_trunc(part, date)",
			returnType:  withArity(2, 2, fixedType(ColumnTypeDatetime)),
		},
		"day": {
			description: "Extract the day component from a date or timestamp.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/datepart#daydate",
			signature:   "day(date)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"dayofweek": {
			description: "Extract the dayofweek component from a date or timestamp.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/datepart#dayofweekdate",
			signature:   "dayofweek(date)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"dayofyear": {
			description: "Extract the dayofyear component from a date or timestamp.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/datepart#dayofyeardate",
			signature:   "dayofyear(date)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"degrees": {
			description: "Converts radians to degrees.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/numeric#degreesx",
			signature:   "degrees(x)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"dense_rank": {
			description: "The rank of the current row without gaps.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/window_functions#dense_rank",
			signature:   "dense_rank()",
			returnType:  withArity(0, 0, fixedType(ColumnTypeNumber)),
		},
		"ends_with": {
			description: "Returns true if string ends with search_string.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/text#suffixstring-search_string",
			signature:   "ends_with(string, search_string)",
			returnType:  withArity(2, 2, fixedType(ColumnTypeBoolean)),
		},
		"entropy": {
			description: "Returns the log-2 entropy of count input-values.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/aggregates#entropyx",
			signature:   "entropy(x)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"epoch": {
			description: "Extract the epoch component from a temporal type.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/timestamp#epochtimestamp",
			signature:   "epoch(timestamp)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"epoch_ms": {
			description: "Extract the epoch component in milliseconds from a temporal type.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/timestamp#epoch_mstimestamp",
			signature:   "epoch_ms(timestamp)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"exp": {
			description: "Computes e to the power of x.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/numeric#expx",
			signature:   "exp(x)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"first": {
			description: "Returns the first value (NULL or non-NULL) from arg. This function is affected by ordering.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/aggregates#firstarg",
			signature:   "first(arg)",
			returnType:  withArity(1, 1, preservesArgTypeAndHint),
		},
		"first_value": {
			description: "The first value of expr in the frame. Can IGNORE or RESPECT NULLS.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/window_functions#first_valueexpr-order-by-ordering-ignore-nulls",
			signature:   "first_value(expr[ ORDER BY ordering][ IGNORE NULLS])",
			returnType:  withArity(1, 1, preservesArgTypeAndHint),
		},
		"flatten": {
			description: "Flattens a nested list by one level.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/list#flattennested_list",
			signature:   "flatten(nested_list)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeJSON)),
		},
		"floor": {
			description: "Rounds the number down.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/numeric#floorx",
			signature:   "floor(x)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"format": {
			description: "Formats a string using the fmt syntax.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/text#formatformat-",
			signature:   "format(format, ...)",
			returnType:  withArity(1, -1, fixedType(ColumnTypeString)),
		},
		"generate_series": {
			description: "Creates a list of values between start and stop - the stop parameter is inclusive.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/list#generate_seriesstart-stop-step",
			signature:   "generate_series(start[, stop][, step])",
			returnType:  withArity(1, 3, fixedType(ColumnTypeJSON)),
		},
		"greatest": {
			description: "Returns the largest value. For strings lexicographical ordering is used. Note that lowercase characters are considered \"larger\" than uppercase characters and collations are not supported.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/text#greatestarg1-",
			signature:   "greatest(arg1, ...)",
			returnType:  withArity(1, -1, preservesArgType),
		},
		"hash": {
			description: "Returns a UBIGINT with the hash of the value. Note that this is not a cryptographic hash.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/utility#hashvalue",
			signature:   "hash(value)",
			returnType:  withArity(1, -1, fixedType(ColumnTypeNumber)),
		},
		"hour": {
			description: "Extract the hour component from a date or timestamp.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/datepart#hourdate",
			signature:   "hour(date)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"if": {
			description: "Returns then if condition evaluates to true, otherwise returns the else value.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/utility#ifa-b-c",
			signature:   "if(a, b, c)",
			returnType:  withArity(3, 3, ifReturnType),
		},
		"ifnull": {
			description: "Returns expr1 if it's not NULL, otherwise expr2.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/utility#ifnullexpr-other",
			signature:   "ifnull(expr, other)",
			returnType:  withArity(2, 2, preservesArgType),
		},
		"instr": {
			description: "Returns location of first occurrence of search_string in string, counting from 1. Returns 0 if no match found.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/text#instrstring-search_string",
			signature:   "instr(string, search_string)",
			returnType:  withArity(2, 2, fixedType(ColumnTypeNumber)),
		},
		"isodow": {
			description: "Extract the isodow component from a date or timestamp.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/datepart#isodowdate",
			signature:   "isodow(date)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"isoyear": {
			description: "Extract the isoyear component from a date or timestamp.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/datepart#isoyeardate",
			signature:   "isoyear(date)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"jaccard": {
			description: "The Jaccard similarity between two strings. Characters of different cases (e.g., a and A) are considered different. Returns a number between 0 and 1.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/text#jaccards1-s2",
			signature:   "jaccard(s1, s2)",
			returnType:  withArity(2, 2, fixedType(ColumnTypeNumber)),
		},
		"json_array": {
			description: "Creates a JSON array from a list of arguments.",
			docsURL:     "https://duckdb.org/docs/current/data/json/creating_json",
			signature:   "json_array(any, ...)",
			returnType:  withArity(0, -1, fixedType(ColumnTypeJSON)),
		},
		"json_array_length": {
			description: "Returns the number of elements in a JSON array, or 0 if not an array.",
			docsURL:     "https://duckdb.org/docs/current/data/json/json_functions#json-scalar-functions",
			signature:   "json_array_length(json[, path])",
			returnType:  withArity(1, 2, fixedType(ColumnTypeNumber)),
		},
		"json_contains": {
			description: "Returns true if a JSON value contains the specified search value.",
			docsURL:     "https://duckdb.org/docs/current/data/json/json_functions#json-scalar-functions",
			signature:   "json_contains(json_haystack, json_needle)",
			returnType:  withArity(2, 2, fixedType(ColumnTypeBoolean)),
		},
		"json_exists": {
			description: "Returns true if a JSON path exists in the JSON value.",
			docsURL:     "https://duckdb.org/docs/current/data/json/json_functions#json-extraction-functions",
			signature:   "json_exists(json, path)",
			returnType:  withArity(2, 2, fixedType(ColumnTypeBoolean)),
		},
		"json_extract": {
			description: "Extracts the JSON value at the given path or key.",
			docsURL:     "https://duckdb.org/docs/current/data/json/json_functions#json-extraction-functions",
			signature:   "json_extract(json, path)",
			returnType:  withArity(2, 2, fixedType(ColumnTypeJSON)),
		},
		"json_extract_path": {
			description: "Extracts the JSON value at the given path (alias for json_extract).",
			docsURL:     "https://duckdb.org/docs/current/data/json/json_functions#json-extraction-functions",
			signature:   "json_extract_path(json, path)",
			returnType:  withArity(2, 2, fixedType(ColumnTypeJSON)),
		},
		"json_extract_string": {
			description: "Extracts the value at the given path or key as a string.",
			docsURL:     "https://duckdb.org/docs/current/data/json/json_functions#json-extraction-functions",
			signature:   "json_extract_string(json, path)",
			returnType:  withArity(2, 2, fixedType(ColumnTypeString)),
		},
		"json_group_array": {
			description: "Aggregates values into a JSON array.",
			docsURL:     "https://duckdb.org/docs/current/data/json/json_functions#json-aggregate-functions",
			signature:   "json_group_array(any)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeJSON)),
		},
		"json_group_object": {
			description: "Aggregates key/value pairs into a JSON object.",
			docsURL:     "https://duckdb.org/docs/current/data/json/json_functions#json-aggregate-functions",
			signature:   "json_group_object(key, value)",
			returnType:  withArity(2, 2, fixedType(ColumnTypeJSON)),
		},
		"json_keys": {
			description: "Returns the keys of a JSON object as a list.",
			docsURL:     "https://duckdb.org/docs/current/data/json/json_functions#json-scalar-functions",
			signature:   "json_keys(json[, path])",
			returnType:  withArity(1, 2, fixedType(ColumnTypeJSON)),
		},
		"json_merge_patch": {
			description: "Merges two JSON documents using RFC 7396 JSON Merge Patch semantics.",
			docsURL:     "https://duckdb.org/docs/current/data/json/creating_json",
			signature:   "json_merge_patch(json, json)",
			returnType:  withArity(2, -1, fixedType(ColumnTypeJSON)),
		},
		"json_object": {
			description: "Creates a JSON object from a list of key/value argument pairs.",
			docsURL:     "https://duckdb.org/docs/current/data/json/creating_json",
			signature:   "json_object(key, value, ...)",
			returnType:  withArity(0, -1, fixedType(ColumnTypeJSON)),
		},
		"json_quote": {
			description: "Converts a value to a quoted JSON value (alias for to_json).",
			docsURL:     "https://duckdb.org/docs/current/data/json/creating_json",
			signature:   "json_quote(any)",
			returnType:  withArity(0, -1, fixedType(ColumnTypeJSON)),
		},
		"json_structure": {
			description: "Returns the structure (a JSON schema-like description) of a JSON value.",
			docsURL:     "https://duckdb.org/docs/current/data/json/json_functions#json-scalar-functions",
			signature:   "json_structure(json)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeJSON)),
		},
		"json_type": {
			description: "Returns the type of a JSON value as a string.",
			docsURL:     "https://duckdb.org/docs/current/data/json/json_functions#json-scalar-functions",
			signature:   "json_type(json[, path])",
			returnType:  withArity(1, 2, fixedType(ColumnTypeString)),
		},
		"json_valid": {
			description: "Returns true if the string is valid JSON.",
			docsURL:     "https://duckdb.org/docs/current/data/json/json_functions#json-scalar-functions",
			signature:   "json_valid(json)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeBoolean)),
		},
		"json_value": {
			description: "Extracts a JSON scalar value at the given path as a string.",
			docsURL:     "https://duckdb.org/docs/current/data/json/json_functions#json-extraction-functions",
			signature:   "json_value(json, path)",
			returnType:  withArity(2, 2, fixedType(ColumnTypeString)),
		},
		"kurtosis": {
			description: "Returns the excess kurtosis (Fisher's definition) of all input values, with a bias correction according to the sample size.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/aggregates#kurtosisx",
			signature:   "kurtosis(x)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"lag": {
			description: "The value of expr n rows after the current row, or the default. Can IGNORE or RESPECT NULLS.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/window_functions#lagexpr-offset-default-order-by-ordering-ignore-nulls",
			signature:   "lag(expr[, offset[, default]][ ORDER BY ordering][ IGNORE NULLS])",
			returnType:  withArity(1, 3, preservesArgTypeAndHint),
		},
		"last": {
			description: "Returns the last value of a column. This function is affected by ordering.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/aggregates#lastarg",
			signature:   "last(arg)",
			returnType:  withArity(1, 1, preservesArgTypeAndHint),
		},
		"last_day": {
			description: "Returns the last day of the month.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/date#last_daydate",
			signature:   "last_day(date)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeDatetime)),
		},
		"last_value": {
			description: "The last value of expr in the frame. Can IGNORE or RESPECT NULLS.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/window_functions#last_valueexpr-order-by-ordering-ignore-nulls",
			signature:   "last_value(expr[ ORDER BY ordering][ IGNORE NULLS])",
			returnType:  withArity(1, 1, preservesArgTypeAndHint),
		},
		"lead": {
			description: "The value of expr n rows after the current row, or the default. Can IGNORE or RESPECT NULLS.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/window_functions#leadexpr-offset-default-order-by-ordering-ignore-nulls",
			signature:   "lead(expr[, offset[, default]][ ORDER BY ordering][ IGNORE NULLS])",
			returnType:  withArity(1, 3, preservesArgTypeAndHint),
		},
		"least": {
			description: "Returns the smallest value. For strings lexicographical ordering is used. Note that uppercase characters are considered \"smaller\" than lowercase characters, and collations are not supported.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/text#leastarg1-",
			signature:   "least(arg1, ...)",
			returnType:  withArity(1, -1, preservesArgType),
		},
		"left": {
			description: "Extracts the left-most count characters.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/text#leftstring-count",
			signature:   "left(string, count)",
			returnType:  withArity(2, 2, fixedType(ColumnTypeString)),
		},
		"len": {
			description: "Number of characters in string.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/text#lengthstring",
			signature:   "len(string)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"length": {
			description: "Number of characters in string.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/text#lengthstring",
			signature:   "length(string)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"levenshtein": {
			description: "The minimum number of single-character edits (insertions, deletions or substitutions) required to change one string to the other. Characters of different cases (e.g., a and A) are considered different.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/text#levenshteins1-s2",
			signature:   "levenshtein(s1, s2)",
			returnType:  withArity(2, 2, fixedType(ColumnTypeNumber)),
		},
		"list": {
			description: "Returns a LIST containing all the values of a column.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/aggregates#listarg",
			signature:   "list(arg)",
			returnType:  withArity(1, 1, listReturnType),
		},
		"list_aggregate": {
			description: "Executes the aggregate function function_name on the elements of list.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/list#list_aggregatelist-function_name-",
			signature:   "list_aggregate(list, function_name, ...)",
			returnType:  withArity(2, -1, listAggregateReturnType),
		},
		"list_concat": {
			description: "Concatenates lists. NULL inputs are skipped. See also operator ||.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/list#list_concatlist_1--list_n",
			signature:   "list_concat(list_1, ..., list_n)",
			returnType:  withArity(0, -1, fixedType(ColumnTypeJSON)),
		},
		"list_contains": {
			description: "Returns true if the list contains the element.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/list#list_containslist-element",
			signature:   "list_contains(list, element)",
			returnType:  withArity(2, 2, fixedType(ColumnTypeBoolean)),
		},
		"list_distinct": {
			description: "Removes all duplicates and NULL values from a list. Does not preserve the original order.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/list#list_distinctlist",
			signature:   "list_distinct(list)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeJSON)),
		},
		"list_has_all": {
			description: "Returns true if all elements of list2 are in list1. NULLs are ignored.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/list#list_has_alllist1-list2",
			signature:   "list_has_all(list1, list2)",
			returnType:  withArity(2, 2, fixedType(ColumnTypeBoolean)),
		},
		"list_has_any": {
			description: "Returns true if the lists have any element in common. NULLs are ignored.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/list#list_has_anylist1-list2",
			signature:   "list_has_any(list1, list2)",
			returnType:  withArity(2, 2, fixedType(ColumnTypeBoolean)),
		},
		"list_intersect": {
			description: "Returns a list containing the distinct elements that are present in both list1 and list2.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/list#list_intersectlist1-list2",
			signature:   "list_intersect(list1, list2)",
			returnType:  withArity(2, 2, fixedType(ColumnTypeJSON)),
		},
		"list_position": {
			description: "Returns the index of the element if the list contains the element. If the element is not found, it returns NULL.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/list#list_positionlist-element",
			signature:   "list_position(list, element)",
			returnType:  withArity(2, 2, fixedType(ColumnTypeNumber)),
		},
		"list_reverse_sort": {
			description: "Sorts the elements of the list in reverse order.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/list#list_reverse_sortlist-col1",
			signature:   "list_reverse_sort(list[, col1])",
			returnType:  withArity(1, 2, fixedType(ColumnTypeJSON)),
		},
		"list_slice": {
			description: "Extracts a sublist or substring using slice conventions. Negative values are accepted.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/list#list_slicelist-begin-end",
			signature:   "list_slice(list, begin, end)",
			returnType:  withArity(3, 4, fixedType(ColumnTypeJSON)),
		},
		"list_sort": {
			description: "Sorts the elements of the list.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/list#list_sortlist-col1-col2",
			signature:   "list_sort(list[, col1][, col2])",
			returnType:  withArity(1, 3, fixedType(ColumnTypeJSON)),
		},
		"list_unique": {
			description: "Counts the unique elements of a list.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/list#list_uniquelist",
			signature:   "list_unique(list)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"list_value": {
			description: "Creates a LIST containing the argument values.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/list#list_valuearg-",
			signature:   "list_value(arg, ...)",
			returnType:  withArity(0, -1, fixedType(ColumnTypeJSON)),
		},
		"ln": {
			description: "Computes the natural logarithm of x.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/numeric#lnx",
			signature:   "ln(x)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"log": {
			description: "Computes the logarithm of x to base b. b may be omitted, in which case the default 10.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/numeric#logx",
			signature:   "log(x)",
			returnType:  withArity(1, 2, fixedType(ColumnTypeNumber)),
		},
		"log10": {
			description: "Computes the 10-log of x.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/numeric#log10x",
			signature:   "log10(x)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"log2": {
			description: "Computes the 2-log of x.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/numeric#log2x",
			signature:   "log2(x)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"lower": {
			description: "Converts string to lower case.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/text#lowerstring",
			signature:   "lower(string)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeString)),
		},
		"lpad": {
			description: "Pads the string with the character on the left until it has count characters. Truncates the string on the right if it has more than count characters.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/text#lpadstring-count-character",
			signature:   "lpad(string, count, character)",
			returnType:  withArity(3, 3, fixedType(ColumnTypeString)),
		},
		"ltrim": {
			description: "Removes any occurrences of any of the characters from the left side of the string. characters defaults to space.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/text#ltrimstring-characters",
			signature:   "ltrim(string[, characters])",
			returnType:  withArity(1, 2, fixedType(ColumnTypeString)),
		},
		"make_date": {
			description: "The date for the given parts.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/date#make_dateyear-month-day",
			signature:   "make_date(year, month, day)",
			returnType:  withArity(1, 3, fixedType(ColumnTypeDatetime)),
		},
		"make_timestamp": {
			description: "The timestamp for the given parts.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/timestamp#make_timestampbigint-bigint-bigint-bigint-bigint-double",
			signature:   "make_timestamp(bigint, bigint, bigint, bigint, bigint, double)",
			returnType:  withArity(1, 6, fixedType(ColumnTypeDatetime)),
		},
		"make_timestamp_ms": {
			description: "Constructs a timestamp from the given number of milliseconds since the epoch.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/timestamp#make_timestamp_msmilliseconds",
			signature:   "make_timestamp_ms(milliseconds)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeDatetime)),
		},
		"map_entries": {
			description: "Returns the map entries as a list of keys/values.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/map#map_entriesmap",
			signature:   "map_entries(map)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeJSON)),
		},
		"map_extract": {
			description: "Returns a list containing the value for a given key, or an empty list if the key is not contained in the map.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/map#map_extractmap-key",
			signature:   "map_extract(map, key)",
			returnType:  withArity(2, 2, fixedType(ColumnTypeJSON)),
		},
		"map_keys": {
			description: "Returns the keys of a map as a list.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/map#map_keysmap",
			signature:   "map_keys(map)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeJSON)),
		},
		"max": {
			description: "Returns the maximum value present in arg.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/aggregates#maxarg",
			signature:   "max(arg)",
			returnType:  withArity(1, 2, preservesArgTypeAndHint),
		},
		"md5": {
			description: "Returns the MD5 hash of the string as a VARCHAR.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/utility#md5string",
			signature:   "md5(string)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeString)),
		},
		"median": {
			description: "Returns the middle value of the set. NULL values are ignored. For even value counts, interpolate-able types (numeric, date/time) return the average of the two middle values. Non-interpolate-able types (everything else) return the lower of the two middle values.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/aggregates#medianx",
			signature:   "median(x)",
			returnType:  withArity(1, 1, preservesArgType),
		},
		"min": {
			description: "Returns the minimum value present in arg.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/aggregates#minarg",
			signature:   "min(arg)",
			returnType:  withArity(1, 2, preservesArgTypeAndHint),
		},
		"minute": {
			description: "Extract the minute component from a date or timestamp.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/datepart#minutedate",
			signature:   "minute(date)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"mod": {
			description: "The remainder of x divided by y.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/numeric",
			signature:   "mod(x, y)",
			returnType:  withArity(2, 2, fixedType(ColumnTypeNumber)),
		},
		"mode": {
			description: "Returns the most frequent value for the values within x. NULL values are ignored.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/aggregates#modex",
			signature:   "mode(x)",
			returnType:  withArity(1, 1, preservesArgTypeAndHint),
		},
		"month": {
			description: "Extract the month component from a date or timestamp.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/datepart#monthdate",
			signature:   "month(date)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"now": {
			description: "Returns the current timestamp.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/date",
			signature:   "now()",
			returnType:  withArity(0, 0, fixedType(ColumnTypeDatetime)),
		},
		"nth_value": {
			description: "The nth value of expr in the frame. Can IGNORE or RESPECT NULLS.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/window_functions#nth_valueexpr-nth-order-by-ordering-ignore-nulls",
			signature:   "nth_value(expr, nth[ ORDER BY ordering][ IGNORE NULLS])",
			returnType:  withArity(2, 2, preservesArgTypeAndHint),
		},
		"ntile": {
			description: "The row bucket in a window partition for a given bucket count.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/window_functions#ntilenum_buckets-order-by-ordering",
			signature:   "ntile(num_buckets[ ORDER BY ordering])",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"percent_rank": {
			description: "The relative rank of the current row as a fraction.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/window_functions#percent_rankorder-by-ordering",
			signature:   "percent_rank([ORDER BY ordering])",
			returnType:  withArity(0, 0, fixedType(ColumnTypeNumber)),
		},
		"pi": {
			description: "Returns the value of pi.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/numeric#pi",
			signature:   "pi()",
			returnType:  withArity(0, 0, fixedType(ColumnTypeNumber)),
		},
		"pow": {
			description: "Computes x to the power of y.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/numeric#powx-y",
			signature:   "pow(x, y)",
			returnType:  withArity(2, 2, fixedType(ColumnTypeNumber)),
		},
		"power": {
			description: "Computes x to the power of y.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/numeric#powerx-y",
			signature:   "power(x, y)",
			returnType:  withArity(2, 2, fixedType(ColumnTypeNumber)),
		},
		"printf": {
			description: "Formats a string using printf syntax.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/text#printfformat-",
			signature:   "printf(format, ...)",
			returnType:  withArity(1, -1, fixedType(ColumnTypeString)),
		},
		"product": {
			description: "Calculates the product of all tuples in arg.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/aggregates#productarg",
			signature:   "product(arg)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"quantile_cont": {
			description: "Returns the interpolated quantile number between 0 and 1 . If pos is a LIST of FLOATs, then the result is a LIST of the corresponding interpolated quantiles.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/aggregates#quantile_contx-pos",
			signature:   "quantile_cont(x, pos)",
			returnType:  withArity(2, 2, preservesArgType),
		},
		"quantile_disc": {
			description: "Returns the exact quantile number between 0 and 1 . If pos is a LIST of FLOATs, then the result is a LIST of the corresponding exact quantiles.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/aggregates#quantile_discx-pos",
			signature:   "quantile_disc(x, pos)",
			returnType:  withArity(1, 2, preservesArgTypeAndHint),
		},
		"quarter": {
			description: "Extract the quarter component from a date or timestamp.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/datepart#quarterdate",
			signature:   "quarter(date)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"radians": {
			description: "Converts degrees to radians.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/numeric#radiansx",
			signature:   "radians(x)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"range": {
			description: "Creates a list of values between start and stop - the stop parameter is exclusive.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/list#rangestart-stop-step",
			signature:   "range(start[, stop][, step])",
			returnType:  withArity(1, 3, fixedType(ColumnTypeJSON)),
		},
		"rank": {
			description: "The rank of the current row with gaps.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/window_functions#rankorder-by-ordering",
			signature:   "rank([ORDER BY ordering])",
			returnType:  withArity(0, 0, fixedType(ColumnTypeNumber)),
		},
		"regexp_extract": {
			description: "If string contains the regex pattern, returns the capturing group specified by optional parameter group; otherwise, returns the empty string. The group must be a constant value. If no group is given, it defaults to 0. A set of optional regex options can be set.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/regular_expressions#regexp_extractstring-pattern-group--0-options",
			signature:   "regexp_extract(string, pattern[, group = 0][, options])",
			returnType:  withArity(2, 4, fixedType(ColumnTypeString)),
		},
		"regexp_extract_all": {
			description: "Finds non-overlapping occurrences of the regex in the string and returns the corresponding values of the capturing group. A set of optional regex options can be set.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/regular_expressions#regexp_extract_allstring-regex-group--0-options",
			signature:   "regexp_extract_all(string, regex[, group = 0][, options])",
			returnType:  withArity(2, 4, fixedType(ColumnTypeJSON)),
		},
		"regexp_full_match": {
			description: "Returns true if the entire string matches the regex. A set of optional regex options can be set.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/regular_expressions#regexp_full_matchstring-regex-options",
			signature:   "regexp_full_match(string, regex[, options])",
			returnType:  withArity(2, 3, fixedType(ColumnTypeBoolean)),
		},
		"regexp_matches": {
			description: "Returns true if string contains the regex, false otherwise. A set of optional regex options can be set.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/regular_expressions#regexp_matchesstring-pattern-options",
			signature:   "regexp_matches(string, pattern[, options])",
			returnType:  withArity(2, 3, fixedType(ColumnTypeBoolean)),
		},
		"regexp_replace": {
			description: "If string contains the regex, replaces the matching part with replacement. A set of optional regex options can be set.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/regular_expressions#regexp_replacestring-pattern-replacement-options",
			signature:   "regexp_replace(string, pattern, replacement[, options])",
			returnType:  withArity(3, 4, fixedType(ColumnTypeString)),
		},
		"regexp_split_to_array": {
			description: "Splits the string along the regex. A set of optional regex options can be set.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/regular_expressions#regexp_split_to_arraystring-regex-options",
			signature:   "regexp_split_to_array(string, regex[, options])",
			returnType:  withArity(2, 3, fixedType(ColumnTypeJSON)),
		},
		"repeat": {
			description: "Repeats the string count number of times.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/text#repeatstring-count",
			signature:   "repeat(string, count)",
			returnType:  withArity(2, 2, fixedType(ColumnTypeString)),
		},
		"replace": {
			description: "Replaces any occurrences of the source with target in string.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/text#replacestring-source-target",
			signature:   "replace(string, source, target)",
			returnType:  withArity(3, 3, fixedType(ColumnTypeString)),
		},
		"reverse": {
			description: "Reverses the string.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/text#reversestring",
			signature:   "reverse(string)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeString)),
		},
		"right": {
			description: "Extract the right-most count characters.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/text#rightstring-count",
			signature:   "right(string, count)",
			returnType:  withArity(2, 2, fixedType(ColumnTypeString)),
		},
		"round": {
			description: "Rounds x to s decimal places.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/numeric#roundv-numeric-s-integer",
			signature:   "round(v NUMERIC, s INTEGER)",
			returnType:  withArity(1, 2, fixedType(ColumnTypeNumber)),
		},
		"row_number": {
			description: "The row number in a window partition.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/window_functions#row_numberorder-by-ordering",
			signature:   "row_number([ORDER BY ordering])",
			returnType:  withArity(0, 0, fixedType(ColumnTypeNumber)),
		},
		"rpad": {
			description: "Pads the string with the character on the right until it has count characters. Truncates the string on the right if it has more than count characters.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/text#rpadstring-count-character",
			signature:   "rpad(string, count, character)",
			returnType:  withArity(3, 3, fixedType(ColumnTypeString)),
		},
		"rtrim": {
			description: "Removes any occurrences of any of the characters from the right side of the string. characters defaults to space.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/text#rtrimstring-characters",
			signature:   "rtrim(string[, characters])",
			returnType:  withArity(1, 2, fixedType(ColumnTypeString)),
		},
		"second": {
			description: "Extract the second component from a date or timestamp.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/datepart#seconddate",
			signature:   "second(date)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"sha256": {
			description: "Returns a VARCHAR with the SHA-256 hash of the value.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/utility#sha256string",
			signature:   "sha256(string)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeString)),
		},
		"sign": {
			description: "Returns the sign of x as -1, 0 or 1.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/numeric#signx",
			signature:   "sign(x)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"sin": {
			description: "Computes the sine of x.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/numeric#sinx",
			signature:   "sin(x)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"sinh": {
			description: "Computes the hyperbolic sine of x.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/numeric",
			signature:   "sinh(x)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"skewness": {
			description: "Returns the skewness of all input values.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/aggregates#skewnessx",
			signature:   "skewness(x)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"split_part": {
			description: "Splits the string along the separator and returns the data at the (1-based) index.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/text#split_partstring-separator-index",
			signature:   "split_part(string, separator, index)",
			returnType:  withArity(3, 3, fixedType(ColumnTypeString)),
		},
		"sqrt": {
			description: "Returns the square root of x.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/numeric#sqrtx",
			signature:   "sqrt(x)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"starts_with": {
			description: "Returns true if string begins with search_string.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/text#starts_withstring-search_string",
			signature:   "starts_with(string, search_string)",
			returnType:  withArity(2, 2, fixedType(ColumnTypeBoolean)),
		},
		"stddev": {
			description: "Returns the sample standard deviation.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/aggregates#stddev_sampx",
			signature:   "stddev(x)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"stddev_pop": {
			description: "Returns the population standard deviation.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/aggregates#stddev_popx",
			signature:   "stddev_pop(x)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"stddev_samp": {
			description: "Returns the sample standard deviation.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/aggregates#stddev_sampx",
			signature:   "stddev_samp(x)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"strftime": {
			description: "Converts a date to a string according to the format string.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/date#strftimedate-format",
			signature:   "strftime(date, format)",
			returnType:  withArity(2, 2, fixedType(ColumnTypeString)),
		},
		"string_agg": {
			description: "Concatenates the column string values with an optional separator.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/aggregates#string_aggarg-sep",
			signature:   "string_agg(arg)",
			returnType:  withArity(1, 2, fixedType(ColumnTypeString)),
		},
		"string_split": {
			description: "Splits the string along the separator.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/text#string_splitstring-separator",
			signature:   "string_split(string, separator)",
			returnType:  withArity(2, 2, fixedType(ColumnTypeJSON)),
		},
		"strip_accents": {
			description: "Strips accents from string.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/text#strip_accentsstring",
			signature:   "strip_accents(string)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeString)),
		},
		"strptime": {
			description: "Converts the string text to timestamp according to the format string. Throws an error on failure. To return NULL on failure, use try_strptime.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/timestamp#strptimetext-format-list",
			signature:   "strptime(text, format-list)",
			returnType:  withArity(2, 2, fixedType(ColumnTypeDatetime)),
		},
		"struct_extract": {
			description: "Extract the named entry from the STRUCT.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/struct#struct_extractstruct-entry",
			signature:   "struct_extract(struct, 'entry')",
			returnType:  withArity(2, 2, fixedType(ColumnTypeJSON)),
		},
		"struct_keys": {
			description: "Returns the field names of a STRUCT as a list.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/struct",
			signature:   "struct_keys(struct)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeJSON)),
		},
		"struct_pack": {
			description: "Create a STRUCT containing the argument values. The entry name will be the bound variable name.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/struct#struct_packname--any-",
			signature:   "struct_pack(name := any, ...)",
			returnType:  withArity(0, -1, fixedType(ColumnTypeJSON)),
		},
		"substr": {
			description: "Extracts substring starting from character start up to the end of the string. If optional argument length is set, extracts a substring of length characters instead. Note that a start value of 1 refers to the first character of the string.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/text#substringstring-start-length",
			signature:   "substr(string, start[, length])",
			returnType:  withArity(2, 3, fixedType(ColumnTypeString)),
		},
		"sum": {
			description: "Calculates the sum value for all tuples in arg.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/aggregates#sumarg",
			signature:   "sum(arg)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"tan": {
			description: "Computes the tangent of x.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/numeric#tanx",
			signature:   "tan(x)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"tanh": {
			description: "Computes the hyperbolic tangent of x.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/numeric",
			signature:   "tanh(x)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"time_bucket": {
			description: "Truncate TIMESTAMPTZ by the specified interval bucket_width, aligned relative to an origin (defaults to 2000-01-03 for sub-month buckets, 2000-01-01 for month/year buckets).",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/timestamp#time_bucketbucket_width-timestamp-origin",
			signature:   "time_bucket(bucket_width, timestamp[, origin])",
			returnType:  withArity(2, 3, fixedType(ColumnTypeDatetime)),
		},
		"timezone": {
			description: "Extract the timezone component from a date or timestamp.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/datepart#timezonedate",
			signature:   "timezone(date)",
			returnType:  withArity(1, 2, fixedType(ColumnTypeDatetime)),
		},
		"to_json": {
			description: "Converts a value to JSON.",
			docsURL:     "https://duckdb.org/docs/current/data/json/creating_json",
			signature:   "to_json(any)",
			returnType:  withArity(0, -1, fixedType(ColumnTypeJSON)),
		},
		"to_timestamp": {
			description: "Converts secs since epoch to a timestamp with time zone.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/timestamp",
			signature:   "to_timestamp(sec)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeDatetime)),
		},
		"today": {
			description: "Current date (start of current transaction) in the local time zone.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/date#today",
			signature:   "today()",
			returnType:  withArity(0, 0, fixedType(ColumnTypeDatetime)),
		},
		"translate": {
			description: "Replaces each character in string that matches a character in the from set with the corresponding character in the to set. If from is longer than to, occurrences of the extra characters in from are deleted.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/text#translatestring-from-to",
			signature:   "translate(string, from, to)",
			returnType:  withArity(3, 3, fixedType(ColumnTypeString)),
		},
		"trunc": {
			description: "Truncates the number.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/numeric#truncx",
			signature:   "trunc(x)",
			returnType:  withArity(1, 2, fixedType(ColumnTypeNumber)),
		},
		"try_strptime": {
			description: "Converts the string text to timestamp according to the format string. Returns NULL on failure (unlike strptime, which errors).",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/timestamp#try_strptimetext-format-list",
			signature:   "try_strptime(text, format-list)",
			returnType:  withArity(2, 2, fixedType(ColumnTypeDatetime)),
		},
		"typeof": {
			description: "Returns the name of the data type of the result of the expression.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/utility#typeofexpression",
			signature:   "typeof(expression)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeString)),
		},
		"unicode": {
			description: "Returns an INTEGER representing the unicode codepoint of the first character in the string.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/text#unicodestring",
			signature:   "unicode(string)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"unnest": {
			description: "Unnests a list or struct by one level, turning a single row's list into one row per element.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/list#unnestlist",
			signature:   "unnest(list)",
			returnType:  withArity(1, 1, unnestReturnType),
		},
		"upper": {
			description: "Converts string to upper case.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/text#upperstring",
			signature:   "upper(string)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeString)),
		},
		"var_pop": {
			description: "Returns the population variance.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/aggregates#var_popx",
			signature:   "var_pop(x)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"var_samp": {
			description: "Returns the sample variance of all input values.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/aggregates#var_sampx",
			signature:   "var_samp(x)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"variance": {
			description: "Returns the sample variance of all input values.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/aggregates#var_sampx",
			signature:   "variance(arg, val)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"week": {
			description: "Extract the week component from a date or timestamp.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/datepart#weekdate",
			signature:   "week(date)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
		"year": {
			description: "Extract the year component from a date or timestamp.",
			docsURL:     "https://duckdb.org/docs/current/sql/functions/datepart#yeardate",
			signature:   "year(date)",
			returnType:  withArity(1, 1, fixedType(ColumnTypeNumber)),
		},
	}
}
