//go:generate go run generate_extracted_values.go

package meta

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/util"
	"github.com/oklog/ulid/v2"
	"go.opentelemetry.io/otel/attribute"
)

var BlankAttr = attribute.String("", "")

type SerializableAttrs struct {
	es    *util.ErrSet
	Attrs []SerializableAttr
}

func NewAttrSet(attrs ...SerializableAttr) *SerializableAttrs {
	return &SerializableAttrs{
		Attrs: attrs,
	}
}

type SerializableAttr struct {
	serialize func(any) (attribute.KeyValue, bool)
	key       string
	value     any
}

func AddAttr[T any](r *SerializableAttrs, attr attr[T], value T) {
	r.Attrs = append(r.Attrs, Attr(attr, value))
}

func (r *SerializableAttrs) AddErr(err error) {
	if r.es == nil {
		r.es = util.NewErrSet()
	}

	r.es.Add(err)
}

func (r *SerializableAttrs) Merge(other *SerializableAttrs) *SerializableAttrs {
	es := r.es
	o := other
	if o == nil {
		o = NewAttrSet()
	} else if o.es != nil {
		es = r.es.Merge(o.es)
	}

	return &SerializableAttrs{
		es:    es,
		Attrs: append(r.Attrs, o.Attrs...),
	}
}

func (r *SerializableAttrs) Serialize() []attribute.KeyValue {
	attrs := []attribute.KeyValue{}
	if r.es == nil {
		r.es = util.NewErrSet()
	}

	for _, item := range r.Attrs {
		if kv, ok := item.serialize(item.value); ok {
			attrs = append(attrs, kv)
		} else {
			r.es.Add(fmt.Errorf("failed to serialize value %v for attribute %s", item.value, item.key))
		}
	}

	if r.es.HasErrors() {
		attrs = append(attrs,
			attribute.String(fmt.Sprintf("%s%s", AttrKeyPrefix, InternalError), r.es.Err().Error()),
		)
	}

	return attrs
}

type Serializer interface {
	Key() string
	SerializeValue(any) (attribute.KeyValue, bool)
	DeserializeValue(any) (any, bool)
}

type attr[T any] struct {
	key         string
	serialize   func(T) attribute.KeyValue
	deserialize func(any) (T, bool)
}

func (a attr[T]) Key() string {
	return a.key
}

func (a attr[T]) SerializeValue(v any) (attribute.KeyValue, bool) {
	if val, ok := v.(T); ok {
		return a.serialize(val), true
	}

	// TODO Add an internal error to show that we failed to serialize the value
	return attribute.KeyValue{}, false
}

func (a attr[T]) DeserializeValue(v any) (any, bool) {
	return a.deserialize(v)
}

func Attr[T any](attr attr[T], value T) SerializableAttr {
	return SerializableAttr{
		serialize: func(v any) (attribute.KeyValue, bool) {
			return attr.SerializeValue(v)
		},
		key:   attr.Key(),
		value: value,
	}
}

// Reusable serializers
func withPrefix(key string) string {
	return fmt.Sprintf("%s%s", AttrKeyPrefix, key)
}

func AnyAttr(key string) attr[*any] {
	return attr[*any]{
		key: withPrefix(key),
		serialize: func(v *any) attribute.KeyValue {
			if v == nil {
				return BlankAttr
			}

			return attribute.String(withPrefix(key), fmt.Sprintf("%v", *v))
		},
		deserialize: func(v any) (*any, bool) {
			return &v, true
		},
	}
}

func StringAttr(key string) attr[*string] {
	return attr[*string]{
		key: withPrefix(key),
		serialize: func(v *string) attribute.KeyValue {
			if v == nil {
				return BlankAttr
			}

			return attribute.String(withPrefix(key), *v)
		},
		deserialize: func(v any) (*string, bool) {
			s, ok := v.(string)
			return &s, ok
		},
	}
}

func StringSliceAttr(key string) attr[*[]string] {
	return attr[*[]string]{
		key: withPrefix(key),
		serialize: func(v *[]string) attribute.KeyValue {
			if v == nil || len(*v) == 0 {
				return BlankAttr
			}

			return attribute.StringSlice(withPrefix(key), *v)
		},
		deserialize: func(v any) (*[]string, bool) {
			if slice, ok := v.([]string); ok {
				return &slice, true
			}

			return nil, false
		},
	}
}

func BoolAttr(key string) attr[*bool] {
	return attr[*bool]{
		key: withPrefix(key),
		serialize: func(v *bool) attribute.KeyValue {
			if v == nil {
				return BlankAttr
			}

			return attribute.Bool(withPrefix(key), *v)
		},
		deserialize: func(v any) (*bool, bool) {
			switch v := v.(type) {
			case bool:
				return &v, true
			case string:
				if b, err := strconv.ParseBool(v); err == nil {
					return &b, true
				}
			}
			return nil, false

		},
	}
}

func TimeAttr(key string) attr[*time.Time] {
	return attr[*time.Time]{
		key: withPrefix(key),
		serialize: func(v *time.Time) attribute.KeyValue {
			if v == nil {
				return BlankAttr
			}

			return attribute.Int64(withPrefix(key), v.UnixMilli())
		},
		deserialize: func(v any) (*time.Time, bool) {
			switch v := v.(type) {
			case int64:
				t := time.UnixMilli(v)
				return &t, true
			case float64:
				t := time.UnixMilli(int64(v))
				return &t, true
			case string:
				if f, err := strconv.ParseFloat(v, 64); err == nil {
					t := time.UnixMilli(int64(f))
					return &t, true
				}
			}
			return nil, false
		},
	}
}

func DurationAttr(key string) attr[*time.Duration] {
	return attr[*time.Duration]{
		key: withPrefix(key),
		serialize: func(v *time.Duration) attribute.KeyValue {
			if v == nil {
				return BlankAttr
			}

			return attribute.Int64(withPrefix(key), int64(*v/time.Millisecond))
		},
		deserialize: func(v any) (*time.Duration, bool) {
			switch v := v.(type) {
			case float64:
				d := time.Duration(int64(v)) * time.Millisecond
				return &d, true
			case string:
				if f, err := strconv.ParseFloat(v, 64); err == nil {
					d := time.Duration(int64(f)) * time.Millisecond
					return &d, true
				}
			}
			return nil, false

		},
	}
}

func ULIDAttr(key string) attr[*ulid.ULID] {
	return attr[*ulid.ULID]{
		key: withPrefix(key),
		serialize: func(v *ulid.ULID) attribute.KeyValue {
			if v == nil {
				return BlankAttr
			}

			return attribute.String(withPrefix(key), v.String())
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

func UUIDAttr(key string) attr[*uuid.UUID] {
	return attr[*uuid.UUID]{
		key: withPrefix(key),
		serialize: func(v *uuid.UUID) attribute.KeyValue {
			if v == nil {
				return BlankAttr
			}

			return attribute.String(withPrefix(key), v.String())
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

func IntAttr(key string) attr[*int] {
	return attr[*int]{
		key: withPrefix(key),
		serialize: func(v *int) attribute.KeyValue {
			if v == nil {
				return BlankAttr
			}

			return attribute.Int(withPrefix(key), *v)
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

func StepStatusAttr(key string) attr[*enums.StepStatus] {
	return attr[*enums.StepStatus]{
		key: withPrefix(key),
		serialize: func(v *enums.StepStatus) attribute.KeyValue {
			if v == nil {
				return BlankAttr
			}

			return attribute.String(withPrefix(key), v.String())
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

func StepOpAttr(key string) attr[*enums.Opcode] {
	return attr[*enums.Opcode]{
		key: withPrefix(key),
		serialize: func(v *enums.Opcode) attribute.KeyValue {
			if v == nil {
				return BlankAttr
			}

			return attribute.String(withPrefix(key), v.String())
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

func HttpHeaderAttr(key string) attr[*http.Header] {
	return attr[*http.Header]{
		key: withPrefix(key),
		serialize: func(v *http.Header) attribute.KeyValue {
			if v == nil || len(*v) == 0 {
				return BlankAttr
			}

			headerByt, _ := json.Marshal(v)

			return attribute.String(withPrefix(key), string(headerByt))
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

// ExtractTypedValues uses reflection to extract typed pointer values from the
// Attrs struct given a map of attribute key-value pairs. It returns a properly
// typed ExtractedValues struct with IDE support and compile-time type checking.
func ExtractTypedValues(ctx context.Context, attrs map[string]any) (*ExtractedValues, error) {
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

			if key == "" {
				// The serializer exists in the struct definition but hasn't
				// actually been set. This is an operator error and needs to
				// be addressed!
				return nil, fmt.Errorf("span attribute serializer for '%s' is empty - a span attribute has been defined but no (de)serializer has been set; this attribute will never be persisted or deserialized", fieldName)
			}

			if value, exists := attrs[key]; exists {
				if deserializedValue, success := serializer.DeserializeValue(value); success {
					// Set the deserialized value in the result struct
					resultField.Set(reflect.ValueOf(deserializedValue))
				}
			}
		}
	}

	return result, nil
}
