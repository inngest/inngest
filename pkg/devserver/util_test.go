package devserver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAnyToString(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected string
		err      error
	}{
		{
			name:     "should convert int values",
			value:    10,
			expected: "10",
			err:      nil,
		},
		{
			name:     "should convert uint values",
			value:    uint(100),
			expected: "100",
			err:      nil,
		},
		{
			name:     "should convert bool values",
			value:    false,
			expected: "false",
			err:      nil,
		},
		{
			name:     "should assign string as is",
			value:    "hello",
			expected: "hello",
			err:      nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := anyToString(test.value)

			assert.Equal(t, test.expected, actual)
			assert.Equal(t, test.err, err)
		})
	}
}
