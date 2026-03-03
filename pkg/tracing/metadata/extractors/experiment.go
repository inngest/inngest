package extractors

import (
	"context"
	"encoding/json"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/tracing/metadata"
	tracev1 "go.opentelemetry.io/proto/otlp/trace/v1"
)

//tygo:generate
const (
	KindInngestExperiment metadata.Kind = "inngest.experiment"
)

//tygo:generate
type ExperimentMetadata struct {
	ExperimentName    string         `json:"experiment_name"`
	VariantSelected   string         `json:"variant_selected"`
	SelectionStrategy string         `json:"selection_strategy"`
	AvailableVariants []string       `json:"available_variants"`
	VariantWeights    map[string]int `json:"variant_weights,omitempty"`
}

func (ms ExperimentMetadata) Kind() metadata.Kind {
	return KindInngestExperiment
}

func (ms ExperimentMetadata) Op() metadata.Opcode {
	return enums.MetadataOpcodeMerge
}

func (ms ExperimentMetadata) Serialize() (metadata.Values, error) {
	var rawMetadata metadata.Values
	err := rawMetadata.FromStruct(ms)
	if err != nil {
		return nil, err
	}

	return rawMetadata, nil
}

type ExperimentMetadataExtractor struct{}

func NewExperimentMetadataExtractor() *ExperimentMetadataExtractor {
	return &ExperimentMetadataExtractor{}
}

func (e *ExperimentMetadataExtractor) ExtractSpanMetadata(ctx context.Context, span *tracev1.Span) ([]metadata.Structured, error) {
	if !e.isExperimentSpan(span) {
		return nil, nil
	}

	md := e.extractExperimentMetadata(span)
	return []metadata.Structured{md}, nil
}

var experimentAttributeKeys = map[string]bool{
	"inngest.experiment.name":               true,
	"inngest.experiment.variant_selected":    true,
	"inngest.experiment.selection_strategy":  true,
	"inngest.experiment.available_variants":  true,
	"inngest.experiment.variant_weights":     true,
}

func (e *ExperimentMetadataExtractor) isExperimentSpan(span *tracev1.Span) bool {
	for _, attr := range span.Attributes {
		if experimentAttributeKeys[attr.Key] {
			return true
		}
	}
	return false
}

func (e *ExperimentMetadataExtractor) extractExperimentMetadata(span *tracev1.Span) ExperimentMetadata {
	var md ExperimentMetadata

	for _, attr := range span.Attributes {
		switch attr.Key {
		case "inngest.experiment.name":
			md.ExperimentName = attr.Value.GetStringValue()
		case "inngest.experiment.variant_selected":
			md.VariantSelected = attr.Value.GetStringValue()
		case "inngest.experiment.selection_strategy":
			md.SelectionStrategy = attr.Value.GetStringValue()
		case "inngest.experiment.available_variants":
			if arrVal := attr.Value.GetArrayValue(); arrVal != nil {
				for _, v := range arrVal.GetValues() {
					if s := v.GetStringValue(); s != "" {
						md.AvailableVariants = append(md.AvailableVariants, s)
					}
				}
			}
		case "inngest.experiment.variant_weights":
			// Variant weights are stored as a JSON-encoded string attribute
			if s := attr.Value.GetStringValue(); s != "" {
				var weights map[string]int
				if err := json.Unmarshal([]byte(s), &weights); err == nil {
					md.VariantWeights = weights
				}
			}
		}
	}

	return md
}
