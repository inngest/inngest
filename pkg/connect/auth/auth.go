package auth

import (
	"context"
	"github.com/google/uuid"
	"github.com/inngest/inngest/proto/gen/connect/v1"
	"time"
)

type Response struct {
	AccountID uuid.UUID
	EnvID     uuid.UUID
}

type Handler func(context.Context, *connect.WorkerConnectRequestData) (*Response, error)

type SessionTokenSigner interface {
	SignSessionToken(accountId uuid.UUID, envId uuid.UUID, expireAfter time.Duration) (string, error)
}
