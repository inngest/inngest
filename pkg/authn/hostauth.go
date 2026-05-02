package authn

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	hostAuthCookieName = "inngest_session"
	hostAuthIssuer     = "inngest-host"
	hostAuthExpiry     = 24 * time.Hour
	hostAuthSalt       = "inngest-host-auth-v1"
)

type HostAuthConfig struct {
	Email    string
	Password string
}

func NewHostAuthConfig() *HostAuthConfig {
	email := os.Getenv("INNGEST_HOST_EMAIL")
	password := os.Getenv("INNGEST_HOST_PASSWORD")
	if email == "" || password == "" {
		return nil
	}
	return &HostAuthConfig{
		Email:    email,
		Password: password,
	}
}

func (c *HostAuthConfig) IsEnabled() bool {
	return c != nil && c.Email != "" && c.Password != ""
}

func (c *HostAuthConfig) ValidateCredentials(email, password string) bool {
	if !c.IsEnabled() {
		return false
	}
	emailMatch := strings.EqualFold(email, c.Email)
	passwordMatch := subtle.ConstantTimeCompare([]byte(password), []byte(c.Password)) == 1
	return emailMatch && passwordMatch
}

func (c *HostAuthConfig) JWTSecret() []byte {
	mac := hmac.New(sha256.New, []byte(hostAuthSalt))
	mac.Write([]byte(c.Password))
	return mac.Sum(nil)
}

func (c *HostAuthConfig) CreateToken(email string) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   email,
		Issuer:    hostAuthIssuer,
		ExpiresAt: jwt.NewNumericDate(now.Add(hostAuthExpiry)),
		IssuedAt:  jwt.NewNumericDate(now),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(c.JWTSecret())
}

func (c *HostAuthConfig) ValidateToken(tokenString string) (*jwt.RegisteredClaims, error) {
	claims := &jwt.RegisteredClaims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return c.JWTSecret(), nil
	}, jwt.WithIssuer(hostAuthIssuer), jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return nil, err
	}
	return claims, nil
}
