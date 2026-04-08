package openapi3

import "encoding/json"

// StringMap is a map[string]string that ignores the origin in the underlying json representation.
type StringMap[V any] map[string]V

// UnmarshalJSON sets StringMap to a copy of data.
func (stringMap *StringMap[V]) UnmarshalJSON(data []byte) (err error) {
	*stringMap, err = unmarshalStringMap[V](data)
	return
}

// unmarshalStringMapP unmarshals given json into a map[string]*V
func unmarshalStringMapP[V any](data []byte) (map[string]*V, error) {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}

	result := make(map[string]*V, len(m))
	for k, v := range m {
		value, err := deepCast[V](v)
		if err != nil {
			return nil, err
		}
		result[k] = value
	}

	return result, nil
}

// unmarshalStringMap unmarshals given json into a map[string]V
func unmarshalStringMap[V any](data []byte) (map[string]V, error) {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}

	result := make(map[string]V, len(m))
	for k, v := range m {
		value, err := deepCast[V](v)
		if err != nil {
			return nil, err
		}
		result[k] = *value
	}

	return result, nil
}

// deepCast casts any value to a value of type V.
func deepCast[V any](value any) (*V, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}

	var result V
	if err = json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
