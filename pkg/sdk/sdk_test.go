package sdk

import (
	"encoding/json"
	"testing"

	sdkfeatureobs "github.com/inngest/inngest/proto/gen/sdk_feature_observations/v1"
	"github.com/stretchr/testify/require"
)

func TestRegisterRequestChecksum(t *testing.T) {
	t.Run("excludes feature observations", func(t *testing.T) {
		r := require.New(t)

		base := RegisterRequest{
			V:       "1",
			URL:     "http://localhost:3000/api/inngest",
			AppName: "test-app",
			SDK:     "js:v4.13.0",
		}

		withoutObservations := base
		withObservations := base
		withObservations.FeatureObservations = FeatureObservationsFromProto(&sdkfeatureobs.FeatureObservations{
			AiMetadataExtraction: &sdkfeatureobs.AIMetadataExtraction{
				ReadinessReason: sdkfeatureobs.AIMetadataExtractionReadinessReason_AI_METADATA_EXTRACTION_READINESS_REASON_OTEL_PROVIDER_MISSING,
			},
		})

		withoutChecksum, err := withoutObservations.Checksum()
		r.NoError(err)

		withChecksum, err := withObservations.Checksum()
		r.NoError(err)
		r.Equal(withoutChecksum, withChecksum)
	})
}

func TestRegisterRequestFeatureObservationsJSONKey(t *testing.T) {
	r := require.New(t)

	req := RegisterRequest{
		FeatureObservations: FeatureObservationsFromProto(&sdkfeatureobs.FeatureObservations{
			AiMetadataExtraction: &sdkfeatureobs.AIMetadataExtraction{
				ReadinessReason: sdkfeatureobs.AIMetadataExtractionReadinessReason_AI_METADATA_EXTRACTION_READINESS_REASON_READY,
			},
		}),
	}

	byt, err := json.Marshal(req)
	r.NoError(err)
	r.Contains(string(byt), `"featureObservations"`)
	r.NotContains(string(byt), `"feature_observations"`)

	var decoded RegisterRequest
	r.NoError(json.Unmarshal([]byte(`{
		"featureObservations": {
			"aiMetadataExtraction": {
				"readinessReason": 1
			}
		}
	}`), &decoded))
	r.NotNil(decoded.FeatureObservations)
	r.Equal(
		sdkfeatureobs.AIMetadataExtractionReadinessReason_AI_METADATA_EXTRACTION_READINESS_REASON_READY,
		decoded.FeatureObservations.GetAiMetadataExtraction().GetReadinessReason(),
	)
}
