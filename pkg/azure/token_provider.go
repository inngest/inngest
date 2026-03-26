package azure

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/jackc/pgx/v5"
	"github.com/redis/rueidis"
)

const (
	// AzurePostgresScope is the OAuth2 scope for Azure Database for PostgreSQL.
	AzurePostgresScope = "https://ossrdbms-aad.database.windows.net/.default"

	// AzureRedisScope is the OAuth2 scope for Azure Cache for Redis.
	AzureRedisScope = "https://redis.azure.com/.default"
)

// IsAzureAuthEnabled returns true if Azure Workload Identity authentication
// should be used. This is determined by the presence of the AZURE_POSTGRESQL_HOST
// environment variable.
func IsAzureAuthEnabled() bool {
	return os.Getenv("AZURE_POSTGRESQL_HOST") != ""
}

// AzurePostgresConfig holds the connection parameters for Azure PostgreSQL
// when using Workload Identity authentication.
type AzurePostgresConfig struct {
	Host     string
	Port     uint16
	Database string
	User     string
	Schema   string
}

// LoadAzurePostgresConfig reads Azure PostgreSQL connection parameters from
// environment variables and validates that all required values are present.
func LoadAzurePostgresConfig() (AzurePostgresConfig, error) {
	port := uint16(5432)
	if p := os.Getenv("AZURE_POSTGRESQL_PORT"); p != "" {
		v, err := strconv.ParseUint(p, 10, 16)
		if err != nil {
			return AzurePostgresConfig{}, fmt.Errorf("invalid AZURE_POSTGRESQL_PORT %q: %w", p, err)
		}
		port = uint16(v)
		if port == 0 {
			return AzurePostgresConfig{}, fmt.Errorf("invalid AZURE_POSTGRESQL_PORT: port 0 is not allowed")
		}
	}

	cfg := AzurePostgresConfig{
		Host:     os.Getenv("AZURE_POSTGRESQL_HOST"),
		Port:     port,
		Database: os.Getenv("AZURE_POSTGRESQL_DATABASE"),
		User:     os.Getenv("AZURE_POSTGRESQL_USER"),
		Schema:   os.Getenv("AZURE_POSTGRESQL_SCHEMA"),
	}

	if cfg.Host == "" || cfg.Database == "" || cfg.User == "" {
		return AzurePostgresConfig{}, fmt.Errorf(
			"azure workload identity auth enabled but missing required env vars: AZURE_POSTGRESQL_HOST=%q, AZURE_POSTGRESQL_DATABASE=%q, AZURE_POSTGRESQL_USER=%q",
			cfg.Host, cfg.Database, cfg.User,
		)
	}

	return cfg, nil
}

// NewBeforeConnectHook creates a pgx BeforeConnect callback that acquires an
// Azure AD token and sets it as the connection password. The Azure SDK handles
// token caching internally, so each call to GetToken is efficient when the
// cached token is still valid.
func NewBeforeConnectHook() (func(ctx context.Context, cc *pgx.ConnConfig) error, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure credential: %w", err)
	}

	return func(ctx context.Context, cc *pgx.ConnConfig) error {
		token, err := cred.GetToken(ctx, policy.TokenRequestOptions{
			Scopes: []string{AzurePostgresScope},
		})
		if err != nil {
			return fmt.Errorf("failed to acquire Azure AD token for PostgreSQL: %w", err)
		}
		cc.Password = token.Token
		return nil
	}, nil
}

// IsAzureRedisAuthEnabled returns true if Azure Workload Identity authentication
// should be used for Redis. This is determined by the presence of the AZURE_REDIS_HOST
// environment variable.
func IsAzureRedisAuthEnabled() bool {
	return os.Getenv("AZURE_REDIS_HOST") != ""
}

// AzureRedisConfig holds the connection parameters for Azure Cache for Redis
// when using Workload Identity authentication.
type AzureRedisConfig struct {
	Host string
	Port uint16
	User string
}

// LoadAzureRedisConfig reads Azure Cache for Redis connection parameters from
// environment variables and validates that all required values are present.
func LoadAzureRedisConfig() (AzureRedisConfig, error) {
	// Azure Cache for Redis defaults to port 6380 for TLS connections.
	port := uint16(6380)
	if p := os.Getenv("AZURE_REDIS_PORT"); p != "" {
		v, err := strconv.ParseUint(p, 10, 16)
		if err != nil {
			return AzureRedisConfig{}, fmt.Errorf("invalid AZURE_REDIS_PORT %q: %w", p, err)
		}
		port = uint16(v)
		if port == 0 {
			return AzureRedisConfig{}, fmt.Errorf("invalid AZURE_REDIS_PORT: port 0 is not allowed")
		}
	}

	cfg := AzureRedisConfig{
		Host: os.Getenv("AZURE_REDIS_HOST"),
		Port: port,
		User: os.Getenv("AZURE_REDIS_USER"),
	}

	if cfg.Host == "" || cfg.User == "" {
		return AzureRedisConfig{}, fmt.Errorf(
			"azure workload identity redis auth enabled but missing required env vars: AZURE_REDIS_HOST=%q, AZURE_REDIS_USER=%q",
			cfg.Host, cfg.User,
		)
	}

	return cfg, nil
}

// NewRedisClientOption builds a rueidis.ClientOption configured for Azure
// Workload Identity authentication. It uses AuthCredentialsFn to inject a
// fresh Azure AD token on every Redis connection attempt. The Azure SDK
// caches tokens internally, so this is efficient.
func NewRedisClientOption() (rueidis.ClientOption, error) {
	cfg, err := LoadAzureRedisConfig()
	if err != nil {
		return rueidis.ClientOption{}, err
	}

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return rueidis.ClientOption{}, fmt.Errorf("failed to create Azure credential for Redis: %w", err)
	}

	return rueidis.ClientOption{
		InitAddress: []string{fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)},
		TLSConfig: &tls.Config{
			ServerName: cfg.Host,
			MinVersion: tls.VersionTLS12,
		},
		AuthCredentialsFn: func(_ rueidis.AuthCredentialsContext) (rueidis.AuthCredentials, error) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			token, err := cred.GetToken(ctx, policy.TokenRequestOptions{
				Scopes: []string{AzureRedisScope},
			})
			if err != nil {
				return rueidis.AuthCredentials{}, fmt.Errorf("failed to acquire Azure AD token for Redis: %w", err)
			}
			return rueidis.AuthCredentials{
				Username: cfg.User,
				Password: token.Token,
			}, nil
		},
	}, nil
}
