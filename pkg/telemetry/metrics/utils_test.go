package metrics

import (
	"github.com/google/uuid"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
)

func TestParseAttributes(t *testing.T) {
	tests := []struct {
		name     string
		attrs    map[string]any
		expected []attribute.KeyValue
	}{
		{
			name: "should be able to parse multiple types of data",
			attrs: map[string]any{
				"string": "world",
				"int":    200,
				"int64":  100000000000000,
				"uint":   uint(10),
				"float":  0.2,
				"bool":   true,
				"uuid":   uuid.MustParse("a8267138-39c2-4a36-8fa3-ed530b765006"),
			},
			expected: []attribute.KeyValue{
				{Key: attribute.Key("string"), Value: attribute.StringValue("world")},
				{Key: attribute.Key("int"), Value: attribute.IntValue(200)},
				{Key: attribute.Key("int64"), Value: attribute.IntValue(100000000000000)},
				{Key: attribute.Key("uint"), Value: attribute.Int64Value(10)},
				{Key: attribute.Key("float"), Value: attribute.Float64Value(0.2)},
				{Key: attribute.Key("bool"), Value: attribute.BoolValue(true)},
				{Key: attribute.Key("uuid"), Value: attribute.StringValue("a8267138-39c2-4a36-8fa3-ed530b765006")},
			},
		},
		{
			name: "unsupported type will be ignored",
			attrs: map[string]any{
				"string":  "world",
				"int":     200,
				"unknown": map[string]string{},
			},
			expected: []attribute.KeyValue{
				{Key: attribute.Key("string"), Value: attribute.StringValue("world")},
				{Key: attribute.Key("int"), Value: attribute.IntValue(200)},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := parseAttributes(test.attrs)
			require.ElementsMatch(t, test.expected, result)
		})
	}
}
