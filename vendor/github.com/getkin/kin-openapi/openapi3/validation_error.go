package openapi3

import "fmt"

// ValidationError is the embedded base for every typed validation error
// emitted by the document validation walker (T.Validate, Info.Validate,
// Paths.Validate, etc.). Three layers of granularity are exposed; pick
// whichever the caller needs:
//
//  1. Base — *ValidationError. Catchall for "this is a validation
//     issue, here is the message". Reachable from any leaf via the As
//     method that each leaf implements.
//  2. Cluster — types like *RequiredFieldError or
//     *FieldVersionMismatchError. Group families of related failures
//     and expose the family-level metadata (Field, MinVersion, ...).
//     Wrap the underlying leaf via Unwrap, so errors.As can still walk
//     to the leaf.
//  3. Leaf — one type per call site (e.g. *InfoVersionRequired,
//     *LicenseIdentifierFieldFor31Plus). Lets callers match an exact
//     failure point without string comparison.
//
// All three are reachable from the same returned error through
// standard Go error wrapping (errors.As, errors.Is, errors.Unwrap),
// so a caller that only needs "is it a validation error?" stops at
// the base and a caller that wants "is it specifically license.identifier
// being used in 3.0?" matches the leaf.
//
// Backward compatibility: every site that today returns errors.New(msg)
// migrates to a leaf type that embeds ValidationError with Message set
// to the original string. (*ValidationError).Error() returns Message
// unchanged, so existing string-matching consumers see identical output.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

// asValidationError is a small helper used by every leaf type's As
// method to expose the embedded *ValidationError to errors.As. Defined
// once here so the leaf-side boilerplate stays a single line.
func asValidationError(target any, ve *ValidationError) bool {
	t, ok := target.(**ValidationError)
	if !ok {
		return false
	}
	*t = ve
	return true
}

// ---------------------------------------------------------------------
// Cluster types — group families of related failures.
// ---------------------------------------------------------------------

// RequiredFieldError clusters "X must be a non-empty value" failures
// across the spec (info.title, info.version, license.name, the openapi
// version string, server.url). Carries the field path, wraps the
// per-site leaf so callers can match either:
//
//	var rfe *RequiredFieldError
//	if errors.As(err, &rfe) { /* knows the field */ }
//
//	var ivr *InfoVersionRequired
//	if errors.As(err, &ivr) { /* knows it's exactly info.version */ }
type RequiredFieldError struct {
	// Field is the JSON-pointer-style path of the required field
	// (e.g. "info.version", "license.name", "openapi", "server.url").
	Field string
	// Cause is the underlying leaf error. Walked by errors.Unwrap.
	Cause error
	// Origin is the source location of the offending element when the
	// document was loaded with Loader.IncludeOrigin = true. Nil for
	// document-root fields (Loader doesn't track Origin on *T) and on
	// loads where origin tracking was off.
	Origin *Origin
}

func (e *RequiredFieldError) Error() string { return e.Cause.Error() }
func (e *RequiredFieldError) Unwrap() error { return e.Cause }

// SchemaValueError clusters failures of "this schema's <kind> value
// doesn't satisfy the schema's own constraints" — example, default,
// examples[i], etc. checked against the schema during document
// validation. Wraps the underlying error from VisitJSON (a
// *SchemaError or a MultiError of them) so callers can match either:
//
//	var sve *SchemaValueError
//	if errors.As(err, &sve) { /* knows ValueKind = "example" */ }
//
//	var se *SchemaError
//	if errors.As(err, &se) { /* full schema-validation detail */ }
//
// Cause is typed as error (not *SchemaError) because VisitJSON can
// return either a single SchemaError or a MultiError aggregating
// several. errors.As walks both shapes transparently.
type SchemaValueError struct {
	// ValueKind identifies the schema sub-field whose value failed
	// (e.g. "example", "default").
	ValueKind string
	// Cause is the underlying error from schema.VisitJSON — either a
	// *SchemaError or a MultiError of them. Walked by errors.Unwrap.
	Cause error
	// Origin is the source location of the offending element when the
	// document was loaded with Loader.IncludeOrigin = true. Nil when
	// origin tracking is off.
	Origin *Origin
}

func (e *SchemaValueError) Error() string {
	return fmt.Sprintf("invalid %s: %s", e.ValueKind, e.Cause.Error())
}

func (e *SchemaValueError) Unwrap() error { return e.Cause }

// PathParametersError clusters "operation declares fewer/more path
// parameters than appear in the path template" failures. Carries the
// path template, method, and the list of missing parameter names so
// callers can render or filter without parsing the message.
type PathParametersError struct {
	// Path is the path template (e.g. "/api/{domain}/{project}/...").
	Path string
	// Method is the HTTP method (e.g. "POST").
	Method string
	// Missing names path-template variables (or operation parameters)
	// that don't have a corresponding declaration on the other side.
	Missing []string
	// Origin is the source location of the path item when the document
	// was loaded with Loader.IncludeOrigin = true.
	Origin *Origin
}

func (e *PathParametersError) Error() string {
	return fmt.Sprintf("operation %s %s must define exactly all path parameters (missing: %v)",
		e.Method, e.Path, e.Missing)
}

// FieldVersionMismatchError clusters "field X is for OpenAPI >=Y"
// failures (3.1+ keywords used in 3.0 documents). Carries the field
// name and minimum version, wraps the per-site leaf.
type FieldVersionMismatchError struct {
	// Field is the field name flagged (e.g. "summary", "identifier",
	// "$defs", "prefixItems", "contains", ...).
	Field string
	// MinVersion is the minimum OpenAPI version that allows the field
	// (e.g. "3.1").
	MinVersion string
	// Cause is the underlying leaf error. Walked by errors.Unwrap.
	Cause error
	// Origin is the source location of the offending element when the
	// document was loaded with Loader.IncludeOrigin = true. Nil for
	// document-root fields (Loader doesn't track Origin on *T) and on
	// loads where origin tracking was off.
	Origin *Origin
}

func (e *FieldVersionMismatchError) Error() string { return e.Cause.Error() }
func (e *FieldVersionMismatchError) Unwrap() error { return e.Cause }

// ---------------------------------------------------------------------
// Leaf types — one per call site. Each embeds ValidationError for
// Error() and As-to-base, and is wrapped in its cluster type when
// returned from a validator.
//
// Naming convention: <Subject><Action> for required fields,
// <Subject>FieldFor31Plus for 3.1-only fields used in 3.0 docs.
// Subjects use Go-identifier-friendly transliterations of OAS field
// paths ("$defs" -> "Defs", "$dynamicAnchor" -> "DynamicAnchor").
// ---------------------------------------------------------------------

// RequiredFieldError leaves.

type InfoVersionRequired struct{ ValidationError }

func (e *InfoVersionRequired) As(target any) bool {
	return asValidationError(target, &e.ValidationError)
}

type InfoTitleRequired struct{ ValidationError }

func (e *InfoTitleRequired) As(target any) bool {
	return asValidationError(target, &e.ValidationError)
}

type LicenseNameRequired struct{ ValidationError }

func (e *LicenseNameRequired) As(target any) bool {
	return asValidationError(target, &e.ValidationError)
}

type OpenAPIVersionRequired struct{ ValidationError }

func (e *OpenAPIVersionRequired) As(target any) bool {
	return asValidationError(target, &e.ValidationError)
}

type ServerURLRequired struct{ ValidationError }

func (e *ServerURLRequired) As(target any) bool {
	return asValidationError(target, &e.ValidationError)
}

// FieldVersionMismatchError leaves — non-schema fields.

type InfoSummaryFieldFor31Plus struct{ ValidationError }

func (e *InfoSummaryFieldFor31Plus) As(target any) bool {
	return asValidationError(target, &e.ValidationError)
}

type LicenseIdentifierFieldFor31Plus struct{ ValidationError }

func (e *LicenseIdentifierFieldFor31Plus) As(target any) bool {
	return asValidationError(target, &e.ValidationError)
}

type WebhooksFieldFor31Plus struct{ ValidationError }

func (e *WebhooksFieldFor31Plus) As(target any) bool {
	return asValidationError(target, &e.ValidationError)
}

type JSONSchemaDialectFieldFor31Plus struct{ ValidationError }

func (e *JSONSchemaDialectFieldFor31Plus) As(target any) bool {
	return asValidationError(target, &e.ValidationError)
}

// FieldVersionMismatchError leaves — schema fields (rejected by
// schema.go's reject() helper when a 3.0 doc uses 3.1 keywords).

type ConstFieldFor31Plus struct{ ValidationError }

func (e *ConstFieldFor31Plus) As(target any) bool {
	return asValidationError(target, &e.ValidationError)
}

type ExamplesFieldFor31Plus struct{ ValidationError }

func (e *ExamplesFieldFor31Plus) As(target any) bool {
	return asValidationError(target, &e.ValidationError)
}

type PrefixItemsFieldFor31Plus struct{ ValidationError }

func (e *PrefixItemsFieldFor31Plus) As(target any) bool {
	return asValidationError(target, &e.ValidationError)
}

type ContainsFieldFor31Plus struct{ ValidationError }

func (e *ContainsFieldFor31Plus) As(target any) bool {
	return asValidationError(target, &e.ValidationError)
}

type MinContainsFieldFor31Plus struct{ ValidationError }

func (e *MinContainsFieldFor31Plus) As(target any) bool {
	return asValidationError(target, &e.ValidationError)
}

type MaxContainsFieldFor31Plus struct{ ValidationError }

func (e *MaxContainsFieldFor31Plus) As(target any) bool {
	return asValidationError(target, &e.ValidationError)
}

type PatternPropertiesFieldFor31Plus struct{ ValidationError }

func (e *PatternPropertiesFieldFor31Plus) As(target any) bool {
	return asValidationError(target, &e.ValidationError)
}

type DependentSchemasFieldFor31Plus struct{ ValidationError }

func (e *DependentSchemasFieldFor31Plus) As(target any) bool {
	return asValidationError(target, &e.ValidationError)
}

type PropertyNamesFieldFor31Plus struct{ ValidationError }

func (e *PropertyNamesFieldFor31Plus) As(target any) bool {
	return asValidationError(target, &e.ValidationError)
}

type UnevaluatedItemsFieldFor31Plus struct{ ValidationError }

func (e *UnevaluatedItemsFieldFor31Plus) As(target any) bool {
	return asValidationError(target, &e.ValidationError)
}

type UnevaluatedPropertiesFieldFor31Plus struct{ ValidationError }

func (e *UnevaluatedPropertiesFieldFor31Plus) As(target any) bool {
	return asValidationError(target, &e.ValidationError)
}

type IfFieldFor31Plus struct{ ValidationError }

func (e *IfFieldFor31Plus) As(target any) bool {
	return asValidationError(target, &e.ValidationError)
}

type ThenFieldFor31Plus struct{ ValidationError }

func (e *ThenFieldFor31Plus) As(target any) bool {
	return asValidationError(target, &e.ValidationError)
}

type ElseFieldFor31Plus struct{ ValidationError }

func (e *ElseFieldFor31Plus) As(target any) bool {
	return asValidationError(target, &e.ValidationError)
}

type DependentRequiredFieldFor31Plus struct{ ValidationError }

func (e *DependentRequiredFieldFor31Plus) As(target any) bool {
	return asValidationError(target, &e.ValidationError)
}

type ContentEncodingFieldFor31Plus struct{ ValidationError }

func (e *ContentEncodingFieldFor31Plus) As(target any) bool {
	return asValidationError(target, &e.ValidationError)
}

type ContentMediaTypeFieldFor31Plus struct{ ValidationError }

func (e *ContentMediaTypeFieldFor31Plus) As(target any) bool {
	return asValidationError(target, &e.ValidationError)
}

type ContentSchemaFieldFor31Plus struct{ ValidationError }

func (e *ContentSchemaFieldFor31Plus) As(target any) bool {
	return asValidationError(target, &e.ValidationError)
}

type DefsFieldFor31Plus struct{ ValidationError }

func (e *DefsFieldFor31Plus) As(target any) bool {
	return asValidationError(target, &e.ValidationError)
}

type SchemaFieldFor31Plus struct{ ValidationError }

func (e *SchemaFieldFor31Plus) As(target any) bool {
	return asValidationError(target, &e.ValidationError)
}

type CommentFieldFor31Plus struct{ ValidationError }

func (e *CommentFieldFor31Plus) As(target any) bool {
	return asValidationError(target, &e.ValidationError)
}

type IDFieldFor31Plus struct{ ValidationError }

func (e *IDFieldFor31Plus) As(target any) bool {
	return asValidationError(target, &e.ValidationError)
}

type AnchorFieldFor31Plus struct{ ValidationError }

func (e *AnchorFieldFor31Plus) As(target any) bool {
	return asValidationError(target, &e.ValidationError)
}

type DynamicAnchorFieldFor31Plus struct{ ValidationError }

func (e *DynamicAnchorFieldFor31Plus) As(target any) bool {
	return asValidationError(target, &e.ValidationError)
}

type DynamicRefFieldFor31Plus struct{ ValidationError }

func (e *DynamicRefFieldFor31Plus) As(target any) bool {
	return asValidationError(target, &e.ValidationError)
}

// ---------------------------------------------------------------------
// Constructors — the validator-side entry points. Build the leaf, wrap
// in the cluster, return the cluster (which exposes the leaf via
// Unwrap and the base via the leaf's As method).
// ---------------------------------------------------------------------

func newRequiredField(field string, leaf error, origin *Origin) error {
	return &RequiredFieldError{Field: field, Cause: leaf, Origin: origin}
}

func newInfoVersionRequired(origin *Origin) error {
	return newRequiredField("info.version",
		&InfoVersionRequired{ValidationError{Message: "value of version must be a non-empty string"}}, origin)
}

func newInfoTitleRequired(origin *Origin) error {
	return newRequiredField("info.title",
		&InfoTitleRequired{ValidationError{Message: "value of title must be a non-empty string"}}, origin)
}

func newLicenseNameRequired(origin *Origin) error {
	return newRequiredField("license.name",
		&LicenseNameRequired{ValidationError{Message: "value of license name must be a non-empty string"}}, origin)
}

// newOpenAPIVersionRequired has no Origin parameter: the OpenAPI version
// string lives on the document root *T, which the loader doesn't track.
func newOpenAPIVersionRequired() error {
	return newRequiredField("openapi",
		&OpenAPIVersionRequired{ValidationError{Message: "value of openapi must be a non-empty string"}}, nil)
}

func newServerURLRequired(origin *Origin) error {
	return newRequiredField("server.url",
		&ServerURLRequired{ValidationError{Message: "value of url must be a non-empty string"}}, origin)
}

// newSchemaValueError wraps the result of schema.VisitJSON in a
// *SchemaValueError cluster, identifying which schema sub-field
// (example, default, ...) carried the offending value. cause is
// either a *SchemaError or a MultiError of them.
func newSchemaValueError(valueKind string, cause error, origin *Origin) error {
	return &SchemaValueError{ValueKind: valueKind, Cause: cause, Origin: origin}
}

// newFieldVersionMismatch wraps leaf in a FieldVersionMismatchError for the
// given field at minimum version 3.1. Used by per-call-site constructors
// (newInfoSummaryFieldFor31Plus, etc.) and by the dispatch helper
// newFieldFor31Plus that schema.go's reject closure goes through.
func newFieldVersionMismatch(field string, leaf error, origin *Origin) error {
	return &FieldVersionMismatchError{
		Field:      field,
		MinVersion: "3.1",
		Cause:      leaf,
		Origin:     origin,
	}
}

// Per-call-site constructors for the four non-schema FieldFor31Plus sites
// (info.summary, license.identifier, doc.webhooks, doc.jsonSchemaDialect).
// The schema fields go through fieldFor31PlusLeaves below because they're
// dispatched from a runtime-parameterised closure in schema.go's reject.

func newInfoSummaryFieldFor31Plus(origin *Origin) error {
	const msg = "field summary is for OpenAPI >=3.1"
	return newFieldVersionMismatch("summary",
		&InfoSummaryFieldFor31Plus{ValidationError{Message: msg}}, origin)
}

func newLicenseIdentifierFieldFor31Plus(origin *Origin) error {
	const msg = "field identifier is for OpenAPI >=3.1"
	return newFieldVersionMismatch("identifier",
		&LicenseIdentifierFieldFor31Plus{ValidationError{Message: msg}}, origin)
}

// newWebhooksFieldFor31Plus and newJSONSchemaDialectFieldFor31Plus have no
// Origin parameter: both fields live on the document root *T, which the
// loader doesn't track.
func newWebhooksFieldFor31Plus() error {
	const msg = "field webhooks is for OpenAPI >=3.1"
	return newFieldVersionMismatch("webhooks",
		&WebhooksFieldFor31Plus{ValidationError{Message: msg}}, nil)
}

func newJSONSchemaDialectFieldFor31Plus() error {
	const msg = "field jsonschemadialect is for OpenAPI >=3.1"
	return newFieldVersionMismatch("jsonschemadialect",
		&JSONSchemaDialectFieldFor31Plus{ValidationError{Message: msg}}, nil)
}

// fieldFor31PlusLeaves maps field names (as passed to errFieldFor31Plus)
// to their typed leaf constructors. Only schema-keyword fields are in
// the table — those are dispatched at runtime from schema.go's reject
// closure. The four non-schema fields (summary, identifier, webhooks,
// jsonschemadialect) have direct constructors above. Any field not in
// the map falls back to a bare *ValidationError, so callers still get
// the cluster + base layers — only the per-leaf type is missing.
var fieldFor31PlusLeaves = map[string]func(msg string) error{
	"const":                 func(m string) error { return &ConstFieldFor31Plus{ValidationError{Message: m}} },
	"examples":              func(m string) error { return &ExamplesFieldFor31Plus{ValidationError{Message: m}} },
	"prefixItems":           func(m string) error { return &PrefixItemsFieldFor31Plus{ValidationError{Message: m}} },
	"contains":              func(m string) error { return &ContainsFieldFor31Plus{ValidationError{Message: m}} },
	"minContains":           func(m string) error { return &MinContainsFieldFor31Plus{ValidationError{Message: m}} },
	"maxContains":           func(m string) error { return &MaxContainsFieldFor31Plus{ValidationError{Message: m}} },
	"patternProperties":     func(m string) error { return &PatternPropertiesFieldFor31Plus{ValidationError{Message: m}} },
	"dependentSchemas":      func(m string) error { return &DependentSchemasFieldFor31Plus{ValidationError{Message: m}} },
	"propertyNames":         func(m string) error { return &PropertyNamesFieldFor31Plus{ValidationError{Message: m}} },
	"unevaluatedItems":      func(m string) error { return &UnevaluatedItemsFieldFor31Plus{ValidationError{Message: m}} },
	"unevaluatedProperties": func(m string) error { return &UnevaluatedPropertiesFieldFor31Plus{ValidationError{Message: m}} },
	"if":                    func(m string) error { return &IfFieldFor31Plus{ValidationError{Message: m}} },
	"then":                  func(m string) error { return &ThenFieldFor31Plus{ValidationError{Message: m}} },
	"else":                  func(m string) error { return &ElseFieldFor31Plus{ValidationError{Message: m}} },
	"dependentRequired":     func(m string) error { return &DependentRequiredFieldFor31Plus{ValidationError{Message: m}} },
	"contentEncoding":       func(m string) error { return &ContentEncodingFieldFor31Plus{ValidationError{Message: m}} },
	"contentMediaType":      func(m string) error { return &ContentMediaTypeFieldFor31Plus{ValidationError{Message: m}} },
	"contentSchema":         func(m string) error { return &ContentSchemaFieldFor31Plus{ValidationError{Message: m}} },
	"$defs":                 func(m string) error { return &DefsFieldFor31Plus{ValidationError{Message: m}} },
	"$schema":               func(m string) error { return &SchemaFieldFor31Plus{ValidationError{Message: m}} },
	"$comment":              func(m string) error { return &CommentFieldFor31Plus{ValidationError{Message: m}} },
	"$id":                   func(m string) error { return &IDFieldFor31Plus{ValidationError{Message: m}} },
	"$anchor":               func(m string) error { return &AnchorFieldFor31Plus{ValidationError{Message: m}} },
	"$dynamicAnchor":        func(m string) error { return &DynamicAnchorFieldFor31Plus{ValidationError{Message: m}} },
	"$dynamicRef":           func(m string) error { return &DynamicRefFieldFor31Plus{ValidationError{Message: m}} },
}

// newFieldFor31Plus dispatches errFieldFor31Plus's per-field message
// to the right typed leaf and wraps it in a FieldVersionMismatchError.
// Fields not in fieldFor31PlusLeaves fall back to a bare
// *ValidationError so the caller still gets a stable Message and the
// cluster + base layers; only the per-leaf type is missing.
//
// Reached only from schema.go's reject closure with a runtime field
// name; the four non-schema sites use direct constructors instead.
func newFieldFor31Plus(field string, origin *Origin) error {
	msg := "field " + field + " is for OpenAPI >=3.1"
	var leaf error
	if ctor, ok := fieldFor31PlusLeaves[field]; ok {
		leaf = ctor(msg)
	} else {
		leaf = &ValidationError{Message: msg}
	}
	return newFieldVersionMismatch(field, leaf, origin)
}
