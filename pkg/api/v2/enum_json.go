package apiv2

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/inngest/inngest/pkg/api/v2/apiv2base"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var unknownProtoJSONFieldRegexp = regexp.MustCompile(`unknown field "([^"]+)"`)

var responseEnumPrefixes = []string{
	"APP_METHOD_",
	"FUNCTION_RUN_STATUS_",
	"FUNCTION_TRIGGER_TYPE_",
	"FUNCTION_CONCURRENCY_SCOPE_",
	"FUNCTION_SINGLETON_MODE_",
	"TRACE_SPAN_STATUS_",
	"TRACE_STEP_OP_",
	"SANDBOX_STATUS_",
	"SANDBOX_PROCESS_STATE_",
	"SANDBOX_LOG_STREAM_",
}

type responseEnumMarshaler struct {
	*runtime.JSONPb
}

func newResponseEnumMarshaler() runtime.Marshaler {
	return NewResponseEnumMarshaler()
}

func NewResponseEnumMarshaler() runtime.Marshaler {
	return responseEnumMarshaler{JSONPb: &runtime.JSONPb{
		UnmarshalOptions: protojson.UnmarshalOptions{DiscardUnknown: false},
	}}
}

func (responseEnumMarshaler) StreamContentType(any) string {
	return "application/x-ndjson"
}

func (m responseEnumMarshaler) Marshal(v any) ([]byte, error) {
	data, err := m.JSONPb.Marshal(v)
	if err != nil {
		return nil, err
	}

	if !containsProtoResponse(v) {
		return data, nil
	}

	return shortenResponseEnumNames(data)
}

func containsProtoResponse(v any) bool {
	if _, ok := v.(proto.Message); ok {
		return true
	}
	if values, ok := v.(map[string]any); ok {
		_, ok := values["result"].(proto.Message)
		return ok
	}
	return false
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

// NewDecoder preserves grpc-gateway's default protobuf JSON decoding while
// translating unknown-field errors into REST v2's structured error format.
func (m responseEnumMarshaler) NewDecoder(r io.Reader) runtime.Decoder {
	return requestBodyDecoder{Decoder: m.JSONPb.NewDecoder(r)}
}

type requestBodyDecoder struct {
	runtime.Decoder
}

func (d requestBodyDecoder) Decode(v any) error {
	err := d.Decoder.Decode(v)
	if err == nil {
		return nil
	}
	if match := unknownProtoJSONFieldRegexp.FindStringSubmatch(err.Error()); len(match) == 2 {
		return unexpectedRequestBodyFieldError{field: match[1]}
	}
	return err
}

type unexpectedRequestBodyFieldError struct {
	field string
}

func (e unexpectedRequestBodyFieldError) Error() string {
	response := apiv2base.ErrorResponse{Errors: []apiv2base.ErrorItem{{
		Code:    apiv2base.ErrorInvalidRequest,
		Message: fmt.Sprintf("Unexpected request body field %q", e.field),
	}}}
	data, _ := json.Marshal(response)
	return string(data)
}

func shortenResponseEnumNames(data []byte) ([]byte, error) {
	var body any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&body); err != nil {
		return nil, err
	}

	shortenResponseEnumValue(body)

	var out bytes.Buffer
	encoder := json.NewEncoder(&out)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(body); err != nil {
		return nil, err
	}

	return bytes.TrimSuffix(out.Bytes(), []byte("\n")), nil
}

func shortenResponseEnumValue(value any) {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if key == "input" || key == "output" {
				continue
			}
			if str, ok := child.(string); ok {
				typed[key] = shortenResponseEnumString(str)
				continue
			}
			shortenResponseEnumValue(child)
		}
	case []any:
		for i, child := range typed {
			if str, ok := child.(string); ok {
				typed[i] = shortenResponseEnumString(str)
				continue
			}
			shortenResponseEnumValue(child)
		}
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
