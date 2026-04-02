package openapi3

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	"github.com/go-openapi/jsonpointer"
)

// T is the root of an OpenAPI v3 document
// See https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.0.3.md#openapi-object
type T struct {
	Extensions map[string]any `json:"-" yaml:"-"`

	OpenAPI      string               `json:"openapi" yaml:"openapi"` // Required
	Components   *Components          `json:"components,omitempty" yaml:"components,omitempty"`
	Info         *Info                `json:"info" yaml:"info"`   // Required
	Paths        *Paths               `json:"paths" yaml:"paths"` // Required
	Security     SecurityRequirements `json:"security,omitempty" yaml:"security,omitempty"`
	Servers      Servers              `json:"servers,omitempty" yaml:"servers,omitempty"`
	Tags         Tags                 `json:"tags,omitempty" yaml:"tags,omitempty"`
	ExternalDocs *ExternalDocs        `json:"externalDocs,omitempty" yaml:"externalDocs,omitempty"`

	visited visitedComponent
	url     *url.URL

	// Document-scoped format validators
	// These validators are automatically used by all schemas in this document
	stringFormats  map[string]StringFormatValidator
	numberFormats  map[string]NumberFormatValidator
	integerFormats map[string]IntegerFormatValidator
}

var _ jsonpointer.JSONPointable = (*T)(nil)

// JSONLookup implements https://pkg.go.dev/github.com/go-openapi/jsonpointer#JSONPointable
func (doc *T) JSONLookup(token string) (any, error) {
	switch token {
	case "openapi":
		return doc.OpenAPI, nil
	case "components":
		return doc.Components, nil
	case "info":
		return doc.Info, nil
	case "paths":
		return doc.Paths, nil
	case "security":
		return doc.Security, nil
	case "servers":
		return doc.Servers, nil
	case "tags":
		return doc.Tags, nil
	case "externalDocs":
		return doc.ExternalDocs, nil
	}

	v, _, err := jsonpointer.GetForToken(doc.Extensions, token)
	return v, err
}

// MarshalJSON returns the JSON encoding of T.
func (doc *T) MarshalJSON() ([]byte, error) {
	x, err := doc.MarshalYAML()
	if err != nil {
		return nil, err
	}
	return json.Marshal(x)
}

// MarshalYAML returns the YAML encoding of T.
func (doc *T) MarshalYAML() (any, error) {
	if doc == nil {
		return nil, nil
	}
	m := make(map[string]any, 4+len(doc.Extensions))
	for k, v := range doc.Extensions {
		m[k] = v
	}
	m["openapi"] = doc.OpenAPI
	if x := doc.Components; x != nil {
		m["components"] = x
	}
	m["info"] = doc.Info
	m["paths"] = doc.Paths
	if x := doc.Security; len(x) != 0 {
		m["security"] = x
	}
	if x := doc.Servers; len(x) != 0 {
		m["servers"] = x
	}
	if x := doc.Tags; len(x) != 0 {
		m["tags"] = x
	}
	if x := doc.ExternalDocs; x != nil {
		m["externalDocs"] = x
	}
	return m, nil
}

// UnmarshalJSON sets T to a copy of data.
func (doc *T) UnmarshalJSON(data []byte) error {
	type TBis T
	var x TBis
	if err := json.Unmarshal(data, &x); err != nil {
		return unmarshalError(err)
	}
	_ = json.Unmarshal(data, &x.Extensions)
	delete(x.Extensions, "openapi")
	delete(x.Extensions, "components")
	delete(x.Extensions, "info")
	delete(x.Extensions, "paths")
	delete(x.Extensions, "security")
	delete(x.Extensions, "servers")
	delete(x.Extensions, "tags")
	delete(x.Extensions, "externalDocs")
	if len(x.Extensions) == 0 {
		x.Extensions = nil
	}
	*doc = T(x)
	return nil
}

func (doc *T) AddOperation(path string, method string, operation *Operation) {
	if doc.Paths == nil {
		doc.Paths = NewPaths()
	}
	pathItem := doc.Paths.Value(path)
	if pathItem == nil {
		pathItem = &PathItem{}
		doc.Paths.Set(path, pathItem)
	}
	pathItem.SetOperation(method, operation)
}

func (doc *T) AddServer(server *Server) {
	doc.Servers = append(doc.Servers, server)
}

func (doc *T) AddServers(servers ...*Server) {
	doc.Servers = append(doc.Servers, servers...)
}

// SetStringFormatValidators sets document-scoped string format validators.
// These validators are automatically used by all schemas in this document.
func (doc *T) SetStringFormatValidators(validators map[string]StringFormatValidator) {
	doc.stringFormats = validators
}

// SetStringFormatValidator sets a single document-scoped string format validator.
func (doc *T) SetStringFormatValidator(name string, validator StringFormatValidator) {
	if doc.stringFormats == nil {
		doc.stringFormats = make(map[string]StringFormatValidator)
	}
	doc.stringFormats[name] = validator
}

// SetNumberFormatValidators sets document-scoped number format validators.
// These validators are automatically used by all schemas in this document.
func (doc *T) SetNumberFormatValidators(validators map[string]NumberFormatValidator) {
	doc.numberFormats = validators
}

// SetNumberFormatValidator sets a single document-scoped number format validator.
func (doc *T) SetNumberFormatValidator(name string, validator NumberFormatValidator) {
	if doc.numberFormats == nil {
		doc.numberFormats = make(map[string]NumberFormatValidator)
	}
	doc.numberFormats[name] = validator
}

// SetIntegerFormatValidators sets document-scoped integer format validators.
// These validators are automatically used by all schemas in this document.
func (doc *T) SetIntegerFormatValidators(validators map[string]IntegerFormatValidator) {
	doc.integerFormats = validators
}

// SetIntegerFormatValidator sets a single document-scoped integer format validator.
func (doc *T) SetIntegerFormatValidator(name string, validator IntegerFormatValidator) {
	if doc.integerFormats == nil {
		doc.integerFormats = make(map[string]IntegerFormatValidator)
	}
	doc.integerFormats[name] = validator
}

// GetSchemaValidationOptions returns SchemaValidationOptions that include
// this document's format validators. Use this when validating schemas from this document.
func (doc *T) GetSchemaValidationOptions() []SchemaValidationOption {
	var opts []SchemaValidationOption
	if doc.stringFormats != nil {
		opts = append(opts, WithStringFormatValidators(doc.stringFormats))
	}
	if doc.numberFormats != nil {
		opts = append(opts, WithNumberFormatValidators(doc.numberFormats))
	}
	if doc.integerFormats != nil {
		opts = append(opts, WithIntegerFormatValidators(doc.integerFormats))
	}
	return opts
}

// Validate returns an error if T does not comply with the OpenAPI spec.
// Validations Options can be provided to modify the validation behavior.
func (doc *T) Validate(ctx context.Context, opts ...ValidationOption) error {
	ctx = WithValidationOptions(ctx, opts...)

	if doc.OpenAPI == "" {
		return errors.New("value of openapi must be a non-empty string")
	}

	var wrap func(error) error

	wrap = func(e error) error { return fmt.Errorf("invalid components: %w", e) }
	if v := doc.Components; v != nil {
		if err := v.Validate(ctx); err != nil {
			return wrap(err)
		}
	}

	wrap = func(e error) error { return fmt.Errorf("invalid info: %w", e) }
	if v := doc.Info; v != nil {
		if err := v.Validate(ctx); err != nil {
			return wrap(err)
		}
	} else {
		return wrap(errors.New("must be an object"))
	}

	wrap = func(e error) error { return fmt.Errorf("invalid paths: %w", e) }
	if v := doc.Paths; v != nil {
		if err := v.Validate(ctx); err != nil {
			return wrap(err)
		}
	} else {
		return wrap(errors.New("must be an object"))
	}

	wrap = func(e error) error { return fmt.Errorf("invalid security: %w", e) }
	if v := doc.Security; v != nil {
		if err := v.Validate(ctx); err != nil {
			return wrap(err)
		}
	}

	wrap = func(e error) error { return fmt.Errorf("invalid servers: %w", e) }
	if v := doc.Servers; v != nil {
		if err := v.Validate(ctx); err != nil {
			return wrap(err)
		}
	}

	wrap = func(e error) error { return fmt.Errorf("invalid tags: %w", e) }
	if v := doc.Tags; v != nil {
		if err := v.Validate(ctx); err != nil {
			return wrap(err)
		}
	}

	wrap = func(e error) error { return fmt.Errorf("invalid external docs: %w", e) }
	if v := doc.ExternalDocs; v != nil {
		if err := v.Validate(ctx); err != nil {
			return wrap(err)
		}
	}

	return validateExtensions(ctx, doc.Extensions)
}

// ValidateSchemaJSON validates data against a schema using this document's format validators.
// This is a convenience method that automatically applies the document's format validators.
func (doc *T) ValidateSchemaJSON(schema *Schema, value any, opts ...SchemaValidationOption) error {
	// Combine document's validators with any additional options
	allOpts := append(doc.GetSchemaValidationOptions(), opts...)
	return schema.VisitJSON(value, allOpts...)
}
