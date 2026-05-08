package openapi3

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
)

// License is specified by OpenAPI/Swagger standard version 3.
// See https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.0.3.md#license-object
// and https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.1.0.md#license-object
type License struct {
	Extensions map[string]any `json:"-" yaml:"-"`
	Origin     *Origin        `json:"-" yaml:"-"`

	Name string `json:"name" yaml:"name"` // Required
	URL  string `json:"url,omitempty" yaml:"url,omitempty"`

	// Identifier is an SPDX license expression for the API (OpenAPI 3.1)
	// Either url or identifier can be specified, not both
	Identifier string `json:"identifier,omitempty" yaml:"identifier,omitempty"` // OpenAPI >=3.1
}

// MarshalJSON returns the JSON encoding of License.
func (license License) MarshalJSON() ([]byte, error) {
	x, err := license.MarshalYAML()
	if err != nil {
		return nil, err
	}
	return json.Marshal(x)
}

// MarshalYAML returns the YAML encoding of License.
func (license License) MarshalYAML() (any, error) {
	m := make(map[string]any, 3+len(license.Extensions))
	maps.Copy(m, license.Extensions)
	m["name"] = license.Name
	if x := license.URL; x != "" {
		m["url"] = x
	}
	if x := license.Identifier; x != "" {
		m["identifier"] = x
	}
	return m, nil
}

// UnmarshalJSON sets License to a copy of data.
func (license *License) UnmarshalJSON(data []byte) error {
	type LicenseBis License
	var x LicenseBis
	if err := json.Unmarshal(data, &x); err != nil {
		return unmarshalError(err)
	}
	_ = json.Unmarshal(data, &x.Extensions)
	delete(x.Extensions, "name")
	delete(x.Extensions, "url")
	delete(x.Extensions, "identifier")
	if len(x.Extensions) == 0 {
		x.Extensions = nil
	}
	*license = License(x)
	return nil
}

// Validate returns an error if License does not comply with the OpenAPI spec.
func (license *License) Validate(ctx context.Context, opts ...ValidationOption) error {
	ctx = WithValidationOptions(ctx, opts...)

	if license.Identifier != "" && !getValidationOptions(ctx).isOpenAPI31OrLater {
		return newLicenseIdentifierFieldFor31Plus(license.Origin)
	}

	if license.Name == "" {
		return newLicenseNameRequired(license.Origin)
	}

	if license.URL != "" && license.Identifier != "" {
		return errors.New("license must not specify both 'url' and 'identifier'")
	}

	return validateExtensions(ctx, license.Extensions)
}
