package apiv2

import (
	"encoding/json"
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
