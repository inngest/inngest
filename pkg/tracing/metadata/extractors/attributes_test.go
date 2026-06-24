package extractors

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "go.opentelemetry.io/proto/otlp/common/v1"
)

func strAttr(key, value string) *v1.KeyValue {
	return &v1.KeyValue{
		Key:   key,
		Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: value}},
	}
}

func intAttr(key string, value int64) *v1.KeyValue {
	return &v1.KeyValue{
		Key:   key,
		Value: &v1.AnyValue{Value: &v1.AnyValue_IntValue{IntValue: value}},
	}
}

func arrAttr(key string, values ...string) *v1.KeyValue {
	vals := make([]*v1.AnyValue, len(values))
	for i, s := range values {
		vals[i] = &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: s}}
	}
	return &v1.KeyValue{
		Key: key,
		Value: &v1.AnyValue{
			Value: &v1.AnyValue_ArrayValue{ArrayValue: &v1.ArrayValue{Values: vals}},
		},
	}
}

// TestProviderPrefersProviderNameOverDeprecatedSystem verifies that when both
// the deprecated gen_ai.system and its replacement gen_ai.provider.name are
// present, the replacement wins, regardless of attribute order.
func TestProviderPrefersProviderNameOverDeprecatedSystem(t *testing.T) {
	t.Parallel()

	orders := [][]*v1.KeyValue{
		{strAttr("gen_ai.system", "openai"), strAttr("gen_ai.provider.name", "anthropic")},
		{strAttr("gen_ai.provider.name", "anthropic"), strAttr("gen_ai.system", "openai")},
	}

	for _, attrs := range orders {
		var md AIMetadata
		foundAny := extractAIMetadataFromAttributes(attrs, &md)
		assert.True(t, foundAny)
		assert.Equal(t, "anthropic", md.System,
			"gen_ai.provider.name should win over deprecated gen_ai.system")
	}
}

// TestFinishReasons verifies the array path, the scalar-string fallback, and
// that empty entries are dropped (leaving the field unset when none remain).
func TestFinishReasons(t *testing.T) {
	t.Parallel()

	t.Run("array drops empty entries", func(t *testing.T) {
		var md AIMetadata
		ok := extractAIMetadataFromAttributes(
			[]*v1.KeyValue{arrAttr("gen_ai.response.finish_reasons", "stop", "", "length")}, &md)
		assert.True(t, ok)
		assert.Equal(t, []string{"stop", "length"}, md.FinishReasons)
	})

	t.Run("scalar string is wrapped", func(t *testing.T) {
		var md AIMetadata
		ok := extractAIMetadataFromAttributes(
			[]*v1.KeyValue{strAttr("gen_ai.response.finish_reasons", "tool_calls")}, &md)
		assert.True(t, ok)
		assert.Equal(t, []string{"tool_calls"}, md.FinishReasons)
	})

	t.Run("empty scalar leaves field unset", func(t *testing.T) {
		var md AIMetadata
		extractAIMetadataFromAttributes(
			[]*v1.KeyValue{strAttr("gen_ai.response.finish_reasons", "")}, &md)
		assert.Nil(t, md.FinishReasons)
	})
}
