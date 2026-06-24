package apiv2

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/inngest/inngest/pkg/api/v2/apiv2base"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	defaultExperimentsLimit = 20
	maxExperimentsLimit     = 100
	defaultSessionsLimit    = 20
	maxSessionsLimit        = 100
	defaultSessionRunsLimit = 20
	maxSessionRunsLimit     = 100
)

func (s *Service) ListExperiments(ctx context.Context, req *apiv2.ListExperimentsRequest) (*apiv2.ListExperimentsResponse, error) {
	if result := s.rateLimiter.CheckRateLimit(ctx, apiv2.V2_ListExperiments_FullMethodName); result.Limited {
		return nil, s.base.NewError(http.StatusTooManyRequests, apiv2base.ErrorRateLimited,
			"API rate limit exceeded. The request was rejected and no experiments were fetched.")
	}

	if s.experiments == nil {
		return nil, s.base.NewError(http.StatusNotImplemented, apiv2base.ErrorNotImplemented, "Experiments not implemented in OSS")
	}

	cursor, limit, err := experimentsPageOpts(req.GetCursor(), req.GetLimit())
	if err != nil {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, err.Error())
	}

	result, err := s.experiments.ListExperiments(ctx, ExperimentListOpts{Cursor: cursor, Limit: limit})
	if err != nil {
		return nil, s.base.NewError(http.StatusInternalServerError, apiv2base.ErrorInternalError, "Unable to fetch experiments")
	}
	if result == nil {
		result = &ExperimentListResult{}
	}

	data := make([]*apiv2.Experiment, 0, len(result.Experiments))
	for _, experiment := range result.Experiments {
		data = append(data, toAPIExperiment(experiment))
	}

	return &apiv2.ListExperimentsResponse{
		Data:     data,
		Metadata: &apiv2.ResponseMetadata{FetchedAt: timestamppb.Now()},
		Page:     experimentsPage(result.Experiments, limit, result.HasMore),
	}, nil
}

func (s *Service) GetExperiment(ctx context.Context, req *apiv2.GetExperimentRequest) (*apiv2.GetExperimentResponse, error) {
	if req.FunctionId == "" || req.ExperimentName == "" {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorMissingField, "Function ID and experiment name are required")
	}

	if result := s.rateLimiter.CheckRateLimit(ctx, apiv2.V2_GetExperiment_FullMethodName); result.Limited {
		return nil, s.base.NewError(http.StatusTooManyRequests, apiv2base.ErrorRateLimited,
			"API rate limit exceeded. The request was rejected and no experiment was fetched.")
	}

	if s.experiments == nil {
		return nil, s.base.NewError(http.StatusNotImplemented, apiv2base.ErrorNotImplemented, "Experiments not implemented in OSS")
	}

	detail, err := s.experiments.GetExperiment(ctx, ExperimentDetailOpts{
		FunctionID:     decodePathParam(req.FunctionId),
		ExperimentName: decodePathParam(req.ExperimentName),
		Variant:        req.Variant,
	})
	if err != nil {
		return nil, s.base.NewError(http.StatusInternalServerError, apiv2base.ErrorInternalError, "Unable to fetch experiment")
	}

	return &apiv2.GetExperimentResponse{
		Data:     toAPIExperimentDetail(detail),
		Metadata: &apiv2.ResponseMetadata{FetchedAt: timestamppb.Now()},
	}, nil
}

func (s *Service) ListSessions(ctx context.Context, req *apiv2.ListSessionsRequest) (*apiv2.ListSessionsResponse, error) {
	if req.SessionKey == "" {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorMissingField, "Session key is required")
	}

	if result := s.rateLimiter.CheckRateLimit(ctx, apiv2.V2_ListSessions_FullMethodName); result.Limited {
		return nil, s.base.NewError(http.StatusTooManyRequests, apiv2base.ErrorRateLimited,
			"API rate limit exceeded. The request was rejected and no sessions were fetched.")
	}

	if s.sessions == nil {
		return nil, s.base.NewError(http.StatusNotImplemented, apiv2base.ErrorNotImplemented, "Sessions not implemented in OSS")
	}

	cursor, limit, err := sessionsPageOpts(req.GetCursor(), req.GetLimit())
	if err != nil {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, err.Error())
	}

	result, err := s.sessions.ListSessions(ctx, SessionListOpts{
		SessionKey: decodePathParam(req.SessionKey),
		Cursor:     cursor,
		Limit:      limit,
	})
	if err != nil {
		return nil, s.base.NewError(http.StatusInternalServerError, apiv2base.ErrorInternalError, "Unable to fetch sessions")
	}
	if result == nil {
		result = &SessionListResult{}
	}

	data := make([]*apiv2.SessionGroup, 0, len(result.Sessions))
	for _, session := range result.Sessions {
		data = append(data, toAPISessionGroup(session))
	}

	return &apiv2.ListSessionsResponse{
		Data:     data,
		Metadata: &apiv2.ResponseMetadata{FetchedAt: timestamppb.Now()},
		Page:     sessionsPage(result.Sessions, limit, result.HasMore),
	}, nil
}

func (s *Service) ListSessionRuns(ctx context.Context, req *apiv2.ListSessionRunsRequest) (*apiv2.ListSessionRunsResponse, error) {
	if req.SessionKey == "" || req.SessionId == "" {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorMissingField, "Session key and session ID are required")
	}

	if result := s.rateLimiter.CheckRateLimit(ctx, apiv2.V2_ListSessionRuns_FullMethodName); result.Limited {
		return nil, s.base.NewError(http.StatusTooManyRequests, apiv2base.ErrorRateLimited,
			"API rate limit exceeded. The request was rejected and no session runs were fetched.")
	}

	if s.sessions == nil {
		return nil, s.base.NewError(http.StatusNotImplemented, apiv2base.ErrorNotImplemented, "Sessions not implemented in OSS")
	}

	cursor, limit, err := sessionRunsPageOpts(req.GetCursor(), req.GetLimit())
	if err != nil {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, err.Error())
	}

	result, err := s.sessions.ListSessionRuns(ctx, SessionRunsOpts{
		SessionKey: decodePathParam(req.SessionKey),
		SessionID:  decodePathParam(req.SessionId),
		Cursor:     cursor,
		Limit:      limit,
	})
	if err != nil {
		return nil, s.base.NewError(http.StatusInternalServerError, apiv2base.ErrorInternalError, "Unable to fetch session runs")
	}
	if result == nil {
		result = &SessionRunsResult{}
	}

	data := make([]*apiv2.SessionRun, 0, len(result.Runs))
	for _, run := range result.Runs {
		data = append(data, toAPISessionRun(run))
	}

	return &apiv2.ListSessionRunsResponse{
		Data:     data,
		Metadata: &apiv2.ResponseMetadata{FetchedAt: timestamppb.Now()},
		Page:     sessionRunsPage(result.Runs, limit, result.HasMore),
	}, nil
}

func experimentsPageOpts(cursor string, requestedLimit int32) (*ExperimentCursor, int, error) {
	limit, err := boundedLimit(requestedLimit, defaultExperimentsLimit, maxExperimentsLimit)
	if err != nil {
		return nil, 0, err
	}
	if cursor == "" {
		return nil, limit, nil
	}
	var parsed ExperimentCursor
	if err := decodeCursor(cursor, &parsed); err != nil || parsed.LastSeen.IsZero() || parsed.FunctionID == "" || parsed.Name == "" {
		return nil, 0, fmt.Errorf("Cursor is invalid")
	}
	return &parsed, limit, nil
}

func sessionsPageOpts(cursor string, requestedLimit int32) (*SessionCursor, int, error) {
	limit, err := boundedLimit(requestedLimit, defaultSessionsLimit, maxSessionsLimit)
	if err != nil {
		return nil, 0, err
	}
	if cursor == "" {
		return nil, limit, nil
	}
	var parsed SessionCursor
	if err := decodeCursor(cursor, &parsed); err != nil || parsed.LastActiveAt.IsZero() || parsed.SessionID == "" {
		return nil, 0, fmt.Errorf("Cursor is invalid")
	}
	return &parsed, limit, nil
}

func sessionRunsPageOpts(cursor string, requestedLimit int32) (*SessionRunCursor, int, error) {
	limit, err := boundedLimit(requestedLimit, defaultSessionRunsLimit, maxSessionRunsLimit)
	if err != nil {
		return nil, 0, err
	}
	if cursor == "" {
		return nil, limit, nil
	}
	var parsed SessionRunCursor
	if err := decodeCursor(cursor, &parsed); err != nil || parsed.QueuedAt.IsZero() || parsed.RunID == "" {
		return nil, 0, fmt.Errorf("Cursor is invalid")
	}
	return &parsed, limit, nil
}

func boundedLimit(requestedLimit int32, defaultLimit, maxLimit int) (int, error) {
	limit := int(requestedLimit)
	if limit == 0 {
		limit = defaultLimit
	}
	if limit < 1 {
		return 0, fmt.Errorf("Limit must be at least 1")
	}
	if limit > maxLimit {
		return 0, fmt.Errorf("Limit cannot exceed %d", maxLimit)
	}
	return limit, nil
}

func experimentsPage(experiments []Experiment, limit int, hasMore bool) *apiv2.Page {
	page := &apiv2.Page{HasMore: hasMore, Limit: int32(limit)}
	if hasMore && len(experiments) > 0 {
		cursor := encodeCursor(ExperimentCursor{
			LastSeen:   experiments[len(experiments)-1].LastSeen,
			FunctionID: experiments[len(experiments)-1].FunctionID,
			Name:       experiments[len(experiments)-1].Name,
		})
		page.Cursor = &cursor
	}
	return page
}

func sessionsPage(sessions []SessionGroup, limit int, hasMore bool) *apiv2.Page {
	page := &apiv2.Page{HasMore: hasMore, Limit: int32(limit)}
	if hasMore && len(sessions) > 0 {
		cursor := encodeCursor(SessionCursor{
			LastActiveAt: sessions[len(sessions)-1].LastActiveAt,
			SessionID:    sessions[len(sessions)-1].SessionID,
		})
		page.Cursor = &cursor
	}
	return page
}

func sessionRunsPage(runs []SessionRun, limit int, hasMore bool) *apiv2.Page {
	page := &apiv2.Page{HasMore: hasMore, Limit: int32(limit)}
	if hasMore && len(runs) > 0 {
		cursor := encodeCursor(SessionRunCursor{
			QueuedAt: runs[len(runs)-1].QueuedAt,
			RunID:    runs[len(runs)-1].ID,
		})
		page.Cursor = &cursor
	}
	return page
}

func encodeCursor(v any) string {
	data, _ := json.Marshal(v)
	return base64.RawURLEncoding.EncodeToString(data)
}

func decodeCursor(cursor string, out any) error {
	data, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, out)
}

func toAPIExperiment(experiment Experiment) *apiv2.Experiment {
	return &apiv2.Experiment{
		Name:              experiment.Name,
		FunctionId:        experiment.FunctionID,
		FunctionSlug:      experiment.FunctionSlug,
		SelectionStrategy: experiment.SelectionStrategy,
		Variants:          experiment.Variants,
		VariantCount:      int32(len(experiment.Variants)),
		TotalRuns:         int32(experiment.TotalRuns),
		FirstSeen:         optionalTimestamp(experiment.FirstSeen),
		LastSeen:          optionalTimestamp(experiment.LastSeen),
	}
}

func toAPIExperimentDetail(detail *ExperimentDetail) *apiv2.ExperimentDetail {
	if detail == nil {
		return nil
	}
	variants := make([]*apiv2.ExperimentVariantMetrics, 0, len(detail.Variants))
	for _, variant := range detail.Variants {
		metrics := make([]*apiv2.ExperimentVariantMetric, 0, len(variant.Metrics))
		for _, metric := range variant.Metrics {
			metrics = append(metrics, &apiv2.ExperimentVariantMetric{
				Key: metric.Key,
				Avg: metric.Avg,
				Min: metric.Min,
				Max: metric.Max,
			})
		}
		variants = append(variants, &apiv2.ExperimentVariantMetrics{
			VariantName: variant.VariantName,
			RunCount:    int32(variant.RunCount),
			Metrics:     metrics,
		})
	}
	weights := make([]*apiv2.ExperimentVariantWeight, 0, len(detail.VariantWeights))
	for _, weight := range detail.VariantWeights {
		weights = append(weights, &apiv2.ExperimentVariantWeight{
			VariantName: weight.VariantName,
			Weight:      weight.Weight,
		})
	}
	return &apiv2.ExperimentDetail{
		Name:              detail.Name,
		Variants:          variants,
		VariantWeights:    weights,
		FirstSeen:         optionalTimestamp(detail.FirstSeen),
		LastSeen:          optionalTimestamp(detail.LastSeen),
		SelectionStrategy: detail.SelectionStrategy,
	}
}

func toAPISessionGroup(session SessionGroup) *apiv2.SessionGroup {
	functions := make([]*apiv2.SessionFunction, 0, len(session.Functions))
	for _, fn := range session.Functions {
		functions = append(functions, &apiv2.SessionFunction{
			Slug: fn.Slug,
			Name: fn.Name,
		})
	}
	return &apiv2.SessionGroup{
		SessionKey:     session.SessionKey,
		SessionId:      session.SessionID,
		RunCount:       int32(session.RunCount),
		FailedRunCount: int32(session.FailedRunCount),
		FailureRate:    session.FailureRate,
		LastActiveAt:   optionalTimestamp(session.LastActiveAt),
		Functions:      functions,
	}
}

func toAPISessionRun(run SessionRun) *apiv2.SessionRun {
	return &apiv2.SessionRun{
		Id:           run.ID,
		FunctionSlug: run.FunctionSlug,
		EventName:    run.EventName,
		Status:       run.Status,
		QueuedAt:     optionalTimestamp(run.QueuedAt),
		StartedAt:    optionalTimestampPtr(run.StartedAt),
		EndedAt:      optionalTimestampPtr(run.EndedAt),
	}
}

func optionalTimestamp(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}

func optionalTimestampPtr(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return optionalTimestamp(*t)
}
