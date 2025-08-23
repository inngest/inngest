package apiv2

import (
	"context"

	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
)

// ValidatingService wraps the actual service and adds validation to all methods
type ValidatingService struct {
	*Service
}

// NewValidatingService creates a service that validates all incoming requests
func NewValidatingService() *ValidatingService {
	return &ValidatingService{
		Service: NewService(),
	}
}

// Health wraps the health check with validation (though HealthRequest has no validation rules)
func (v *ValidatingService) Health(ctx context.Context, req *apiv2.HealthRequest) (*apiv2.HealthResponse, error) {
	if err := validateRequest(req); err != nil {
		return nil, err
	}
	return v.Service.Health(ctx, req)
}

// CreateAccount wraps the create account method with validation
func (v *ValidatingService) CreateAccount(ctx context.Context, req *apiv2.CreateAccountRequest) (*apiv2.CreateAccountResponse, error) {
	if err := validateRequest(req); err != nil {
		return nil, err
	}
	return v.Service.CreateAccount(ctx, req)
}

// FetchAccounts wraps the fetch accounts method with validation
func (v *ValidatingService) FetchAccounts(ctx context.Context, req *apiv2.FetchAccountsRequest) (*apiv2.FetchAccountsResponse, error) {
	if err := validateRequest(req); err != nil {
		return nil, err
	}
	return v.Service.FetchAccounts(ctx, req)
}

// Ensure ValidatingService implements the V2Server interface
var _ apiv2.V2Server = (*ValidatingService)(nil)