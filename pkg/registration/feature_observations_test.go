package registration

import (
	"encoding/json"
	"testing"

	"github.com/inngest/inngest/pkg/sdk"
	sdkfeatureobs "github.com/inngest/inngest/proto/gen/sdk_feature_observations/v1"
	"github.com/stretchr/testify/require"
)

func TestFeatureObservationsAppMetadata(t *testing.T) {
	r := require.New(t)

	registerReq := sdk.RegisterRequest{
		FeatureObservations: sdk.FeatureObservationsFromProto(&sdkfeatureobs.FeatureObservations{
			AiMetadataExtraction: &sdkfeatureobs.AIMetadataExtraction{
				ReadinessReason: sdkfeatureobs.AIMetadataExtractionReadinessReason_AI_METADATA_EXTRACTION_READINESS_REASON_READY,
			},
		}),
	}

	var metadata map[string]string
	r.NoError(json.Unmarshal([]byte(AppMetadataForRegisterRequest(registerReq)), &metadata))

	observations, err := FeatureObservationsFromAppMetadata(metadata)
	r.NoError(err)
	r.NotNil(observations)
	r.Equal(
		sdkfeatureobs.AIMetadataExtractionReadinessReason_AI_METADATA_EXTRACTION_READINESS_REASON_READY,
		observations.GetAiMetadataExtraction().GetReadinessReason(),
	)

	observations, err = FeatureObservationsFromAppMetadata(nil)
	r.NoError(err)
	r.Nil(observations)

	invalidMetadata := map[string]string{}
	byt, err := json.Marshal(appMetadata{FeatureObservations: "nope"})
	r.NoError(err)
	r.NoError(json.Unmarshal(byt, &invalidMetadata))

	_, err = FeatureObservationsFromAppMetadata(invalidMetadata)
	r.Error(err)
}
