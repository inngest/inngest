package cqrs

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Environment struct {
	ID        uuid.UUID `json:"internal_id"`
	Slug      string    `json:"slug"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	AccountID uuid.UUID `json:"account_id"`
}

// EnvironmentReader provides the function to retrieve environments
type EnvironmentReader interface {
	// GetEnvironmentBySlug returns the environment with the provided slug
	GetEnvironmentBySlug(ctx context.Context, accountID uuid.UUID, slug string) (*Environment, error)
	// GetEnvironmentByUUID returns the environment with the provided ID
	GetEnvironmentByUUID(ctx context.Context, id uuid.UUID) (*Environment, error)
}
