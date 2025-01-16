package realtime

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
)

const (
	Issuer        = "rt.inngest.com"
	DefaultExpiry = time.Minute
)

type JWTClaims struct {
	jwt.RegisteredClaims
	Env    uuid.UUID `json:"env"`
	Topics []Topic   `json:"topics"`
}

func (j JWTClaims) AccountID() uuid.UUID {
	return uuid.MustParse(j.RegisteredClaims.Subject)
}

func (j JWTClaims) WorkspaceID() uuid.UUID {
	return j.Env
}

func ValidateJWT(ctx context.Context, secret []byte, token string) (*JWTClaims, error) {
	claims := &JWTClaims{}
	_, err := jwt.ParseWithClaims(
		token,
		claims,
		func(token *jwt.Token) (interface{}, error) {
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
		jwt.WithIssuer(Issuer),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, fmt.Errorf("invalid realtime token: %w", err)
	}
	return claims, nil
}

// TopicsFromJWT takes a JWT string, a secret, and returns any topics that the
// JWT allows access to.
//
// JWTs are used to register interest and subscribe to topics.
func TopicsFromJWT(ctx context.Context, secret []byte, token string) ([]Topic, error) {
	claims, err := ValidateJWT(ctx, secret, token)
	if err != nil {
		return nil, err
	}
	return claims.Topics, nil
}

// NewJWT returns a new JWT used to subscribe to topics as an unauthenticated user.
//
// JWTs are made using a pre-shared key, and can then be passed to the frontend to
// subscribe to the JWT's encoded topics.
func NewJWT(ctx context.Context, secret []byte, accountID, envID uuid.UUID, topics []Topic) (string, error) {
	now := time.Now()

	id, err := ulid.New(ulid.Now(), rand.Reader)
	if err != nil {
		return "", fmt.Errorf("could not generate session token ID: %w", err)
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    Issuer,
			Subject:   accountID.String(),
			ExpiresAt: jwt.NewNumericDate(now.Add(DefaultExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        id.String(),
		},
		Env:    envID,
		Topics: topics,
	})
	signed, err := t.SignedString(secret)
	if err != nil {
		return "", fmt.Errorf("could not sign session token: %w", err)
	}
	return signed, nil
}
