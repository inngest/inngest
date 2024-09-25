package run

import (
	"context"
	"testing"

	sq "github.com/doug-martin/goqu/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToSQLEventFilters(t *testing.T) {
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
			cel:  []string{`event.name == 'test/hello' && event.name == 'test/yolo'`, `event.ts > 1727291508963`},
			expected: []sq.Expression{
				sq.And(
					sq.C("event_name").Eq("test/hello"),
					sq.C("event_name").Eq("test/yolo"),
				),
				sq.C("event_ts").Gt(int64(1727291508963)),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler, err := NewExpressionHandler(ctx, WithExpressionHandlerExpressions(test.cel))
			require.NoError(t, err)

			filters, err := handler.ToSQLEventFilters(ctx)
			require.NoError(t, err)

			assert.ElementsMatch(t, test.expected, filters)
		})
	}
}
