package apiv2

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestResponseEnumMarshalerShortensAPIEnumPrefixes(t *testing.T) {
	output, err := structpb.NewStruct(map[string]any{
		"literal": "FUNCTION_RUN_STATUS_FAILED",
	})
	require.NoError(t, err)

	marshaler := newResponseEnumMarshaler()
	data, err := marshaler.Marshal(&apiv2.GetFunctionRunResponse{
		Data: &apiv2.FunctionRun{
			Id:     "run-id",
			Status: apiv2.FunctionRunStatus_FUNCTION_RUN_STATUS_COMPLETED,
			Output: output,
		},
	})
	require.NoError(t, err)

	var body map[string]any
	require.NoError(t, json.Unmarshal(data, &body))

	run := body["data"].(map[string]any)
	require.Equal(t, "COMPLETED", run["status"])
	require.Equal(t, "FUNCTION_RUN_STATUS_FAILED", run["output"].(map[string]any)["literal"])
}

func TestResponseEnumMarshalerShortensTraceEnumPrefixes(t *testing.T) {
	marshaler := newResponseEnumMarshaler()
	data, err := marshaler.Marshal(&apiv2.GetFunctionTraceResponse{
		Data: &apiv2.FunctionTrace{
			RunId: "run-id",
			RootSpan: &apiv2.TraceSpan{
				Id:     "span-id",
				Status: apiv2.TraceSpanStatus_TRACE_SPAN_STATUS_COMPLETED,
				StepOp: func() *apiv2.TraceStepOp {
					op := apiv2.TraceStepOp_TRACE_STEP_OP_SEND_EVENT
					return &op
				}(),
				Children: []*apiv2.TraceSpan{
					{
						Id:     "child-span-id",
						Status: apiv2.TraceSpanStatus_TRACE_SPAN_STATUS_WAITING,
					},
				},
			},
		},
	})
	require.NoError(t, err)

	var body map[string]any
	require.NoError(t, json.Unmarshal(data, &body))

	root := body["data"].(map[string]any)["rootSpan"].(map[string]any)
	require.Equal(t, "COMPLETED", root["status"])
	require.Equal(t, "SEND_EVENT", root["stepOp"])
	require.Equal(t, "WAITING", root["children"].([]any)[0].(map[string]any)["status"])
}

func TestResponseEnumMarshalerLeavesNonProtoValuesUnchanged(t *testing.T) {
	marshaler := newResponseEnumMarshaler()

	data, err := marshaler.Marshal(map[string]string{
		"status": "FUNCTION_RUN_STATUS_COMPLETED",
	})
	require.NoError(t, err)

	var body map[string]string
	require.NoError(t, json.Unmarshal(data, &body))
	require.Equal(t, "FUNCTION_RUN_STATUS_COMPLETED", body["status"])
}

func TestResponseEnumMarshalerReturnsMarshalErrors(t *testing.T) {
	marshaler := newResponseEnumMarshaler()

	data, err := marshaler.Marshal(func() {})
	require.Nil(t, data)
	require.Error(t, err)
}

func TestResponseEnumMarshalerEncoderWritesDelimitedJSON(t *testing.T) {
	var buf bytes.Buffer
	marshaler := newResponseEnumMarshaler()
	encoder := marshaler.NewEncoder(&buf)

	err := encoder.Encode(&apiv2.GetFunctionRunResponse{
		Data: &apiv2.FunctionRun{
			Id:     "run-id",
			Status: apiv2.FunctionRunStatus_FUNCTION_RUN_STATUS_COMPLETED,
		},
	})
	require.NoError(t, err)

	require.JSONEq(t, `{"data":{"id":"run-id","status":"COMPLETED"}}`, buf.String())
	require.True(t, bytes.HasSuffix(buf.Bytes(), []byte("\n")))
}

func TestResponseEnumMarshalerEncoderReturnsWriteErrors(t *testing.T) {
	marshaler := newResponseEnumMarshaler()
	encoder := marshaler.NewEncoder(failingWriter{})

	err := encoder.Encode(&apiv2.GetFunctionRunResponse{
		Data: &apiv2.FunctionRun{
			Id:     "run-id",
			Status: apiv2.FunctionRunStatus_FUNCTION_RUN_STATUS_COMPLETED,
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "write failed")
}

func TestResponseEnumMarshalerEncoderReturnsMarshalErrors(t *testing.T) {
	marshaler := newResponseEnumMarshaler()
	encoder := marshaler.NewEncoder(&bytes.Buffer{})

	err := encoder.Encode(func() {})
	require.Error(t, err)
}

func TestResponseEnumMarshalerEncoderReturnsDelimiterWriteErrors(t *testing.T) {
	marshaler := newResponseEnumMarshaler()
	encoder := marshaler.NewEncoder(&failingSecondWriter{})

	err := encoder.Encode(&apiv2.GetFunctionRunResponse{
		Data: &apiv2.FunctionRun{
			Id:     "run-id",
			Status: apiv2.FunctionRunStatus_FUNCTION_RUN_STATUS_COMPLETED,
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "delimiter write failed")
}

func TestShortenResponseEnumNamesHandlesNestedArraysAndSkipsPayloads(t *testing.T) {
	data, err := shortenResponseEnumNames([]byte(`{
		"values": ["TRACE_STEP_OP_SEND_EVENT", "TRACE_SPAN_STATUS_FAILED"],
		"input": "TRACE_SPAN_STATUS_COMPLETED",
		"output": "FUNCTION_RUN_STATUS_FAILED",
		"metadata": [{"status": "TRACE_SPAN_STATUS_WAITING"}]
	}`))
	require.NoError(t, err)

	var body map[string]any
	require.NoError(t, json.Unmarshal(data, &body))

	values := body["values"].([]any)
	require.Equal(t, "SEND_EVENT", values[0])
	require.Equal(t, "FAILED", values[1])
	require.Equal(t, "TRACE_SPAN_STATUS_COMPLETED", body["input"])
	require.Equal(t, "FUNCTION_RUN_STATUS_FAILED", body["output"])
	require.Equal(t, "WAITING", body["metadata"].([]any)[0].(map[string]any)["status"])
}

func TestShortenResponseEnumNamesReturnsDecodeErrors(t *testing.T) {
	data, err := shortenResponseEnumNames([]byte(`{`))
	require.Nil(t, data)
	require.Error(t, err)
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

type failingSecondWriter struct {
	writes int
}

func (w *failingSecondWriter) Write(data []byte) (int, error) {
	if w.writes > 0 {
		return 0, errors.New("delimiter write failed")
	}
	w.writes++
	return len(data), nil
}
