package registration

import (
	"encoding/json"
	"fmt"

	"github.com/inngest/inngest/pkg/sdk"
)

type appMetadata struct {
	FeatureObservations string `json:"featureObservations,omitempty"`
}

// AppMetadataForRegisterRequest adapts protobuf-backed SDK observations to the
// Dev Server's app metadata storage, which is currently map[string]string JSON.
// The single metadata value is a protobuf JSON object; GraphQL derives any
// feature-specific readiness fields from that source.
func AppMetadataForRegisterRequest(r sdk.RegisterRequest) string {
	metadata := appMetadata{}

	if r.FeatureObservations != nil && r.FeatureObservations.HasAny() {
		if byt, err := json.Marshal(r.FeatureObservations); err == nil {
			metadata.FeatureObservations = string(byt)
		}
	}

	byt, err := json.Marshal(metadata)
	if err != nil {
		return "{}"
	}

	return string(byt)
}

// FeatureObservationsFromAppMetadata extracts the SDK observations from the
// Dev Server's app metadata representation.
func FeatureObservationsFromAppMetadata(metadata map[string]string) (*sdk.FeatureObservations, error) {
	byt, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("invalid app metadata: %w", err)
	}

	var appMetadata appMetadata
	if err := json.Unmarshal(byt, &appMetadata); err != nil {
		return nil, fmt.Errorf("invalid app metadata: %w", err)
	}

	if appMetadata.FeatureObservations == "" {
		return nil, nil
	}

	observations := &sdk.FeatureObservations{}
	if err := json.Unmarshal([]byte(appMetadata.FeatureObservations), observations); err != nil {
		return nil, fmt.Errorf("invalid SDK feature observations: %w", err)
	}

	return observations, nil
}
