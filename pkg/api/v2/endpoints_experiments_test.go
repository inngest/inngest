package apiv2

import (
	"context"
	"testing"

	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Experiments is an Insights/ClickHouse-backed product that the OSS dev
// server cannot serve, so its endpoints always return 501 in OSS. Sessions
// moved out of this file once implemented -- see endpoints_sessions_test.go.
func TestExperimentsNotImplementedInOSS(t *testing.T) {
	service := NewService(ServiceOptions{})

	t.Run("ListExperiments", func(t *testing.T) {
		_, err := service.ListExperiments(context.Background(), &apiv2.ListExperimentsRequest{})
		require.Equal(t, codes.Unimplemented, status.Code(err))
		require.ErrorContains(t, err, "Experiments not implemented in OSS")
	})
	t.Run("GetExperiment", func(t *testing.T) {
		_, err := service.GetExperiment(context.Background(), &apiv2.GetExperimentRequest{FunctionId: "fn", ExperimentId: "exp"})
		require.Equal(t, codes.Unimplemented, status.Code(err))
		require.ErrorContains(t, err, "Experiments not implemented in OSS")
	})
}
