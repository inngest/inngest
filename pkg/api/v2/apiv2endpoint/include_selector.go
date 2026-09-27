package apiv2endpoint

import "fmt"

// IncludeSelector parses API values into the generated semantic values for an
// RPC's include query parameter.
type IncludeSelector[T any] struct {
	values map[string]T
}

func NewIncludeSelector[T any](values map[string]T) IncludeSelector[T] {
	return IncludeSelector[T]{values: values}
}

func (s IncludeSelector[T]) Parse(value string) (T, error) {
	if parsed, ok := s.values[value]; ok {
		return parsed, nil
	}
	var zero T
	return zero, fmt.Errorf("unsupported include value %q", value)
}
