package apiv2

import (
	"context"
	"fmt"
	"net/http"

	"github.com/inngest/inngest/pkg/api/v2/apiv2base"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/duckdb/dashboards"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	defaultSessionKeysLimit = 20
	maxSessionKeysLimit     = 100
	defaultSessionsLimit    = 20
	maxSessionsLimit        = 100
	defaultSessionRunsLimit = 20
	maxSessionRunsLimit     = 100
)

// ListSessionKeys backs GET /v2/sessions, calling directly into
// pkg/duckdb/dashboards' query-key registry against s.duckDB -- the same
// dual-write connection QueryInsights runs against (endpoints_insights.go),
// gated the same way: a nil s.duckDB means Sessions requires --duckdb dual-
// write to be enabled, not that the environment genuinely has none.
func (s *Service) ListSessionKeys(ctx context.Context, req *apiv2.ListSessionKeysRequest) (*apiv2.ListSessionKeysResponse, error) {
	if result := s.rateLimiter.CheckRateLimit(ctx, apiv2.V2_ListSessionKeys_FullMethodName); result.Limited {
		return nil, s.base.NewError(http.StatusTooManyRequests, apiv2base.ErrorRateLimited,
			"API rate limit exceeded. The request was rejected and no session keys were fetched.")
	}
	if s.duckDB == nil {
		return nil, s.base.NewError(http.StatusNotImplemented, apiv2base.ErrorNotImplemented, "Sessions requires dual-write (--duckdb) to be enabled")
	}

	limit, err := boundedLimit(req.Limit, defaultSessionKeysLimit, maxSessionKeysLimit)
	if err != nil {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, err.Error())
	}

	result, err := dashboards.ListSessionKeys(ctx, s.duckDB, consts.DevServerAccountID, consts.DevServerEnvID, dashboards.SessionKeysOpts{
		Search: req.GetSearch(),
		Cursor: req.GetCursor(),
		Limit:  limit,
	})
	if err != nil {
		return nil, s.base.NewError(http.StatusInternalServerError, apiv2base.ErrorInternalError, "Unable to fetch session keys")
	}

	data := make([]*apiv2.SessionKey, len(result.Keys))
	for i, key := range result.Keys {
		data[i] = &apiv2.SessionKey{
			Id:        key.Key,
			CreatedAt: timestamppb.New(key.CreatedAt),
		}
	}

	var nextCursor string
	if len(result.Keys) > 0 {
		nextCursor = result.Keys[len(result.Keys)-1].Cursor
	}

	return &apiv2.ListSessionKeysResponse{
		Data:     data,
		Metadata: &apiv2.ResponseMetadata{FetchedAt: timestamppb.Now()},
		Page:     sessionsPage(limit, result.HasMore, nextCursor),
	}, nil
}

// ListSessions backs GET /v2/sessions/{sessionKey}.
func (s *Service) ListSessions(ctx context.Context, req *apiv2.ListSessionsRequest) (*apiv2.ListSessionsResponse, error) {
	if req.SessionKey == "" {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorMissingField, "Session key is required")
	}
	if result := s.rateLimiter.CheckRateLimit(ctx, apiv2.V2_ListSessions_FullMethodName); result.Limited {
		return nil, s.base.NewError(http.StatusTooManyRequests, apiv2base.ErrorRateLimited,
			"API rate limit exceeded. The request was rejected and no sessions were fetched.")
	}
	if s.duckDB == nil {
		return nil, s.base.NewError(http.StatusNotImplemented, apiv2base.ErrorNotImplemented, "Sessions requires dual-write (--duckdb) to be enabled")
	}

	limit, err := boundedLimit(req.Limit, defaultSessionsLimit, maxSessionsLimit)
	if err != nil {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, err.Error())
	}
	from, err := optionalTimestamp(req.From, "from")
	if err != nil {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, err.Error())
	}
	until, err := optionalTimestamp(req.Until, "until")
	if err != nil {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, err.Error())
	}

	opts := dashboards.SessionsOpts{
		Key:      req.SessionKey,
		IDSearch: req.GetSearch(),
		Cursor:   req.GetCursor(),
		Limit:    limit,
	}
	if from != nil {
		opts.From = *from
	}
	if until != nil {
		opts.Until = *until
	}

	result, err := dashboards.ListSessions(ctx, s.duckDB, consts.DevServerAccountID, consts.DevServerEnvID, opts)
	if err != nil {
		return nil, s.base.NewError(http.StatusInternalServerError, apiv2base.ErrorInternalError, "Unable to fetch sessions")
	}

	data := make([]*apiv2.SessionGroup, len(result.Sessions))
	for i, group := range result.Sessions {
		data[i] = toAPISessionGroup(group)
	}

	var nextCursor string
	if len(result.Sessions) > 0 {
		nextCursor = result.Sessions[len(result.Sessions)-1].Cursor
	}

	return &apiv2.ListSessionsResponse{
		Data:     data,
		Metadata: &apiv2.ResponseMetadata{FetchedAt: timestamppb.Now()},
		Page:     sessionsPage(limit, result.HasMore, nextCursor),
	}, nil
}

// ListSessionRuns backs GET /v2/sessions/{sessionKey}/{sessionId}/runs.
func (s *Service) ListSessionRuns(ctx context.Context, req *apiv2.ListSessionRunsRequest) (*apiv2.ListSessionRunsResponse, error) {
	if req.SessionKey == "" || req.SessionId == "" {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorMissingField, "Session key and session ID are required")
	}
	if result := s.rateLimiter.CheckRateLimit(ctx, apiv2.V2_ListSessionRuns_FullMethodName); result.Limited {
		return nil, s.base.NewError(http.StatusTooManyRequests, apiv2base.ErrorRateLimited,
			"API rate limit exceeded. The request was rejected and no session runs were fetched.")
	}
	if s.duckDB == nil {
		return nil, s.base.NewError(http.StatusNotImplemented, apiv2base.ErrorNotImplemented, "Sessions requires dual-write (--duckdb) to be enabled")
	}

	limit, err := boundedLimit(req.Limit, defaultSessionRunsLimit, maxSessionRunsLimit)
	if err != nil {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, err.Error())
	}
	from, err := optionalTimestamp(req.From, "from")
	if err != nil {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, err.Error())
	}
	until, err := optionalTimestamp(req.Until, "until")
	if err != nil {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, err.Error())
	}

	opts := dashboards.SessionRunsOpts{
		Key:    req.SessionKey,
		ID:     req.SessionId,
		Cursor: req.GetCursor(),
		Limit:  limit,
	}
	if from != nil {
		opts.From = *from
	}
	if until != nil {
		opts.Until = *until
	}

	result, err := dashboards.ListSessionRuns(ctx, s.duckDB, consts.DevServerAccountID, consts.DevServerEnvID, opts)
	if err != nil {
		return nil, s.base.NewError(http.StatusInternalServerError, apiv2base.ErrorInternalError, "Unable to fetch session runs")
	}

	data := make([]*apiv2.SessionRun, len(result.Runs))
	for i, run := range result.Runs {
		data[i] = toAPISessionRun(run)
	}

	var nextCursor string
	if len(result.Runs) > 0 {
		nextCursor = result.Runs[len(result.Runs)-1].Cursor
	}

	return &apiv2.ListSessionRunsResponse{
		Data:     data,
		Metadata: &apiv2.ResponseMetadata{FetchedAt: timestamppb.Now()},
		Page:     sessionsPage(limit, result.HasMore, nextCursor),
	}, nil
}

// boundedLimit mirrors parseRunsPageOpts' limit handling (endpoints_runs.go):
// 0/unset falls back to defaultLimit, anything above maxLimit errors rather
// than silently clamping.
func boundedLimit(requested *int32, defaultLimit, maxLimit int) (int, error) {
	limit := defaultLimit
	if requested != nil {
		limit = int(*requested)
	}
	if limit < 1 {
		return 0, fmt.Errorf("Limit must be at least 1")
	}
	if limit > maxLimit {
		return 0, fmt.Errorf("Limit cannot exceed %d", maxLimit)
	}
	return limit, nil
}

func sessionsPage(limit int, hasMore bool, nextCursor string) *apiv2.Page {
	page := &apiv2.Page{
		HasMore: hasMore,
		Limit:   int32(limit),
	}
	if hasMore && nextCursor != "" {
		page.Cursor = &nextCursor
	}
	return page
}

func toAPIFunctionRef(fn dashboards.FunctionRef) *apiv2.FunctionRef {
	return &apiv2.FunctionRef{
		Id:  fn.Slug,
		App: &apiv2.AppRef{Id: fn.AppID.String()},
	}
}

func toAPISessionGroup(group dashboards.SessionGroup) *apiv2.SessionGroup {
	functions := make([]*apiv2.FunctionRef, len(group.Functions))
	for i, fn := range group.Functions {
		functions[i] = toAPIFunctionRef(fn)
	}
	return &apiv2.SessionGroup{
		Id:             group.ID,
		RunCount:       int32(group.RunCount),
		FailedRunCount: int32(group.FailedRunCount),
		FailureRate:    group.FailureRate(),
		LastActiveAt:   timestamppb.New(group.LastActiveAt),
		Functions:      functions,
	}
}

func toAPISessionRun(run dashboards.SessionRun) *apiv2.SessionRun {
	result := &apiv2.SessionRun{
		Id:       run.RunID,
		Function: toAPIFunctionRef(run.Function),
		Status:   toFunctionRunStatus(run.Status),
		QueuedAt: timestamppb.New(run.QueuedAt),
	}
	if run.EventName != "" {
		eventName := run.EventName
		result.EventName = &eventName
	}
	if !run.StartedAt.IsZero() {
		result.StartedAt = timestamppb.New(run.StartedAt)
	}
	if !run.EndedAt.IsZero() {
		result.EndedAt = timestamppb.New(run.EndedAt)
	}
	return result
}
