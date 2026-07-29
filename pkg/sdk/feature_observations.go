package sdk

import (
	"bytes"
	"encoding/json"

	sdkfeatureobs "github.com/inngest/inngest/proto/gen/sdk_feature_observations/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

type FeatureObservations []*sdkfeatureobs.FeatureObservation

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

func (f FeatureObservations) MarshalJSON() ([]byte, error) {
	if f == nil {
		return []byte("null"), nil
	}

	marshal := protojson.MarshalOptions{UseEnumNumbers: true}
	buf := bytes.NewBufferString("[")
	for i, observation := range f {
		if i > 0 {
			buf.WriteByte(',')
		}

		if observation == nil {
			buf.WriteString("null")
			continue
		}

		byt, err := marshal.Marshal(observation)
		if err != nil {
			return nil, err
		}
		buf.Write(byt)
	}
	buf.WriteByte(']')

	return buf.Bytes(), nil
}

func (f *FeatureObservations) UnmarshalJSON(byt []byte) error {
	byt = bytes.TrimSpace(byt)
	if bytes.Equal(byt, []byte("null")) {
		*f = nil
		return nil
	}

	var rawObservations []json.RawMessage
	if err := json.Unmarshal(byt, &rawObservations); err != nil {
		return err
	}

	unmarshal := protojson.UnmarshalOptions{DiscardUnknown: true}
	observations := make(FeatureObservations, 0, len(rawObservations))
	for _, raw := range rawObservations {
		raw = bytes.TrimSpace(raw)
		if bytes.Equal(raw, []byte("null")) {
			observations = append(observations, nil)
			continue
		}

		observation := &sdkfeatureobs.FeatureObservation{}
		if err := unmarshal.Unmarshal(raw, observation); err != nil {
			return err
		}
		observations = append(observations, observation)
	}

	*f = observations
	return nil
}

func (f RegisterRequest) AIMetadataExtractionReadinessReason() *int {
	for _, observation := range f.FeatureObservations {
		aiMetadataExtraction := observation.GetAiMetadataExtraction()
		if aiMetadataExtraction == nil {
			continue
		}

		reason := int(aiMetadataExtraction.GetReadinessReason())
		return &reason
	}

	return nil
}

func (f RegisterRequest) ExtendedTracesReadinessReason() *int {
	for _, observation := range f.FeatureObservations {
		extendedTraces := observation.GetExtendedTraces()
		if extendedTraces == nil {
			continue
		}

		reason := int(extendedTraces.GetReadinessReason())
		return &reason
	}

	return nil
}
