package apiv2

import (
	"net/url"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/grpc-ecosystem/grpc-gateway/v2/utilities"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestV2QueryParameterParser(t *testing.T) {
	parser := v2QueryParameterParser{defaultParser: &runtime.DefaultQueryParser{}}

	t.Run("preserves nested and repeated field parsing", func(t *testing.T) {
		rerun := &apiv2.RerunRequest{}
		require.NoError(t, parser.Parse(rerun, url.Values{
			"fromStep.stepId": {"step-1"},
		}, utilities.NewDoubleArray(nil)))
		require.Equal(t, "step-1", rerun.GetFromStep().GetStepId())

		runs := &apiv2.ListRunsRequest{}
		require.NoError(t, parser.Parse(runs, url.Values{
			"status": {"COMPLETED", "FAILED"},
		}, utilities.NewDoubleArray(nil)))
		require.Equal(t, []string{"COMPLETED", "FAILED"}, runs.GetStatus())
	})

	t.Run("preserves default behavior outside API v2", func(t *testing.T) {
		require.NoError(t, parser.Parse(&emptypb.Empty{}, url.Values{
			"ignored": {"value"},
		}, utilities.NewDoubleArray(nil)))
	})
}
