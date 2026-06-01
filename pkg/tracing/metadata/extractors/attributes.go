package extractors

import (
	"cmp"
	"slices"

	v1 "go.opentelemetry.io/proto/otlp/common/v1"
)

// convention identifies a namespace of attributes and orders which should win
// when multiple namespaces are present. Lower numbers will override higher
// numbers.
type convention int

const (
	semconv       convention = 1
	openinference convention = 2
)

// attrMapping records the canonical field a source attribute key maps to, the
// convention it belongs to, and a keyRank tiebreak used to order keys within
// the same convention (lower wins; 0 = default).
type attrMapping struct {
	field      string
	convention convention
	keyRank    int
}

var keyFieldMap = map[string]attrMapping{
	// ---------------------------------------------------------------------------
	// Open Telementry Semantic Conventions
	// ---------------------------------------------------------------------------
	"gen_ai.usage.input_tokens": {
		field:      "inputTokens",
		convention: semconv,
	},
	"gen_ai.usage.output_tokens": {
		field:      "outputTokens",
		convention: semconv,
	},
	"gen_ai.request.model": {
		field:      "model",
		convention: semconv,
	},
	"gen_ai.provider.name": {
		field:      "providerName",
		convention: semconv,
	},
	"gen_ai.system": {
		// deprecated in semconv in favor of gen_ai.provider.name; both are
		// semconv, so this keyRank places it behind its replacement.
		field:      "providerName",
		convention: semconv,
		keyRank:    1,
	},
	"gen_ai.operation.name": {
		field:      "operationName",
		convention: semconv,
	},

	// ---------------------------------------------------------------------------
	// Open Inference
	// ---------------------------------------------------------------------------
	"llm.token_count.prompt": {
		field:      "inputTokens",
		convention: openinference,
	},
	"llm.token_count.completion": {
		field:      "outputTokens",
		convention: openinference,
	},
	"llm.model_name": {
		field:      "model",
		convention: openinference,
	},
	// llm.system identifies the AI product/vendor (openai, anthropic, ...),
	// matching the semantics of the deprecated semconv gen_ai.system.
	"llm.system": {
		field:      "providerName",
		convention: openinference,
	},
}

var metadataFieldSetters = map[string]func(v *v1.AnyValue, md *AIMetadata){
	"inputTokens": func(v *v1.AnyValue, md *AIMetadata) {
		md.InputTokens = v.GetIntValue()
	},
	"outputTokens": func(v *v1.AnyValue, md *AIMetadata) {
		md.OutputTokens = v.GetIntValue()
	},
	"model": func(v *v1.AnyValue, md *AIMetadata) {
		md.Model = v.GetStringValue()
	},
	"providerName": func(v *v1.AnyValue, md *AIMetadata) {
		md.System = v.GetStringValue()
	},
	"operationName": func(v *v1.AnyValue, md *AIMetadata) {
		md.OperationName = v.GetStringValue()
	},
}

// parsedAttr records a key, value, the convention it came from, and its
// keyRank tiebreak within that convention.
type parsedAttr struct {
	value      *v1.AnyValue
	convention convention
	keyRank    int
}

// compareByRank computes the overall ordering of two parsedAttrs.
// We order first by convention and berak ties with keyRank.
func compareByRank(a, b parsedAttr) int {
	return cmp.Or(
		cmp.Compare(a.convention, b.convention),
		cmp.Compare(a.keyRank, b.keyRank),
	)
}

func extractAIMetadataFromAttributes(attributes []*v1.KeyValue, md *AIMetadata) (foundAny bool) {
	potentialAttrs := map[string][]parsedAttr{}

	for _, attr := range attributes {
		if mapping, ok := keyFieldMap[attr.Key]; ok {
			potentialAttrs[mapping.field] = append(
				potentialAttrs[mapping.field],
				parsedAttr{
					value:      attr.Value,
					convention: mapping.convention,
					keyRank:    mapping.keyRank,
				},
			)
		}
	}

	if len(potentialAttrs) == 0 {
		return false
	}

	for field, attrs := range potentialAttrs {
		if len(attrs) == 0 {
			continue
		}

		var chosenAttr parsedAttr

		if len(attrs) == 1 {
			chosenAttr = attrs[0]
		} else {
			// Reduce our list of attrs to the highest-priority one.
			chosenAttr = slices.MinFunc(attrs, compareByRank)
		}

		metadataFieldSetter, ok := metadataFieldSetters[field]
		if !ok {
			// log an error for the unexpected field
			continue
		}

		metadataFieldSetter(chosenAttr.value, md)
		foundAny = true
	}

	return foundAny
}
