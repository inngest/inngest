package parser

import (
	"strconv"
	"strings"
)

// TypeName is a type as written in a cast (`x::INT[]`, `CAST(x AS ...)`) or
// a typed literal (`DATE '2024-01-01'`), built from the grammar's own Type
// rule (common.gram) rather than kept as source text.
//
// Every Expr a type can carry (modifiers, array sizes) is a child, so Walk
// sees it. The adapter only accepts number and string literals there, so a
// type can't smuggle a subquery or function call past a visitor either way.
type TypeName struct {
	baseExpr
	// Name is the base type name. When Keyword is set it's the grammar's
	// own keyword spelling, uppercased and space-joined into one element
	// (`INTEGER`, `CHARACTER VARYING`, `DOUBLE PRECISION`, `STRUCT`, `MAP`,
	// `TIMESTAMP`). Otherwise it's an identifier, possibly catalog/schema
	// qualified, one element per part, decoded but otherwise as written
	// (`json`, `main.my_enum`, `"varchar"` as `varchar`).
	Name    []string
	Keyword bool
	// Modifiers are the parenthesized type parameters (`DECIMAL(10, 2)`,
	// `VARCHAR(10)`, `TIMESTAMP(3)`, `FLOAT(24)`), which for ENUM are its
	// values (`ENUM('a', 'b')`). Always a *Literal of kind LitNumber or
	// LitString. An empty list (`VARCHAR()`) is the same as none.
	Modifiers []Expr
	// Fields are a STRUCT/ROW/UNION's members, in order. A bare STRUCT/ROW
	// (no parens) has none.
	Fields []*TypeField
	// Params are the member types of MAP (key, value) and TUPLE.
	Params []*TypeName
	// TimeZone is `WITH TIME ZONE` or `WITHOUT TIME ZONE` on TIME/TIMESTAMP.
	TimeZone string
	// Interval is an INTERVAL's qualifier as written, uppercased
	// (`YEAR TO MONTH`, `DAYS`). DuckDB ignores it.
	Interval string
	// Setof is set for `SETOF T`, which DuckDB parses as plain T. The
	// inner type's array bounds and any after it are both in Arrays, in
	// source order (`SETOF INT[] []` is INT[][]).
	Setof bool
	// Arrays are the type's list/array dimensions in source order:
	// `INT[3][]` (a list of 3-element arrays) is [{3}, {nil}]. `ARRAY`
	// and `ARRAY[n]` spellings are the same as `[]`/`[n]`.
	Arrays []ArrayBound
}

// ArrayBound is one `[]` (Size nil, a LIST) or `[n]` (a fixed-size ARRAY)
// dimension.
type ArrayBound struct {
	Size Expr
}

// TypeField is one `name Type` member of a STRUCT/ROW/UNION.
type TypeField struct {
	baseExpr
	Name string
	Type *TypeName
}

func (f *TypeField) Children() []Node { return []Node{f.Type} }
func (f *TypeField) String() string   { return "TypeField(" + f.Name + ")" }

func (t *TypeName) Children() []Node {
	var out []Node
	for _, m := range t.Modifiers {
		out = append(out, m)
	}
	for _, f := range t.Fields {
		out = append(out, f)
	}
	for _, p := range t.Params {
		out = append(out, p)
	}
	for _, b := range t.Arrays {
		if b.Size != nil {
			out = append(out, b.Size)
		}
	}
	return out
}

// String shows t's own fields, not its children: an identifier name is
// quoted (`TypeName("json")`) so it reads differently from a keyword
// spelling (`TypeName(INTEGER)`), and each array dimension is `[]` or
// `[n]`.
func (t *TypeName) String() string {
	var b strings.Builder
	b.WriteString("TypeName(")
	if t.Setof {
		b.WriteString("SETOF ")
	}
	for i, p := range t.Name {
		if i > 0 {
			b.WriteByte('.')
		}
		if t.Keyword {
			b.WriteString(p)
		} else {
			b.WriteString(strconv.Quote(p))
		}
	}
	if t.TimeZone != "" {
		b.WriteString(" " + t.TimeZone)
	}
	if t.Interval != "" {
		b.WriteString(" " + t.Interval)
	}
	for _, a := range t.Arrays {
		b.WriteByte('[')
		if lit, ok := a.Size.(*Literal); ok {
			b.WriteString(lit.Text)
		}
		b.WriteByte(']')
	}
	b.WriteByte(')')
	return b.String()
}

// BaseTypeName resolves t's base type, ignoring Arrays and Modifiers, to
// the name DuckDB itself reports for it (`typeof(NULL::<type>)`): aliases
// like INT4, TEXT, CHARACTER VARYING, DOUBLE PRECISION and ROW resolve to
// INTEGER, VARCHAR, VARCHAR, DOUBLE and STRUCT, and TIMESTAMP WITH TIME
// ZONE stays as is. A qualified name resolves by its last part, and a name
// DuckDB doesn't know as a builtin (a user type) comes back uppercased.
//
// Modifiers are ignored even where DuckDB folds them into the name, so
// DECIMAL(10,2) is DECIMAL and TIMESTAMP(3) is TIMESTAMP (DuckDB reports
// TIMESTAMP_MS).
func (t *TypeName) BaseTypeName() string {
	if len(t.Name) == 0 {
		return ""
	}
	name := strings.ToUpper(t.Name[len(t.Name)-1])
	if t.TimeZone == "WITH TIME ZONE" {
		name += " WITH TIME ZONE"
	}
	if canonical, ok := typeAliases[name]; ok {
		return canonical
	}
	return name
}

// typeAliases maps every type spelling DuckDB accepts for a builtin to the
// canonical name DuckDB reports for it: every duckdb_types() name that
// isn't already canonical, plus the SQL-standard multi-word spellings the
// grammar has its own rules for (CharacterType, DoubleType, BitType).
// TestTypeAliasesMatchDuckDB checks it against a real duckdb.
var typeAliases = map[string]string{
	"CHAR": "VARCHAR", "BPCHAR": "VARCHAR", "TEXT": "VARCHAR", "STRING": "VARCHAR", "NVARCHAR": "VARCHAR",
	"CHARACTER": "VARCHAR", "CHARACTER VARYING": "VARCHAR", "CHAR VARYING": "VARCHAR",
	"NCHAR": "VARCHAR", "NCHAR VARYING": "VARCHAR",
	"NATIONAL CHARACTER": "VARCHAR", "NATIONAL CHARACTER VARYING": "VARCHAR",
	"NATIONAL CHAR": "VARCHAR", "NATIONAL CHAR VARYING": "VARCHAR",
	"BYTEA": "BLOB", "BINARY": "BLOB", "VARBINARY": "BLOB",
	"BITSTRING": "BIT", "BIT VARYING": "BIT",
	"GUID": "UUID",
	"BOOL": "BOOLEAN", "LOGICAL": "BOOLEAN",
	"INT1": "TINYINT",
	"INT2": "SMALLINT", "INT16": "SMALLINT", "SHORT": "SMALLINT",
	"INT": "INTEGER", "INT4": "INTEGER", "INT32": "INTEGER", "SIGNED": "INTEGER", "INTEGRAL": "INTEGER",
	"INT8": "BIGINT", "INT64": "BIGINT", "LONG": "BIGINT", "OID": "BIGINT",
	"INT128": "HUGEINT",
	"UINT8":  "UTINYINT", "UINT16": "USMALLINT", "UINT32": "UINTEGER", "UINT64": "UBIGINT", "UINT128": "UHUGEINT",
	"REAL": "FLOAT", "FLOAT4": "FLOAT",
	"FLOAT8": "DOUBLE", "DOUBLE PRECISION": "DOUBLE",
	"NUMERIC": "DECIMAL", "DEC": "DECIMAL",
	"VARINT":   "BIGNUM",
	"DATETIME": "TIMESTAMP", "TIMESTAMP_US": "TIMESTAMP",
	"TIMESTAMPTZ": "TIMESTAMP WITH TIME ZONE", "TIMETZ": "TIME WITH TIME ZONE",
	"ROW": "STRUCT",
}
