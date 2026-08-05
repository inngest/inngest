package sdk

import (
	"bytes"

	sdkfeatureobs "github.com/inngest/inngest/proto/gen/sdk_feature_observations/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

type FeatureObservations struct {
	observations *sdkfeatureobs.FeatureObservations
}

func FeatureObservationsFromProto(observations *sdkfeatureobs.FeatureObservations) *FeatureObservations {
	if observations == nil {
		return nil
	}

	return &FeatureObservations{observations: observations}
}

func (f *FeatureObservations) Proto() *sdkfeatureobs.FeatureObservations {
	if f == nil || f.observations == nil {
		return nil
	}

	return f.observations
}

func (f *FeatureObservations) HasAny() bool {
	return f.GetAiMetadataExtraction() != nil || f.GetExtendedTraces() != nil || f.GetSendEvents() != nil
}

func (f *FeatureObservations) GetAiMetadataExtraction() *sdkfeatureobs.AIMetadataExtraction {
	if f == nil {
		return nil
	}

	return f.Proto().GetAiMetadataExtraction()
}

func (f *FeatureObservations) GetExtendedTraces() *sdkfeatureobs.ExtendedTraces {
	if f == nil {
		return nil
	}

	return f.Proto().GetExtendedTraces()
}

func (f *FeatureObservations) GetSendEvents() *sdkfeatureobs.SendEvents {
	if f == nil {
		return nil
	}

	return f.Proto().GetSendEvents()
}

const (
	AIMetadataExtractionReadinessReasonUnknown                   = int(sdkfeatureobs.AIMetadataExtractionReadinessReason_AI_METADATA_EXTRACTION_READINESS_REASON_UNSPECIFIED)
	AIMetadataExtractionReadinessReasonReady                     = int(sdkfeatureobs.AIMetadataExtractionReadinessReason_AI_METADATA_EXTRACTION_READINESS_REASON_READY)
	AIMetadataExtractionReadinessReasonDisabledByUser            = int(sdkfeatureobs.AIMetadataExtractionReadinessReason_AI_METADATA_EXTRACTION_READINESS_REASON_DISABLED_BY_USER)
	AIMetadataExtractionReadinessReasonOtelProviderMissing       = int(sdkfeatureobs.AIMetadataExtractionReadinessReason_AI_METADATA_EXTRACTION_READINESS_REASON_OTEL_PROVIDER_MISSING)
	AIMetadataExtractionReadinessReasonOtelSpanProcessorNotAdded = int(sdkfeatureobs.AIMetadataExtractionReadinessReason_AI_METADATA_EXTRACTION_READINESS_REASON_OTEL_SPAN_PROCESSOR_NOT_ADDED)
)

const (
	ExtendedTracesReadinessReasonUnknown                    = int(sdkfeatureobs.ExtendedTracesReadinessReason_EXTENDED_TRACES_READINESS_REASON_UNSPECIFIED)
	ExtendedTracesReadinessReasonReady                      = int(sdkfeatureobs.ExtendedTracesReadinessReason_EXTENDED_TRACES_READINESS_REASON_READY)
	ExtendedTracesReadinessReasonNotEnabledByUser           = int(sdkfeatureobs.ExtendedTracesReadinessReason_EXTENDED_TRACES_READINESS_REASON_NOT_ENABLED_BY_USER)
	ExtendedTracesReadinessReasonDisabledByUser             = int(sdkfeatureobs.ExtendedTracesReadinessReason_EXTENDED_TRACES_READINESS_REASON_DISABLED_BY_USER)
	ExtendedTracesReadinessReasonOtelProviderMissing        = int(sdkfeatureobs.ExtendedTracesReadinessReason_EXTENDED_TRACES_READINESS_REASON_OTEL_PROVIDER_MISSING)
	ExtendedTracesReadinessReasonOtelSpanProcessorNotAdded  = int(sdkfeatureobs.ExtendedTracesReadinessReason_EXTENDED_TRACES_READINESS_REASON_OTEL_SPAN_PROCESSOR_NOT_ADDED)
	ExtendedTracesReadinessReasonOtelProviderCreationFailed = int(sdkfeatureobs.ExtendedTracesReadinessReason_EXTENDED_TRACES_READINESS_REASON_OTEL_PROVIDER_CREATION_FAILED)
)

func (f *FeatureObservations) MarshalJSON() ([]byte, error) {
	if f == nil {
		return []byte("null"), nil
	}

	marshal := protojson.MarshalOptions{UseEnumNumbers: true}
	return marshal.Marshal(f.Proto())
}

func (f *FeatureObservations) UnmarshalJSON(byt []byte) error {
	byt = bytes.TrimSpace(byt)
	if bytes.Equal(byt, []byte("null")) {
		f.observations = nil
		return nil
	}

	unmarshal := protojson.UnmarshalOptions{DiscardUnknown: true}
	observations := &sdkfeatureobs.FeatureObservations{}
	if err := unmarshal.Unmarshal(byt, observations); err != nil {
		return err
	}

	f.observations = observations
	return nil
}
