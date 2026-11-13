package start

import (
	"context"
	"errors"
	"fmt"
	"time"

	localconfig "github.com/inngest/inngest/cmd/internal/config"
	"github.com/inngest/inngest/cmd/internal/server"
	"github.com/inngest/inngest/pkg/authn"
	"github.com/inngest/inngest/pkg/config"
	"github.com/inngest/inngest/pkg/devserver"
	"github.com/inngest/inngest/pkg/headers"
	"github.com/urfave/cli/v3"
)

func action(ctx context.Context, cmd *cli.Command) error {
	conf, err := config.Dev(ctx)
	if err != nil {
		return err
	}

	if err = localconfig.InitStartConfig(ctx, cmd); err != nil {
		return err
	}

	server.SetAddress(cmd, conf)
	conf.ServerKind = headers.ServerKindCloud

	signingKey := localconfig.GetValue(cmd, "signing-key", "")
	if err := validateSigningKey(signingKey); err != nil {
		return err
	}

	eventKeys := localconfig.GetStringSlice(cmd, "event-key")
	if len(eventKeys) == 0 {
		return errors.New("at least one event-key is required")
	}

	// Read through localconfig so env vars (INNGEST_POSTGRES_*) and config
	// files are honored, not just flags.
	postgresMaxIdleConns := localconfig.GetIntValue(cmd, "postgres-max-idle-conns", 10)
	postgresMaxOpenConns := localconfig.GetIntValue(cmd, "postgres-max-open-conns", 100)
	if err := validatePostgresPool(postgresMaxIdleConns, postgresMaxOpenConns); err != nil {
		return err
	}

	connectGRPCConfig, err := server.ConnectGRPCConfig(ctx, cmd)
	if err != nil {
		return err
	}

	tick := localconfig.GetIntValue(cmd, "tick", devserver.DefaultTick)
	opts := devserver.StartOpts{
		Config:                  *conf,
		ConnectGatewayHost:      conf.CoreAPI.Addr,
		ConnectGatewayPort:      localconfig.GetIntValue(cmd, "connect-gateway-port", devserver.DefaultConnectGatewayPort),
		ConnectGRPCConfig:       connectGRPCConfig,
		EventKeys:               eventKeys,
		NoUI:                    localconfig.GetBoolValue(cmd, "no-ui", false),
		EnableDuckDB:            localconfig.GetBoolValue(cmd, "duckdb", false),
		EnableDuckDBReads:       localconfig.GetBoolValue(cmd, "duckdb-reads", false),
		Persist:                 true,
		PollInterval:            localconfig.GetIntValue(cmd, "poll-interval", devserver.DefaultPollInterval),
		PostgresConnMaxIdleTime: localconfig.GetIntValue(cmd, "postgres-conn-max-idle-time", 5),
		PostgresConnMaxLifetime: localconfig.GetIntValue(cmd, "postgres-conn-max-lifetime", 30),
		PostgresMaxIdleConns:    postgresMaxIdleConns,
		PostgresMaxOpenConns:    postgresMaxOpenConns,
		PostgresURI:             localconfig.GetValue(cmd, "postgres-uri", ""),
		QueueWorkers:            localconfig.GetIntValue(cmd, "queue-workers", devserver.DefaultQueueWorkers),
		RedisURI:                localconfig.GetValue(cmd, "redis-uri", ""),
		RequireKeys:             true,
		RetryInterval:           localconfig.GetIntValue(cmd, "retry-interval", 0),
		SigningKey:              &signingKey,
		SQLiteDir:               localconfig.GetValue(cmd, "sqlite-dir", ""),
		Tick:                    time.Duration(tick) * time.Millisecond,
		URLs:                    localconfig.GetStringSlice(cmd, "sdk-url"),
	}

	closeTracers, err := server.SetupTracers(ctx, conf.EventAPI.Port, "", "")
	if err != nil {
		return err
	}
	defer closeTracers()

	// the first signal stops the server gracefully.  a second signal exits at
	// once.
	server.ExitAfterSignals(2)

	return devserver.New(ctx, opts)
}

// validateSigningKey checks that the signing key is set and is hex with an
// even number of characters.
func validateSigningKey(signingKey string) error {
	if signingKey == "" {
		return errors.New("signing-key is required")
	}
	// Check if the signing key has an even number of characters (required for hex decoding)
	if len(signingKey)%2 != 0 {
		return errors.New("signing-key must be hex string with even number of chars")
	}
	if _, err := authn.HashedSigningKey(signingKey); err != nil {
		return fmt.Errorf("signing-key must be valid hex string: %w", err)
	}
	return nil
}

// validatePostgresPool checks the Postgres connection pool settings.
func validatePostgresPool(maxIdleConns, maxOpenConns int) error {
	if maxOpenConns <= 1 {
		return fmt.Errorf("postgres-max-open-conns (%d) must be greater than 1", maxOpenConns)
	}
	if maxIdleConns > maxOpenConns {
		return fmt.Errorf("postgres-max-idle-conns (%d) cannot be greater than postgres-max-open-conns (%d)", maxIdleConns, maxOpenConns)
	}
	return nil
}
