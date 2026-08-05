package resolvers

import (
	"encoding/json"
	"testing"

	"github.com/inngest/inngest/pkg/registration"
	"github.com/inngest/inngest/pkg/sdk"
	sdkfeatureobs "github.com/inngest/inngest/proto/gen/sdk_feature_observations/v1"
	"github.com/stretchr/testify/require"
)

func TestSdkFeatureReadinessFromMetadata(t *testing.T) {
	r := require.New(t)

	metadata := appMetadataForRegisterRequest(t, sdk.RegisterRequest{
		FeatureObservations: sdk.FeatureObservationsFromProto(&sdkfeatureobs.FeatureObservations{
			AiMetadataExtraction: &sdkfeatureobs.AIMetadataExtraction{
				ReadinessReason: sdkfeatureobs.AIMetadataExtractionReadinessReason_AI_METADATA_EXTRACTION_READINESS_REASON_READY,
			},
			ExtendedTraces: &sdkfeatureobs.ExtendedTraces{
				ReadinessReason: sdkfeatureobs.ExtendedTracesReadinessReason_EXTENDED_TRACES_READINESS_REASON_OTEL_PROVIDER_MISSING,
			},
		}),
	})

	readiness, err := sdkFeatureReadinessFromMetadata(metadata)
	r.NoError(err)
	r.NotNil(readiness)
	r.NotNil(readiness.AiMetadataExtraction)
	r.True(readiness.AiMetadataExtraction.Ready)
	r.NotNil(readiness.AiMetadataExtraction.Reason)
	r.Equal(1, *readiness.AiMetadataExtraction.Reason)
	r.NotNil(readiness.ExtendedTraces)
	r.False(readiness.ExtendedTraces.Ready)
	r.NotNil(readiness.ExtendedTraces.Reason)
	r.Equal(4, *readiness.ExtendedTraces.Reason)

	readiness, err = sdkFeatureReadinessFromMetadata(nil)
	r.NoError(err)
	r.NotNil(readiness)
	r.Nil(readiness.AiMetadataExtraction)
	r.Nil(readiness.ExtendedTraces)

	status := sdkFeatureStatusFromReadinessReason(2, sdk.AIMetadataExtractionReadinessReasonReady)
	r.False(status.Ready)
	r.NotNil(status.Reason)
	r.Equal(2, *status.Reason)
}

func appMetadataForRegisterRequest(t *testing.T, r sdk.RegisterRequest) map[string]string {
	t.Helper()

	var metadata map[string]string
	require.NoError(t, json.Unmarshal([]byte(registration.AppMetadataForRegisterRequest(r)), &metadata))
	return metadata
}
