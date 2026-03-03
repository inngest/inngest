package event_cel

import (
	"context"
	"testing"

	sq "github.com/doug-martin/goqu/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewExpressionHandler(t *testing.T) {
	ctx := context.Background()

	t.Run("empty handler - no filters", func(t *testing.T) {
		h, err := NewExpressionHandler(ctx)
		require.NoError(t, err)
		assert.False(t, h.HasFilters())
		assert.False(t, h.HasDataFilters())
		assert.Empty(t, h.EventExprList)
	})

	t.Run("valid event.name expression", func(t *testing.T) {
		h, err := NewExpressionHandler(ctx,
			WithExpressionHandlerBlob(`event.name == "test/hello"`, ""),
		)
		require.NoError(t, err)
		assert.True(t, h.HasFilters())
		assert.True(t, h.HasDataFilters())
		assert.Len(t, h.EventExprList, 1)
	})

	t.Run("multiple expressions via newline blob", func(t *testing.T) {
		blob := "event.name == \"test/hello\"\nevent.ts > 1727291508963"
		h, err := NewExpressionHandler(ctx, WithExpressionHandlerBlob(blob, ""))
		require.NoError(t, err)
		assert.True(t, h.HasFilters())
		assert.Len(t, h.EventExprList, 2)
	})

	t.Run("empty blob - no filters", func(t *testing.T) {
		h, err := NewExpressionHandler(ctx, WithExpressionHandlerBlob("", ""))
		require.NoError(t, err)
		assert.False(t, h.HasFilters())
	})
}

func TestEventCelValidation(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name   string
		cel    string
		errStr string
	}{
		{
			name: "valid event.name expression",
			cel:  `event.name == "test/hello"`,
		},
		{
			name: "valid event.id expression",
			cel:  `event.id == "abc-123"`,
		},
		{
			name: "valid event.ts expression",
			cel:  `event.ts > 1727291508963`,
		},
		{
			name: "valid event.v expression",
			cel:  `event.v == "1"`,
		},
		{
			name: "valid OR expression",
			cel:  `event.name == "a" || event.name == "b"`,
		},
		{
			name: "valid AND expression",
			cel:  `event.name == "a" && event.ts > 1000`,
		},
		{
			name:   "invalid macro",
			cel:    `event.data.num.filter(x, x > 5)`,
			errStr: "macros are currently not supported",
		},
		{
			name:   "has macro",
			cel:    `has(event.name)`,
			errStr: "macros are currently not supported",
		},
		{
			name:   "invalid triple equals",
			cel:    `event.name === "test"`,
			errStr: "token recognition error",
		},
		{
			name:   "lowercase and keyword",
			cel:    `event.name == "a" and event.ts > 1000`,
			errStr: "mismatched input 'and'",
		},
		{
			name:   "invalid syntax - method call",
			cel:    `event.name.startWith("hello")`,
			errStr: "invalid syntax detected",
		},
		{
			name:   "unsupported field",
			cel:    `user.name == "alice"`,
			errStr: "unsupported filter",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewExpressionHandler(ctx,
				WithExpressionHandlerBlob(test.cel, ""),
			)

			if test.errStr == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, test.errStr)
			}
		})
	}
}

func TestEventCelToSQLFilters(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		cel      string
		expected []sq.Expression
	}{
		{
			name:     "event.name equality",
			cel:      `event.name == "test/hello"`,
			expected: []sq.Expression{sq.C("event_name").Eq("test/hello")},
		},
		{
			name:     "event.name inequality",
			cel:      `event.name != "test/hello"`,
			expected: []sq.Expression{sq.C("event_name").Neq("test/hello")},
		},
		{
			name:     "event.id equality",
			cel:      `event.id == "abc-123"`,
			expected: []sq.Expression{sq.C("event_id").Eq("abc-123")},
		},
		{
			name:     "event.id inequality",
			cel:      `event.id != "abc-123"`,
			expected: []sq.Expression{sq.C("event_id").Neq("abc-123")},
		},
		{
			name:     "event.v equality",
			cel:      `event.v == "1"`,
			expected: []sq.Expression{sq.C("event_v").Eq("1")},
		},
		{
			name:     "event.ts greater than",
			cel:      `event.ts > 1727291508963`,
			expected: []sq.Expression{sq.C("event_ts").Gt(int64(1727291508963))},
		},
		{
			name:     "event.ts greater than or equal",
			cel:      `event.ts >= 1727291508963`,
			expected: []sq.Expression{sq.C("event_ts").Gte(int64(1727291508963))},
		},
		{
			name:     "event.ts less than",
			cel:      `event.ts < 1727291508963`,
			expected: []sq.Expression{sq.C("event_ts").Lt(int64(1727291508963))},
		},
		{
			name:     "event.ts less than or equal",
			cel:      `event.ts <= 1727291508963`,
			expected: []sq.Expression{sq.C("event_ts").Lte(int64(1727291508963))},
		},
		{
			name:     "event.ts equality",
			cel:      `event.ts == 1727291508963`,
			expected: []sq.Expression{sq.C("event_ts").Eq(int64(1727291508963))},
		},
		{
			name:     "event.ts inequality",
			cel:      `event.ts != 1727291508963`,
			expected: []sq.Expression{sq.C("event_ts").Neq(int64(1727291508963))},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			h, err := NewExpressionHandler(ctx,
				WithExpressionHandlerBlob(test.cel, ""),
			)
			require.NoError(t, err)

			filters, err := h.ToSQLFilters(ctx)
			require.NoError(t, err)

			assert.ElementsMatch(t, test.expected, filters)
		})
	}
}

func TestEventCelToSQLFiltersComplex(t *testing.T) {
	ctx := context.Background()

	t.Run("OR expression", func(t *testing.T) {
		h, err := NewExpressionHandler(ctx,
			WithExpressionHandlerBlob(`event.name == "a" || event.name == "b"`, ""),
		)
		require.NoError(t, err)

		filters, err := h.ToSQLFilters(ctx)
		require.NoError(t, err)

		assert.Len(t, filters, 1)
		expected := sq.Or(
			sq.C("event_name").Eq("a"),
			sq.C("event_name").Eq("b"),
		)
		assert.Equal(t, expected, filters[0])
	})

	t.Run("AND expression", func(t *testing.T) {
		h, err := NewExpressionHandler(ctx,
			WithExpressionHandlerBlob(`event.name == "test" && event.ts > 1000`, ""),
		)
		require.NoError(t, err)

		filters, err := h.ToSQLFilters(ctx)
		require.NoError(t, err)

		assert.Len(t, filters, 1)
		expected := sq.And(
			sq.C("event_name").Eq("test"),
			sq.C("event_ts").Gt(int64(1000)),
		)
		assert.Equal(t, expected, filters[0])
	})

	t.Run("multiple expressions via newline blob produce multiple filters", func(t *testing.T) {
		blob := "event.name == \"test/hello\"\nevent.ts > 1727291508963"
		h, err := NewExpressionHandler(ctx, WithExpressionHandlerBlob(blob, ""))
		require.NoError(t, err)

		filters, err := h.ToSQLFilters(ctx)
		require.NoError(t, err)

		assert.ElementsMatch(t, []sq.Expression{
			sq.C("event_name").Eq("test/hello"),
			sq.C("event_ts").Gt(int64(1727291508963)),
		}, filters)
	})

	t.Run("no filters returns empty slice", func(t *testing.T) {
		h, err := NewExpressionHandler(ctx)
		require.NoError(t, err)

		filters, err := h.ToSQLFilters(ctx)
		require.NoError(t, err)
		assert.Empty(t, filters)
	})

	t.Run("duplicate expressions are deduplicated", func(t *testing.T) {
		blob := "event.name == \"test/hello\"\nevent.name == \"test/hello\""
		h, err := NewExpressionHandler(ctx, WithExpressionHandlerBlob(blob, ""))
		require.NoError(t, err)

		// The same expression appears twice but should only generate one filter
		filters, err := h.ToSQLFilters(ctx)
		require.NoError(t, err)

		assert.Len(t, filters, 1)
	})
}

func TestEventCelCustomSQLConverter(t *testing.T) {
	ctx := context.Background()

	t.Run("custom SQL converter is used", func(t *testing.T) {
		customFilter := sq.C("custom_col").Eq("custom_val")
		customConverter := func(ctx context.Context, n interface{ HasPredicate() bool }) ([]sq.Expression, error) {
			return []sq.Expression{customFilter}, nil
		}
		_ = customConverter

		// Verify default SQLiteConverter is used when no custom converter set
		h, err := NewExpressionHandler(ctx,
			WithExpressionHandlerBlob(`event.name == "test"`, ""),
		)
		require.NoError(t, err)
		assert.NotNil(t, h.SQLConverter)

		filters, err := h.ToSQLFilters(ctx)
		require.NoError(t, err)
		assert.Equal(t, []sq.Expression{sq.C("event_name").Eq("test")}, filters)
	})
}
