package extractors

import (
	v1 "go.opentelemetry.io/proto/otlp/common/v1"

	"github.com/inngest/inngest/pkg/util"
)

// extractAIMetadataFromAttributes reads the OpenTelemetry GenAI semantic
// convention (`gen_ai.*`) attributes off a span into md, returning false when
// the span carries none of the recognized attributes.
//
// Each supported attribute appears at most once on a span, so we index by key
// and read the fields we care about; everything else is ignored.
func extractAIMetadataFromAttributes(attributes []*v1.KeyValue, md *AIMetadata) (foundAny bool) {
	byKey := make(map[string]*v1.AnyValue, len(attributes))
	for _, attr := range attributes {
		byKey[attr.Key] = attr.Value
	}

	read := func(key string, set func(v *v1.AnyValue)) {
		if v, ok := byKey[key]; ok {
			set(v)
			foundAny = true
		}
	}

	read("gen_ai.request.model", func(v *v1.AnyValue) { md.RequestModel = v.GetStringValue() })
	read("gen_ai.operation.name", func(v *v1.AnyValue) { md.OperationName = v.GetStringValue() })
	read("gen_ai.response.model", func(v *v1.AnyValue) { md.ResponseModel = v.GetStringValue() })
	read("gen_ai.response.id", func(v *v1.AnyValue) { md.ResponseID = v.GetStringValue() })

	read("gen_ai.usage.input_tokens", func(v *v1.AnyValue) { md.InputTokens = v.GetIntValue() })
	read("gen_ai.usage.output_tokens", func(v *v1.AnyValue) { md.OutputTokens = v.GetIntValue() })
	read("gen_ai.usage.total_tokens", func(v *v1.AnyValue) { md.TotalTokens = util.ToPtr(v.GetIntValue()) })

	read("gen_ai.response.finish_reasons", func(v *v1.AnyValue) {
		// semconv defines finish_reasons as an array (one entry per choice), but
		// some instrumentations (e.g. LangSmith) emit a single scalar string;
		// both are accepted. Empty entries are dropped and the field is left
		// unset when none remain. Values are stored raw (no tool_call/tool_calls
		// normalization).
		var reasons []string
		if arr := v.GetArrayValue(); arr != nil {
			for _, val := range arr.GetValues() {
				if s := val.GetStringValue(); s != "" {
					reasons = append(reasons, s)
				}
			}
		} else if s := v.GetStringValue(); s != "" {
			reasons = []string{s}
		}
		if len(reasons) > 0 {
			md.FinishReasons = reasons
		}
	})

	// Provider: gen_ai.provider.name is canonical and gen_ai.system is its
	// deprecated predecessor. Read system first so the canonical key overwrites
	// it whenever both are present.
	read("gen_ai.system", func(v *v1.AnyValue) { md.Provider = v.GetStringValue() })
	read("gen_ai.provider.name", func(v *v1.AnyValue) { md.Provider = v.GetStringValue() })

	return foundAny
}
