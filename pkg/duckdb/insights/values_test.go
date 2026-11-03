package insights

import (
	"math"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestJSONSafeValue(t *testing.T) {
	ts := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	huge, _ := new(big.Int).SetString("170141183460469231731687303715884105727", 10)

	cases := []struct {
		name string
		in   any
		want any
	}{
		{"nil", nil, nil},
		{"string", "x", "x"},
		{"finite float", 1.5, 1.5},
		{"nan", math.NaN(), "NaN"},
		{"inf", math.Inf(1), "Infinity"},
		{"-inf", math.Inf(-1), "-Infinity"},
		{"float32 inf", float32(math.Inf(1)), "Infinity"},
		{"safe int", int64(maxSafeInteger), int64(maxSafeInteger)},
		{"unsafe int", int64(9007199254740993), "9007199254740993"},
		{"unsafe negative int", int64(-9007199254740993), "-9007199254740993"},
		{"unsafe uint", uint64(18000000000000000000), "18000000000000000000"},
		{"big int", huge, "170141183460469231731687303715884105727"},
		{"small big int", big.NewInt(7), int64(7)},
		{"time passes through", ts, ts},
		{"list", []any{math.NaN(), int64(1)}, []any{"NaN", int64(1)}},
		{"struct", map[string]any{"a": math.Inf(-1)}, map[string]any{"a": "-Infinity"}},
		{"map w/ non-string keys", map[int]float64{1: math.Inf(1)}, map[string]any{"1": "Infinity"}},
		{"typed list", []float64{math.NaN()}, []any{"NaN"}},
		{"bytes pass through", []byte("ab"), []byte("ab")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, JSONSafeValue(tc.in))
		})
	}
}
