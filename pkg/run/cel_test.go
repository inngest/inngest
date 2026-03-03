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
		{
			name: "output string equality",
			cel:  []string{`output.status == "ok"`},
			expected: []sq.Expression{
				sq.L("((spans.output#>>'{}')::jsonb->'data')#>>'{status}'").Eq("ok"),
			},
		},
		{
			name: "output numeric greater than",
			cel:  []string{`output.count > 10`},
			expected: []sq.Expression{
				sq.L("(((spans.output#>>'{}')::jsonb->'data')#>>'{count}')::numeric").Gt(int64(10)),
			},
		},
		{
			name: "error string equality",
			cel:  []string{`error.message == "something went wrong"`},
			expected: []sq.Expression{
				sq.L("((spans.output#>>'{}')::jsonb->'error')#>>'{message}'").Eq("something went wrong"),
			},
		},
		{
			name: "error not null",
			cel:  []string{`error.code != null`},
			expected: []sq.Expression{
				sq.L("jsonb_typeof(((spans.output#>>'{}')::jsonb->'error')#>'{code}')").Neq("null"),
			},
		},
		{
			name: "event.data string equality",
			cel:  []string{`event.data.status == "active"`},
			expected: []sq.Expression{
				sq.L("(NULLIF(events.event_data, '')::jsonb)#>>'{status}'").Eq("active"),
			},
		},
		{
			name: "event.data numeric comparison",
			cel:  []string{`event.data.count > 5`},
			expected: []sq.Expression{
				sq.L("((NULLIF(events.event_data, '')::jsonb)#>>'{count}')::numeric").Gt(int64(5)),
			},
		},
		{
			name: "event.data nested field",
			cel:  []string{`event.data.nested.field == "value"`},
			expected: []sq.Expression{
				sq.L("(NULLIF(events.event_data, '')::jsonb)#>>'{nested,field}'").Eq("value"),
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

func TestToSQLFiltersWithSQLiteConverterAdditional(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		cel      []string
		expected []sq.Expression
	}{
		{
			name: "output string equality",
			cel:  []string{`output.status == "ok"`},
			expected: []sq.Expression{
				sq.L("json_extract(json_extract(spans.output, '$.data'), '$.status')").Eq("ok"),
			},
		},
		{
			name: "output numeric greater than",
			cel:  []string{`output.count > 10`},
			expected: []sq.Expression{
				sq.L("CAST(json_extract(json_extract(spans.output, '$.data'), '$.count') AS NUMERIC)").Gt(int64(10)),
			},
		},
		{
			name: "error string equality",
			cel:  []string{`error.message == "something went wrong"`},
			expected: []sq.Expression{
				sq.L("json_extract(json_extract(spans.output, '$.error'), '$.message')").Eq("something went wrong"),
			},
		},
		{
			name: "error not null",
			cel:  []string{`error.code != null`},
			expected: []sq.Expression{
				sq.L("json_type(json_extract(spans.output, '$.error'), '$.code')").Neq("null"),
			},
		},
		{
			name: "event.data string equality",
			cel:  []string{`event.data.status == "active"`},
			expected: []sq.Expression{
				sq.L("json_extract(NULLIF(events.event_data, ''), '$.status')").Eq("active"),
			},
		},
		{
			name: "event.data numeric comparison",
			cel:  []string{`event.data.count > 5`},
			expected: []sq.Expression{
				sq.L("CAST(json_extract(NULLIF(events.event_data, ''), '$.count') AS NUMERIC)").Gt(int64(5)),
			},
		},
		{
			name: "event.data nested field",
			cel:  []string{`event.data.nested.field == "value"`},
			expected: []sq.Expression{
				sq.L("json_extract(NULLIF(events.event_data, ''), '$.nested.field')").Eq("value"),
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

func TestHasFilters(t *testing.T) {
	ctx := context.Background()

	t.Run("no expressions - no filters", func(t *testing.T) {
		h, err := NewExpressionHandler(ctx)
		require.NoError(t, err)
		assert.False(t, h.HasFilters())
		assert.False(t, h.HasEventFilters())
		assert.False(t, h.HasOutputFilters())
	})

	t.Run("event expression - has event filter only", func(t *testing.T) {
		h, err := NewExpressionHandler(ctx,
			WithExpressionHandlerExpressions([]string{`event.name == "test/hello"`}),
		)
		require.NoError(t, err)
		assert.True(t, h.HasFilters())
		assert.True(t, h.HasEventFilters())
		assert.False(t, h.HasOutputFilters())
	})

	t.Run("output expression - has output filter only", func(t *testing.T) {
		h, err := NewExpressionHandler(ctx,
			WithExpressionHandlerExpressions([]string{`output.success == true`}),
		)
		require.NoError(t, err)
		assert.True(t, h.HasFilters())
		assert.False(t, h.HasEventFilters())
		assert.True(t, h.HasOutputFilters())
	})

	t.Run("error expression - has output filter only", func(t *testing.T) {
		h, err := NewExpressionHandler(ctx,
			WithExpressionHandlerExpressions([]string{`error.message == "fail"`}),
		)
		require.NoError(t, err)
		assert.True(t, h.HasFilters())
		assert.False(t, h.HasEventFilters())
		assert.True(t, h.HasOutputFilters())
	})

	t.Run("mixed event and output - has both", func(t *testing.T) {
		h, err := NewExpressionHandler(ctx,
			WithExpressionHandlerExpressions([]string{
				`event.name == "test/hello"`,
				`output.success == true`,
			}),
		)
		require.NoError(t, err)
		assert.True(t, h.HasFilters())
		assert.True(t, h.HasEventFilters())
		assert.True(t, h.HasOutputFilters())
	})
}

func TestWithExpressionHandlerBlob(t *testing.T) {
	ctx := context.Background()

	t.Run("newline-delimited blob", func(t *testing.T) {
		blob := "event.name == \"test/hello\"\nevent.ts > 1727291508963"
		h, err := NewExpressionHandler(ctx, WithExpressionHandlerBlob(blob, ""))
		require.NoError(t, err)
		assert.True(t, h.HasEventFilters())
		assert.Len(t, h.EventExprList, 2)
	})

	t.Run("custom delimiter blob", func(t *testing.T) {
		blob := `event.name == "test/hello"|output.success == true`
		h, err := NewExpressionHandler(ctx, WithExpressionHandlerBlob(blob, "|"))
		require.NoError(t, err)
		assert.True(t, h.HasEventFilters())
		assert.True(t, h.HasOutputFilters())
	})

	t.Run("empty blob", func(t *testing.T) {
		h, err := NewExpressionHandler(ctx, WithExpressionHandlerBlob("", ""))
		require.NoError(t, err)
		assert.False(t, h.HasFilters())
	})

	t.Run("single expression blob", func(t *testing.T) {
		blob := `event.name == "test/hello"`
		h, err := NewExpressionHandler(ctx, WithExpressionHandlerBlob(blob, "\n"))
		require.NoError(t, err)
		assert.True(t, h.HasEventFilters())
		assert.Len(t, h.EventExprList, 1)
	})
}

func TestMatchOutputExpressions(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		expr     string
		output   []byte
		expected bool
	}{
		{
			name:     "should match boolean true",
			expr:     `output.success == true`,
			output:   []byte(`{"success": true}`),
			expected: true,
		},
		{
			name:     "should not match boolean false vs true",
			expr:     `output.success == true`,
			output:   []byte(`{"success": false}`),
			expected: false,
		},
		{
			name:     "should match string equality",
			expr:     `output.status == "ok"`,
			output:   []byte(`{"status": "ok"}`),
			expected: true,
		},
		{
			name:     "should not match different string",
			expr:     `output.status == "ok"`,
			output:   []byte(`{"status": "fail"}`),
			expected: false,
		},
		{
			name:     "should match numeric comparison",
			expr:     `output.count > 5`,
			output:   []byte(`{"count": 10}`),
			expected: true,
		},
		{
			name:     "should not match numeric comparison",
			expr:     `output.count > 5`,
			output:   []byte(`{"count": 3}`),
			expected: false,
		},
		{
			name:     "should not match when output is empty",
			expr:     `output.success == true`,
			output:   []byte(``),
			expected: false,
		},
		{
			name:     "should match nested field",
			expr:     `output.data.value == "hello"`,
			output:   []byte(`{"data": {"value": "hello"}}`),
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler, err := NewExpressionHandler(ctx,
				WithExpressionHandlerExpressions([]string{test.expr}),
			)
			require.NoError(t, err)

			ok, err := handler.MatchOutputExpressions(ctx, test.output)
			require.NoError(t, err)
			assert.Equal(t, test.expected, ok)
		})
	}
}
