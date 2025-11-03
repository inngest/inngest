package meta

import "encoding/json"

type RawMetadata map[string]json.RawMessage

type StructuredMetadata interface {
	Kind() MetadataKind
	Serialize() (RawMetadata, error)
}

func AnyStructuredMetadata(kind MetadataKind, data any) StructuredMetadata {
	return anyStructuredMetadata{
		kind: kind,
		data: data,
	}
}

type anyStructuredMetadata struct {
	kind MetadataKind
	data any
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

type MetadataKind string

type MetadataOp string // TODO real enum

const (
	MetadataOpMerge MetadataOp = "merge"
)
