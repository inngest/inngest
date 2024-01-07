package expressions

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInterpolated(t *testing.T) {
	tests := []struct {
		exprInput    string
		exprExpected string
		vars         map[string]any
	}{
		{
			// int
			exprInput:    `event.data.id == async.data.id && async.data.val <= event.data.value`,
			exprExpected: `"ab_1" == async.data.id && async.data.val <= 1295`,
			vars: map[string]any{
				"event": map[string]any{
					"data": map[string]any{
						"id":    "ab_1",
						"value": 1295,
					},
				},
			},
		},
		{
			// float
			exprInput:    `event.data.id == async.data.id && async.data.val <= event.data.value`,
			exprExpected: `"ab_1" == async.data.id && async.data.val <= 1295.0`,
			vars: map[string]any{
				"event": map[string]any{
					"data": map[string]any{
						"id":    "ab_1",
						"value": 1295.0,
					},
				},
			},
		},
		{
			// float
			exprInput:    `event.data.id == async.data.id && async.data.val <= event.data.value`,
			exprExpected: `"ab_1" == async.data.id && async.data.val <= 1295.0`,
			vars: map[string]any{
				"event": map[string]any{
					"data": map[string]any{
						"id":    "ab_1",
						"value": float64(1295),
					},
				},
			},
		},
		{
			exprInput: `event.data.id == async.data.id && 500 <= event.data.value`,
			// NOTE: 500 <= 1295 evaluates to true and should be missing.
			exprExpected: `"ab_1" == async.data.id`,
			vars: map[string]any{
				"event": map[string]any{
					"data": map[string]any{
						"id":    "ab_1",
						"value": 1295,
					},
				},
			},
		},
		{
			exprInput: `event.data.id == async.data.id && event.data.foo == async.data.foo`,
			// event.data.foo is not present and should be left.
			exprExpected: `"ab_1" == async.data.id && event.data.foo == async.data.foo`,
			vars: map[string]any{
				"event": map[string]any{
					"data": map[string]any{
						"id": "ab_1",
					},
				},
			},
		},
	}

	for _, test := range tests {
		actual, err := Interpolated(context.Background(), test.exprInput, test.vars)
		require.NoError(t, err)
		fmt.Println(actual)
		require.EqualValues(t, test.exprExpected, actual)
	}
}
