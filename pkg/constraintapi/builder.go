package constraintapi

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
)

type leaseRequestBuilder struct {
	req *CapacityLeaseRequest
}

func (b *leaseRequestBuilder) IdempotencyKey(idempotencyKey string) *leaseRequestBuilder {
	b.req.IdempotencyKey = idempotencyKey
	return b
}

func (b *leaseRequestBuilder) AccountID(accountID uuid.UUID) *leaseRequestBuilder {
	b.req.AccountID = accountID
	return b
}

func (b *leaseRequestBuilder) EnvID(envID uuid.UUID) *leaseRequestBuilder {
	b.req.EnvID = envID
	return b
}

func (b *leaseRequestBuilder) FunctionID(functionID uuid.UUID) *leaseRequestBuilder {
	b.req.FunctionID = functionID
	return b
}

func (b *leaseRequestBuilder) Configuration(config ConstraintConfig) *leaseRequestBuilder {
	b.req.Configuration = config
	return b
}

func (b *leaseRequestBuilder) CurrentTime(currentTime time.Time) *leaseRequestBuilder {
	b.req.CurrentTime = currentTime
	return b
}

func (b *leaseRequestBuilder) Duration(duration time.Duration) *leaseRequestBuilder {
	b.req.Duration = duration
	return b
}

func (b *leaseRequestBuilder) MaximumLifetime(maximumLifetime time.Duration) *leaseRequestBuilder {
	b.req.MaximumLifetime = maximumLifetime
	return b
}

func (b *leaseRequestBuilder) BlockingThreshold(blockingThreshold time.Duration) *leaseRequestBuilder {
	b.req.BlockingThreshold = blockingThreshold
	return b
}

func (b *leaseRequestBuilder) Valid() error {
	var errs error

	if err := b.req.Valid(); err != nil {
		errs = multierror.Append(errs, err)
	}

	return errs
}

func (b *leaseRequestBuilder) Build() (*CapacityLeaseRequest, error) {
	if err := b.Valid(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	return b.req, nil
}

func NewLeaseRequest() *leaseRequestBuilder {
	b := &leaseRequestBuilder{
		req: &CapacityLeaseRequest{},
	}

	return b
}
