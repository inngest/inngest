package openapi3

const originKey = "__origin__"

// Origin contains the origin of a collection.
// Key is the location of the collection itself.
// Fields is a map of the location of each scalar field in the collection.
// Sequences is a map of the location of each item in sequence-valued fields.
type Origin struct {
	Key       *Location             `json:"key,omitempty" yaml:"key,omitempty"`
	Fields    map[string]Location   `json:"fields,omitempty" yaml:"fields,omitempty"`
	Sequences map[string][]Location `json:"sequences,omitempty" yaml:"sequences,omitempty"`
}

// Location is a struct that contains the location of a field.
type Location struct {
	File   string `json:"file,omitempty" yaml:"file,omitempty"`
	Line   int    `json:"line,omitempty" yaml:"line,omitempty"`
	Column int    `json:"column,omitempty" yaml:"column,omitempty"`
	Name   string `json:"name,omitempty" yaml:"name,omitempty"`
}

// stripOriginFromAny recursively removes the __origin__ key from any
// map[string]any value. This is needed for interface{}/any-typed fields
// (e.g. Schema.Enum, Schema.Default, Parameter.Example) that have no
// dedicated UnmarshalJSON to consume the origin metadata injected by
// the YAML origin-tracking loader.
func stripOriginFromAny(v any) any {
	m, ok := v.(map[string]any)
	if !ok {
		return v
	}
	delete(m, originKey)
	for k, val := range m {
		m[k] = stripOriginFromAny(val)
	}
	return m
}
