package apiv2

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	apiv2base "github.com/inngest/inngest/pkg/api/v2/apiv2base"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestResponseEnumMarshalerShortensAPIEnumPrefixes(t *testing.T) {
	output, err := structpb.NewStruct(map[string]any{
		"literal": "FUNCTION_RUN_STATUS_FAILED",
		"fields": map[string]any{
			"x": map[string]any{"nullValue": "FUNCTION_RUN_STATUS_FAILED"},
		},
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
	require.Equal(t, "FUNCTION_RUN_STATUS_FAILED", run["output"].(map[string]any)["fields"].(map[string]any)["x"].(map[string]any)["nullValue"])
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

func TestResponseEnumMarshalerUsesSandboxJSONContract(t *testing.T) {
	outcome := apiv2.SandboxOutcome_SANDBOX_OUTCOME_TIMED_OUT
	marshaler := newResponseEnumMarshaler()
	data, err := marshaler.Marshal(&apiv2.GetSandboxResponse{
		Data: &apiv2.Sandbox{
			Id:           "22222222-2222-2222-2222-222222222222",
			VpcId:        "11111111-1111-1111-1111-111111111111",
			Name:         "test-sandbox",
			Generation:   3,
			DesiredState: apiv2.SandboxDesiredState_SANDBOX_DESIRED_STATE_TERMINATED,
			Phase:        apiv2.SandboxPhase_SANDBOX_PHASE_TERMINAL,
			Outcome:      &outcome,
			CleanupState: apiv2.SandboxCleanupState_SANDBOX_CLEANUP_STATE_IN_PROGRESS,
		},
	})
	require.NoError(t, err)

	var body map[string]any
	require.NoError(t, json.Unmarshal(data, &body))
	sandbox := body["data"].(map[string]any)
	require.Equal(t, "11111111-1111-1111-1111-111111111111", sandbox["vpcId"])
	require.Equal(t, float64(3), sandbox["generation"])
	require.Equal(t, "TERMINATED", sandbox["desiredState"])
	require.Equal(t, "TERMINAL", sandbox["phase"])
	require.Equal(t, "TIMED_OUT", sandbox["outcome"])
	require.Equal(t, "IN_PROGRESS", sandbox["cleanupState"])
	require.NotContains(t, sandbox, "desired_state")
	require.NotContains(t, sandbox, "cleanup_state")
}

func TestResponseEnumMarshalerOmitsUnknownSandboxOutcome(t *testing.T) {
	marshaler := newResponseEnumMarshaler()
	data, err := marshaler.Marshal(&apiv2.GetSandboxResponse{
		Data: &apiv2.Sandbox{
			Id:           "22222222-2222-2222-2222-222222222222",
			DesiredState: apiv2.SandboxDesiredState_SANDBOX_DESIRED_STATE_RUNNING,
			Phase:        apiv2.SandboxPhase_SANDBOX_PHASE_PENDING,
			CleanupState: apiv2.SandboxCleanupState_SANDBOX_CLEANUP_STATE_NOT_REQUIRED,
		},
	})
	require.NoError(t, err)

	var body map[string]any
	require.NoError(t, json.Unmarshal(data, &body))
	sandbox := body["data"].(map[string]any)
	require.NotContains(t, sandbox, "outcome")
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

func TestResponseEnumMarshalerPreservesExecOutputExactly(t *testing.T) {
	stdout := "FUNCTION_RUN_STATUS_FAILED\nTRACE_STEP_OP_SEND_EVENT\x00 café 👩🏽‍💻"
	stderr := "SANDBOX_OUTCOME_TIMED_OUT\nreplacement: �"
	data, err := newResponseEnumMarshaler().Marshal(&apiv2.ExecSandboxResponse{Data: &apiv2.ExecSandboxData{
		Stdout: &stdout,
		Stderr: &stderr,
	}})
	require.NoError(t, err)

	var body struct {
		Data struct {
			Stdout string `json:"stdout"`
			Stderr string `json:"stderr"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(data, &body))
	require.Equal(t, stdout, body.Data.Stdout)
	require.Equal(t, stderr, body.Data.Stderr)
}

func TestResponseEnumMarshalerRejectsInvalidExecOutputEncoding(t *testing.T) {
	invalid := string([]byte{0xff})
	data, err := newResponseEnumMarshaler().Marshal(&apiv2.ExecSandboxResponse{Data: &apiv2.ExecSandboxData{Stdout: &invalid}})
	require.Nil(t, data)
	require.Equal(t, codes.DataLoss, status.Code(err))
	require.Contains(t, status.Convert(err).Message(), apiv2base.ErrorOutputEncodingInvalid)
}

func TestShortenResponseEnumNamesReturnsDecodeErrors(t *testing.T) {
	descriptor := (&apiv2.ExecSandboxResponse{}).ProtoReflect().Descriptor()
	data, err := shortenResponseEnumNames([]byte(`{`), descriptor)
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
