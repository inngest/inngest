package sdk

import (
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
		withObservations.FeatureObservations = FeatureObservations{
			{
				Feature: &sdkfeatureobs.FeatureObservation_AiMetadataExtraction{
					AiMetadataExtraction: &sdkfeatureobs.AIMetadataExtraction{
						ReadinessReason: sdkfeatureobs.AIMetadataExtractionReadinessReason_AI_METADATA_EXTRACTION_READINESS_REASON_OTEL_PROVIDER_MISSING,
					},
				},
			},
		}

		withoutChecksum, err := withoutObservations.Checksum()
		r.NoError(err)

		withChecksum, err := withObservations.Checksum()
		r.NoError(err)
		r.Equal(withoutChecksum, withChecksum)
	})
}
