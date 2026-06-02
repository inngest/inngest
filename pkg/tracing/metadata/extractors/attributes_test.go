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

func strArrAttr(key string, values ...string) *v1.KeyValue {
	vals := make([]*v1.AnyValue, len(values))
	for i, v := range values {
		vals[i] = &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: v}}
	}
	return &v1.KeyValue{
		Key:   key,
		Value: &v1.AnyValue{Value: &v1.AnyValue_ArrayValue{ArrayValue: &v1.ArrayValue{Values: vals}}},
	}
}

// TestCanonicalKeysHaveFieldSetters guards against drift between
// canonicalKeyMapping and metadataFieldSetters: every canonical key referenced
// by a mapping must have a setter, otherwise the attribute is silently dropped.
func TestCanonicalKeysHaveFieldSetters(t *testing.T) {
	t.Parallel()

	for sourceKey, mapping := range keyFieldMap {
		_, ok := metadataFieldSetters[mapping.field]
		assert.Truef(t, ok,
			"field %q (mapped from %q) has no entry in metadataFieldSetters",
			mapping.field, sourceKey)
	}
}

// TestRankingPrefersProviderNameOverDeprecatedSystem verifies that when both
// the deprecated gen_ai.system and its replacement gen_ai.provider.name are
// present, the deprecated key's keyRank demotes it so the replacement wins,
// regardless of attribute order.
func TestRankingPrefersProviderNameOverDeprecatedSystem(t *testing.T) {
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
