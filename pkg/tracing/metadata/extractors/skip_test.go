package extractors

import (
	"context"
	"testing"
	"github.com/stretchr/testify/require"
	commonv1 "go.opentelemetry.io/proto/otlp/common/v1"
	tracev1 "go.opentelemetry.io/proto/otlp/trace/v1"
)

func TestSkipMetadataExtractor(t *testing.T) {
	extractor := NewSkipMetadataExtractor()
	
	t.Run("extracts skip reason from inngest/function span", func(t *testing.T) {
		span := &tracev1.Span{
			Name: "inngest/function",
			Attributes: []*commonv1.KeyValue{{
				Key: "sys.function.skip.reason",
				Value: &commonv1.AnyValue{
					Value: &commonv1.AnyValue_StringValue{
						StringValue: "function_paused",
					},
				},
			}},
		}
		
		metadata, err := extractor.ExtractSpanMetadata(context.Background(), span)
		require.NoError(t, err)
		require.Len(t, metadata, 1)
		require.Equal(t, "function_paused", metadata[0].(SkipMetadata).Reason)
	})
	
	t.Run("returns nil when skip reason attribute not found", func(t *testing.T) {
		span := &tracev1.Span{
			Name: "inngest/function",
			Attributes: []*commonv1.KeyValue{{
				Key: "other.attribute",
				Value: &commonv1.AnyValue{
					Value: &commonv1.AnyValue_StringValue{
						StringValue: "value",
					},
				},
			}},
		}
		
		metadata, err := extractor.ExtractSpanMetadata(context.Background(), span)
		require.NoError(t, err)
		require.Nil(t, metadata)
	})
}