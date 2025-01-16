package base_cqrs

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMarshalJSONAsString(t *testing.T) {
	tests := []struct {
		input    any
		expected string
	}{
		{
			nil,
			`null`,
		},
		{
			`"hello"`,
			`"hello"`,
		},
		{
			1,
			`1`,
		},
		{
			&struct {
				Test string `json:"test"`
			}{
				Test: "test",
			},
			`{"test":"test"}`,
		},
	}

	for _, i := range tests {
		out, err := marshalJSONAsString(i.input)
		require.NoError(t, err)
		require.EqualValues(t, i.expected, out)
	}
}
