package parser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// typeSpellings is a broad table of type spellings the grammar accepts and
// DuckDB's parser does too, each with the canonical form Write prints.
var typeSpellings = []struct{ in, want string }{
	{"INTEGER", "INTEGER"},
	{"int", "INT"},
	{"int4", "int4"},
	{"BigInt", "BIGINT"},
	{"json", "json"},
	{"uuid", "uuid"},
	{"double", "double"},
	{"double   precision", "DOUBLE PRECISION"},
	{"REAL", "REAL"},
	{"BOOLEAN", "BOOLEAN"},
	{"varchar", "VARCHAR"},
	{"VARCHAR(10)", "VARCHAR(10)"},
	{"VARCHAR ( 10 )", "VARCHAR(10)"},
	{"VARCHAR()", "VARCHAR"},
	{"character", "CHARACTER"},
	{"character varying", "CHARACTER VARYING"},
	{"CHARACTER VARYING(3)", "CHARACTER VARYING(3)"},
	{"char varying", "CHAR VARYING"},
	{"national character varying", "NATIONAL CHARACTER VARYING"},
	{"national char(2)", "NATIONAL CHAR(2)"},
	{"nchar varying", "NCHAR VARYING"},
	{"text", "text"},
	{"string", "string"},
	{"bpchar", "bpchar"},
	{"DECIMAL", "DECIMAL"},
	{"decimal(10,2)", "DECIMAL(10, 2)"},
	{"DEC(5)", "DEC(5)"},
	{"numeric (18, 3)", "NUMERIC(18, 3)"},
	{"FLOAT", "FLOAT"},
	{"float(24)", "FLOAT(24)"},
	{"date", "date"},
	{"TIME", "TIME"},
	{"time with time zone", "TIME WITH TIME ZONE"},
	{"TIME WITHOUT TIME ZONE", "TIME WITHOUT TIME ZONE"},
	{"TIMESTAMP", "TIMESTAMP"},
	{"timestamp(3)", "TIMESTAMP(3)"},
	{"timestamp  with  time zone", "TIMESTAMP WITH TIME ZONE"},
	{"TIMESTAMP(3) WITHOUT TIME ZONE", "TIMESTAMP(3) WITHOUT TIME ZONE"},
	{"timestamptz", "timestamptz"},
	{"TIMESTAMP_NS", "TIMESTAMP_NS"},
	{"interval", "INTERVAL"},
	{"interval year", "INTERVAL YEAR"},
	{"INTERVAL YEARS TO MONTHS", "INTERVAL YEARS TO MONTHS"},
	{"interval day to second", "INTERVAL DAY TO SECOND"},
	{"INTERVAL MILLENNIA", "INTERVAL MILLENNIA"},
	{"BIT", "BIT"},
	{"bit varying(3)", "BIT VARYING(3)"},
	{"blob", "blob"},
	{"VARIANT", "VARIANT"},
	{"GEOMETRY", "GEOMETRY"},
	{"geometry('OGC:CRS84')", "GEOMETRY('OGC:CRS84')"},
	{"ENUM('a', 'b')", "ENUM('a', 'b')"},
	{"enum ( 'it''s' )", "enum('it''s')"},
	{"INT[]", "INT[]"},
	{"INT [ ]", "INT[]"},
	{"INT[3]", "INT[3]"},
	{"INT [ 3 ] [ ]", "INT[3][]"},
	{"INT[][3]", "INT[][3]"},
	{"int array", "INT[]"},
	{"INT ARRAY[3]", "INT[3]"},
	{"INT ARRAY ARRAY", "INT[][]"},
	{"INT[3] ARRAY", "INT[3][]"},
	{"SETOF INT", "SETOF INT"},
	{"SETOF INT[] []", "SETOF INT[][]"},
	{"STRUCT", "STRUCT"},
	{"ROW", "ROW"},
	{"struct (a int)", "STRUCT(a INT)"},
	{"STRUCT(a INT,)", "STRUCT(a INT)"},
	{"ROW(a INT, b VARCHAR)", "ROW(a INT, b VARCHAR)"},
	{`STRUCT("a b" INT, "select" VARCHAR, "Key" json)`, `STRUCT("a b" INT, "select" VARCHAR, Key json)`},
	{"STRUCT(a INT[], b MAP(VARCHAR, STRUCT(c INT)))[]", "STRUCT(a INT[], b MAP(VARCHAR, STRUCT(c INT)))[]"},
	{"STRUCT(key VARCHAR, id VARCHAR)[]", "STRUCT(key VARCHAR, id VARCHAR)[]"},
	{"STRUCT(a STRUCT(b STRUCT(c DECIMAL(10, 2)[2])[])[3])", "STRUCT(a STRUCT(b STRUCT(c DECIMAL(10, 2)[2])[])[3])"},
	{"map(varchar, int)", "MAP(VARCHAR, INT)"},
	{"MAP(VARCHAR, INT[])[]", "MAP(VARCHAR, INT[])[]"},
	{"MAP(INT, MAP(INT, STRUCT(x INT)))", "MAP(INT, MAP(INT, STRUCT(x INT)))"},
	{"UNION(a INT, b VARCHAR)", "UNION(a INT, b VARCHAR)"},
	{"union (num INT, str VARCHAR)[]", "UNION(num INT, str VARCHAR)[]"},
	{"TUPLE(INT, VARCHAR)", "TUPLE(INT, VARCHAR)"},
	{"main.integer", "main.\"integer\""},
	{"memory.main.my_type", "memory.main.my_type"},
	{`"int"`, `"int"`},
	{`"struct"`, `"struct"`},
	{`"varchar"`, `"varchar"`},
	{`"My Type"`, `"My Type"`},
}

func TestWriteTypeNameRoundTrips(t *testing.T) {
	for _, c := range typeSpellings {
		t.Run(c.in, func(t *testing.T) {
			typ, err := ParseType(c.in)
			require.NoError(t, err)
			got := String(typ)
			require.Equal(t, c.want, got)

			reparsed, err := ParseType(got)
			require.NoError(t, err, "reparsing %q", got)
			require.Equal(t, Dump(typ), Dump(reparsed), "%q reparses differently", got)
			require.Equal(t, got, String(reparsed))

			// The same type as a cast and as a typed literal in a full
			// statement round-trips too.
			for _, sql := range []string{"SELECT x::" + c.in + " FROM t", "SELECT CAST(x AS " + c.in + ")", "SELECT TRY_CAST(x AS " + c.in + ")"} {
				stmt, err := ParseString(sql)
				require.NoError(t, err, sql)
				out := String(stmt)
				require.Contains(t, out, c.want)
				reparsedStmt, err := ParseString(out)
				require.NoError(t, err, out)
				require.Equal(t, Dump(stmt), Dump(reparsedStmt), "%q reparses differently", out)
			}
		})
	}
}

func TestTypeNameStructure(t *testing.T) {
	typ, err := ParseType(`STRUCT(a INT[3], "b c" MAP(VARCHAR, DECIMAL(10, 2)))[]`)
	require.NoError(t, err)
	require.Equal(t, []string{"STRUCT"}, typ.Name)
	require.True(t, typ.Keyword)
	require.Len(t, typ.Arrays, 1)
	require.Nil(t, typ.Arrays[0].Size)
	require.Len(t, typ.Fields, 2)

	a := typ.Fields[0]
	require.Equal(t, "a", a.Name)
	require.Equal(t, []string{"INT"}, a.Type.Name)
	require.Len(t, a.Type.Arrays, 1)
	require.Equal(t, &Literal{baseExpr: a.Type.Arrays[0].Size.(*Literal).baseExpr, Kind: LitNumber, Text: "3"}, a.Type.Arrays[0].Size)

	b := typ.Fields[1]
	require.Equal(t, "b c", b.Name)
	require.Equal(t, []string{"MAP"}, b.Type.Name)
	require.Len(t, b.Type.Params, 2)
	require.Equal(t, []string{"VARCHAR"}, b.Type.Params[0].Name)
	dec := b.Type.Params[1]
	require.Equal(t, []string{"DECIMAL"}, dec.Name)
	require.Len(t, dec.Modifiers, 2)
	require.Equal(t, "10", dec.Modifiers[0].(*Literal).Text)
	require.Equal(t, "2", dec.Modifiers[1].(*Literal).Text)

	require.Equal(t, `TypeName(STRUCT[])
  TypeField(a)
    TypeName(INT[3])
      Literal(1,3)
  TypeField(b c)
    TypeName(MAP)
      TypeName(VARCHAR)
      TypeName(DECIMAL)
        Literal(1,10)
        Literal(1,2)
`, Dump(typ))

	for _, c := range []struct {
		in   string
		want *TypeName
	}{
		{"main.my_enum", &TypeName{Name: []string{"main", "my_enum"}}},
		{`"My Type"`, &TypeName{Name: []string{"My Type"}}},
		{"double precision", &TypeName{Name: []string{"DOUBLE PRECISION"}, Keyword: true}},
		{"timestamp with time zone", &TypeName{Name: []string{"TIMESTAMP"}, Keyword: true, TimeZone: "WITH TIME ZONE"}},
		{"interval day to second", &TypeName{Name: []string{"INTERVAL"}, Keyword: true, Interval: "DAY TO SECOND"}},
		{"bit varying", &TypeName{Name: []string{"BIT VARYING"}, Keyword: true}},
		{"setof int", &TypeName{Name: []string{"INT"}, Keyword: true, Setof: true}},
		{"STRUCT", &TypeName{Name: []string{"STRUCT"}, Keyword: true}},
	} {
		t.Run(c.in, func(t *testing.T) {
			typ, err := ParseType(c.in)
			require.NoError(t, err)
			typ.baseExpr = baseExpr{}
			require.Equal(t, c.want, typ)
		})
	}

	enum, err := ParseType("ENUM('a', 'b')")
	require.NoError(t, err)
	require.Equal(t, []string{"ENUM"}, enum.Name)
	require.False(t, enum.Keyword)
	require.Equal(t, LitString, enum.Modifiers[0].(*Literal).Kind)
	require.Equal(t, "b", enum.Modifiers[1].(*Literal).Text)
}

// TestTypeNameModifiersAreWalked checks every Expr a type carries
// (modifiers and array sizes, at any nesting depth) is visible to Walk.
func TestTypeNameModifiersAreWalked(t *testing.T) {
	stmt, err := ParseString("SELECT x::STRUCT(a VARCHAR(7), b MAP(INT, DECIMAL(8, 9)[10]))[11], DATE(12) '2024-01-01' FROM t")
	require.NoError(t, err)
	var seen []string
	Walk(literalCollector{&seen}, stmt)
	require.Equal(t, []string{"7", "8", "9", "10", "11", "12"}, seen)
}

type literalCollector struct{ seen *[]string }

func (c literalCollector) Visit(n Node) Visitor {
	if lit, ok := n.(*Literal); ok && lit.Kind == LitNumber {
		*c.seen = append(*c.seen, lit.Text)
	}
	return c
}

func TestParseTypeErrors(t *testing.T) {
	for _, c := range []struct{ in, msg string }{
		{"INT garbage", "unexpected trailing input"},
		{"", "failed to parse Type"},
		{"VARCHAR((SELECT 1))", "A type modifier other than a number or string literal is not supported"},
		{"INT[f(1)]", "A type modifier other than a number or string literal is not supported"},
		{"MAP(INT, VARCHAR(x))", "A type modifier other than a number or string literal is not supported"},
		{"LIST(INT)", "A type modifier other than a number or string literal is not supported"},
	} {
		_, err := ParseType(c.in)
		var perr *ParseError
		require.ErrorAs(t, err, &perr, c.in)
		require.Equal(t, c.msg, perr.Message, c.in)
	}
}

func TestBaseTypeName(t *testing.T) {
	for in, want := range map[string]string{
		"INT":                         "INTEGER",
		"int4":                        "INTEGER",
		"integral":                    "INTEGER",
		"character varying(10)":       "VARCHAR",
		"NATIONAL CHAR VARYING":       "VARCHAR",
		"text":                        "VARCHAR",
		"double precision":            "DOUBLE",
		"float8":                      "DOUBLE",
		"real":                        "FLOAT",
		"numeric(10,2)":               "DECIMAL",
		"timestamp with time zone":    "TIMESTAMP WITH TIME ZONE",
		"timestamp without time zone": "TIMESTAMP",
		"timestamptz":                 "TIMESTAMP WITH TIME ZONE",
		"time with time zone":         "TIME WITH TIME ZONE",
		"interval year to month":      "INTERVAL",
		"ROW(a INT)":                  "STRUCT",
		"struct(a int)[]":             "STRUCT",
		"setof int":                   "INTEGER",
		"main.int8":                   "BIGINT",
		"json":                        "JSON",
		"my_type":                     "MY_TYPE",
		"bit varying":                 "BIT",
	} {
		typ, err := ParseType(in)
		require.NoError(t, err, in)
		require.Equal(t, want, typ.BaseTypeName(), in)
	}
}

// runDuckDB runs each statement through a duckdb CLI on PATH (skipping the
// test without one) and returns each successful statement's rows, keyed by
// its "i" column. A statement that errors has no entry.
func runDuckDB(t *testing.T, stmts []string) map[int]map[string]any {
	t.Helper()
	bin, err := exec.LookPath("duckdb")
	if err != nil {
		t.Skip("duckdb binary not found on PATH; skipping")
	}
	cmd := exec.CommandContext(t.Context(), bin, "-jsonlines")
	cmd.Stdin = strings.NewReader(strings.Join(stmts, ";\n") + ";\n")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	_ = cmd.Run() // a failing statement makes the exit status nonzero
	out := map[int]map[string]any{}
	for _, line := range strings.Split(stdout.String(), "\n") {
		if line == "" {
			continue
		}
		var row map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &row), line)
		out[int(row["i"].(float64))] = row
	}
	return out
}

func sqlString(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

// stripQueryLocations drops json_serialize_sql's source offsets, which
// differ between two spellings of the same type.
func stripQueryLocations(v any) any {
	switch x := v.(type) {
	case map[string]any:
		for k := range x {
			if strings.HasPrefix(k, "query_location") {
				delete(x, k)
				continue
			}
			x[k] = stripQueryLocations(x[k])
		}
	case []any:
		for i := range x {
			x[i] = stripQueryLocations(x[i])
		}
	}
	return v
}

// TestWriteTypeNameMatchesDuckDB checks DuckDB's own parser gives every
// spelling in typeSpellings and Write's canonical form of it the identical
// type (json_serialize_sql's unbound cast_type, minus source offsets).
func TestWriteTypeNameMatchesDuckDB(t *testing.T) {
	var stmts []string
	for i, c := range typeSpellings {
		stmts = append(stmts, fmt.Sprintf("SELECT %d AS i, json_serialize_sql(%s) AS a, json_serialize_sql(%s) AS b",
			i, sqlString("SELECT NULL::"+c.in), sqlString("SELECT NULL::"+c.want)))
	}
	rows := runDuckDB(t, stmts)
	for i, c := range typeSpellings {
		t.Run(c.in, func(t *testing.T) {
			row, ok := rows[i]
			require.True(t, ok, "duckdb failed on %q", c.in)
			a, b := row["a"].(map[string]any), row["b"].(map[string]any)
			require.Equal(t, false, a["error"], "duckdb rejects %q: %v", c.in, a)
			require.Equal(t, stripQueryLocations(a), stripQueryLocations(b), "%q and %q", c.in, c.want)
		})
	}
}

// typeSpellingsWithoutCatalogEntry are type spellings DuckDB's parser
// accepts that duckdb_types() doesn't list.
var typeSpellingsWithoutCatalogEntry = []string{
	"CHARACTER", "CHARACTER VARYING", "CHAR VARYING", "NCHAR VARYING",
	"NATIONAL CHARACTER", "NATIONAL CHAR", "NATIONAL CHARACTER VARYING", "NATIONAL CHAR VARYING",
	"DOUBLE PRECISION", "BIT VARYING", "TIME WITHOUT TIME ZONE", "TIMESTAMP WITHOUT TIME ZONE",
	"INTERVAL YEAR", "INTERVAL DAY TO SECOND", "ROW(a INT)", "SETOF INT",
}

// TestTypeAliasesMatchDuckDB checks BaseTypeName against the name a real
// duckdb reports (typeof(NULL::<name>), minus any parameter list) for every
// name duckdb_types() lists plus typeSpellingsWithoutCatalogEntry. Names
// that need type parameters (LIST, MAP, ENUM, ...) error in DuckDB and are
// skipped.
func TestTypeAliasesMatchDuckDB(t *testing.T) {
	names := runDuckDB(t, []string{"SELECT 0 AS i, list(DISTINCT type_name ORDER BY type_name) AS names FROM duckdb_types()"})
	var all []string
	for _, n := range names[0]["names"].([]any) {
		all = append(all, n.(string))
	}
	require.NotEmpty(t, all)
	all = append(all, typeSpellingsWithoutCatalogEntry...)

	var stmts []string
	for i, name := range all {
		if _, err := ParseType(name); err != nil {
			// A name the grammar can't spell bare (`null`, `type`).
			t.Logf("skipping %q: %v", name, err)
			continue
		}
		stmts = append(stmts, fmt.Sprintf("SELECT %d AS i, typeof(NULL::%s) AS t", i, name))
	}
	rows := runDuckDB(t, stmts)
	checked := 0
	for i, name := range all {
		row, ok := rows[i]
		if !ok {
			continue
		}
		typ, err := ParseType(name)
		require.NoError(t, err)
		want := row["t"].(string)
		if j := strings.IndexByte(want, '('); j >= 0 {
			want = want[:j]
		}
		require.Equal(t, want, typ.BaseTypeName(), "%q is %q in duckdb", name, row["t"])
		checked++
	}
	require.Greater(t, checked, 60)
}
