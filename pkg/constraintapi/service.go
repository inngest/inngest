package constraintapi

import (
	"context"

	"github.com/inngest/inngest/pkg/service"
)

type constraintApiSvc struct{}

func (c *constraintApiSvc) Name() string {
	return "constraintapi"
}

func (c *constraintApiSvc) Pre(ctx context.Context) error {
	return nil
}

func (c *constraintApiSvc) Run(ctx context.Context) error {
	return nil
}

func (c *constraintApiSvc) Stop(ctx context.Context) error {
	return nil
}

func NewConstraintAPI() service.Service {
	return &constraintApiSvc{}
}
