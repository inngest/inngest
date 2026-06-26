package apiv2

import (
	"context"
	"testing"
	"time"

	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type stubExperimentProvider struct {
	list      []Experiment
	detail    *ExperimentDetail
	detailErr error
	opts      ExperimentListOpts
	more      bool
}

func (s *stubExperimentProvider) ListExperiments(ctx context.Context, opts ExperimentListOpts) (*ExperimentListResult, error) {
	s.opts = opts
	return &ExperimentListResult{Experiments: s.list, HasMore: s.more}, nil
}

func (s *stubExperimentProvider) GetExperiment(ctx context.Context, opts ExperimentDetailOpts) (*ExperimentDetail, error) {
	return s.detail, s.detailErr
}

type stubSessionProvider struct {
	list     []SessionGroup
	runs     []SessionRun
	opts     SessionListOpts
	ropts    SessionRunsOpts
	listMore bool
	runsMore bool
}

func (s *stubSessionProvider) ListSessions(ctx context.Context, opts SessionListOpts) (*SessionListResult, error) {
	s.opts = opts
	return &SessionListResult{Sessions: s.list, HasMore: s.listMore}, nil
}

func (s *stubSessionProvider) ListSessionRuns(ctx context.Context, opts SessionRunsOpts) (*SessionRunsResult, error) {
	s.ropts = opts
	return &SessionRunsResult{Runs: s.runs, HasMore: s.runsMore}, nil
}

func TestService_ListExperiments(t *testing.T) {
	firstSeen := time.Date(2026, 6, 22, 10, 0, 0, 0, time.UTC)
	lastSeen := firstSeen.Add(time.Hour)
	provider := &stubExperimentProvider{
		list: []Experiment{{
			Name:              "quality",
			FunctionID:        "fn-id",
			FunctionSlug:      "app-quality",
			SelectionStrategy: "weighted",
			Variants:          []string{"control", "test"},
			TotalRuns:         12,
			FirstSeen:         firstSeen,
			LastSeen:          lastSeen,
		}},
		more: true,
	}
	service := NewService(ServiceOptions{
		Experiments: provider,
	})

	resp, err := service.ListExperiments(context.Background(), &apiv2.ListExperimentsRequest{})

	require.NoError(t, err)
	require.Len(t, resp.Data, 1)
	require.Equal(t, "quality", resp.Data[0].Name)
	require.Equal(t, int32(2), resp.Data[0].VariantCount)
	require.Equal(t, int32(12), resp.Data[0].TotalRuns)
	require.Equal(t, firstSeen, resp.Data[0].FirstSeen.AsTime())
	require.True(t, resp.Page.HasMore)
	require.Equal(t, int32(defaultExperimentsLimit), resp.Page.Limit)
	require.NotNil(t, resp.Page.Cursor)
	require.Equal(t, defaultExperimentsLimit, provider.opts.Limit)
}

func TestService_GetExperiment(t *testing.T) {
	service := NewService(ServiceOptions{
		Experiments: &stubExperimentProvider{
			detail: &ExperimentDetail{
				Name: "quality",
				Variants: []ExperimentVariantMetrics{{
					VariantName: "control",
					RunCount:    5,
					Metrics: []ExperimentVariantMetric{{
						Key: "accuracy",
						Avg: 0.8,
						Min: 0.5,
						Max: 1,
					}},
				}},
				VariantWeights: []ExperimentVariantWeight{{VariantName: "control", Weight: 50}},
			},
		},
	})

	resp, err := service.GetExperiment(context.Background(), &apiv2.GetExperimentRequest{
		FunctionId:     "fn-id",
		ExperimentName: "quality",
	})

	require.NoError(t, err)
	require.Equal(t, "quality", resp.Data.Name)
	require.Len(t, resp.Data.Variants, 1)
	require.Equal(t, "accuracy", resp.Data.Variants[0].Metrics[0].Key)
	require.Equal(t, 50.0, resp.Data.VariantWeights[0].Weight)
}

func TestService_GetExperiment_NotFound(t *testing.T) {
	service := NewService(ServiceOptions{
		Experiments: &stubExperimentProvider{detailErr: ErrExperimentNotFound},
	})

	_, err := service.GetExperiment(context.Background(), &apiv2.GetExperimentRequest{
		FunctionId:     "fn-id",
		ExperimentName: "quality",
	})

	require.ErrorContains(t, err, "Experiment not found")
	require.Equal(t, codes.NotFound, status.Code(err))
}

func TestService_GetExperiment_NilDetail(t *testing.T) {
	service := NewService(ServiceOptions{
		Experiments: &stubExperimentProvider{detail: nil},
	})

	_, err := service.GetExperiment(context.Background(), &apiv2.GetExperimentRequest{
		FunctionId:     "fn-id",
		ExperimentName: "quality",
	})

	require.ErrorContains(t, err, "Experiment not found")
	require.Equal(t, codes.NotFound, status.Code(err))
}

func TestService_ListSessions_SessionKeyNotPathDecoded(t *testing.T) {
	provider := &stubSessionProvider{}
	service := NewService(ServiceOptions{Sessions: provider})

	// session_key is a query param on GET /sessions, so it must reach the
	// provider verbatim rather than being URL path-unescaped a second time.
	_, err := service.ListSessions(context.Background(), &apiv2.ListSessionsRequest{SessionKey: "team%2Fapp"})

	require.NoError(t, err)
	require.Equal(t, "team%2Fapp", provider.opts.SessionKey)
}

func TestService_ListSessions(t *testing.T) {
	lastActiveAt := time.Date(2026, 6, 22, 10, 0, 0, 0, time.UTC)
	provider := &stubSessionProvider{
		list: []SessionGroup{{
			SessionKey:     "conversation_id",
			SessionID:      "conv-1",
			RunCount:       10,
			FailedRunCount: 2,
			FailureRate:    20,
			LastActiveAt:   lastActiveAt,
			Functions:      []SessionFunction{{Slug: "app-fn", Name: "Function"}},
		}},
	}
	service := NewService(ServiceOptions{
		Sessions: provider,
	})

	resp, err := service.ListSessions(context.Background(), &apiv2.ListSessionsRequest{SessionKey: "conversation_id"})

	require.NoError(t, err)
	require.Len(t, resp.Data, 1)
	require.Equal(t, "conv-1", resp.Data[0].SessionId)
	require.Equal(t, int32(2), resp.Data[0].FailedRunCount)
	require.Equal(t, "app-fn", resp.Data[0].Functions[0].Slug)
	require.Equal(t, defaultSessionsLimit, provider.opts.Limit)
}

func TestService_ListSessionRuns(t *testing.T) {
	eventName := "message.created"
	queuedAt := time.Date(2026, 6, 22, 10, 0, 0, 0, time.UTC)
	provider := &stubSessionProvider{
		runs: []SessionRun{{
			ID:           "run-id",
			FunctionSlug: "app-fn",
			EventName:    &eventName,
			Status:       "Completed",
			QueuedAt:     queuedAt,
		}},
	}
	service := NewService(ServiceOptions{
		Sessions: provider,
	})

	resp, err := service.ListSessionRuns(context.Background(), &apiv2.ListSessionRunsRequest{
		SessionKey: "conversation_id",
		SessionId:  "conv-1",
	})

	require.NoError(t, err)
	require.Len(t, resp.Data, 1)
	require.Equal(t, "run-id", resp.Data[0].Id)
	require.Equal(t, eventName, resp.Data[0].GetEventName())
	require.Equal(t, queuedAt, resp.Data[0].QueuedAt.AsTime())
	require.Equal(t, defaultSessionRunsLimit, provider.ropts.Limit)
}
