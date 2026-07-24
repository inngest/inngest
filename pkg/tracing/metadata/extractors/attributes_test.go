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

func dblAttr(key string, value float64) *v1.KeyValue {
	return &v1.KeyValue{
		Key:   key,
		Value: &v1.AnyValue{Value: &v1.AnyValue_DoubleValue{DoubleValue: value}},
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
		assert.Equal(t, "anthropic", md.Provider,
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

// TestGranularUsageAndRequestParams verifies extraction of the granular usage
// and request-parameter attributes into their pointer fields.
func TestGranularUsageAndRequestParams(t *testing.T) {
	t.Parallel()

	var md AIMetadata
	ok := extractAIMetadataFromAttributes([]*v1.KeyValue{
		intAttr("gen_ai.usage.cache_read.input_tokens", 100),
		intAttr("gen_ai.usage.cache_creation.input_tokens", 25),
		intAttr("gen_ai.usage.reasoning.output_tokens", 40),
		dblAttr("gen_ai.request.temperature", 0.7),
		dblAttr("gen_ai.request.top_p", 0.9),
		intAttr("gen_ai.request.max_tokens", 1024),
		dblAttr("gen_ai.request.frequency_penalty", 0.5),
		dblAttr("gen_ai.request.presence_penalty", -0.25),
		intAttr("gen_ai.request.seed", 42),
	}, &md)

	assert.True(t, ok)
	assert.Equal(t, int64(100), *md.CacheReadTokens)
	assert.Equal(t, int64(25), *md.CacheCreationTokens)
	assert.Equal(t, int64(40), *md.ReasoningTokens)
	assert.Equal(t, 0.7, *md.Temperature)
	assert.Equal(t, 0.9, *md.TopP)
	assert.Equal(t, int64(1024), *md.MaxTokens)
	assert.Equal(t, 0.5, *md.FrequencyPenalty)
	assert.Equal(t, -0.25, *md.PresencePenalty)
	assert.Equal(t, int64(42), *md.Seed)
}

// TestNumericCoercion verifies the int/double/string fallbacks, since OTLP
// encoders are inconsistent about how they encode numeric attribute values.
func TestNumericCoercion(t *testing.T) {
	t.Parallel()

	t.Run("int field accepts double and string", func(t *testing.T) {
		var md AIMetadata
		ok := extractAIMetadataFromAttributes([]*v1.KeyValue{
			dblAttr("gen_ai.request.max_tokens", 2048),
			strAttr("gen_ai.request.seed", "7"),
		}, &md)
		assert.True(t, ok)
		assert.Equal(t, int64(2048), *md.MaxTokens)
		assert.Equal(t, int64(7), *md.Seed)
	})

	t.Run("float field accepts int and string", func(t *testing.T) {
		var md AIMetadata
		ok := extractAIMetadataFromAttributes([]*v1.KeyValue{
			intAttr("gen_ai.request.temperature", 1),
			strAttr("gen_ai.request.top_p", "0.95"),
		}, &md)
		assert.True(t, ok)
		assert.Equal(t, 1.0, *md.Temperature)
		assert.Equal(t, 0.95, *md.TopP)
	})

	t.Run("unparseable string leaves field unset", func(t *testing.T) {
		var md AIMetadata
		extractAIMetadataFromAttributes([]*v1.KeyValue{
			strAttr("gen_ai.request.max_tokens", "not-a-number"),
		}, &md)
		assert.Nil(t, md.MaxTokens)
	})
}
