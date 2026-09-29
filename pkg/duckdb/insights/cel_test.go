package insights

import (
	"context"
	"testing"

	sq "github.com/doug-martin/goqu/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCELEventTableFilters(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		cel      []string
		expected []sq.Expression
	}{
		{
			name: "event.id equals targets the column directly",
			cel:  []string{`event.id == "abc"`},
			expected: []sq.Expression{
				sq.L("event_id").Eq("abc"),
			},
		},
		{
			name: "event.name not equals targets the column directly",
			cel:  []string{`event.name != "test/hello"`},
			expected: []sq.Expression{
				sq.L("event_name").Neq("test/hello"),
			},
		},
		{
			name: "event.v equals targets the column directly",
			cel:  []string{`event.v == "2024-01-01"`},
			expected: []sq.Expression{
				sq.L("event_v").Eq("2024-01-01"),
			},
		},
		{
			name: "event.ts casts via epoch_ms",
			cel:  []string{`event.ts > 1727291508963`},
			expected: []sq.Expression{
				sq.L("epoch_ms(event_ts)").Gt(int64(1727291508963)),
			},
		},
		{
			name: "event.data targets the event_data JSON column directly",
			cel:  []string{`event.data.foo == "bar"`},
			expected: []sq.Expression{
				sq.L("(event_data::JSON->>?)", "$.foo").Eq("bar"),
			},
		},
		{
			name: "event.data nested path",
			cel:  []string{`event.data.nested.value == "x"`},
			expected: []sq.Expression{
				sq.L("(event_data::JSON->>?)", "$.nested.value").Eq("x"),
			},
		},
		{
			name: "event.data null",
			cel:  []string{`event.data.n == null`},
			expected: []sq.Expression{
				sq.L("(event_data::JSON->?) = 'null'::JSON", "$.n"),
			},
		},
		{
			name: "AND combines both predicates",
			cel:  []string{`event.name == "x" && event.data.foo == "bar"`},
			expected: []sq.Expression{
				sq.And(
					sq.L("event_name").Eq("x"),
					sq.L("(event_data::JSON->>?)", "$.foo").Eq("bar"),
				),
			},
		},
		{
			name: "event.id rejects a non-string literal",
			cel:  []string{`event.id == 123`},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			filters, err := CELEventTableFilters(ctx, test.cel)
			if test.expected == nil {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.ElementsMatch(t, test.expected, filters)
		})
	}
}

func TestCELEventFilters(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		cel      []string
		expected []sq.Expression
	}{
		{
			name: "event.id equals",
			cel:  []string{`event.id == "abc"`},
			expected: []sq.Expression{
				sq.L("(x::JSON->>?)", "$.id").Eq("abc"),
			},
		},
		{
			name: "event.name not equals",
			cel:  []string{`event.name != "test/hello"`},
			expected: []sq.Expression{
				sq.L("(x::JSON->>?)", "$.name").Neq("test/hello"),
			},
		},
		{
			name: "event.v equals",
			cel:  []string{`event.v == "2024-01-01"`},
			expected: []sq.Expression{
				sq.L("(x::JSON->>?)", "$.v").Eq("2024-01-01"),
			},
		},
		{
			name: "event.ts greater than",
			cel:  []string{`event.ts > 1727291508963`},
			expected: []sq.Expression{
				sq.L("CAST((x::JSON->>?) AS DOUBLE)", "$.ts").Gt(int64(1727291508963)),
			},
		},
		{
			name: "event.data string field",
			cel:  []string{`event.data.foo == "bar"`},
			expected: []sq.Expression{
				sq.L("(x::JSON->>?)", "$.data.foo").Eq("bar"),
			},
		},
		{
			name: "event.data boolean true",
			cel:  []string{`event.data.b == true`},
			expected: []sq.Expression{
				sq.L("(x::JSON->>?)", "$.data.b").Eq("true"),
			},
		},
		{
			name: "event.data null",
			cel:  []string{`event.data.n == null`},
			expected: []sq.Expression{
				sq.L("(x::JSON->?) = 'null'::JSON", "$.data.n"),
			},
		},
		{
			name: "event.data not null",
			cel:  []string{`event.data.n != null`},
			expected: []sq.Expression{
				sq.L("(x::JSON->?) != 'null'::JSON", "$.data.n"),
			},
		},
		{
			name: "event.data nested path",
			cel:  []string{`event.data.nested.value == "x"`},
			expected: []sq.Expression{
				sq.L("(x::JSON->>?)", "$.data.nested.value").Eq("x"),
			},
		},
		{
			name: "AND combines both predicates",
			cel:  []string{`event.name == "x" && event.data.foo == "bar"`},
			expected: []sq.Expression{
				sq.And(
					sq.L("(x::JSON->>?)", "$.name").Eq("x"),
					sq.L("(x::JSON->>?)", "$.data.foo").Eq("bar"),
				),
			},
		},
		{
			name:     "output.*/error.* predicates are excluded",
			cel:      []string{`output.ok == true`},
			expected: []sq.Expression{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			filters, err := CELEventFilters(ctx, test.cel)
			require.NoError(t, err)
			assert.ElementsMatch(t, test.expected, filters)
		})
	}
}

func TestCELOutputFilters(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		cel      []string
		expected []sq.Expression
	}{
		{
			name: "output boolean true targets the data envelope",
			cel:  []string{`output.success == true`},
			expected: []sq.Expression{
				sq.L("(output::JSON->>?)", "$.data.success").Eq("true"),
			},
		},
		{
			name: "output null",
			cel:  []string{`output.result == null`},
			expected: []sq.Expression{
				sq.L("(output::JSON->?) = 'null'::JSON", "$.data.result"),
			},
		},
		{
			name: "output numeric comparison",
			cel:  []string{`output.count >= 3`},
			expected: []sq.Expression{
				sq.L("CAST((output::JSON->>?) AS DOUBLE)", "$.data.count").Gte(int64(3)),
			},
		},
		{
			name: "output nested path",
			cel:  []string{`output.nested.value == "x"`},
			expected: []sq.Expression{
				sq.L("(output::JSON->>?)", "$.data.nested.value").Eq("x"),
			},
		},
		{
			name: "error targets the error envelope",
			cel:  []string{`error.message == "boom"`},
			expected: []sq.Expression{
				sq.L("(output::JSON->>?)", "$.error.message").Eq("boom"),
			},
		},
		{
			name:     "event.* predicates are excluded",
			cel:      []string{`event.name == "x"`},
			expected: []sq.Expression{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			filters, err := CELOutputFilters(ctx, test.cel)
			require.NoError(t, err)
			assert.ElementsMatch(t, test.expected, filters)
		})
	}
}

func TestCELEventAndOutputFiltersSplitAMixedExpression(t *testing.T) {
	ctx := context.Background()
	cel := []string{`event.name == "x" && output.ok == true`}

	eventFilters, err := CELEventFilters(ctx, cel)
	require.NoError(t, err)
	assert.Equal(t, []sq.Expression{sq.L("(x::JSON->>?)", "$.name").Eq("x")}, eventFilters)

	outputFilters, err := CELOutputFilters(ctx, cel)
	require.NoError(t, err)
	assert.Equal(t, []sq.Expression{sq.L("(output::JSON->>?)", "$.data.ok").Eq("true")}, outputFilters)
}

func TestCELEventAndOutputFiltersKeepAMixedOrWhole(t *testing.T) {
	// An || across event.* and output.* can't be split into two ANDed
	// halves: nothing is pushed down pre-collapse, and the output half
	// evaluates the whole OR, matching event.* against inputs itself.
	ctx := context.Background()
	cel := []string{`event.name == "x" || output.ok == true`}

	eventFilters, err := CELEventFilters(ctx, cel)
	require.NoError(t, err)
	assert.Empty(t, eventFilters)

	outputFilters, err := CELOutputFilters(ctx, cel)
	require.NoError(t, err)
	sqlText, args, err := RenderWhereSQL(outputFilters)
	require.NoError(t, err)
	assert.Equal(t, `(len(list_filter(json_transform(inputs, '["JSON"]'), lambda x: ((x::JSON->>?) = ?))) > 0 OR ((output::JSON->>?) = ?))`, sqlText)
	assert.Equal(t, []any{"$.name", "x", "$.data.ok", "true"}, args)
}

func TestCELEventTableFiltersNestedOrGroup(t *testing.T) {
	// The (a && (b || c)) branch must stay one ANDed unit inside the
	// outer OR, not be flattened into it as a || (b || c).
	ctx := context.Background()
	filters, err := CELEventTableFilters(ctx, []string{`(event.name == "x" && (event.data.a == 1 || event.data.a == 3)) || event.name == "z"`})
	require.NoError(t, err)
	sqlText, args, err := RenderWhereSQL(filters)
	require.NoError(t, err)
	assert.Equal(t, "(((event_name = ?) AND ((CAST((event_data::JSON->>?) AS DOUBLE) = ?) OR (CAST((event_data::JSON->>?) AS DOUBLE) = ?))) OR (event_name = ?))", sqlText)
	assert.Equal(t, []any{"x", "$.a", int64(1), "$.a", int64(3), "z"}, args)
}

func TestCELEventTableFiltersOrWithUnconvertibleBranchIsUnconstrained(t *testing.T) {
	// output.* has no meaning for an events-table search; dropping just
	// that branch would narrow the OR to event.name == "x".
	filters, err := CELEventTableFilters(context.Background(), []string{`event.name == "x" || output.ok == true`})
	require.NoError(t, err)
	assert.Empty(t, filters)
}

func TestCELJSONPathKeysAreBoundNotSpliced(t *testing.T) {
	// expr lifts string literals (a bracketed key included) out of an
	// expression unless it already mentions "vars.", so the second
	// predicate here is what lets the raw key reach handleJSONFilter.
	ctx := context.Background()
	cel := []string{`event.data.foo["x') OR 1=1 OR ('"] == "y" && event.data.vars.q == "z"`}

	filters, err := CELEventTableFilters(ctx, cel)
	require.NoError(t, err)
	sqlText, args, err := RenderWhereSQL(filters)
	require.NoError(t, err)
	assert.Equal(t, "(((event_data::JSON->>?) = ?) AND ((event_data::JSON->>?) = ?))", sqlText)
	assert.Equal(t, []any{`$.foo."x') OR 1=1 OR ('"`, "y", "$.vars.q", "z"}, args)
}

func TestCELJSONPath(t *testing.T) {
	tests := map[string]string{
		"":                "$",
		"id":              "$.id",
		"data.foo.bar":    "$.data.foo.bar",
		"data.items[0]":   "$.data.items[0]",
		"data.items[0].x": "$.data.items[0].x",
		"data.foo[a'b]":   `$.data.foo."a'b"`,
		"data.foo[a.b]":   `$.data.foo."a.b"`,
		`data.foo[a"b\c]`: `$.data.foo."a\"b\\c"`,
		"data.foo[a b]":   `$.data.foo."a b"`,
	}
	for in, want := range tests {
		t.Run(in, func(t *testing.T) {
			assert.Equal(t, want, celJSONPath(in))
		})
	}
}

func TestRenderWhereSQL(t *testing.T) {
	tests := []struct {
		name     string
		filters  []sq.Expression
		wantSQL  string
		wantArgs []any
	}{
		{
			name:     "no filters renders nothing",
			filters:  nil,
			wantSQL:  "",
			wantArgs: nil,
		},
		{
			name: "single filter",
			filters: []sq.Expression{
				sq.L("(output::JSON->>'$.data.ok')").Eq("true"),
			},
			wantSQL:  "((output::JSON->>'$.data.ok') = ?)",
			wantArgs: []any{"true"},
		},
		{
			name: "multiple filters AND together with positional args in order",
			filters: []sq.Expression{
				sq.L("(x::JSON->>'$.name')").Eq("test/hello"),
				sq.L("CAST((x::JSON->>'$.ts') AS DOUBLE)").Gt(int64(123)),
			},
			wantSQL:  "(((x::JSON->>'$.name') = ?) AND (CAST((x::JSON->>'$.ts') AS DOUBLE) > ?))",
			wantArgs: []any{"test/hello", int64(123)},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sqlText, args, err := RenderWhereSQL(test.filters)
			require.NoError(t, err)
			assert.Equal(t, test.wantSQL, sqlText)
			assert.Equal(t, test.wantArgs, args)
		})
	}
}
