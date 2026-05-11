package start

import (
	"os"
	"path/filepath"
	"testing"

	localconfig "github.com/inngest/inngest/cmd/internal/config"
	"github.com/inngest/inngest/pkg/devserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
)

func TestValidatePostgreSQLConnectionPool(t *testing.T) {
	tests := []struct {
		name                  string
		postgresMaxIdleConns  int
		postgresMaxOpenConns  int
		expectError           bool
		expectedErrorContains string
	}{
		{
			name:                 "valid connection pool settings",
			postgresMaxIdleConns: 10,
			postgresMaxOpenConns: 100,
			expectError:          false,
		},
		{
			name:                  "max open connections too low (0)",
			postgresMaxIdleConns:  10,
			postgresMaxOpenConns:  0,
			expectError:           true,
			expectedErrorContains: "postgres-max-open-conns (0) must be greater than 1",
		},
		{
			name:                  "max open connections too low (1)",
			postgresMaxIdleConns:  10,
			postgresMaxOpenConns:  1,
			expectError:           true,
			expectedErrorContains: "postgres-max-open-conns (1) must be greater than 1",
		},
		{
			name:                 "max open connections at boundary (2)",
			postgresMaxIdleConns: 1,
			postgresMaxOpenConns: 2,
			expectError:          false,
		},
		{
			name:                  "max idle connections greater than max open connections",
			postgresMaxIdleConns:  50,
			postgresMaxOpenConns:  25,
			expectError:           true,
			expectedErrorContains: "postgres-max-idle-conns (50) cannot be greater than postgres-max-open-conns (25)",
		},
		{
			name:                 "max idle connections equal to max open connections",
			postgresMaxIdleConns: 25,
			postgresMaxOpenConns: 25,
			expectError:          false,
		},
		{
			name:                 "zero max idle connections with valid max open connections",
			postgresMaxIdleConns: 0,
			postgresMaxOpenConns: 10,
			expectError:          false,
		},
		{
			name:                 "default values should be valid",
			postgresMaxIdleConns: 10,
			postgresMaxOpenConns: 100,
			expectError:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePostgreSQLConnectionPool(tt.postgresMaxIdleConns, tt.postgresMaxOpenConns)

			if tt.expectError {
				assert.Error(t, err)
				if tt.expectedErrorContains != "" {
					assert.Contains(t, err.Error(), tt.expectedErrorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPostgresConnectionPoolConfigFromCommandUsesEnvBackedConfig(t *testing.T) {
	configFile := filepath.Join(t.TempDir(), "inngest.yml")
	require.NoError(t, os.WriteFile(configFile, []byte("{}\n"), 0644))

	t.Setenv("INNGEST_CONFIG", configFile)
	t.Setenv("INNGEST_POSTGRES_MAX_IDLE_CONNS", "17")
	t.Setenv("INNGEST_POSTGRES_MAX_OPEN_CONNS", "171")
	t.Setenv("INNGEST_POSTGRES_CONN_MAX_IDLE_TIME", "11")
	t.Setenv("INNGEST_POSTGRES_CONN_MAX_LIFETIME", "61")

	cmd := &cli.Command{
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "config"},
			&cli.IntFlag{Name: "postgres-max-idle-conns", Value: devserver.DefaultPostgresMaxIdleConns},
			&cli.IntFlag{Name: "postgres-max-open-conns", Value: devserver.DefaultPostgresMaxOpenConns},
			&cli.IntFlag{Name: "postgres-conn-max-idle-time", Value: devserver.DefaultPostgresConnMaxIdleTime},
			&cli.IntFlag{Name: "postgres-conn-max-lifetime", Value: devserver.DefaultPostgresConnMaxLifetime},
		},
	}

	err := localconfig.InitStartConfig(t.Context(), cmd)
	require.NoError(t, err)

	pool := postgresConnectionPoolConfigFromCommand(cmd)
	assert.Equal(t, 17, pool.maxIdleConns)
	assert.Equal(t, 171, pool.maxOpenConns)
	assert.Equal(t, 11, pool.connMaxIdleTime)
	assert.Equal(t, 61, pool.connMaxLifetime)
}

func TestValidatePostgresConnectionPoolConfigSkipsSQLite(t *testing.T) {
	err := validatePostgresConnectionPoolConfig("", postgresConnectionPoolConfig{
		maxIdleConns: 10,
		maxOpenConns: 1,
	})

	require.NoError(t, err)
}
