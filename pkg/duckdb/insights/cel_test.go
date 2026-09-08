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
				sq.L("(event_data->>'$.foo')").Eq("bar"),
			},
		},
		{
			name: "event.data nested path",
			cel:  []string{`event.data.nested.value == "x"`},
			expected: []sq.Expression{
				sq.L("(event_data->>'$.nested.value')").Eq("x"),
			},
		},
		{
			name: "event.data null",
			cel:  []string{`event.data.n == null`},
			expected: []sq.Expression{
				sq.L("(event_data->'$.n') = 'null'::JSON"),
			},
		},
		{
			name: "AND combines both predicates",
			cel:  []string{`event.name == "x" && event.data.foo == "bar"`},
			expected: []sq.Expression{
				sq.And(
					sq.L("event_name").Eq("x"),
					sq.L("(event_data->>'$.foo')").Eq("bar"),
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
				sq.L("(x->>'$.id')").Eq("abc"),
			},
		},
		{
			name: "event.name not equals",
			cel:  []string{`event.name != "test/hello"`},
			expected: []sq.Expression{
				sq.L("(x->>'$.name')").Neq("test/hello"),
			},
		},
		{
			name: "event.v equals",
			cel:  []string{`event.v == "2024-01-01"`},
			expected: []sq.Expression{
				sq.L("(x->>'$.v')").Eq("2024-01-01"),
			},
		},
		{
			name: "event.ts greater than",
			cel:  []string{`event.ts > 1727291508963`},
			expected: []sq.Expression{
				sq.L("CAST((x->>'$.ts') AS DOUBLE)").Gt(int64(1727291508963)),
			},
		},
		{
			name: "event.data string field",
			cel:  []string{`event.data.foo == "bar"`},
			expected: []sq.Expression{
				sq.L("(x->>'$.data.foo')").Eq("bar"),
			},
		},
		{
			name: "event.data boolean true",
			cel:  []string{`event.data.b == true`},
			expected: []sq.Expression{
				sq.L("(x->>'$.data.b')").Eq("true"),
			},
		},
		{
			name: "event.data null",
			cel:  []string{`event.data.n == null`},
			expected: []sq.Expression{
				sq.L("(x->'$.data.n') = 'null'::JSON"),
			},
		},
		{
			name: "event.data not null",
			cel:  []string{`event.data.n != null`},
			expected: []sq.Expression{
				sq.L("(x->'$.data.n') != 'null'::JSON"),
			},
		},
		{
			name: "event.data nested path",
			cel:  []string{`event.data.nested.value == "x"`},
			expected: []sq.Expression{
				sq.L("(x->>'$.data.nested.value')").Eq("x"),
			},
		},
		{
			name: "AND combines both predicates",
			cel:  []string{`event.name == "x" && event.data.foo == "bar"`},
			expected: []sq.Expression{
				sq.And(
					sq.L("(x->>'$.name')").Eq("x"),
					sq.L("(x->>'$.data.foo')").Eq("bar"),
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
				sq.L("(output->>'$.data.success')").Eq("true"),
			},
		},
		{
			name: "output null",
			cel:  []string{`output.result == null`},
			expected: []sq.Expression{
				sq.L("(output->'$.data.result') = 'null'::JSON"),
			},
		},
		{
			name: "output numeric comparison",
			cel:  []string{`output.count >= 3`},
			expected: []sq.Expression{
				sq.L("CAST((output->>'$.data.count') AS DOUBLE)").Gte(int64(3)),
			},
		},
		{
			name: "output nested path",
			cel:  []string{`output.nested.value == "x"`},
			expected: []sq.Expression{
				sq.L("(output->>'$.data.nested.value')").Eq("x"),
			},
		},
		{
			name: "error targets the error envelope",
			cel:  []string{`error.message == "boom"`},
			expected: []sq.Expression{
				sq.L("(output->>'$.error.message')").Eq("boom"),
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
	assert.Equal(t, []sq.Expression{sq.L("(x->>'$.name')").Eq("x")}, eventFilters)

	outputFilters, err := CELOutputFilters(ctx, cel)
	require.NoError(t, err)
	assert.Equal(t, []sq.Expression{sq.L("(output->>'$.data.ok')").Eq("true")}, outputFilters)
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
				sq.L("(output->>'$.data.ok')").Eq("true"),
			},
			wantSQL:  "((output->>'$.data.ok') = ?)",
			wantArgs: []any{"true"},
		},
		{
			name: "multiple filters AND together with positional args in order",
			filters: []sq.Expression{
				sq.L("(x->>'$.name')").Eq("test/hello"),
				sq.L("CAST((x->>'$.ts') AS DOUBLE)").Gt(int64(123)),
			},
			wantSQL:  "(((x->>'$.name') = ?) AND (CAST((x->>'$.ts') AS DOUBLE) > ?))",
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
