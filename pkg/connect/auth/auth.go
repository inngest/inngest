package auth

import (
	"context"
	"github.com/google/uuid"
	"github.com/inngest/inngest/proto/gen/connect/v1"
	"time"
)

type Entitlements struct {
	ConnectionAllowed bool `json:"-"`
	AppsPerConnection int  `json:"appsPerConnection"`
}

type Response struct {
	AccountID uuid.UUID
	EnvID     uuid.UUID

	// Entitlements are included in short-lived token
	Entitlements Entitlements
}

type Handler func(context.Context, *connect.WorkerConnectRequestData) (*Response, error)

type SessionTokenSigner interface {
	// SignSessionToken issues a short-lived session token including entitlements
	SignSessionToken(accountId uuid.UUID, envId uuid.UUID, expireAfter time.Duration, entitlements Entitlements) (string, error)
}
