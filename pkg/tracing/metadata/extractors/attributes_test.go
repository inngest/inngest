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

// TestCanonicalKeysHaveFieldSetters guards against drift between
// canonicalKeyMapping and metadataFieldSetters: every canonical key referenced
// by a scalar mapping must have a setter, otherwise the attribute is silently
// dropped. Composite mappings (expand != nil) carry no field of their own — they
// explode into synthetic children that are matched back through keyFieldMap, so
// those children are covered by their own scalar entries in this same loop.
func TestCanonicalKeysHaveFieldSetters(t *testing.T) {
	t.Parallel()

	for sourceKey, mapping := range keyFieldMap {
		if mapping.expand != nil {
			assert.Emptyf(t, mapping.field,
				"composite mapping %q must not also set a scalar field", sourceKey)
			continue
		}

		_, ok := metadataFieldSetters[mapping.field]
		assert.Truef(t, ok,
			"field %q (mapped from %q) has no entry in metadataFieldSetters",
			mapping.field, sourceKey)
	}
}

func TestAttrMappings_FieldAndExpandAreExclusive(t *testing.T) {
	t.Parallel()
	for key, mapping := range keyFieldMap {
		assert.Truef(
			t,
			(mapping.field != "") != (mapping.expand != nil),
			"key %s cannot have both field and expand set", key,
		)
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

// TestLangfusePrecedenceAndUsageExpansion verifies two things no captured
// fixture exercises (real Langfuse spans carry no gen_ai.*): on a span carrying
// both namespaces, langfuse.* wins (it ranks first), and the usage_details JSON
// blob expands into input/output/total while unmapped sub-keys are ignored.
func TestLangfusePrecedenceAndUsageExpansion(t *testing.T) {
	t.Parallel()

	orders := [][]*v1.KeyValue{
		{
			strAttr("gen_ai.response.model", "gpt-4.1-nano-another"),
			intAttr("gen_ai.usage.input_tokens", 100),
			strAttr("langfuse.observation.model.name", "gpt-4.1-nano-2025-04-14"),
			strAttr("langfuse.observation.usage_details",
				`{"input":22,"output":6,"total":28,"input_cached_tokens":5}`),
		},
		// reversed, to prove order-independence
		{
			strAttr("langfuse.observation.usage_details",
				`{"input":22,"output":6,"total":28,"input_cached_tokens":5}`),
			strAttr("langfuse.observation.model.name", "gpt-4.1-nano-2025-04-14"),
			intAttr("gen_ai.usage.input_tokens", 100),
			strAttr("gen_ai.response.model", "gpt-4.1-nano-another"),
		},
	}

	for _, attrs := range orders {
		var md AIMetadata
		foundAny := extractAIMetadataFromAttributes(attrs, &md)
		assert.True(t, foundAny)
		// langfuse ranks first, so its values win over the co-present gen_ai.*.
		assert.Equal(t, "gpt-4.1-nano-2025-04-14", md.ResponseModel)
		assert.Equal(t, int64(22), md.InputTokens)
		assert.Equal(t, int64(6), md.OutputTokens)
		// usage_details supplies the total; input_cached_tokens is unmapped and
		// dropped, not summed.
		if assert.NotNil(t, md.TotalTokens) {
			assert.Equal(t, int64(28), *md.TotalTokens)
		}
	}
}

// TestSkipsVercelRollupSpan verifies that the Vercel AI SDK's framework rollup
// span (e.g. `ai.generateText`) extracts to nothing so its `ai.usage.*` isn't
// double-counted against its provider-call child (`ai.generateText.doGenerate`),
// which carries the same usage and the documented `.do*` segment.
func TestSkipsVercelRollupSpan(t *testing.T) {
	t.Parallel()

	// The rollup span (no `.do*` segment) is skipped entirely.
	rollup := []*v1.KeyValue{
		strAttr("ai.operationId", "ai.generateText"),
		strAttr("ai.model.id", "gpt-4.1-nano"),
		intAttr("ai.usage.inputTokens", 17),
	}
	var rollupMd AIMetadata
	assert.False(t, extractAIMetadataFromAttributes(rollup, &rollupMd))
	assert.Equal(t, AIMetadata{}, rollupMd)

	// The provider-call (leaf) span is still extracted normally.
	leaf := []*v1.KeyValue{
		strAttr("ai.operationId", "ai.generateText.doGenerate"),
		strAttr("ai.model.id", "gpt-4.1-nano"),
		intAttr("ai.usage.inputTokens", 17),
	}
	var leafMd AIMetadata
	assert.True(t, extractAIMetadataFromAttributes(leaf, &leafMd))
	assert.Equal(t, "gpt-4.1-nano", leafMd.Model)
	assert.Equal(t, int64(17), leafMd.InputTokens)
}
