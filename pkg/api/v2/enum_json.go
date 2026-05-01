package apiv2

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/protobuf/proto"
)

var responseEnumPrefixes = []string{
	"FUNCTION_RUN_STATUS_",
	"TRACE_SPAN_STATUS_",
	"TRACE_STEP_OP_",
}

type responseEnumMarshaler struct {
	*runtime.JSONPb
}

func newResponseEnumMarshaler() runtime.Marshaler {
	return responseEnumMarshaler{JSONPb: &runtime.JSONPb{}}
}

func (m responseEnumMarshaler) Marshal(v any) ([]byte, error) {
	data, err := m.JSONPb.Marshal(v)
	if err != nil {
		return nil, err
	}

	if _, ok := v.(proto.Message); !ok {
		return data, nil
	}

	return shortenResponseEnumNames(data)
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
