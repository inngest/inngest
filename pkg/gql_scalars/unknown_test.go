package types

import (
	"bytes"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMarshalUnknown(t *testing.T) {
	cases := []struct {
		name string
		v    any
		want string
	}{
		{"object", map[string]any{"a": 1}, `{"a":1}`},
		{"nil", nil, `null`},
		{"nan does not panic", math.NaN(), `null`},
		{"nested inf does not panic", []any{1, math.Inf(1)}, `null`},
		{"unsupported type does not panic", make(chan int), `null`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			require.NotPanics(t, func() { MarshalUnknown(tc.v).MarshalGQL(&buf) })
			require.Equal(t, tc.want, buf.String())
		})
	}
}
