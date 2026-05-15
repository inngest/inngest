package apiv1auth

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/oklog/ulid/v2"
)

const (
	RunClaimsIssuer  = "api.inngest.com"
	RunClaimsExpiry  = time.Hour
	RunClaimsSubject = "run-claim"
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

// CreateRunJWT creates a JWT for viewing the output of a specific run.
func CreateRunJWT(secret []byte, envID uuid.UUID, runID ulid.ULID) (string, error) {
	now := time.Now()
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, RunClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: RunClaimsIssuer,
			// using a specific subject allows us to check whether this is a run claim JWT immediately.
			Subject:   RunClaimsSubject,
			ExpiresAt: jwt.NewNumericDate(now.Add(RunClaimsExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
		Env:   envID,
		RunID: runID,
	})
	signed, err := t.SignedString(secret)
	if err != nil {
		return "", fmt.Errorf("could not sign session token: %w", err)
	}
	return signed, nil
}

// VerifyRunJWT verifies a run claim JWT.
func VerifyRunJWT(ctx context.Context, secret []byte, token string) (*RunClaims, error) {
	claims := &RunClaims{}
	_, err := jwt.ParseWithClaims(
		token,
		claims,
		func(token *jwt.Token) (any, error) {
			// Here, we could check if the token contains the signing key and
			// verify whether the signing key is valid in order to auth the token,
			// letting people use signing keys instead of pre-signed tokens.
			//
			// This is not currently permitted, and we currently always require
			// pre-signed tokens for realtime subscriptions.
			return secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}),
		jwt.WithStrictDecoding(),
		jwt.WithIssuedAt(),
		jwt.WithIssuer(RunClaimsIssuer),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, fmt.Errorf("invalid realtime token: %w", err)
	}
	if claims.Subject != RunClaimsSubject {
		return nil, nil
	}
	return claims, nil
}

// RunClaims represents claims to view a specific run, given an environment hash.
// Note that this is embedded within a JWT.
type RunClaims struct {
	jwt.RegisteredClaims

	// Env is the environment UUID that this run belongs to.
	Env uuid.UUID
	// RunID is the run ID that the claims grant access to
	RunID ulid.ULID
}
