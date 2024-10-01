package run

import (
	"context"
	"testing"

	sq "github.com/doug-martin/goqu/v9"
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
