package apiv2

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	apiv2base "github.com/inngest/inngest/pkg/api/v2/apiv2base"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var responseEnumPrefixes = []string{
	"APP_METHOD_",
	"FUNCTION_RUN_STATUS_",
	"FUNCTION_TRIGGER_TYPE_",
	"FUNCTION_CONCURRENCY_SCOPE_",
	"FUNCTION_SINGLETON_MODE_",
	"TRACE_SPAN_STATUS_",
	"TRACE_STEP_OP_",
	"SANDBOX_DESIRED_STATE_",
	"SANDBOX_PHASE_",
	"SANDBOX_OUTCOME_",
	"SANDBOX_CLEANUP_STATE_",
}

type responseEnumMarshaler struct {
	*runtime.JSONPb
}

func newResponseEnumMarshaler() runtime.Marshaler {
	return NewResponseEnumMarshaler()
}

func NewResponseEnumMarshaler() runtime.Marshaler {
	return responseEnumMarshaler{JSONPb: &runtime.JSONPb{}}
}

func (m responseEnumMarshaler) Marshal(v any) ([]byte, error) {
	if response, ok := v.(*apiv2.ExecSandboxResponse); ok {
		if data := response.GetData(); data != nil &&
			((data.Stdout != nil && !utf8.ValidString(*data.Stdout)) ||
				(data.Stderr != nil && !utf8.ValidString(*data.Stderr))) {
			return nil, apiv2base.NewError(
				http.StatusBadGateway,
				apiv2base.ErrorOutputEncodingInvalid,
				"sandbox Exec output is not valid UTF-8",
			)
		}
	}

	data, err := m.JSONPb.Marshal(v)
	if err != nil {
		return nil, err
	}

	message, ok := v.(proto.Message)
	if !ok {
		return data, nil
	}

	return shortenResponseEnumNames(data, message.ProtoReflect().Descriptor())
}

func (m responseEnumMarshaler) NewEncoder(w io.Writer) runtime.Encoder {
	return runtime.EncoderFunc(func(v any) error {
		data, err := m.Marshal(v)
		if err != nil {
			return err
		}
		if _, err := w.Write(data); err != nil {
			return err
		}
		_, err = w.Write(m.Delimiter())
		return err
	})
}

func shortenResponseEnumNames(data []byte, descriptor protoreflect.MessageDescriptor) ([]byte, error) {
	var body any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&body); err != nil {
		return nil, err
	}

	shortenResponseEnumMessage(body, descriptor)

	var out bytes.Buffer
	encoder := json.NewEncoder(&out)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(body); err != nil {
		return nil, err
	}

	return bytes.TrimSuffix(out.Bytes(), []byte("\n")), nil
}

func shortenResponseEnumMessage(value any, descriptor protoreflect.MessageDescriptor) {
	switch descriptor.FullName() {
	case "google.protobuf.Struct", "google.protobuf.Value", "google.protobuf.ListValue", "google.protobuf.Any":
		// These well-known types use dynamic ProtoJSON shapes. Their strings are
		// payload, not statically typed enum fields.
		return
	}
	object, ok := value.(map[string]any)
	if !ok || descriptor == nil {
		return
	}
	fields := descriptor.Fields()
	for name, child := range object {
		field := fields.ByJSONName(name)
		if field == nil {
			field = fields.ByName(protoreflect.Name(name))
		}
		if field != nil {
			shortenResponseEnumField(object, name, child, field)
		}
	}
}

func shortenResponseEnumField(object map[string]any, name string, value any, field protoreflect.FieldDescriptor) {
	if field.IsMap() {
		entries, ok := value.(map[string]any)
		if !ok {
			return
		}
		for key, entry := range entries {
			shortenResponseEnumSingular(entries, key, entry, field.MapValue())
		}
		return
	}
	if field.IsList() {
		values, ok := value.([]any)
		if !ok {
			return
		}
		for i, entry := range values {
			shortenResponseEnumListValue(values, i, entry, field)
		}
		return
	}
	shortenResponseEnumSingular(object, name, value, field)
}

func shortenResponseEnumListValue(values []any, index int, value any, field protoreflect.FieldDescriptor) {
	if field.Kind() == protoreflect.EnumKind {
		if enumName, ok := value.(string); ok {
			values[index] = shortenResponseEnumString(enumName)
		}
		return
	}
	if field.Kind() == protoreflect.MessageKind || field.Kind() == protoreflect.GroupKind {
		shortenResponseEnumMessage(value, field.Message())
	}
}

func shortenResponseEnumSingular(object map[string]any, name string, value any, field protoreflect.FieldDescriptor) {
	switch field.Kind() {
	case protoreflect.EnumKind:
		if enumName, ok := value.(string); ok {
			object[name] = shortenResponseEnumString(enumName)
		}
	case protoreflect.MessageKind, protoreflect.GroupKind:
		shortenResponseEnumMessage(value, field.Message())
	}
}

func shortenResponseEnumString(value string) string {
	for _, prefix := range responseEnumPrefixes {
		if trimmed, ok := strings.CutPrefix(value, prefix); ok {
			return trimmed
		}
	}
	return value
}
