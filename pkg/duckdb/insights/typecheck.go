package insights

import (
	"strings"

	"github.com/inngest/inngest/pkg/duckdb/parser"
)

// inferType computes expr's ColumnType against scope, covering every AST
// shape validate's allowlist can actually produce. This is what lets
// deriveTable assign a real type to a CTE's or subquery's computed output
// columns instead of requiring an alias for anything that isn't a bare
// column reference.
//
// Unlike a general-purpose SQL type checker, this never needs to model
// implicit-cast promotion across arbitrary types, unknown functions, or
// DDL — the function allowlist is small and fixed, so every case below is
// exhaustive against it. Anything this function genuinely can't determine
// returns ColumnTypeUnknown rather than guessing.
func inferType(expr parser.Expr, scope *tableScope) ColumnType {
	switch e := expr.(type) {
	case *parser.Literal:
		switch e.Kind {
		case parser.LitString:
			return ColumnTypeString
		case parser.LitNumber:
			return ColumnTypeNumber
		case parser.LitTrue, parser.LitFalse:
			return ColumnTypeBoolean
		default: // LitNull: no type information in a bare NULL
			return ColumnTypeUnknown
		}
	case *parser.Ident:
		return identType(e, scope)
	case *parser.CastExpr:
		return DuckDBToColumnType(e.Type)
	case *parser.TypeLiteral:
		return DuckDBToColumnType(e.Type)
	case *parser.IntervalExpr:
		return ColumnTypeDatetime
	case *parser.CaseExpr:
		return caseExprType(e, scope)
	case *parser.FunctionExpr:
		return functionReturnType(strings.ToLower(strings.Join(e.Name, ".")), e.Args, scope)
	case *parser.BinaryExpr:
		return binaryExprType(e, scope)
	case *parser.UnaryExpr:
		return unaryExprType(e, scope)
	case *parser.LambdaExpr:
		// A single-param LambdaExpr is how this parser represents a bare
		// "col -> 'key'" JSON access, which returns JSON in DuckDB (unlike
		// "->>", which returns VARCHAR).
		return ColumnTypeJSON
	case *parser.DotExpr:
		// A struct-field/JSON-path access — the exact underlying type
		// isn't tracked at this granularity (this package doesn't model
		// STRUCT field schemas), so this reports the same generic JSON
		// bucket DuckDBToColumnType gives any composite type.
		return ColumnTypeJSON
	case *parser.NullTest, *parser.IsExpr, *parser.BetweenExpr, *parser.LikeExpr,
		*parser.InExpr, *parser.DistinctFromExpr:
		return ColumnTypeBoolean
	case *parser.ListExpr, *parser.StructExpr, *parser.MapExpr:
		return ColumnTypeJSON
	case *parser.ListComprehensionExpr:
		// Always produces a LIST, same JSON bucket as any other composite
		// type above -- regardless of what its Expr/Filter reference.
		return ColumnTypeJSON
	default:
		// *parser.SubqueryExpr, *parser.Parameter, *parser.StarExpr (never
		// meaningful as a single type; deriveTable expands it before ever
		// calling inferType), *parser.PositionalExpr, *parser.DefaultExpr,
		// *parser.GroupingExpr, *parser.SliceExpr: none of these have a
		// type this package can determine.
		return ColumnTypeUnknown
	}
}

// identType resolves a (possibly table-qualified) column reference to its
// declared knownColumn.colType — mirrors resolveIdentHint's (Task 5)
// bare/qualified resolution exactly, but for type instead of hint.
func identType(id *parser.Ident, scope *tableScope) ColumnType {
	switch len(id.Parts) {
	case 1:
		col, ok := scope.uniqueColumn(id.Parts[0])
		if !ok {
			return ColumnTypeUnknown
		}
		return col.colType
	case 2:
		tbl, ok := scope.lookup(id.Parts[0])
		if !ok {
			return ColumnTypeUnknown
		}
		col, ok := tbl.columns[id.Parts[1]]
		if !ok {
			return ColumnTypeUnknown
		}
		return col.colType
	default:
		return ColumnTypeUnknown
	}
}

// caseExprType unifies a CASE expression's WHEN results and ELSE branch:
// if every branch present agrees on type, that's the CASE's type;
// otherwise (including "no branches produced a determinable type")
// ColumnTypeUnknown.
func caseExprType(c *parser.CaseExpr, scope *tableScope) ColumnType {
	var (
		result ColumnType
		set    bool
	)
	consider := func(t ColumnType) bool {
		if !set {
			result, set = t, true
			return true
		}
		return t == result
	}
	for _, w := range c.Whens {
		if !consider(inferType(w.Result, scope)) {
			return ColumnTypeUnknown
		}
	}
	if c.Else != nil {
		if !consider(inferType(c.Else, scope)) {
			return ColumnTypeUnknown
		}
	}
	if !set {
		return ColumnTypeUnknown
	}
	return result
}

// binaryExprType categorizes a BinaryExpr by its operator. Arithmetic
// operators report DATETIME when either operand is DATETIME-typed (date
// +/- interval, the common case this package's tables actually support —
// e.g. queued_at - started_at), NUMBER otherwise.
func binaryExprType(b *parser.BinaryExpr, scope *tableScope) ColumnType {
	switch strings.ToUpper(b.Op) {
	case "=", "==", "<>", "!=", "<", ">", "<=", ">=", "AND", "OR":
		return ColumnTypeBoolean
	case "||":
		return ColumnTypeString
	case "->>":
		return ColumnTypeString
	case "->":
		return ColumnTypeJSON
	case "+", "-", "*", "/", "%", "^", "//":
		if inferType(b.Left, scope) == ColumnTypeDatetime || inferType(b.Right, scope) == ColumnTypeDatetime {
			return ColumnTypeDatetime
		}
		return ColumnTypeNumber
	case "&", "|", "~", "<<", ">>":
		return ColumnTypeNumber
	default:
		return ColumnTypeUnknown
	}
}

// unaryExprType categorizes a UnaryExpr by its operator.
func unaryExprType(u *parser.UnaryExpr, scope *tableScope) ColumnType {
	switch strings.ToUpper(u.Op) {
	case "NOT":
		return ColumnTypeBoolean
	case "-", "+", "~":
		return inferType(u.X, scope)
	default:
		return ColumnTypeUnknown
	}
}

// functionReturnType has one case per key in allowedFunctions (functions.go)
// — if you add a function there, add its return-type rule here too.
// "Pick one of my arguments" functions (MIN/MAX/COALESCE/NULLIF/
// GREATEST/LEAST/ANY_VALUE) preserve their first argument's type rather
// than converting it, matching what they actually do in DuckDB.
func functionReturnType(name string, args []parser.Expr, scope *tableScope) ColumnType {
	switch name {
	case "count", "length", "len", "round", "abs",
		"date_part", "date_diff", "epoch", "json_array_length", "sum", "avg",
		"year", "month", "day", "hour", "minute", "second",
		// Statistical aggregates that don't preserve their input's type
		// (unlike MEDIAN/QUANTILE_*, grouped with MIN/MAX below).
		"stddev", "stddev_samp", "stddev_pop", "variance", "var_samp", "var_pop",
		"corr", "covar_samp", "covar_pop", "entropy", "skewness", "kurtosis",
		"approx_quantile", "approx_count_distinct", "product",
		// Window functions with no representative argument to preserve.
		"row_number", "rank", "dense_rank", "ntile", "cume_dist", "percent_rank",
		// Scalar numeric.
		"instr", "ascii", "unicode", "levenshtein", "jaccard", "hash",
		"ceil", "ceiling", "floor", "trunc", "sign", "power", "pow", "sqrt",
		"exp", "ln", "log", "log10", "log2", "mod", "pi", "cbrt",
		"isodow", "isoyear", "week", "quarter", "dayofweek", "dayofyear",
		"array_length", "list_unique", "list_position", "array_position":
		return ColumnTypeNumber
	case "lower", "upper", "concat", "substr", "ltrim",
		"rtrim", "replace", "split_part", "string_agg", "strftime",
		"json_extract_string", "regexp_replace", "regexp_extract",
		"lpad", "rpad", "reverse", "repeat", "json_type", "typeof",
		"left", "right", "format", "printf", "md5", "sha256", "translate",
		"chr", "strip_accents", "array_to_string", "json_value":
		return ColumnTypeString
	case "date_trunc", "now", "age", "make_date", "make_timestamp",
		"timezone", "last_day", "date_add", "date_sub",
		"strptime", "to_timestamp":
		return ColumnTypeDatetime
	case "json_valid", "regexp_matches", "starts_with", "contains", "ends_with",
		"json_contains", "json_exists",
		"bool_and", "bool_or", "list_contains", "array_contains",
		"list_has_any", "list_has_all", "regexp_full_match":
		return ColumnTypeBoolean
	case "json_extract", "to_json", "array_agg", "list", "unnest",
		"json_keys", "json_structure", "json_merge_patch", "json_quote",
		"json_group_array", "json_group_object",
		"json_array", "json_object", "json_extract_path",
		"list_distinct", "list_sort", "list_reverse_sort", "list_slice",
		"list_concat", "list_intersect", "flatten", "list_value",
		"string_split", "regexp_split_to_array":
		return ColumnTypeJSON
	// "Pick one of my arguments" functions preserve their representative
	// argument's type rather than converting it, matching what they
	// actually do in DuckDB (confirmed empirically for MEDIAN/QUANTILE_*,
	// which return the input's own type for ordinal types like TIMESTAMP,
	// not always a DOUBLE) -- FIRST/LAST/ARG_MAX/ARG_MIN/BIT_*/window
	// functions like LAG/LEAD all behave the same way: the value column
	// (always args[0]) is the representative type.
	case "min", "max", "greatest", "least", "any_value", "ifnull", "nvl",
		"median", "mode", "quantile_cont", "quantile_disc",
		"first", "last", "arg_max", "arg_min",
		"bit_and", "bit_or", "bit_xor",
		"lag", "lead", "first_value", "last_value", "nth_value":
		if len(args) == 0 {
			return ColumnTypeUnknown
		}
		return inferType(args[0], scope)
	case "if":
		// IF(condition, then, else) -- the "then" branch (args[1]) is the
		// representative type, matching how caseExprType treats a CASE's
		// branches.
		if len(args) < 2 {
			return ColumnTypeUnknown
		}
		return inferType(args[1], scope)
	case "list_aggregate", "array_aggregate":
		// list_aggregate(list, name) applies the aggregate named by args[1]
		// (e.g. 'sum', 'string_agg', 'bit_and') to args[0] -- statically
		// typeable only when that name is a literal string constant (by far
		// the common case: written directly in the query, not built from an
		// expression). Recurses into functionReturnType with the resolved
		// name and args[0] alone, so e.g. list_aggregate(x, 'string_agg')
		// reports the same type STRING_AGG(x) would, reusing every rule
		// above instead of duplicating it. A dynamic (non-literal) name, or
		// one that doesn't match any known aggregate, is genuinely
		// unknowable without asking DuckDB -- reported as such rather than
		// guessed.
		if len(args) < 2 {
			return ColumnTypeUnknown
		}
		lit, ok := args[1].(*parser.Literal)
		if !ok || lit.Kind != parser.LitString {
			return ColumnTypeUnknown
		}
		return functionReturnType(strings.ToLower(lit.Text), args[:1], scope)
	default:
		return ColumnTypeUnknown
	}
}
