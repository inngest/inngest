package apiv2

import (
	"context"
	"net/http"
	"time"

	"github.com/inngest/inngest/pkg/api/v2/apiv2base"
	"github.com/inngest/inngest/pkg/consts"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Health implements the health check endpoint for gRPC (used by grpc-gateway)
func (s *Service) Health(ctx context.Context, req *apiv2.HealthRequest) (*apiv2.HealthResponse, error) {
	return &apiv2.HealthResponse{
		Data: &apiv2.HealthData{
			Status: "ok",
		},
		Metadata: &apiv2.ResponseMetadata{
			FetchedAt:   timestamppb.New(time.Now()),
			CachedUntil: nil, // Health responses are not cached
		},
	}, nil
}

// CreatePartnerAccount implements a protected endpoint that requires authorization
func (s *Service) CreatePartnerAccount(ctx context.Context, req *apiv2.CreateAccountRequest) (*apiv2.CreateAccountResponse, error) {
	// Return multiple errors for the not implemented functionality
	return nil, s.base.NewErrors(http.StatusNotImplemented,
		apiv2base.ErrorItem{Code: apiv2base.ErrorNotImplemented, Message: "Accounts not implemented in OSS"},
		apiv2base.ErrorItem{Code: apiv2base.ErrorNotImplemented, Message: "Partners not implemented in OSS"},
	)
}

func (s *Service) CreateEnv(ctx context.Context, req *apiv2.CreateEnvRequest) (*apiv2.CreateEnvResponse, error) {
	// Validate required fields
	if req.Name == "" {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorMissingField, "Environment name is required")
	}

	// For now, return not implemented since this is OSS
	return nil, s.base.NewError(http.StatusNotImplemented, apiv2base.ErrorNotImplemented, "Environments not implemented in OSS")
}

func (s *Service) FetchPartnerAccounts(ctx context.Context, req *apiv2.FetchAccountsRequest) (*apiv2.FetchAccountsResponse, error) {
	return nil, s.base.NewError(http.StatusNotImplemented, apiv2base.ErrorNotImplemented, "Accounts not implemented in OSS")
}

func (s *Service) FetchAccount(ctx context.Context, req *apiv2.FetchAccountRequest) (*apiv2.FetchAccountResponse, error) {
	// First commit date: 2021-05-13 09:30:04 -0700
	firstCommitTime, err := time.Parse("2006-01-02 15:04:05", "2021-05-13 09:30:04")
	if err != nil {
		return nil, err // s.base.NewError something something
	}

	// Return the default dev server account
	account := &apiv2.Account{
		Id:        consts.DevServerAccountID.String(),
		Email:     "dev@inngest.local",
		Name:      "Dev Server",
		CreatedAt: timestamppb.New(firstCommitTime),
		UpdatedAt: timestamppb.New(firstCommitTime),
	}

	return &apiv2.FetchAccountResponse{
		Data: account,
		Metadata: &apiv2.ResponseMetadata{
			FetchedAt:   timestamppb.New(time.Now()),
			CachedUntil: nil,
		},
	}, nil
}

func (s *Service) FetchAccountEventKeys(ctx context.Context, req *apiv2.FetchAccountEventKeysRequest) (*apiv2.FetchAccountEventKeysResponse, error) {
	// Extract environment from X-Inngest-Env header
	envName := s.base.GetInngestEnvHeader(ctx)

	// Validate pagination parameters
	if req.Limit != nil {
		if *req.Limit < 1 {
			return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, "Limit must be at least 1")
		}
		if *req.Limit > 100 {
			return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, "Limit cannot exceed 100")
		}
	}

	// If no event keys provider is configured, return empty list
	// This happens in dev mode where event keys aren't required
	if s.eventKeys == nil {
		return &apiv2.FetchAccountEventKeysResponse{
			Data: []*apiv2.EventKey{},
			Metadata: &apiv2.ResponseMetadata{
				FetchedAt:   timestamppb.New(time.Now()),
				CachedUntil: nil,
			},
			Page: &apiv2.Page{
				HasMore: false,
			},
		}, nil
	}

	// Get event keys from the provider
	keys, err := s.eventKeys.GetEventKeys(ctx)
	if err != nil {
		return nil, s.base.NewError(http.StatusInternalServerError, apiv2base.ErrorInternalError, "Failed to fetch event keys")
	}

	// Filter by environment if specified
	var filteredKeys []*apiv2.EventKey
	for _, key := range keys {
		if envName == "" || key.Environment == envName {
			filteredKeys = append(filteredKeys, key)
		}
	}

	// For now, return all keys without pagination
	// In a real implementation, you'd handle cursor-based pagination here

	return &apiv2.FetchAccountEventKeysResponse{
		Data: filteredKeys,
		Metadata: &apiv2.ResponseMetadata{
			FetchedAt:   timestamppb.New(time.Now()),
			CachedUntil: nil,
		},
		Page: &apiv2.Page{
			HasMore: false,
		},
	}, nil
}

func (s *Service) FetchAccountEnvs(ctx context.Context, req *apiv2.FetchAccountEnvsRequest) (*apiv2.FetchAccountEnvsResponse, error) {
	// Validate pagination parameters
	if req.Limit != nil {
		if *req.Limit < 1 {
			return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, "Limit must be at least 1")
		}
		if *req.Limit > 250 {
			return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, "Limit cannot exceed 250")
		}
	}

	// First commit date: 2021-05-13 09:30:04 -0700
	firstCommitTime, err := time.Parse("2006-01-02 15:04:05", "2021-05-13 09:30:04")
	if err != nil {
		return nil, err // s.base.NewError something something
	}

	// Return the default dev server environment
	defaultEnv := &apiv2.Env{
		Id:        consts.DevServerEnvID.String(),
		Name:      "dev",
		Type:      apiv2.EnvType_TEST,
		CreatedAt: timestamppb.New(firstCommitTime),
	}

	return &apiv2.FetchAccountEnvsResponse{
		Data: []*apiv2.Env{defaultEnv},
		Metadata: &apiv2.ResponseMetadata{
			FetchedAt:   timestamppb.New(time.Now()),
			CachedUntil: nil,
		},
		Page: &apiv2.Page{
			HasMore: false,
		},
	}, nil
}

func (s *Service) FetchAccountSigningKeys(ctx context.Context, req *apiv2.FetchAccountSigningKeysRequest) (*apiv2.FetchAccountSigningKeysResponse, error) {
	// Extract environment from X-Inngest-Env header
	envName := s.base.GetInngestEnvHeader(ctx)

	// Validate pagination parameters
	if req.Limit != nil {
		if *req.Limit < 1 {
			return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, "Limit must be at least 1")
		}
		if *req.Limit > 100 {
			return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, "Limit cannot exceed 100")
		}
	}

	// If no signing keys provider is configured, return empty list
	// This happens in dev mode where signing keys aren't required
	if s.signingKeys == nil {
		return &apiv2.FetchAccountSigningKeysResponse{
			Data: []*apiv2.SigningKey{},
			Metadata: &apiv2.ResponseMetadata{
				FetchedAt:   timestamppb.New(time.Now()),
				CachedUntil: nil,
			},
			Page: &apiv2.Page{
				HasMore: false,
			},
		}, nil
	}

	// Get signing keys from the provider
	keys, err := s.signingKeys.GetSigningKeys(ctx)
	if err != nil {
		return nil, s.base.NewError(http.StatusInternalServerError, apiv2base.ErrorInternalError, "Failed to fetch signing keys")
	}

	// Filter by environment if specified
	var filteredKeys []*apiv2.SigningKey
	for _, key := range keys {
		if envName == "" || key.Environment == envName {
			filteredKeys = append(filteredKeys, key)
		}
	}

	// For now, return all keys without pagination
	// In a real implementation, you'd handle cursor-based pagination here

	return &apiv2.FetchAccountSigningKeysResponse{
		Data: filteredKeys,
		Metadata: &apiv2.ResponseMetadata{
			FetchedAt:   timestamppb.New(time.Now()),
			CachedUntil: nil,
		},
		Page: &apiv2.Page{
			HasMore: false,
		},
	}, nil
}

func (s *Service) CreateWebhook(ctx context.Context, req *apiv2.CreateWebhookRequest) (*apiv2.CreateWebhookResponse, error) {
	// Extract environment from X-Inngest-Env header
	envName := s.base.GetInngestEnvHeader(ctx)
	if envName == "" {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorMissingField, "X-Inngest-Env header is required")
	}

	// Validate required fields
	if req.Name == "" {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorMissingField, "Webhook name is required")
	}

	if req.Transform == "" {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorMissingField, "Transform function is required")
	}

	// For now, return not implemented since this is OSS
	// In a real implementation, this would:
	// 1. Generate a unique URL for the webhook
	// 2. Validate JavaScript syntax for transform and response functions
	// 3. Save to database and return the created webhook with generated URL
	return nil, s.base.NewError(http.StatusNotImplemented, apiv2base.ErrorNotImplemented, "Webhooks not implemented in OSS")
}

func (s *Service) ListWebhooks(ctx context.Context, req *apiv2.ListWebhooksRequest) (*apiv2.ListWebhooksResponse, error) {
	// Extract environment from X-Inngest-Env header
	envName := s.base.GetInngestEnvHeader(ctx)
	if envName == "" {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorMissingField, "X-Inngest-Env header is required")
	}

	// Validate pagination parameters
	if req.Limit != nil {
		if *req.Limit < 1 {
			return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, "Limit must be at least 1")
		}
		if *req.Limit > 100 {
			return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, "Limit cannot exceed 100")
		}
	}

	// For now, return not implemented since this is OSS
	// In a real implementation, this would:
	// 1. Query the database for webhooks in the specified environment
	// 2. Apply cursor-based pagination
	// 3. Return the list with proper pagination metadata
	return nil, s.base.NewError(http.StatusNotImplemented, apiv2base.ErrorNotImplemented, "Webhooks not implemented in OSS")
}

func (s *Service) PatchEnv(ctx context.Context, req *apiv2.PatchEnvRequest) (*apiv2.PatchEnvsResponse, error) {
	return nil, s.base.NewError(http.StatusNotImplemented, apiv2base.ErrorNotImplemented, "Environments not implemented in OSS")
}

func (s *Service) ListRuns(ctx context.Context, req *apiv2.ListRunsRequest) (*apiv2.ListRunsResponse, error) {
	// Check if data manager is available
	if s.data == nil {
		return nil, s.base.NewError(http.StatusNotImplemented, apiv2base.ErrorNotImplemented, "Run history not available")
	}

	// Validate pagination parameters
	limit := int32(50) // default
	if req.Limit != nil {
		if *req.Limit < 1 {
			return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, "Limit must be at least 1")
		}
		if *req.Limit > 250 {
			return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, "Limit cannot exceed 250")
		}
		limit = *req.Limit
	}

	// Parse and validate time range
	var startTime, endTime time.Time
	var err error

	if req.StartTime != nil && *req.StartTime != "" {
		startTime, err = time.Parse(time.RFC3339, *req.StartTime)
		if err != nil {
			return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, "Invalid startTime format, must be RFC3339")
		}
	} else {
		// Default to 24 hours ago if not specified
		startTime = time.Now().Add(-24 * time.Hour)
	}

	if req.EndTime != nil && *req.EndTime != "" {
		endTime, err = time.Parse(time.RFC3339, *req.EndTime)
		if err != nil {
			return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, "Invalid endTime format, must be RFC3339")
		}
	} else {
		endTime = time.Now()
	}

	orderBy := s.mapListRunsOrderBy(req)
	filter, err := s.mapListRunsFilter(req, startTime, endTime)
	if err != nil {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, err.Error())
	}

	// TODO: Load 1 extra item and check if there are more items to load
	opts := s.toRunsQueryOpt(int(limit), req.Cursor, orderBy, filter)
	traceRuns, err := s.data.GetTraceRuns(ctx, opts)
	if err != nil {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, "Invalid endTime format, must be RFC3339")
	}

	runs := make([]*apiv2.Run, 0, len(traceRuns))
	for _, traceRun := range traceRuns {
		// TODO: This could be optimized to reduce queries
		app, err := s.data.GetAppByID(ctx, traceRun.AppID)
		if err != nil {
			return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, "Failed to load app")
		}
		fn, err := s.data.GetFunctionByInternalUUID(ctx, traceRun.FunctionID)
		if err != nil {
			return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, "Failed to load function")
		}
		run := &apiv2.Run{
			Id:         traceRun.RunID,
			AppId:      app.Name,
			FunctionId: s.cleanFunctionId(fn.Slug, app.Name),
			Status:     apiv2.RunStatus(traceRun.Status),
			QueuedAt:   timestamppb.New(traceRun.QueuedAt),
			StartedAt:  timestamppb.New(traceRun.StartedAt),
			EndedAt:    timestamppb.New(traceRun.EndedAt),
		}
		runs = append(runs, run)
	}

	var cursor string
	if len(traceRuns) > 0 {
		cursor = traceRuns[len(traceRuns)-1].Cursor
	}

	return &apiv2.ListRunsResponse{
		Data: runs,
		Metadata: &apiv2.ResponseMetadata{
			FetchedAt:   timestamppb.New(time.Now()),
			CachedUntil: nil,
		},
		Page: &apiv2.Page{
			Cursor:  &cursor,
			HasMore: len(runs) == int(limit),
			Limit:   limit,
		},
	}, nil
}
