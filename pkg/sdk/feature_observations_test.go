package sdk

import (
	"encoding/json"
	"testing"

	sdkfeatureobs "github.com/inngest/inngest/proto/gen/sdk_feature_observations/v1"
	"github.com/stretchr/testify/require"
)

func TestFeatureObservationsJSON(t *testing.T) {
	r := require.New(t)

	observations := FeatureObservationsFromProto(&sdkfeatureobs.FeatureObservations{
		AiMetadataExtraction: &sdkfeatureobs.AIMetadataExtraction{
			ReadinessReason: sdkfeatureobs.AIMetadataExtractionReadinessReason_AI_METADATA_EXTRACTION_READINESS_REASON_OTEL_PROVIDER_MISSING,
			OtelSetup: &sdkfeatureobs.OTelSetup{
				ProviderFound:             true,
				ProviderSource:            sdkfeatureobs.OTelProviderSource_OTEL_PROVIDER_SOURCE_FIRST_PARTY,
				AddSpanProcessorAttempted: true,
				Failure:                   sdkfeatureobs.OTelSetupFailure_OTEL_SETUP_FAILURE_NO_PROVIDER,
			},
		},
		ExtendedTraces: &sdkfeatureobs.ExtendedTraces{
			ReadinessReason: sdkfeatureobs.ExtendedTracesReadinessReason_EXTENDED_TRACES_READINESS_REASON_NOT_ENABLED_BY_USER,
			Config: &sdkfeatureobs.ExtendedTracesConfig{
				Behavior: sdkfeatureobs.ExtendedTracesBehavior_EXTENDED_TRACES_BEHAVIOR_AUTO,
			},
		},
	})

	byt, err := json.Marshal(observations)
	r.NoError(err)
	r.JSONEq(`{
		"aiMetadataExtraction": {
			"readinessReason": 3,
			"otelSetup": {
				"providerFound": true,
				"providerSource": 1,
				"addSpanProcessorAttempted": true,
				"failure": 1
			}
		},
		"extendedTraces": {
			"readinessReason": 2,
			"config": {
				"behavior": 3
			}
		}
	}`, string(byt))

	var decoded FeatureObservations
	r.NoError(json.Unmarshal(byt, &decoded))
	r.Equal(int(sdkfeatureobs.AIMetadataExtractionReadinessReason_AI_METADATA_EXTRACTION_READINESS_REASON_OTEL_PROVIDER_MISSING), int(decoded.GetAiMetadataExtraction().GetReadinessReason()))
	r.Equal(int(sdkfeatureobs.ExtendedTracesBehavior_EXTENDED_TRACES_BEHAVIOR_AUTO), int(decoded.GetExtendedTraces().GetConfig().GetBehavior()))
}

func TestFeatureObservationsJSONAcceptsEnumNames(t *testing.T) {
	var observations FeatureObservations
	err := json.Unmarshal([]byte(`{
		"sendEvents": {
			"readinessReason": "SEND_EVENTS_READINESS_REASON_READY",
			"config": {
				"eventKeyConfigured": true,
				"eventApiOriginOverrideConfigured": true
			}
		}
	}`), &observations)

	require.NoError(t, err)
	require.Equal(
		t,
		sdkfeatureobs.SendEventsReadinessReason_SEND_EVENTS_READINESS_REASON_READY,
		observations.GetSendEvents().GetReadinessReason(),
	)
	require.True(t, observations.GetSendEvents().GetConfig().GetEventKeyConfigured())
	require.True(t, observations.GetSendEvents().GetConfig().GetEventApiOriginOverrideConfigured())
}
