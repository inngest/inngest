package apiv2

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/api/v2/apiv2base"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/duckdb/dashboards"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	defaultExperimentsLimit = 20
	maxExperimentsLimit     = 100
)

// ListExperiments backs GET /v2/experiments and its per-function route,
// reading pkg/duckdb/dashboards against s.duckDB like Sessions does.
func (s *Service) ListExperiments(ctx context.Context, req *apiv2.ListExperimentsRequest) (*apiv2.ListExperimentsResponse, error) {
	if result := s.rateLimiter.CheckRateLimit(ctx, apiv2.V2_ListExperiments_FullMethodName); result.Limited {
		return nil, s.base.NewError(http.StatusTooManyRequests, apiv2base.ErrorRateLimited,
			"API rate limit exceeded. The request was rejected and no experiments were fetched.")
	}
	if s.duckDB == nil {
		return nil, s.base.NewError(http.StatusNotImplemented, apiv2base.ErrorNotImplemented, "Experiments requires dual-write (--duckdb) to be enabled")
	}
	if (req.AppId == "") != (req.FunctionId == "") {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorMissingField, "App ID and function ID must be provided together")
	}

	limit, err := boundedLimit(req.Limit, defaultExperimentsLimit, maxExperimentsLimit)
	if err != nil {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, err.Error())
	}
	opts := dashboards.ExperimentsOpts{Cursor: req.GetCursor(), Limit: limit}
	if opts.From, opts.Until, err = experimentTimeRange(req.From, req.Until); err != nil {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, err.Error())
	}
	if req.AppId != "" {
		if opts.FunctionID, err = s.experimentFunctionID(ctx, req.AppId, req.FunctionId); err != nil {
			return nil, err
		}
	}

	result, err := dashboards.ListExperiments(ctx, s.duckDB, consts.DevServerAccountID, consts.DevServerEnvID, opts)
	if err != nil {
		return nil, s.sessionsError(err, "Unable to fetch experiments")
	}

	data := make([]*apiv2.Experiment, len(result.Experiments))
	for i, exp := range result.Experiments {
		data[i] = toAPIExperiment(exp)
	}
	var nextCursor string
	if len(result.Experiments) > 0 {
		nextCursor = result.Experiments[len(result.Experiments)-1].Cursor
	}

	return &apiv2.ListExperimentsResponse{
		Data:     data,
		Metadata: &apiv2.ResponseMetadata{FetchedAt: timestamppb.Now()},
		Page:     sessionsPage(limit, result.HasMore, nextCursor),
	}, nil
}

// GetExperiment backs GET /v2/apps/{app_id}/functions/{function_id}/
// experiments/{experiment_id}. Like Cloud, an experiment w/ no runs in the
// window is a 404, and an unknown variant of a known experiment a 400.
func (s *Service) GetExperiment(ctx context.Context, req *apiv2.GetExperimentRequest) (*apiv2.GetExperimentResponse, error) {
	if req.AppId == "" || req.FunctionId == "" || req.ExperimentId == "" {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorMissingField, "App ID, function ID, and experiment ID are required")
	}
	if result := s.rateLimiter.CheckRateLimit(ctx, apiv2.V2_GetExperiment_FullMethodName); result.Limited {
		return nil, s.base.NewError(http.StatusTooManyRequests, apiv2base.ErrorRateLimited,
			"API rate limit exceeded. The request was rejected and no experiment was fetched.")
	}
	if s.duckDB == nil {
		return nil, s.base.NewError(http.StatusNotImplemented, apiv2base.ErrorNotImplemented, "Experiments requires dual-write (--duckdb) to be enabled")
	}

	// The ID is used as is, like a session ID: the gateway matches on the
	// decoded URL path, and {experiment_id=**} needs no double encoding for
	// a /, so decoding again would mangle a name w/ a literal %.
	opts := dashboards.ExperimentOpts{Name: req.ExperimentId}
	var err error
	if opts.From, opts.Until, err = experimentTimeRange(req.From, req.Until); err != nil {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, err.Error())
	}
	if opts.FunctionID, err = s.experimentFunctionID(ctx, req.AppId, req.FunctionId); err != nil {
		return nil, err
	}

	detail, err := dashboards.GetExperiment(ctx, s.duckDB, consts.DevServerAccountID, consts.DevServerEnvID, opts)
	if err != nil {
		return nil, s.sessionsError(err, "Unable to fetch experiment")
	}
	if len(detail.Variants) == 0 {
		return nil, s.base.NewError(http.StatusNotFound, apiv2base.ErrorNotFound, "Experiment not found")
	}
	if variant := req.GetVariant(); variant != "" {
		var only []dashboards.ExperimentVariant
		for _, v := range detail.Variants {
			if v.Name == variant {
				only = append(only, v)
			}
		}
		if len(only) == 0 {
			return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat,
				fmt.Sprintf("Variant %q was not found for experiment %q", variant, detail.Name))
		}
		detail.Variants = only
	}

	return &apiv2.GetExperimentResponse{
		Data:     toAPIExperimentDetail(detail),
		Metadata: &apiv2.ResponseMetadata{FetchedAt: timestamppb.Now()},
	}, nil
}

// experimentFunctionID resolves a public app + function ID to the
// function's UUID, which run rows carry.
func (s *Service) experimentFunctionID(ctx context.Context, appID, functionID string) (uuid.UUID, error) {
	if s.functions == nil {
		return uuid.Nil, s.base.NewError(http.StatusNotImplemented, apiv2base.ErrorNotImplemented, "Experiments by function are not yet implemented")
	}
	fn, err := s.functions.GetFunctionByApp(ctx, decodePathParam(appID), decodePathParam(functionID))
	if err != nil {
		return uuid.Nil, s.getFunctionError(err)
	}
	return fn.ID, nil
}

// experimentTimeRange is the request's from/until, zero where unset
// (dashboards.ResolveExperimentTimeRange defaults them).
func experimentTimeRange(from, until *timestamppb.Timestamp) (f, u time.Time, err error) {
	fromTS, err := optionalTimestamp(from, "from")
	if err != nil {
		return f, u, err
	}
	untilTS, err := optionalTimestamp(until, "until")
	if err != nil {
		return f, u, err
	}
	if fromTS != nil {
		f = *fromTS
	}
	if untilTS != nil {
		u = *untilTS
	}
	return f, u, nil
}

func toAPIExperiment(exp dashboards.Experiment) *apiv2.Experiment {
	return &apiv2.Experiment{
		Id:                exp.Name,
		Function:          toAPIFunctionRef(exp.Function),
		SelectionStrategy: exp.SelectionStrategy,
		Variants:          exp.Variants,
		VariantCount:      int32(len(exp.Variants)),
		TotalRuns:         int32(exp.TotalRuns),
		FirstSeen:         timestamppb.New(exp.FirstSeen),
		LastSeen:          timestamppb.New(exp.LastSeen),
	}
}

func toAPIExperimentDetail(detail *dashboards.ExperimentDetail) *apiv2.ExperimentDetail {
	variants := make([]*apiv2.ExperimentVariantMetrics, len(detail.Variants))
	for i, v := range detail.Variants {
		metrics := make([]*apiv2.ExperimentVariantMetric, len(v.Metrics))
		for j, m := range v.Metrics {
			metrics[j] = &apiv2.ExperimentVariantMetric{Key: m.Key, Avg: m.Avg, Min: m.Min, Max: m.Max}
		}
		variants[i] = &apiv2.ExperimentVariantMetrics{VariantName: v.Name, RunCount: int32(v.RunCount), Metrics: metrics}
	}
	weights := make([]*apiv2.ExperimentVariantWeight, len(detail.Weights))
	for i, w := range detail.Weights {
		weights[i] = &apiv2.ExperimentVariantWeight{VariantName: w.Variant, Weight: w.Weight}
	}
	return &apiv2.ExperimentDetail{
		Id:                detail.Name,
		Variants:          variants,
		VariantWeights:    weights,
		FirstSeen:         timestamppb.New(detail.FirstSeen),
		LastSeen:          timestamppb.New(detail.LastSeen),
		SelectionStrategy: detail.SelectionStrategy,
	}
}
