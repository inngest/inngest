//go:generate go run generate_extracted_values.go

package meta

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/util"
	"github.com/oklog/ulid/v2"
	"go.opentelemetry.io/otel/attribute"
)

var BlankAttr = attribute.String("", "")

type RawAttrs struct {
	items []anyAttr
}

type anyAttr struct {
	serialize func(any) (attribute.KeyValue, bool)
	key       string
	value     any
}

func AddRawAttr[T any](r *RawAttrs, attr Attr[T], value T) {
	r.items = append(r.items, anyAttr{
		serialize: func(v any) (attribute.KeyValue, bool) {
			return attr.SerializeValue(v)
		},
		key:   attr.Key(),
		value: value,
	})
}

func (r *RawAttrs) Serialize() []attribute.KeyValue {
	attrs := []attribute.KeyValue{}
	es := util.NewErrSet()

	for _, item := range r.items {
		if kv, ok := item.serialize(item.value); ok {
			attrs = append(attrs, kv)
		} else {
			es.Add(fmt.Errorf("failed to serialize value %v for attribute %s", item.value, item.key))
		}
	}

	if es.HasErrors() {
		attrs = append(attrs,
			attribute.String(InternalError, es.Err().Error()),
		)
	}

	return attrs
}

type Serializer interface {
	Key() string
	SerializeValue(any) (attribute.KeyValue, bool)
	DeserializeValue(any) (any, bool)
}

type Attr[T any] struct {
	key         string
	serialize   func(T) attribute.KeyValue
	deserialize func(any) (T, bool)
}

func (a Attr[T]) Key() string {
	return a.key
}

func (a Attr[T]) SerializeValue(v any) (attribute.KeyValue, bool) {
	if val, ok := v.(T); ok {
		return a.serialize(val), true
	}

	// TODO Add an internal error to show that we failed to serialize the value
	return attribute.KeyValue{}, false
}

func (a Attr[T]) DeserializeValue(v any) (any, bool) {
	return a.deserialize(v)
}

// Reusable serializers
func AnyAttr(key string) Attr[*any] {
	return Attr[*any]{
		key: key,
		serialize: func(v *any) attribute.KeyValue {
			if v == nil {
				return BlankAttr
			}

			return attribute.String(key, fmt.Sprintf("%v", *v))
		},
		deserialize: func(v any) (*any, bool) {
			return &v, true
		},
	}
}

func StringAttr(key string) Attr[*string] {
	return Attr[*string]{
		key: key,
		serialize: func(v *string) attribute.KeyValue {
			if v == nil {
				return BlankAttr
			}

			return attribute.String(key, *v)
		},
		deserialize: func(v any) (*string, bool) {
			s, ok := v.(string)
			return &s, ok
		},
	}
}

func BoolAttr(key string) Attr[*bool] {
	return Attr[*bool]{
		key: key,
		serialize: func(v *bool) attribute.KeyValue {
			if v == nil {
				return BlankAttr
			}

			return attribute.Bool(key, *v)
		},
		deserialize: func(v any) (*bool, bool) {
			b, ok := v.(bool)
			return &b, ok
		},
	}
}

func TimeAttr(key string) Attr[*time.Time] {
	return Attr[*time.Time]{
		key: key,
		serialize: func(v *time.Time) attribute.KeyValue {
			if v == nil {
				return BlankAttr
			}

			return attribute.Int64(key, v.UnixMilli())
		},
		deserialize: func(v any) (*time.Time, bool) {
			if ms, ok := v.(float64); ok {
				t := time.UnixMilli(int64(ms))
				return &t, true
			}

			return nil, false
		},
	}
}

func DurationAttr(key string) Attr[*time.Duration] {
	return Attr[*time.Duration]{
		key: key,
		serialize: func(v *time.Duration) attribute.KeyValue {
			if v == nil {
				return BlankAttr
			}

			return attribute.Int64(key, int64(*v/time.Millisecond))
		},
		deserialize: func(v any) (*time.Duration, bool) {
			if ms, ok := v.(float64); ok {
				d := time.Duration(int64(ms)) * time.Millisecond
				return &d, true
			}

			return nil, false
		},
	}
}

func ULIDAttr(key string) Attr[*ulid.ULID] {
	return Attr[*ulid.ULID]{
		key: key,
		serialize: func(v *ulid.ULID) attribute.KeyValue {
			if v == nil {
				return BlankAttr
			}

			return attribute.String(key, v.String())
		},
		deserialize: func(v any) (*ulid.ULID, bool) {
			if s, ok := v.(string); ok {
				if id, err := ulid.Parse(s); err == nil {
					return &id, true
				}
			}

			return nil, false
		},
	}
}

func UUIDAttr(key string) Attr[*uuid.UUID] {
	return Attr[*uuid.UUID]{
		key: key,
		serialize: func(v *uuid.UUID) attribute.KeyValue {
			if v == nil {
				return BlankAttr
			}

			return attribute.String(key, v.String())
		},
		deserialize: func(v any) (*uuid.UUID, bool) {
			if s, ok := v.(string); ok {
				if id, err := uuid.Parse(s); err == nil {
					return &id, true
				}
			}

			return nil, false
		},
	}
}

func IntAttr(key string) Attr[*int] {
	return Attr[*int]{
		key: key,
		serialize: func(v *int) attribute.KeyValue {
			if v == nil {
				return BlankAttr
			}

			return attribute.Int(key, *v)
		},
		deserialize: func(v any) (*int, bool) {
			if i, ok := v.(float64); ok {
				val := int(i)
				return &val, true
			}

			return nil, false
		},
	}
}

func StepStatusAttr(key string) Attr[*enums.StepStatus] {
	return Attr[*enums.StepStatus]{
		key: key,
		serialize: func(v *enums.StepStatus) attribute.KeyValue {
			if v == nil {
				return BlankAttr
			}

			return attribute.String(key, v.String())
		},
		deserialize: func(v any) (*enums.StepStatus, bool) {
			if statusStr, ok := v.(string); ok {
				if status, err := enums.StepStatusString(statusStr); err == nil {
					return &status, true
				}
			}

			return nil, false
		},
	}
}

func StepOpAttr(key string) Attr[*enums.Opcode] {
	return Attr[*enums.Opcode]{
		key: key,
		serialize: func(v *enums.Opcode) attribute.KeyValue {
			if v == nil {
				return BlankAttr
			}

			return attribute.String(key, v.String())
		},
		deserialize: func(v any) (*enums.Opcode, bool) {
			if opStr, ok := v.(string); ok {
				if op, err := enums.OpcodeString(opStr); err == nil {
					return &op, true
				}
			}

			return nil, false
		},
	}
}

func HttpHeaderAttr(key string) Attr[*http.Header] {
	return Attr[*http.Header]{
		key: key,
		serialize: func(v *http.Header) attribute.KeyValue {
			if v == nil || len(*v) == 0 {
				return BlankAttr
			}

			headerByt, _ := json.Marshal(v)

			return attribute.String(key, string(headerByt))
		},
		deserialize: func(v any) (*http.Header, bool) {
			if headerStr, ok := v.(string); ok {
				var headers http.Header
				if err := json.Unmarshal([]byte(headerStr), &headers); err == nil {
					return &headers, true
				}
			}

			return nil, false
		},
	}
}

// ExtractTypedValues uses reflection to extract typed pointer values from the Attrs struct
// given a map of attribute key-value pairs. It returns a properly typed ExtractedValues struct
// with IDE support and compile-time type checking.
func ExtractTypedValues(attrs map[string]any) *ExtractedValues {
	result := &ExtractedValues{}
	resultValue := reflect.ValueOf(result).Elem()
	attrsValue := reflect.ValueOf(Attrs)

	// Iterate through all fields in the Attrs struct
	for i := 0; i < attrsValue.NumField(); i++ {
		field := attrsValue.Field(i)
		fieldType := attrsValue.Type().Field(i)
		fieldName := fieldType.Name

		// Get the corresponding field in our result struct
		resultField := resultValue.FieldByName(fieldName)
		if !resultField.IsValid() || !resultField.CanSet() {
			continue
		}

		// Get the Attr value which has the deserialize function
		attrValue := field.Interface()
		if serializer, ok := attrValue.(Serializer); ok {
			key := serializer.Key()
			if value, exists := attrs[key]; exists {
				if deserializedValue, success := serializer.DeserializeValue(value); success {
					// Set the deserialized value in the result struct
					resultField.Set(reflect.ValueOf(deserializedValue))
				}
			}
		}
	}

	return result
}
