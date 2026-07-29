package apiv2

import (
	"context"
	"testing"

	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Experiments and sessions are Insights/ClickHouse-backed products that the OSS
// dev server cannot serve, so the endpoints always return 501 in OSS.
func TestExperimentsSessionsNotImplementedInOSS(t *testing.T) {
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
	t.Run("ListSessionKeys", func(t *testing.T) {
		_, err := service.ListSessionKeys(context.Background(), &apiv2.ListSessionKeysRequest{})
		require.Equal(t, codes.Unimplemented, status.Code(err))
		require.ErrorContains(t, err, "Sessions not implemented in OSS")
	})
	t.Run("ListSessions", func(t *testing.T) {
		_, err := service.ListSessions(context.Background(), &apiv2.ListSessionsRequest{SessionKey: "k"})
		require.Equal(t, codes.Unimplemented, status.Code(err))
		require.ErrorContains(t, err, "Sessions not implemented in OSS")
	})
	t.Run("ListSessionRuns", func(t *testing.T) {
		_, err := service.ListSessionRuns(context.Background(), &apiv2.ListSessionRunsRequest{SessionKey: "k", SessionId: "id"})
		require.Equal(t, codes.Unimplemented, status.Code(err))
		require.ErrorContains(t, err, "Sessions not implemented in OSS")
	})
}
