package apiv2endpoint

import (
	"fmt"

	"google.golang.org/protobuf/reflect/protoreflect"
)

// IncludeSelector parses the documented response-field names and their
// protobuf text-name aliases for an RPC's include query parameter.
type IncludeSelector struct {
	values map[string]string
}

func NewIncludeSelector(method protoreflect.MethodDescriptor) IncludeSelector {
	if method == nil {
		panic("include RPC does not exist")
	}
	request := method.Input()
	include := request.Fields().ByJSONName("include")
	if include == nil {
		panic("request does not have an include field: " + string(request.FullName()))
	}

	canonicalValues := FieldEnum(include)
	if len(canonicalValues) == 0 {
		panic("include field does not declare supported values: " + string(request.FullName()))
	}
	return newIncludeSelector(method, canonicalValues)
}

func newIncludeSelector(method protoreflect.MethodDescriptor, canonicalValues []string) IncludeSelector {
	selector := IncludeSelector{values: map[string]string{}}
	for _, canonical := range canonicalValues {
		fields := responseFieldsByJSONName(method.Output(), canonical, map[protoreflect.FullName]bool{})
		if len(fields) == 0 {
			panic(fmt.Sprintf("include value %q does not name a response field for %s", canonical, method.Input().FullName()))
		}
		selector.add(canonical, canonical)
		for _, field := range fields {
			selector.add(field.TextName(), canonical)
		}
	}
	return selector
}

func (s IncludeSelector) Parse(value string) (string, error) {
	if parsed, ok := s.values[value]; ok {
		return parsed, nil
	}
	return "", fmt.Errorf("unsupported include value %q", value)
}

func (s IncludeSelector) MustValue(apiValue string) string {
	value, ok := s.values[apiValue]
	if !ok {
		panic(fmt.Sprintf("include value %q is not supported", apiValue))
	}
	return value
}

func (s IncludeSelector) add(apiValue, canonical string) {
	if existing, ok := s.values[apiValue]; ok && existing != canonical {
		panic(fmt.Sprintf("include value %q is ambiguous", apiValue))
	}
	s.values[apiValue] = canonical
}

func responseFieldsByJSONName(message protoreflect.MessageDescriptor, name string, visited map[protoreflect.FullName]bool) []protoreflect.FieldDescriptor {
	if visited[message.FullName()] {
		return nil
	}
	visited[message.FullName()] = true

	var matches []protoreflect.FieldDescriptor
	fields := message.Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		if field.JSONName() == name {
			matches = append(matches, field)
		}
		if field.Message() != nil && !field.IsMap() {
			matches = append(matches, responseFieldsByJSONName(field.Message(), name, visited)...)
		}
	}
	return matches
}
