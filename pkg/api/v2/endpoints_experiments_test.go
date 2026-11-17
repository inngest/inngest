package apiv2

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/inngest"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// insertExperimentRun inserts a dev server run on app "app" assigned to
// variant of experiment name, w/ a run-scoped score per entry in scores.
func insertExperimentRun(t *testing.T, db *sql.DB, appID, functionID uuid.UUID, runID, name, variant string, scores map[string]float64, queuedAt time.Time) {
	t.Helper()
	_, err := db.ExecContext(t.Context(), `
INSERT INTO inngest.runs
  (account_id, env_id, run_id, queued_at, scheduled_at, app_id, app_name, function_id, function_slug, status, attributes, inputs)
VALUES (?, ?, ?, ?, ?, ?, 'app', ?, 'app-fn', 'Completed', '{}', '[]');`,
		consts.DevServerAccountID.String(), consts.DevServerEnvID.String(), runID, queuedAt, queuedAt,
		appID.String(), functionID.String(),
	)
	require.NoError(t, err)

	md := func(kind string, values any) {
		b, err := json.Marshal(values)
		require.NoError(t, err)
		_, err = db.ExecContext(t.Context(), `
INSERT INTO inngest.run_metadata
  (account_id, env_id, run_id, run_queued_at, span_id, scope, kind, is_user, "values", created_at)
VALUES (?, ?, ?, ?, ?, 'run', ?, false, ?, ?);`,
			consts.DevServerAccountID.String(), consts.DevServerEnvID.String(), runID, queuedAt, runID+"-span", kind, string(b), queuedAt,
		)
		require.NoError(t, err)
	}
	md("experiment", map[string]any{"name": name, "variant": variant})
	for k, v := range scores {
		md("score."+k, map[string]any{"value": v})
	}
}

func experimentsService(t *testing.T) (*Service, *mockFunctionProvider, uuid.UUID) {
	db, cleanup := newTestDuckDB(t)
	t.Cleanup(cleanup)

	appID, functionID := uuid.New(), uuid.New()
	now := time.Now().UTC()
	insertExperimentRun(t, db, appID, functionID, "r1", "checkout", "a", map[string]float64{"latency": 10}, now.Add(-2*time.Hour))
	insertExperimentRun(t, db, appID, functionID, "r2", "checkout", "b", map[string]float64{"latency": 30}, now.Add(-time.Hour))

	functions := &mockFunctionProvider{}
	functions.On("GetFunctionByApp", mock.Anything, "app", "fn").Return(inngest.DeployedFunction{ID: functionID}, nil)
	functions.On("GetFunctionByApp", mock.Anything, "app", "missing").Return(inngest.DeployedFunction{}, ErrFunctionNotFound)

	return NewService(ServiceOptions{DuckDB: db, Functions: functions}), functions, functionID
}

func TestExperiments_RequireDuckDB(t *testing.T) {
	service := NewService(ServiceOptions{})

	_, err := service.ListExperiments(context.Background(), &apiv2.ListExperimentsRequest{})
	require.Equal(t, codes.Unimplemented, status.Code(err))
	require.ErrorContains(t, err, "Experiments requires dual-write")

	_, err = service.GetExperiment(context.Background(), &apiv2.GetExperimentRequest{AppId: "app", FunctionId: "fn", ExperimentId: "exp"})
	require.Equal(t, codes.Unimplemented, status.Code(err))
	require.ErrorContains(t, err, "Experiments requires dual-write")
}

func TestListExperiments(t *testing.T) {
	service, _, _ := experimentsService(t)
	ctx := context.Background()

	resp, err := service.ListExperiments(ctx, &apiv2.ListExperimentsRequest{})
	require.NoError(t, err)
	require.False(t, resp.Page.HasMore)
	require.Len(t, resp.Data, 1)
	exp := resp.Data[0]
	require.Equal(t, "checkout", exp.Id)
	require.Equal(t, []string{"a", "b"}, exp.Variants)
	require.EqualValues(t, 2, exp.VariantCount)
	require.EqualValues(t, 2, exp.TotalRuns)
	require.Equal(t, "fn", exp.Function.Id)
	require.Equal(t, "app", exp.Function.App.Id)
	require.Equal(t, "app-fn", exp.Function.GetSlug())

	resp, err = service.ListExperiments(ctx, &apiv2.ListExperimentsRequest{AppId: "app", FunctionId: "fn"})
	require.NoError(t, err)
	require.Len(t, resp.Data, 1)

	_, err = service.ListExperiments(ctx, &apiv2.ListExperimentsRequest{AppId: "app"})
	require.Equal(t, codes.InvalidArgument, status.Code(err))

	_, err = service.ListExperiments(ctx, &apiv2.ListExperimentsRequest{AppId: "app", FunctionId: "missing"})
	require.Equal(t, codes.NotFound, status.Code(err))

	_, err = service.ListExperiments(ctx, &apiv2.ListExperimentsRequest{Cursor: new("nope")})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestGetExperiment(t *testing.T) {
	service, _, _ := experimentsService(t)
	ctx := context.Background()

	resp, err := service.GetExperiment(ctx, &apiv2.GetExperimentRequest{AppId: "app", FunctionId: "fn", ExperimentId: "checkout"})
	require.NoError(t, err)
	require.Equal(t, "checkout", resp.Data.Id)
	require.Len(t, resp.Data.Variants, 2)
	require.Equal(t, "a", resp.Data.Variants[0].VariantName)
	require.EqualValues(t, 1, resp.Data.Variants[0].RunCount)
	require.Equal(t, []*apiv2.ExperimentVariantMetric{{Key: "latency", Avg: 10, Min: 10, Max: 10}}, resp.Data.Variants[0].Metrics)
	require.NotNil(t, resp.Data.VariantWeights, "never null")

	resp, err = service.GetExperiment(ctx, &apiv2.GetExperimentRequest{AppId: "app", FunctionId: "fn", ExperimentId: "checkout", Variant: new("b")})
	require.NoError(t, err)
	require.Len(t, resp.Data.Variants, 1)
	require.Equal(t, "b", resp.Data.Variants[0].VariantName)

	_, err = service.GetExperiment(ctx, &apiv2.GetExperimentRequest{AppId: "app", FunctionId: "fn", ExperimentId: "checkout", Variant: new("c")})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	require.ErrorContains(t, err, "was not found for experiment")

	_, err = service.GetExperiment(ctx, &apiv2.GetExperimentRequest{AppId: "app", FunctionId: "fn", ExperimentId: "nope"})
	require.Equal(t, codes.NotFound, status.Code(err))

	_, err = service.GetExperiment(ctx, &apiv2.GetExperimentRequest{AppId: "app", FunctionId: "missing", ExperimentId: "checkout"})
	require.Equal(t, codes.NotFound, status.Code(err))

	_, err = service.GetExperiment(ctx, &apiv2.GetExperimentRequest{AppId: "app", FunctionId: "fn"})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}
