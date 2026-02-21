package history

import (
	"encoding/json"
	"testing"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/stretchr/testify/require"
)

func TestApplyResponse(t *testing.T) {
	errStr := "something went wrong"

	tests := []struct {
		name           string
		resp           state.DriverResponse
		expectedOutput string
	}{
		{
			name: "OpcodeRunComplete with data",
			resp: state.DriverResponse{
				Generator: []*state.GeneratorOpcode{
					{
						Op:   enums.OpcodeRunComplete,
						ID:   "done",
						Data: json.RawMessage(`{"result":"ok"}`),
					},
				},
			},
			expectedOutput: `{"result":"ok"}`,
		},
		{
			name: "OpcodeSyncRunComplete with data",
			resp: state.DriverResponse{
				Generator: []*state.GeneratorOpcode{
					{
						Op:   enums.OpcodeSyncRunComplete,
						ID:   "done",
						Data: json.RawMessage(`{"result":"ok"}`),
					},
				},
			},
			expectedOutput: `{"result":"ok"}`,
		},
		{
			name: "OpcodeRunComplete with nil data",
			resp: state.DriverResponse{
				Generator: []*state.GeneratorOpcode{
					{
						Op: enums.OpcodeRunComplete,
						ID: "done",
						// Data is nil
					},
				},
			},
			expectedOutput: "",
		},
		{
			name: "Regular generator step (OpcodeStep)",
			resp: state.DriverResponse{
				Generator: []*state.GeneratorOpcode{
					{
						Op:   enums.OpcodeStep,
						ID:   "step1",
						Name: "my step",
						Data: json.RawMessage(`{"data":"step-output"}`),
					},
				},
			},
			// OpcodeStep goes through HistoryVisibleStep -> Output(),
			// which for OpcodeStep returns string(g.Data) directly.
			expectedOutput: `{"data":"step-output"}`,
		},
		{
			name: "Non-generator response (simple function)",
			resp: state.DriverResponse{
				Output: "hello",
			},
			expectedOutput: "hello",
		},
		{
			name: "Error response",
			resp: state.DriverResponse{
				Err: &errStr,
			},
			expectedOutput: errStr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &History{}
			err := applyResponse(h, &tt.resp)
			require.NoError(t, err)
			require.NotNil(t, h.Result)
			require.Equal(t, tt.expectedOutput, h.Result.Output)
		})
	}
}
