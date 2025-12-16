package authn

import (
	"errors"
	"net/http"
	"strings"
)

// AuthContext represents authentication context information
type AuthContext struct {
	// Add fields as needed for your authentication context
	isAuthenticated bool
	// Add other relevant fields
}

var (
	// ErrIncorrectStrategy represents an attempt authenticating via an incorrect strategy,
	// eg. via an API key when no API key is present in the authorization header.
	ErrIncorrectStrategy = errors.New("incorrect auth strategy")

	// ErrNoAuthentication is returned when no authorization strategy was successful and the
	// user is not authenticated.
	ErrNoAuthentication = errors.New("unauthorized: no auth context found")
)

// TokenFromHeader tries to retrieve the token string from the
// "Authorization" reqeust header: "Authorization: BEARER T".
func TokenFromHeader(r *http.Request) string {
	// Get token from authorization header.
	bearer := r.Header.Get("Authorization")
	if len(bearer) > 7 && strings.ToUpper(bearer[0:6]) == "BEARER" {
		return bearer[7:]
	}
	return ""
}
