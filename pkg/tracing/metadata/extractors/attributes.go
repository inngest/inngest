package extractors

import (
	"cmp"
	"encoding/json"
	"regexp"
	"slices"

	"github.com/inngest/inngest/pkg/util"
	v1 "go.opentelemetry.io/proto/otlp/common/v1"
)

// convention identifies a namespace of attributes and orders which should win
// when multiple namespaces are present. Lower numbers will override higher
// numbers.
type convention int

const (
	// langfuse ranks first: Langfuse states its langfuse.* keys always
	// take precedence over the generic conventions on spans it instrumented.
	langfuse      convention = 1
	semconv       convention = 2
	openinference convention = 3
	vercel        convention = 4
)

// langfuseUsagePrefix namespaces the synthetic scalar keys that
// expandLangfuseUsageDetails emits from the langfuse.observation.usage_details
// JSON blob. The double underscore marks them as derived, not wire attributes.
const langfuseUsagePrefix = "__langfuse.usage_details."

// attrMapping records the canonical field a source attribute key maps to, the
// convention it belongs to, and a keyRank tiebreak used to order keys within
// the same convention (lower wins; 0 = default).
//
// A mapping is either scalar (field set, expand nil) or composite (expand set,
// field empty): a composite mapping carries no value of its own — its expand
// func explodes the attribute into synthetic child KeyValues that are matched
// back through keyFieldMap like any other attribute.
type attrMapping struct {
	field      string
	convention convention
	keyRank    int
	expand     func(v *v1.AnyValue) []*v1.KeyValue
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
		field:      "responseModel",
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

	// ---------------------------------------------------------------------------
	// Langfuse (`langfuse.*`, via @langfuse/openai + LangfuseSpanProcessor)
	// ---------------------------------------------------------------------------
	"langfuse.observation.model.name": {
		field:      "responseModel",
		convention: langfuse,
	},
	// usage_details is a single JSON blob ({"input":N,"output":N,"total":N,...}).
	// It carries no value of its own: expand explodes it into synthetic scalar
	// children, which are matched back through keyFieldMap below.
	"langfuse.observation.usage_details": {
		convention: langfuse,
		expand:     expandLangfuseUsageDetails,
	},
	langfuseUsagePrefix + "input": {
		field:      "inputTokens",
		convention: langfuse,
	},
	langfuseUsagePrefix + "output": {
		field:      "outputTokens",
		convention: langfuse,
	},
	langfuseUsagePrefix + "total": {
		field:      "totalTokens",
		convention: langfuse,
	},
}

// expandLangfuseUsageDetails parses the langfuse.observation.usage_details JSON
// blob and emits one synthetic scalar KeyValue per integer entry, keyed under
// langfuseUsagePrefix. keyFieldMap does further processing on the keys emitted.
// matching does.
func expandLangfuseUsageDetails(v *v1.AnyValue) []*v1.KeyValue {
	raw := v.GetStringValue()
	if raw == "" {
		return nil
	}

	var counts map[string]json.Number
	if err := json.Unmarshal([]byte(raw), &counts); err != nil {
		return nil
	}

	out := make([]*v1.KeyValue, 0, len(counts))
	for k, num := range counts {
		n, err := num.Int64()
		if err != nil {
			// non-integer entry (shouldn't happen for token counts); skip.
			continue
		}
		out = append(out, &v1.KeyValue{
			Key:   langfuseUsagePrefix + k,
			Value: &v1.AnyValue{Value: &v1.AnyValue_IntValue{IntValue: n}},
		})
	}
	return out
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

// vercelProviderCallSegment matches the `.do<Op>` segment (e.g. `.doGenerate`,
// `.doStream`, `.doEmbed`) that the Vercel AI SDK appends to the provider-call
// (leaf) span's operationId. The framework rollup span lacks this segment.
//
// @see https://ai-sdk.dev/docs/ai-sdk-core/telemetry
var vercelProviderCallSegment = regexp.MustCompile(`\.do[A-Z]`)

// isVercelRollupSpan reports whether the span is a Vercel AI SDK framework
// rollup (wrapper) span.
//
// The Vercel AI SDK emits a span *tree*: a framework rollup span
// (`ai.generateText`) whose children are the provider-call (leaf) spans
// (`ai.generateText.doGenerate`). Both carry overlapping usage data, so
// extracting from both would double count. We choose to skip the rollup span.
//
// Only Vercel AI SDK spans carry `ai.operationId`; among them, the
// provider-call leaf has a `.do*` segment and the rollup does not.
func isVercelRollupSpan(attributes []*v1.KeyValue) bool {
	for _, attr := range attributes {
		if attr.Key != "ai.operationId" {
			continue
		}
		op := attr.Value.GetStringValue()
		return op != "" && !vercelProviderCallSegment.MatchString(op)
	}
	return false
}

func extractAIMetadataFromAttributes(attributes []*v1.KeyValue, md *AIMetadata) (foundAny bool) {
	// The Vercel AI SDK's rollup span duplicates its provider-call child's
	// usage; skip it to avoid double counting.
	if isVercelRollupSpan(attributes) {
		return false
	}

	potentialAttrs := map[string][]parsedAttr{}

	// addAttr records a matched attribute as a candidate for its canonical field.
	addAttr := func(m attrMapping, value *v1.AnyValue) {
		potentialAttrs[m.field] = append(
			potentialAttrs[m.field],
			parsedAttr{
				value:      value,
				convention: m.convention,
				keyRank:    m.keyRank,
			},
		)
	}

	for _, attr := range attributes {
		mapping, ok := keyFieldMap[attr.Key]
		if !ok {
			continue
		}

		// Composite mapping: explode the attribute into synthetic children and
		// match each back through keyFieldMap, so they flow through the same
		// per-field precedence as everything else. Children carry their own
		// mapping's convention/keyRank, and unmapped children are ignored.
		if mapping.expand != nil {
			for _, child := range mapping.expand(attr.Value) {
				if childMapping, ok := keyFieldMap[child.Key]; ok {
					addAttr(childMapping, child.Value)
				}
			}
			continue
		}

		addAttr(mapping, attr.Value)
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
