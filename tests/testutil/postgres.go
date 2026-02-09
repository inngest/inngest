package testutil

import (
	"fmt"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type PostgresContainer struct {
	testcontainers.Container

	URI      string
	Host     string
	Port     string
	User     string
	Password string
	Database string
}

// PostgresOption represents a configuration option for the PostgreSQL container
type PostgresOption func(*postgresConfig)

// postgresConfig holds the configuration for starting a PostgreSQL container
type postgresConfig struct {
	image    string
	user     string
	password string
	database string
}

func WithPostgresImage(image string) PostgresOption {
	return func(pc *postgresConfig) {
		pc.image = image
	}
}

func WithPostgresUser(user string) PostgresOption {
	return func(pc *postgresConfig) {
		pc.user = user
	}
}

func WithPostgresPassword(password string) PostgresOption {
	return func(pc *postgresConfig) {
		pc.password = password
	}
}

func WithPostgresDatabase(database string) PostgresOption {
	return func(pc *postgresConfig) {
		pc.database = database
	}
}

func StartPostgres(t *testing.T, opts ...PostgresOption) (*PostgresContainer, error) {
	// Apply options with defaults
	config := &postgresConfig{
		image:    PostgresDefaultImage,
		user:     "postgres",
		password: "postgres",
		database: "inngest_test",
	}
	for _, opt := range opts {
		opt(config)
	}

	ctx := t.Context()

	req := testcontainers.ContainerRequest{
		Image:        config.image,
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     config.user,
			"POSTGRES_PASSWORD": config.password,
			"POSTGRES_DB":       config.database,
		},
		WaitingFor: wait.ForAll(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
			wait.ForListeningPort("5432/tcp"),
		).WithDeadline(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start postgres container: %w", err)
	}

	// Get the mapped port
	mappedPort, err := container.MappedPort(ctx, "5432")
	if err != nil {
		return nil, fmt.Errorf("failed to get mapped port: %w", err)
	}

	// Get the host
	host, err := container.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get host: %w", err)
	}

	uri := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		config.user, config.password, host, mappedPort.Port(), config.database)

	return &PostgresContainer{
		Container: container,
		URI:       uri,
		Host:      host,
		Port:      mappedPort.Port(),
		User:      config.user,
		Password:  config.password,
		Database:  config.database,
	}, nil
}
