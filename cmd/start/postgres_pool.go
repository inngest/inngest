package start

import (
	"fmt"

	localconfig "github.com/inngest/inngest/cmd/internal/config"
	"github.com/inngest/inngest/pkg/devserver"
	"github.com/urfave/cli/v3"
)

type postgresConnectionPoolConfig struct {
	maxIdleConns    int
	maxOpenConns    int
	connMaxIdleTime int
	connMaxLifetime int
}

func postgresConnectionPoolConfigFromCommand(cmd *cli.Command) postgresConnectionPoolConfig {
	return postgresConnectionPoolConfig{
		maxIdleConns:    localconfig.GetIntValue(cmd, "postgres-max-idle-conns", devserver.DefaultPostgresMaxIdleConns),
		maxOpenConns:    localconfig.GetIntValue(cmd, "postgres-max-open-conns", devserver.DefaultPostgresMaxOpenConns),
		connMaxIdleTime: localconfig.GetIntValue(cmd, "postgres-conn-max-idle-time", devserver.DefaultPostgresConnMaxIdleTime),
		connMaxLifetime: localconfig.GetIntValue(cmd, "postgres-conn-max-lifetime", devserver.DefaultPostgresConnMaxLifetime),
	}
}

func validatePostgresConnectionPoolConfig(postgresURI string, pool postgresConnectionPoolConfig) error {
	if postgresURI == "" {
		return nil
	}

	return validatePostgreSQLConnectionPool(pool.maxIdleConns, pool.maxOpenConns)
}

func validatePostgreSQLConnectionPool(maxIdleConns, maxOpenConns int) error {
	if maxOpenConns <= 1 {
		return &postgresSQLValidationError{
			message: fmt.Sprintf("postgres-max-open-conns (%d) must be greater than 1", maxOpenConns),
		}
	}
	if maxIdleConns > maxOpenConns {
		return &postgresSQLValidationError{
			message: fmt.Sprintf("postgres-max-idle-conns (%d) cannot be greater than postgres-max-open-conns (%d)",
				maxIdleConns, maxOpenConns),
		}
	}
	return nil
}

type postgresSQLValidationError struct {
	message string
}

func (e *postgresSQLValidationError) Error() string {
	return e.message
}
