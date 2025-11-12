package meta

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"

	tracev1 "go.opentelemetry.io/proto/otlp/trace/v1"
)

var ErrInvalidMetadataOp = errors.New("invalid metadata op")

var DefaultMetadataExtractor MetadataExtractor

type RawMetadata map[string]json.RawMessage

func (m *RawMetadata) FromStruct(v any) error {
	// TODO: reflect stuff so we don't need to remarshal?
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, m)
}

func (m RawMetadata) Combine(o RawMetadata, op MetadataOp) error {
	switch op {
	case MetadataOpMerge:
		maps.Copy(m, o)
		return nil
	case MetadataOpDelete:
		for k := range o {
			delete(m, k)
		}
		return nil
	case MetadataOpAdd:
		for k := range o {
			var a float64
			if err := json.Unmarshal(m[k], &a); err != nil {
				m[k] = o[k]
				continue
			}

			var b float64
			if err := json.Unmarshal(o[k], &b); err != nil {
				continue
			}

			m[k], _ = json.Marshal(a + b)
		}
		return nil
	case MetadataOpSet:
		clear(m)
		maps.Copy(m, o)
		return nil
	default:
		return fmt.Errorf("unrecognized op %q: %w", op, ErrInvalidMetadataOp)
	}
}

type StructuredMetadata interface {
	Kind() MetadataKind
	Serialize() (RawMetadata, error)

	Op() MetadataOp
}

func AnyStructuredMetadata(kind MetadataKind, data any, op MetadataOp) StructuredMetadata {
	return anyStructuredMetadata{
		kind: kind,
		data: data,
		op:   op,
	}
}

type anyStructuredMetadata struct {
	kind MetadataKind
	data any
	op   MetadataOp
}

func (m anyStructuredMetadata) Kind() MetadataKind {
	return m.kind
}

func (m anyStructuredMetadata) Serialize() (RawMetadata, error) {
	b, err := json.Marshal(m.data)
	if err != nil {
		return nil, err
	}

	var raw RawMetadata
	err = json.Unmarshal(b, &raw)
	if err != nil {
		return nil, err
	}

	return raw, nil
}

func (m anyStructuredMetadata) Op() MetadataOp {
	return m.op
}

type MetadataKind string

type MetadataOp string // TODO: real enum

const (
	MetadataOpMerge  MetadataOp = "merge"
	MetadataOpSet    MetadataOp = "set"
	MetadataOpDelete MetadataOp = "delete"
	MetadataOpAdd    MetadataOp = "add"
)

type SpanMetadataExtractor interface {
	ExtractMetadata(ctx context.Context, span *tracev1.Span) ([]StructuredMetadata, error)
}

type SpanMetadataExtractorFunc func(context.Context, *tracev1.Span) ([]StructuredMetadata, error)

func (f SpanMetadataExtractorFunc) ExtractMetadata(ctx context.Context, span *tracev1.Span) ([]StructuredMetadata, error) {
	return f(ctx, span)
}

type SpanMetadataExtractors []SpanMetadataExtractor

func (me SpanMetadataExtractors) ExtractMetadata(ctx context.Context, span *tracev1.Span) ([]StructuredMetadata, error) {
	var metadata []StructuredMetadata
	for _, extractor := range me {
		subMetadata, err := extractor.ExtractMetadata(ctx, span)
		if err != nil {
			return nil, err
		}

		metadata = append(metadata, subMetadata...)
	}

	return metadata, nil
}

type MetadataExtractor struct {
	ExtendedTrace SpanMetadataExtractor
}

func MetadataSpanIDSeed(parentID string, kind MetadataKind) []byte {
	return fmt.Appendf(nil, "%s-metadata-%s", parentID, kind)
}

type MetadataWarningError struct {
	Key string
	Err error
}

func (e *MetadataWarningError) Error() string {
	return e.Err.Error()
}

type WarningMetadata map[string]error

func (wm WarningMetadata) Kind() MetadataKind {
	return "inngest.warnings"
}

func (wm WarningMetadata) Op() MetadataOp {
	return MetadataOpMerge
}

func (wm WarningMetadata) Serialize() (RawMetadata, error) {
	ret := make(RawMetadata)
	for key, warning := range wm {
		ret[key], _ = json.Marshal(warning.Error())
	}

	return ret, nil
}

func ExtractWarningMetadata(err error) WarningMetadata {
	warnings := extractMetadataWarnings(err)

	md := make(WarningMetadata)
	for _, warnings := range warnings {
		md[warnings.Key] = warnings.Err
	}

	return md
}

func extractMetadataWarnings(err error) []*MetadataWarningError {
	var warning *MetadataWarningError
	type joinedError interface{ Unwrap() []error }
	var joinedErr joinedError
	switch {
	case errors.As(err, &joinedErr):
		var ret []*MetadataWarningError
		for _, err := range joinedErr.Unwrap() {
			ret = append(ret, extractMetadataWarnings(err)...)
		}

		return ret
	case errors.As(err, &warning):
		return []*MetadataWarningError{warning}
	default:
		return nil
	}
}
