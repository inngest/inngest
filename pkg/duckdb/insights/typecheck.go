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
		return functionReturnType(strings.ToUpper(strings.Join(e.Name, ".")), e.Args, scope)
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
	case "COUNT", "LENGTH", "LEN", "ROUND", "ABS",
		"DATE_PART", "DATE_DIFF", "EPOCH", "JSON_ARRAY_LENGTH", "SUM", "AVG",
		"YEAR", "MONTH", "DAY", "HOUR", "MINUTE", "SECOND",
		// Statistical aggregates that don't preserve their input's type
		// (unlike MEDIAN/QUANTILE_*, grouped with MIN/MAX below).
		"STDDEV", "STDDEV_SAMP", "STDDEV_POP", "VARIANCE", "VAR_SAMP", "VAR_POP",
		"CORR", "COVAR_SAMP", "COVAR_POP", "ENTROPY", "SKEWNESS", "KURTOSIS",
		"APPROX_QUANTILE", "APPROX_COUNT_DISTINCT", "PRODUCT",
		// Window functions with no representative argument to preserve.
		"ROW_NUMBER", "RANK", "DENSE_RANK", "NTILE", "CUME_DIST", "PERCENT_RANK",
		// Scalar numeric.
		"INSTR", "ASCII", "UNICODE", "LEVENSHTEIN", "JACCARD", "HASH",
		"CEIL", "CEILING", "FLOOR", "TRUNC", "SIGN", "POWER", "POW", "SQRT",
		"EXP", "LN", "LOG", "LOG10", "LOG2", "MOD", "PI", "CBRT",
		"ISODOW", "ISOYEAR", "WEEK", "QUARTER", "DAYOFWEEK", "DAYOFYEAR",
		"ARRAY_LENGTH", "LIST_UNIQUE", "LIST_POSITION", "ARRAY_POSITION":
		return ColumnTypeNumber
	case "LOWER", "UPPER", "CONCAT", "SUBSTR", "LTRIM",
		"RTRIM", "REPLACE", "SPLIT_PART", "STRING_AGG", "STRFTIME",
		"JSON_EXTRACT_STRING", "REGEXP_REPLACE", "REGEXP_EXTRACT",
		"LPAD", "RPAD", "REVERSE", "REPEAT", "JSON_TYPE", "TYPEOF",
		"LEFT", "RIGHT", "FORMAT", "PRINTF", "MD5", "SHA256", "TRANSLATE",
		"CHR", "STRIP_ACCENTS", "ARRAY_TO_STRING", "JSON_VALUE":
		return ColumnTypeString
	case "DATE_TRUNC", "NOW", "AGE", "MAKE_DATE", "MAKE_TIMESTAMP",
		"TIMEZONE", "LAST_DAY", "DATE_ADD", "DATE_SUB",
		"STRPTIME", "TO_TIMESTAMP":
		return ColumnTypeDatetime
	case "JSON_VALID", "REGEXP_MATCHES", "STARTS_WITH", "CONTAINS", "ENDS_WITH",
		"JSON_CONTAINS", "JSON_EXISTS",
		"BOOL_AND", "BOOL_OR", "LIST_CONTAINS", "ARRAY_CONTAINS",
		"LIST_HAS_ANY", "LIST_HAS_ALL", "REGEXP_FULL_MATCH":
		return ColumnTypeBoolean
	case "JSON_EXTRACT", "TO_JSON", "ARRAY_AGG", "LIST", "UNNEST",
		"JSON_KEYS", "JSON_STRUCTURE", "JSON_MERGE_PATCH", "JSON_QUOTE",
		"JSON_GROUP_ARRAY", "JSON_GROUP_OBJECT",
		"JSON_ARRAY", "JSON_OBJECT", "JSON_EXTRACT_PATH",
		"LIST_DISTINCT", "LIST_SORT", "LIST_REVERSE_SORT", "LIST_SLICE",
		"LIST_CONCAT", "LIST_INTERSECT", "FLATTEN", "LIST_VALUE",
		"STRING_SPLIT", "REGEXP_SPLIT_TO_ARRAY":
		return ColumnTypeJSON
	// "Pick one of my arguments" functions preserve their representative
	// argument's type rather than converting it, matching what they
	// actually do in DuckDB (confirmed empirically for MEDIAN/QUANTILE_*,
	// which return the input's own type for ordinal types like TIMESTAMP,
	// not always a DOUBLE) -- FIRST/LAST/ARG_MAX/ARG_MIN/BIT_*/window
	// functions like LAG/LEAD all behave the same way: the value column
	// (always args[0]) is the representative type.
	case "MIN", "MAX", "GREATEST", "LEAST", "ANY_VALUE", "IFNULL", "NVL",
		"MEDIAN", "MODE", "QUANTILE_CONT", "QUANTILE_DISC",
		"FIRST", "LAST", "ARG_MAX", "ARG_MIN",
		"BIT_AND", "BIT_OR", "BIT_XOR",
		"LAG", "LEAD", "FIRST_VALUE", "LAST_VALUE", "NTH_VALUE":
		if len(args) == 0 {
			return ColumnTypeUnknown
		}
		return inferType(args[0], scope)
	case "IF":
		// IF(condition, then, else) -- the "then" branch (args[1]) is the
		// representative type, matching how caseExprType treats a CASE's
		// branches.
		if len(args) < 2 {
			return ColumnTypeUnknown
		}
		return inferType(args[1], scope)
	case "LIST_AGGREGATE", "ARRAY_AGGREGATE":
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
		return functionReturnType(strings.ToUpper(lit.Text), args[:1], scope)
	default:
		return ColumnTypeUnknown
	}
}
