package extractors

import (
	"cmp"
	"slices"

	"github.com/inngest/inngest/pkg/util"
	v1 "go.opentelemetry.io/proto/otlp/common/v1"
)

// convention identifies a namespace of attributes and orders which should win
// when multiple namespaces are present. Lower numbers will override higher
// numbers.
type convention int

const (
	semconv       convention = 1
	openinference convention = 2
	vercel        convention = 3
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
	"gen_ai.usage.total_tokens": {
		field:      "totalTokens",
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
	"gen_ai.response.model": {
		field:      "responseModel",
		convention: semconv,
	},
	"gen_ai.response.id": {
		field:      "responseId",
		convention: semconv,
	},
	"gen_ai.response.finish_reasons": {
		field:      "finishReasons",
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
	"llm.token_count.total": {
		field:      "totalTokens",
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
	// OpenInference emits a single scalar finish reason rather than the semconv
	// gen_ai.response.finish_reasons array.
	"llm.finish_reason": {
		field:      "finishReasons",
		convention: openinference,
	},

	// ---------------------------------------------------------------------------
	// Vercel AI SDK (native telemetry, `ai.*`)
	// ---------------------------------------------------------------------------
	// The AI SDK emits its own spans. A call produces a parent ai.<op> span
	// (ai.* only) plus a child ai.<op>.do<Op> span that ALSO carries a partial
	// gen_ai.* set. vercel ranks below semconv/openinference so gen_ai.* stays
	// authoritative on the child where both are present (their values agree);
	// these mappings are what make the ai.*-only parent and embeddings spans
	// extractable, and they supply ai.usage.totalTokens (gen_ai omits a total).
	"ai.model.id": {
		field:      "model",
		convention: vercel,
	},
	// ai.model.provider is the provider + API surface, e.g. "openai.responses"
	// (not bare "openai"); stored faithfully, no normalization.
	"ai.model.provider": {
		field:      "providerName",
		convention: vercel,
	},
	"ai.usage.inputTokens": {
		field:      "inputTokens",
		convention: vercel,
	},
	"ai.usage.outputTokens": {
		field:      "outputTokens",
		convention: vercel,
	},
	"ai.usage.totalTokens": {
		field:      "totalTokens",
		convention: vercel,
	},
	// Embeddings spans emit only a single ai.usage.tokens count (no input/output
	// split, no gen_ai.*). Map it to inputTokens so the total derives, matching
	// the official-OTel embeddings case.
	"ai.usage.tokens": {
		field:      "inputTokens",
		convention: vercel,
	},
	"ai.response.model": {
		field:      "responseModel",
		convention: vercel,
	},
	"ai.response.id": {
		field:      "responseId",
		convention: vercel,
	},
	// ai.response.finishReason is a single scalar string (like OpenInference);
	// the finishReasons setter already handles the scalar case.
	"ai.response.finishReason": {
		field:      "finishReasons",
		convention: vercel,
	},
}

var metadataFieldSetters = map[string]func(v *v1.AnyValue, md *AIMetadata){
	"inputTokens": func(v *v1.AnyValue, md *AIMetadata) {
		md.InputTokens = v.GetIntValue()
	},
	"outputTokens": func(v *v1.AnyValue, md *AIMetadata) {
		md.OutputTokens = v.GetIntValue()
	},
	"totalTokens": func(v *v1.AnyValue, md *AIMetadata) {
		md.TotalTokens = util.ToPtr(v.GetIntValue())
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
	"responseModel": func(v *v1.AnyValue, md *AIMetadata) {
		md.ResponseModel = v.GetStringValue()
	},
	"responseId": func(v *v1.AnyValue, md *AIMetadata) {
		md.ResponseID = v.GetStringValue()
	},
	"finishReasons": func(v *v1.AnyValue, md *AIMetadata) {
		// semconv gen_ai.response.finish_reasons is an array (one per choice);
		// OpenInference's llm.finish_reason is a single scalar string. Handle
		// both and store the values raw (no tool_call/tool_calls normalization).
		if arr := v.GetArrayValue(); arr != nil {
			reasons := make([]string, 0, len(arr.GetValues()))
			for _, val := range arr.GetValues() {
				reasons = append(reasons, val.GetStringValue())
			}
			md.FinishReasons = reasons
		} else if s := v.GetStringValue(); s != "" {
			md.FinishReasons = []string{s}
		}
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
// We order first by convention and break ties with keyRank.
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

		// Reduce our list of attrs to the highest-priority one.
		chosenAttr := slices.MinFunc(attrs, compareByRank)

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
