package auth

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/oklog/ulid/v2"
)

const wellKnownClaimIssuer = "connect.inngest.com"
const wellKnownClaimAudience = "gateway.connect.inngest.com"

const DefaultExpiry = time.Minute * 5

type claims struct {
	jwt.RegisteredClaims
	Env          uuid.UUID    `json:"env"`
	Entitlements Entitlements `json:"entitlements"`
}

func VerifySessionToken(jwtSecret []byte, tokenString string) (*Response, error) {
	customClaims := &claims{}
	parsedToken, err := jwt.ParseWithClaims(
		tokenString,
		customClaims,
		func(token *jwt.Token) (interface{}, error) { return jwtSecret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}),
		jwt.WithStrictDecoding(),
		jwt.WithIssuedAt(),
		jwt.WithIssuer(wellKnownClaimIssuer),
		jwt.WithAudience(wellKnownClaimAudience),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	if !parsedToken.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	sub, err := customClaims.GetSubject()
	if err != nil {
		return nil, fmt.Errorf("could not get subject from session token")
	}

	accountId, err := uuid.Parse(sub)
	if err != nil {
		return nil, fmt.Errorf("could not parse account ID from session token")
	}

	if customClaims.Env == uuid.Nil {
		return nil, fmt.Errorf("could not parse env ID from session token")
	}

	return &Response{
		AccountID:    accountId,
		EnvID:        customClaims.Env,
		Entitlements: customClaims.Entitlements,
	}, nil
}

type jwtSessionTokenSigner struct {
	jwtSecret []byte
}

func (j jwtSessionTokenSigner) SignSessionToken(accountId uuid.UUID, envId uuid.UUID, expireAfter time.Duration, entitlements Entitlements) (string, error) {
	return signSessionToken(j.jwtSecret, accountId, envId, expireAfter, entitlements)
}

func NewJWTSessionTokenSigner(jwtSecret []byte) SessionTokenSigner {
	return &jwtSessionTokenSigner{jwtSecret: jwtSecret}
}

func signSessionToken(jwtSecret []byte, accountId uuid.UUID, envId uuid.UUID, expireAfter time.Duration, entitlements Entitlements) (string, error) {
	now := time.Now()

	id, err := ulid.New(ulid.Now(), rand.Reader)
	if err != nil {
		return "", fmt.Errorf("could not generate session token ID: %w", err)
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    wellKnownClaimIssuer,
			Subject:   accountId.String(),
			Audience:  []string{wellKnownClaimAudience},
			ExpiresAt: jwt.NewNumericDate(now.Add(expireAfter)),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        id.String(),
		},
		Env:          envId,
		Entitlements: entitlements,
	})
	signed, err := t.SignedString(jwtSecret)
	if err != nil {
		return "", fmt.Errorf("could not sign session token: %w", err)
	}

	return signed, nil
}

func NewJWTAuthHandler(jwtSecret []byte) Handler {
	return func(ctx context.Context, data *connect.WorkerConnectRequestData) (*Response, error) {
		token := data.AuthData.GetSessionToken()
		if token == "" {
			return nil, nil
		}

		verified, err := VerifySessionToken(jwtSecret, token)
		if err != nil {
			return nil, nil
		}

		return &Response{
			AccountID:    verified.AccountID,
			EnvID:        verified.EnvID,
			Entitlements: verified.Entitlements,
		}, nil
	}
}
