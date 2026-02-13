package run

import (
	"context"
	"testing"

	sq "github.com/doug-martin/goqu/v9"
	"github.com/inngest/inngest/pkg/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateExpressionHandler(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name   string
		cel    []string
		errStr string
	}{
		{
			name: "valid CEL expr have no errors",
			cel:  []string{`event.name == "test/hello"`, `output.success == true`},
		},
		{
			name:   "invalid AND",
			cel:    []string{`event.name == "test/hello" and event.ts > 1727291508963`},
			errStr: "mismatched input 'and'",
		},
		{
			name:   "invalid Equal",
			cel:    []string{`event.name === "test/hello"`},
			errStr: "token recognition error at: '= '",
		},
		{
			name:   "macros",
			cel:    []string{`event.data.num.filter(x, x > 5)`},
			errStr: "macros are currently not supported",
		},
		{
			name:   "macros (select)",
			cel:    []string{`has(event.name)`},
			errStr: "macros are currently not supported",
		},
		{
			name:   "invalid syntax",
			cel:    []string{`event.name.startWith("hello")`},
			errStr: "invalid syntax detected",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewExpressionHandler(ctx, WithExpressionHandlerExpressions(test.cel))

			if test.errStr == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, test.errStr)
			}
		})
	}
}

func TestMatchEventExpressions(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		expr     string
		input    event.Event
		expected bool
	}{
		{
			name: "should match",
			expr: `event.name == 'test/hello' && event.data.foo == "bar"`,
			input: event.Event{
				Name: "test/hello",
				Data: map[string]any{"foo": "bar"},
			},
			expected: true,
		},
		{
			name: "should not match",
			expr: `event.data.hello == "world"`,
			input: event.Event{
				Name: "test/hello",
				Data: map[string]any{"foo": "bar"},
			},
			expected: false,
		},
		{
			name: "should match with output as OR",
			expr: `event.data.hello == "world" || output.hello == "world"`,
			input: event.Event{
				Name: "test/hello",
				Data: map[string]any{"hello": "world"},
			},
			expected: true,
		},
		{
			name: "should not match with output as AND",
			expr: `event.data.hello == "world" && output.hello == "world"`,
			input: event.Event{
				Name: "test/hello",
				Data: map[string]any{"hello": "world"},
			},
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler, err := NewExpressionHandler(ctx,
				WithExpressionHandlerExpressions([]string{test.expr}),
			)
			require.NoError(t, err)

			ok, err := handler.MatchEventExpressions(ctx, test.input)
			require.NoError(t, err)

			assert.Equal(t, test.expected, ok)
		})
	}
}

func TestToSQLFilters(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		cel      []string
		expected []sq.Expression
	}{
		{
			name:     "single CEL query",
			cel:      []string{`event.name == "test/hello"`},
			expected: []sq.Expression{sq.C("event_name").Eq("test/hello")},
		},
		{
			name: "multiple CEL queries results in AND statement",
			cel:  []string{`event.name == 'test/hello'`, `event.ts > 1727291508963`},
			expected: []sq.Expression{
				sq.C("event_name").Eq("test/hello"),
				sq.C("event_ts").Gt(int64(1727291508963)),
			},
		},
		{
			name: "OR queries",
			cel:  []string{`event.name == 'test/hello' || event.name == 'test/yolo'`, `event.ts > 1727291508963`},
			expected: []sq.Expression{
				sq.Or(
					sq.C("event_name").Eq("test/hello"),
					sq.C("event_name").Eq("test/yolo"),
				),
				sq.C("event_ts").Gt(int64(1727291508963)),
			},
		},
		{
			name: "AND queries",
			cel:  []string{`event.name == 'test/hello' && event.name == 'test/yolo'`, `event.ts <= 1727291508963`},
			expected: []sq.Expression{
				sq.And(
					sq.C("event_name").Eq("test/hello"),
					sq.C("event_name").Eq("test/yolo"),
				),
				sq.C("event_ts").Lte(int64(1727291508963)),
			},
		},
		{
			name: "AND with nested OR",
			cel:  []string{`event.name != 'test/hello' && (event.name != 'test/yolo' || event.ts >= 1727291508963)`},
			expected: []sq.Expression{
				sq.C("event_name").Neq("test/hello"),
				sq.Or(
					sq.C("event_name").Neq("test/yolo"),
					sq.C("event_ts").Gte(int64(1727291508963)),
				),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler, err := NewExpressionHandler(ctx, WithExpressionHandlerExpressions(test.cel))
			require.NoError(t, err)

			filters, err := handler.ToSQLFilters(ctx)
			require.NoError(t, err)

			assert.ElementsMatch(t, test.expected, filters)
		})
	}
}

func TestToSQLFiltersWithSQLiteConverter(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		cel      []string
		expected []sq.Expression
	}{
		{
			name: "event.data boolean true",
			cel:  []string{`event.data.b == true`},
			expected: []sq.Expression{
				sq.L("json_extract(NULLIF(events.event_data, ''), '$.b')").Eq(1),
			},
		},
		{
			name: "event.data boolean false",
			cel:  []string{`event.data.b2 == false`},
			expected: []sq.Expression{
				sq.L("json_extract(NULLIF(events.event_data, ''), '$.b2')").Eq(0),
			},
		},
		{
			name: "event.data null",
			cel:  []string{`event.data.n == null`},
			expected: []sq.Expression{
				sq.L("json_type(NULLIF(events.event_data, ''), '$.n')").Eq("null"),
			},
		},
		{
			name: "event.data not null",
			cel:  []string{`event.data.n != null`},
			expected: []sq.Expression{
				sq.L("json_type(NULLIF(events.event_data, ''), '$.n')").Neq("null"),
			},
		},
		{
			name: "output boolean true",
			cel:  []string{`output.success == true`},
			expected: []sq.Expression{
				sq.L("json_extract(json_extract(spans.output, '$.data'), '$.success')").Eq(1),
			},
		},
		{
			name: "output null",
			cel:  []string{`output.result == null`},
			expected: []sq.Expression{
				sq.L("json_type(json_extract(spans.output, '$.data'), '$.result')").Eq("null"),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler, err := NewExpressionHandler(ctx,
				WithExpressionHandlerExpressions(test.cel),
				WithExpressionSQLConverter(SpanEventSQLiteConverter),
			)
			require.NoError(t, err)

			filters, err := handler.ToSQLFilters(ctx)
			require.NoError(t, err)

			assert.ElementsMatch(t, test.expected, filters)
		})
	}
}

func TestToSQLFiltersWithPostgresConverter(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		cel      []string
		expected []sq.Expression
	}{
		{
			name: "event.data boolean true",
			cel:  []string{`event.data.b == true`},
			expected: []sq.Expression{
				sq.L("(NULLIF(events.event_data, '')::jsonb)#>>'{b}'").Eq("true"),
			},
		},
		{
			name: "event.data boolean false",
			cel:  []string{`event.data.b2 == false`},
			expected: []sq.Expression{
				sq.L("(NULLIF(events.event_data, '')::jsonb)#>>'{b2}'").Eq("false"),
			},
		},
		{
			name: "event.data null",
			cel:  []string{`event.data.n == null`},
			expected: []sq.Expression{
				sq.L("jsonb_typeof((NULLIF(events.event_data, '')::jsonb)#>'{n}')").Eq("null"),
			},
		},
		{
			name: "event.data not null",
			cel:  []string{`event.data.n != null`},
			expected: []sq.Expression{
				sq.L("jsonb_typeof((NULLIF(events.event_data, '')::jsonb)#>'{n}')").Neq("null"),
			},
		},
		{
			name: "output boolean true",
			cel:  []string{`output.success == true`},
			expected: []sq.Expression{
				sq.L("((spans.output#>>'{}')::jsonb->'data')#>>'{success}'").Eq("true"),
			},
		},
		{
			name: "output null",
			cel:  []string{`output.result == null`},
			expected: []sq.Expression{
				sq.L("jsonb_typeof(((spans.output#>>'{}')::jsonb->'data')#>'{result}')").Eq("null"),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler, err := NewExpressionHandler(ctx,
				WithExpressionHandlerExpressions(test.cel),
				WithExpressionSQLConverter(SpanEventPostgresConverter),
			)
			require.NoError(t, err)

			filters, err := handler.ToSQLFilters(ctx)
			require.NoError(t, err)

			assert.ElementsMatch(t, test.expected, filters)
		})
	}
}
