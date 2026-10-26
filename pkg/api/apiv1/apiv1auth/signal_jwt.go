package apiv1auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	SignalClaimsIssuer  = "api.inngest.com"
	SignalClaimsSubject = "signal-claim"
)

// ErrNotSignalClaims is returned when a valid JWT is not a signal token, for
// example a run token signed with the same secret.
var ErrNotSignalClaims = errors.New("token is not a signal token")

// SignalClaims allows sending exactly one signal into one environment. It does
// not grant access to events, runs, or any other signal.
type SignalClaims struct {
	jwt.RegisteredClaims

	// Env is the environment UUID the signal is sent into.
	Env uuid.UUID
	// SignalID is the only signal the token may send.
	SignalID string
}

// CreateSignalJWT creates a token that can send signalID into envID until
// expiresAt.
func CreateSignalJWT(secret []byte, envID uuid.UUID, signalID string, expiresAt time.Time) (string, error) {
	if signalID == "" {
		return "", errors.New("signal ID is empty")
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, SignalClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    SignalClaimsIssuer,
			Subject:   SignalClaimsSubject,
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		Env:      envID,
		SignalID: signalID,
	})
	signed, err := t.SignedString(secret)
	if err != nil {
		return "", fmt.Errorf("could not sign signal token: %w", err)
	}
	return signed, nil
}

// VerifySignalJWT verifies a signal token and returns its claims.
func VerifySignalJWT(secret []byte, token string) (*SignalClaims, error) {
	claims := &SignalClaims{}
	_, err := jwt.ParseWithClaims(
		token,
		claims,
		func(*jwt.Token) (any, error) { return secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}),
		jwt.WithStrictDecoding(),
		jwt.WithIssuedAt(),
		jwt.WithIssuer(SignalClaimsIssuer),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, fmt.Errorf("invalid signal token: %w", err)
	}
	if claims.Subject != SignalClaimsSubject || claims.SignalID == "" || claims.Env == uuid.Nil {
		return nil, ErrNotSignalClaims
	}
	return claims, nil
}
