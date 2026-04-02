package openapi3

import (
	"context"
	"encoding/json"
)

//go:generate go run refsgenerator.go

// Ref is specified by OpenAPI/Swagger 3.0 standard.
// See https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.0.3.md#reference-object
type Ref struct {
	Ref        string         `json:"$ref" yaml:"$ref"`
	Extensions map[string]any `json:"-" yaml:"-"`
	Origin     *Origin        `json:"__origin__,omitempty" yaml:"__origin__,omitempty"`
}

// MarshalYAML returns the YAML encoding of Ref.
func (x Ref) MarshalYAML() (any, error) {
	m := make(map[string]any, 1+len(x.Extensions))
	for k, v := range x.Extensions {
		m[k] = v
	}
	if x := x.Ref; x != "" {
		m["$ref"] = x
	}
	return m, nil
}

// MarshalJSON returns the JSON encoding of Ref.
func (x Ref) MarshalJSON() ([]byte, error) {
	y, err := x.MarshalYAML()
	if err != nil {
		return nil, err
	}
	return json.Marshal(y)
}

// Validate returns an error if Extensions does not comply with the OpenAPI spec.
func (e *Ref) Validate(ctx context.Context, opts ...ValidationOption) error {
	ctx = WithValidationOptions(ctx, opts...)
	return validateExtensions(ctx, e.Extensions)
}
