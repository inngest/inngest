package apiv1auth

import (
	"context"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
)

// AuthFinder returns auth information from the current context.
type AuthFinder func(ctx context.Context) (V1Auth, error)

// V1Auth represents an object that returns the account and worskpace currently authed.
type V1Auth interface {
	AccountID() uuid.UUID
	WorkspaceID() uuid.UUID
}

// NilAuthFinder is used in the dev server, returning zero auth.
func NilAuthFinder(ctx context.Context) (V1Auth, error) {
	return nilAuth{}, nil
}

type nilAuth struct{}

func (nilAuth) AccountID() uuid.UUID {
	return consts.DevServerAccountID
}

func (nilAuth) WorkspaceID() uuid.UUID {
	return consts.DevServerEnvID
}
