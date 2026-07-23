package extractors

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/tracing/metadata"
	"github.com/inngest/inngest/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAIMetadataEnrich(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		md   AIMetadata
		opts AIEnrichOpts
		want AIMetadata
	}{
		{
			name: "gap-fills total tokens from input and output",
			md: AIMetadata{
				InputTokens:  100,
				OutputTokens: 50,
			},
			want: AIMetadata{
				InputTokens:  100,
				OutputTokens: 50,
				TotalTokens:  util.ToPtr[int64](150),
			},
		},
		{
			name: "no total tokens when both counts are zero",
			md:   AIMetadata{RequestModel: "some-unknown-model"},
			want: AIMetadata{RequestModel: "some-unknown-model"},
		},
		{
			name: "preserves provided total tokens",
			md: AIMetadata{
				InputTokens:  100,
				OutputTokens: 50,
				TotalTokens:  util.ToPtr[int64](999),
			},
			want: AIMetadata{
				InputTokens:  100,
				OutputTokens: 50,
				TotalTokens:  util.ToPtr[int64](999),
			},
		},
		{
			name: "gap-fills cost from request model",
			md: AIMetadata{
				RequestModel: "gpt-4o",
				InputTokens:  1_000_000,
				OutputTokens: 1_000_000,
			},
			want: AIMetadata{
				RequestModel:  "gpt-4o",
				InputTokens:   1_000_000,
				OutputTokens:  1_000_000,
				TotalTokens:   util.ToPtr[int64](2_000_000),
				EstimatedCost: util.ToPtr(12.50),
			},
		},
		{
			name: "prefers response model over request model for cost",
			md: AIMetadata{
				RequestModel:  "gpt-4o",
				ResponseModel: "gpt-4o-mini",
				InputTokens:   1_000_000,
				OutputTokens:  1_000_000,
			},
			want: AIMetadata{
				RequestModel:  "gpt-4o",
				ResponseModel: "gpt-4o-mini",
				InputTokens:   1_000_000,
				OutputTokens:  1_000_000,
				TotalTokens:   util.ToPtr[int64](2_000_000),
				EstimatedCost: util.ToPtr(0.75),
			},
		},
		{
			name: "preserves provided cost",
			md: AIMetadata{
				RequestModel:  "gpt-4o",
				InputTokens:   1_000_000,
				OutputTokens:  1_000_000,
				EstimatedCost: util.ToPtr(42.0),
			},
			want: AIMetadata{
				RequestModel:  "gpt-4o",
				InputTokens:   1_000_000,
				OutputTokens:  1_000_000,
				TotalTokens:   util.ToPtr[int64](2_000_000),
				EstimatedCost: util.ToPtr(42.0),
			},
		},
		{
			name: "no cost for known model when token counts are zero",
			md: AIMetadata{
				RequestModel: "gpt-4o",
				TotalTokens:  util.ToPtr[int64](150),
			},
			want: AIMetadata{
				RequestModel: "gpt-4o",
				TotalTokens:  util.ToPtr[int64](150),
			},
		},
		{
			name: "no cost for unknown model",
			md: AIMetadata{
				RequestModel: "some-unknown-model",
				InputTokens:  100,
				OutputTokens: 50,
			},
			want: AIMetadata{
				RequestModel: "some-unknown-model",
				InputTokens:  100,
				OutputTokens: 50,
				TotalTokens:  util.ToPtr[int64](150),
			},
		},
		{
			name: "gap-fills latency from fallback",
			md:   AIMetadata{InputTokens: 10},
			opts: AIEnrichOpts{FallbackLatencyMs: 1234},
			want: AIMetadata{
				InputTokens: 10,
				TotalTokens: util.ToPtr[int64](10),
				LatencyMs:   util.ToPtr[int64](1234),
			},
		},
		{
			name: "preserves provided latency",
			md: AIMetadata{
				InputTokens: 10,
				LatencyMs:   util.ToPtr[int64](500),
			},
			opts: AIEnrichOpts{FallbackLatencyMs: 1234},
			want: AIMetadata{
				InputTokens: 10,
				TotalTokens: util.ToPtr[int64](10),
				LatencyMs:   util.ToPtr[int64](500),
			},
		},
		{
			name: "no latency when fallback is zero",
			md:   AIMetadata{InputTokens: 10},
			want: AIMetadata{
				InputTokens: 10,
				TotalTokens: util.ToPtr[int64](10),
			},
		},
		{
			name: "does not touch provider or operation name",
			md: AIMetadata{
				Provider:      "custom-provider",
				OperationName: "embed",
			},
			want: AIMetadata{
				Provider:      "custom-provider",
				OperationName: "embed",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			md := tc.md
			md.Enrich(tc.opts)
			assert.Equal(t, tc.want, md)
		})
	}
}

func TestAIMetadataEnrich_Idempotent(t *testing.T) {
	t.Parallel()

	md := AIMetadata{
		RequestModel: "gpt-4o",
		InputTokens:  100,
		OutputTokens: 50,
	}

	md.Enrich(AIEnrichOpts{FallbackLatencyMs: 1000})
	once := md
	md.Enrich(AIEnrichOpts{FallbackLatencyMs: 9999})

	assert.Equal(t, once, md)
}

func TestEnrichAIValues(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		kind metadata.Kind
		op   metadata.Opcode
		in   metadata.Values
		want metadata.Values
	}{
		{
			name: "adds missing total tokens and cost",
			in: metadata.Values{
				"input_tokens":  json.RawMessage(`1000000`),
				"output_tokens": json.RawMessage(`1000000`),
				"request_model": json.RawMessage(`"gpt-4o"`),
			},
			want: metadata.Values{
				"input_tokens":   json.RawMessage(`1000000`),
				"output_tokens":  json.RawMessage(`1000000`),
				"request_model":  json.RawMessage(`"gpt-4o"`),
				"total_tokens":   json.RawMessage(`2000000`),
				"estimated_cost": json.RawMessage(`12.5`),
			},
		},
		{
			name: "preserves caller keys byte-for-byte",
			in: metadata.Values{
				"input_tokens":  json.RawMessage(`100`),
				"output_tokens": json.RawMessage(`50`),
				"request_model": json.RawMessage(`"gpt-4o"`),
				"custom_key":    json.RawMessage(`{"nested": [1, 2, 3]}`),
				"provider":      json.RawMessage(`"my-provider"`),
			},
			want: metadata.Values{
				"input_tokens":   json.RawMessage(`100`),
				"output_tokens":  json.RawMessage(`50`),
				"request_model":  json.RawMessage(`"gpt-4o"`),
				"custom_key":     json.RawMessage(`{"nested": [1, 2, 3]}`),
				"provider":       json.RawMessage(`"my-provider"`),
				"total_tokens":   json.RawMessage(`150`),
				"estimated_cost": json.RawMessage(`0.00075`),
			},
		},
		{
			name: "never overwrites provided total tokens or cost",
			in: metadata.Values{
				"input_tokens":   json.RawMessage(`100`),
				"output_tokens":  json.RawMessage(`50`),
				"request_model":  json.RawMessage(`"gpt-4o"`),
				"total_tokens":   json.RawMessage(`999`),
				"estimated_cost": json.RawMessage(`42`),
			},
			want: metadata.Values{
				"input_tokens":   json.RawMessage(`100`),
				"output_tokens":  json.RawMessage(`50`),
				"request_model":  json.RawMessage(`"gpt-4o"`),
				"total_tokens":   json.RawMessage(`999`),
				"estimated_cost": json.RawMessage(`42`),
			},
		},
		{
			name: "returns input unchanged on unparseable values",
			in: metadata.Values{
				"input_tokens":  json.RawMessage(`"lots"`),
				"output_tokens": json.RawMessage(`50`),
				"request_model": json.RawMessage(`"gpt-4o"`),
			},
			want: metadata.Values{
				"input_tokens":  json.RawMessage(`"lots"`),
				"output_tokens": json.RawMessage(`50`),
				"request_model": json.RawMessage(`"gpt-4o"`),
			},
		},
		{
			name: "no cost for unknown model",
			in: metadata.Values{
				"input_tokens":  json.RawMessage(`100`),
				"output_tokens": json.RawMessage(`50`),
				"request_model": json.RawMessage(`"some-unknown-model"`),
			},
			want: metadata.Values{
				"input_tokens":  json.RawMessage(`100`),
				"output_tokens": json.RawMessage(`50`),
				"request_model": json.RawMessage(`"some-unknown-model"`),
				"total_tokens":  json.RawMessage(`150`),
			},
		},
		{
			name: "no zero cost when only total tokens are provided",
			in: metadata.Values{
				"request_model": json.RawMessage(`"gpt-4o"`),
				"total_tokens":  json.RawMessage(`150`),
			},
			want: metadata.Values{
				"request_model": json.RawMessage(`"gpt-4o"`),
				"total_tokens":  json.RawMessage(`150`),
			},
		},
		{
			name: "no latency added from enrichment",
			in: metadata.Values{
				"input_tokens": json.RawMessage(`100`),
			},
			want: metadata.Values{
				"input_tokens": json.RawMessage(`100`),
				"total_tokens": json.RawMessage(`100`),
			},
		},
		{
			name: "delete op values untouched",
			op:   enums.MetadataOpcodeDelete,
			in: metadata.Values{
				"input_tokens":  json.RawMessage(`100`),
				"output_tokens": json.RawMessage(`50`),
			},
			want: metadata.Values{
				"input_tokens":  json.RawMessage(`100`),
				"output_tokens": json.RawMessage(`50`),
			},
		},
		{
			name: "non-AI kind untouched",
			kind: metadata.Kind("custom"),
			in: metadata.Values{
				"input_tokens":  json.RawMessage(`100`),
				"output_tokens": json.RawMessage(`50`),
			},
			want: metadata.Values{
				"input_tokens":  json.RawMessage(`100`),
				"output_tokens": json.RawMessage(`50`),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			original := make(metadata.Values, len(tc.in))
			for k, v := range tc.in {
				original[k] = append(json.RawMessage(nil), v...)
			}

			kind := tc.kind
			if kind == "" {
				kind = KindInngestAI
			}
			got := EnrichAIValues(context.Background(), kind, tc.op, tc.in)

			require.Equal(t, tc.want, got)
			assert.Equal(t, original, tc.in, "input must never be mutated")
		})
	}
}
