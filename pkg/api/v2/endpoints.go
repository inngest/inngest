package apiv2

import (
	"context"
	"net/http"
	"time"

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
	return nil, NewErrors(http.StatusNotImplemented,
		ErrorItem{Code: ErrorNotImplemented, Message: "Accounts not implemented in OSS"},
		ErrorItem{Code: ErrorNotImplemented, Message: "Partners not implemented in OSS"},
	)
}

func (s *Service) CreateEnv(ctx context.Context, req *apiv2.CreateEnvRequest) (*apiv2.CreateEnvResponse, error) {
	// Validate required fields
	if req.Name == "" {
		return nil, NewError(http.StatusBadRequest, ErrorMissingField, "Environment name is required")
	}

	// For now, return not implemented since this is OSS
	return nil, NewError(http.StatusNotImplemented, ErrorNotImplemented, "Environments not implemented in OSS")
}

func (s *Service) FetchPartnerAccounts(ctx context.Context, req *apiv2.FetchAccountsRequest) (*apiv2.FetchAccountsResponse, error) {
	return nil, NewError(http.StatusNotImplemented, ErrorNotImplemented, "Accounts not implemented in OSS")
}

func (s *Service) FetchAccount(ctx context.Context, req *apiv2.FetchAccountRequest) (*apiv2.FetchAccountResponse, error) {
	// First commit date: 2021-05-13 09:30:04 -0700
	firstCommitTime, err := time.Parse("2006-01-02 15:04:05", "2021-05-13 09:30:04")
	if err != nil {
		return nil, err // NewError something something
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
	// envName := GetInngestEnvHeader(ctx)

	// Validate pagination parameters
	if req.Limit != nil {
		if *req.Limit < 1 {
			return nil, NewError(http.StatusBadRequest, ErrorInvalidFieldFormat, "Limit must be at least 1")
		}
		if *req.Limit > 100 {
			return nil, NewError(http.StatusBadRequest, ErrorInvalidFieldFormat, "Limit cannot exceed 100")
		}
	}

	// For now, return not implemented since this is OSS
	return nil, NewError(http.StatusNotImplemented, ErrorNotImplemented, "event keys not implemented")

}

func (s *Service) FetchAccountEnvs(ctx context.Context, req *apiv2.FetchAccountEnvsRequest) (*apiv2.FetchAccountEnvsResponse, error) {
	// Validate pagination parameters
	if req.Limit != nil {
		if *req.Limit < 1 {
			return nil, NewError(http.StatusBadRequest, ErrorInvalidFieldFormat, "Limit must be at least 1")
		}
		if *req.Limit > 250 {
			return nil, NewError(http.StatusBadRequest, ErrorInvalidFieldFormat, "Limit cannot exceed 250")
		}
	}

	// First commit date: 2021-05-13 09:30:04 -0700
	firstCommitTime, err := time.Parse("2006-01-02 15:04:05", "2021-05-13 09:30:04")
	if err != nil {
		return nil, err // NewError something something
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
	// envName := GetInngestEnvHeader(ctx)

	// Validate pagination parameters
	if req.Limit != nil {
		if *req.Limit < 1 {
			return nil, NewError(http.StatusBadRequest, ErrorInvalidFieldFormat, "Limit must be at least 1")
		}
		if *req.Limit > 100 {
			return nil, NewError(http.StatusBadRequest, ErrorInvalidFieldFormat, "Limit cannot exceed 100")
		}
	}

	// For now, return not implemented since this is OSS
	return nil, NewError(http.StatusNotImplemented, ErrorNotImplemented, "signing keys not implemented")
}