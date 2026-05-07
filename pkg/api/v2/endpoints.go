package apiv2

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/inngest/inngest/pkg/api/v2/apiv2base"
	"github.com/inngest/inngest/pkg/appsync"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/publicerr"
	"github.com/inngest/inngest/pkg/syscode"
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

	// XXX: In the future we can/should add pagination.

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

const syncStatusSuccess = "SUCCESS"

// SyncApp performs an app sync against the SDK at the provided URL.
func (s *Service) SyncApp(
	ctx context.Context,
	req *apiv2.SyncAppRequest,
) (*apiv2.SyncAppResponse, error) {
	ctx = logger.WithStdlib(ctx, logger.From(ctx).With("handler", "SyncApp"))

	if req.Url == "" {
		return nil, s.base.NewError(
			http.StatusBadRequest,
			apiv2base.ErrorMissingField,
			"url is required",
		)
	}
	if req.AppId == "" {
		return nil, s.base.NewError(
			http.StatusBadRequest,
			apiv2base.ErrorMissingField,
			"app_id is required",
		)
	}

	// Sync drives an outbound HTTP request to a caller-supplied URL; rate-limit
	// before any work to bound SSRF / DoS amplification per signing key.
	if result := s.rateLimiter.CheckRateLimit(ctx, apiv2.V2_SyncApp_FullMethodName); result.Limited {
		return nil, s.base.NewError(
			http.StatusTooManyRequests,
			apiv2base.ErrorRateLimited,
			"API rate limit exceeded. The request was rejected and no app was synced.",
		)
	}

	if s.appSyncer == nil {
		return nil, s.base.NewError(
			http.StatusNotImplemented,
			apiv2base.ErrorNotImplemented,
			"App sync not implemented",
		)
	}

	signingKey, err := s.firstSigningKey(ctx)
	if err != nil {
		return nil, err
	}

	syncResp, syscodeErr, err := appsync.Sync(ctx, appsync.Opts{
		AllowInsecureHTTP: s.appSyncAllowInsecure,
		ExpectedAppID:     req.AppId,
		ServerKind:        s.serverKind,
		SigningKey:        signingKey,
		URL:               req.Url,
	})
	if err != nil {
		logger.From(ctx).Error(
			"system error during sync",
			"error", err,
			"url", req.Url,
		)
		return nil, s.base.NewError(
			http.StatusInternalServerError,
			apiv2base.ErrorInternalError,
			"internal error during sync",
		)
	}
	if syscodeErr != nil {
		logger.From(ctx).Warn(
			"sync failed",
			"code", syscodeErr.Code,
			"message", syscodeErr.Message,
			"url", req.Url,
		)
		return nil, s.base.NewError(
			httpStatusForSyncSyscode(syscodeErr.Code),
			syscodeErr.Code,
			syscodeErr.Message,
		)
	}

	reply, err := s.appSyncer.ProcessSync(ctx, *syncResp.ToRegisterRequest())
	if err != nil {
		logger.From(ctx).Error("error processing sync", "error", err)
		return nil, s.processSyncErr(err)
	}

	syncID := ""
	if reply != nil && reply.SyncID != nil {
		syncID = reply.SyncID.String()
	}

	return &apiv2.SyncAppResponse{
		Data: &apiv2.SyncAppData{
			AppId:  req.AppId,
			Id:     syncID,
			Status: syncStatusSuccess,
		},
		Metadata: &apiv2.ResponseMetadata{
			FetchedAt: timestamppb.Now(),
		},
	}, nil
}

// firstSigningKey returns the configured key, or a generic 501 if anything
// is misconfigured. All failure modes log a reason but never surface it, so
// the response can't be used to probe key configuration.
func (s *Service) firstSigningKey(ctx context.Context) (string, error) {
	notImpl := s.base.NewError(
		http.StatusNotImplemented,
		apiv2base.ErrorNotImplemented,
		"App sync not implemented",
	)

	if s.signingKeys == nil {
		logger.From(ctx).Error("no signing keys provider")
		return "", notImpl
	}
	keys, err := s.signingKeys.GetSigningKeys(ctx)
	if err != nil {
		logger.From(ctx).Error("failed to load signing keys", "error", err)
		return "", notImpl
	}
	if len(keys) == 0 || keys[0].Key == "" {
		logger.From(ctx).Error("no signing key")
		return "", notImpl
	}
	if len(keys) > 1 {
		logger.From(ctx).Warn("multiple signing keys")
	}
	return keys[0].Key, nil
}

// processSyncErr maps an AppSyncer error to a v2 response. Known types
// (publicerr.Error, *syscode.Error) propagate status/code/message; others
// collapse to 500 with no internal detail. Caller logs the raw err.
func (s *Service) processSyncErr(err error) error {
	var perr publicerr.Error
	if errors.As(err, &perr) {
		code := perr.Code
		if code == "" {
			code = apiv2base.ErrorAppSyncFailed
		}
		msg := perr.Message
		if msg == "" {
			msg = http.StatusText(perr.Status)
		}
		status := perr.Status
		if status == 0 {
			status = http.StatusInternalServerError
		}
		return s.base.NewError(status, code, msg)
	}

	var syserr *syscode.Error
	if errors.As(err, &syserr) {
		return s.base.NewError(
			http.StatusUnprocessableEntity,
			syserr.Code,
			syserr.Message,
		)
	}

	return s.base.NewError(
		http.StatusInternalServerError,
		apiv2base.ErrorInternalError,
		"failed to process sync",
	)
}

// httpStatusForSyncSyscode maps an appsync.Sync syscode to an HTTP status.
// Caller-input violations are 400; everything else is a 422 (the SDK side of
// the protocol exchange failed).
func httpStatusForSyncSyscode(code string) int {
	switch code {
	case syscode.CodeURLSchemeDenied:
		return http.StatusBadRequest
	default:
		return http.StatusUnprocessableEntity
	}
}
