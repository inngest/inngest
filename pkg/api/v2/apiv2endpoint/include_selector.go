package apiv2endpoint

import "fmt"

// IncludeSelector parses the canonical values and compatibility aliases
// generated from an RPC's protobuf descriptors.
type IncludeSelector struct {
	values map[string]string
}

func NewIncludeSelector(values map[string]string) IncludeSelector {
	return IncludeSelector{values: values}
}

func (s IncludeSelector) Parse(value string) (string, error) {
	if parsed, ok := s.values[value]; ok {
		return parsed, nil
	}
	return "", fmt.Errorf("unsupported include value %q", value)
}
