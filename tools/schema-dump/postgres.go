package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strings"
	"time"

	dbpostgres "github.com/inngest/inngest/pkg/db/postgres"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	tcexec "github.com/testcontainers/testcontainers-go/exec"
	"github.com/testcontainers/testcontainers-go/wait"
)

func dumpPostgresToFiles(ctx context.Context, outputPaths []string, image string, waitTimeout time.Duration) error {
	schema, err := dumpPostgresSchema(ctx, image, waitTimeout)
	if err != nil {
		return err
	}

	if err := writeSchemaFiles(outputPaths, schema); err != nil {
		return err
	}

	fmt.Printf("wrote postgres schema to %s\n", strings.Join(outputPaths, ", "))
	return nil
}

func dumpPostgresSchema(ctx context.Context, image string, waitTimeout time.Duration) (string, error) {
	password, err := randomHex(12)
	if err != nil {
		return "", err
	}

	container, dsn, err := startPostgresContainer(ctx, image, password, waitTimeout)
	if err != nil {
		return "", err
	}
	defer func() { _ = container.Terminate(context.Background()) }()

	db, err := waitForPostgres(ctx, dsn, waitTimeout)
	if err != nil {
		return "", err
	}
	defer db.Close()

	if err := migratePostgres(ctx, db); err != nil {
		return "", err
	}

	dump, err := runContainerCommand(
		ctx,
		container,
		"pg_dump",
		"--schema-only",
		"--no-owner",
		"--no-privileges",
		"--username", defaultPostgresUser,
		"--dbname", defaultPostgresDBName,
	)
	if err != nil {
		return "", fmt.Errorf("dumping postgres schema: %w", err)
	}

	schema := normalizePostgresDump(dump)
	if schema == "" {
		return "", errors.New("postgres schema dump was empty after normalization")
	}

	return schema, nil
}

func startPostgresContainer(ctx context.Context, image, password string, startupTimeout time.Duration) (testcontainers.Container, string, error) {
	req := testcontainers.ContainerRequest{
		Image:        image,
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     defaultPostgresUser,
			"POSTGRES_PASSWORD": password,
			"POSTGRES_DB":       defaultPostgresDBName,
		},
		WaitingFor: wait.ForAll(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
			wait.ForListeningPort("5432/tcp"),
		).WithDeadline(startupTimeout),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, "", fmt.Errorf("starting postgres container: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(context.Background())
		return nil, "", fmt.Errorf("discovering postgres host: %w", err)
	}

	mappedPort, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		_ = container.Terminate(context.Background())
		return nil, "", fmt.Errorf("discovering postgres port: %w", err)
	}

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		defaultPostgresUser,
		password,
		host,
		mappedPort.Port(),
		defaultPostgresDBName,
	)

	return container, dsn, nil
}

func migratePostgres(ctx context.Context, db *sql.DB) error {
	migrationsFS, err := fs.Sub(dbpostgres.MigrationsFS, "migrations")
	if err != nil {
		return err
	}

	provider, err := goose.NewProvider(goose.DialectPostgres, db, migrationsFS)
	if err != nil {
		return err
	}

	_, err = provider.Up(ctx)
	return err
}

func waitForPostgres(ctx context.Context, dsn string, timeout time.Duration) (*sql.DB, error) {
	deadline := time.Now().Add(timeout)
	var lastErr error

	for time.Now().Before(deadline) {
		db, err := sql.Open("pgx", dsn)
		if err != nil {
			lastErr = err
			time.Sleep(500 * time.Millisecond)
			continue
		}

		pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		err = db.PingContext(pingCtx)
		cancel()
		if err == nil {
			return db, nil
		}

		lastErr = err
		_ = db.Close()
		time.Sleep(500 * time.Millisecond)
	}

	if lastErr == nil {
		lastErr = errors.New("timed out waiting for postgres")
	}
	return nil, fmt.Errorf("waiting for postgres: %w", lastErr)
}

func normalizePostgresDump(raw string) string {
	lines := strings.Split(raw, "\n")
	filtered := make([]string, 0, len(lines))
	blankPending := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		switch {
		case trimmed == "":
			blankPending = len(filtered) > 0
			continue
		case strings.HasPrefix(trimmed, "\\restrict "):
			continue
		case strings.HasPrefix(trimmed, "\\unrestrict "):
			continue
		case strings.HasPrefix(trimmed, "-- Dumped from database version "):
			// pg_dump embeds its own version in the dump header. The
			// `postgres:16-alpine` image tag is mutable and minor-version
			// bumps would otherwise produce spurious diffs in the committed
			// schema file. The version is metadata, not schema.
			continue
		case strings.HasPrefix(trimmed, "-- Dumped by pg_dump version "):
			continue
		}

		if blankPending && len(filtered) > 0 {
			filtered = append(filtered, "")
		}
		blankPending = false
		filtered = append(filtered, strings.TrimRight(line, " \t"))
	}

	return strings.TrimSpace(strings.Join(filtered, "\n")) + "\n"
}

func randomHex(byteCount int) (string, error) {
	buf := make([]byte, byteCount)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", buf), nil
}

func runContainerCommand(ctx context.Context, container testcontainers.Container, args ...string) (string, error) {
	exitCode, reader, err := container.Exec(ctx, args, tcexec.Multiplexed())
	if err != nil {
		return "", err
	}

	var output bytes.Buffer
	if reader != nil {
		if _, err := io.Copy(&output, reader); err != nil {
			return "", err
		}
	}

	if exitCode != 0 {
		msg := strings.TrimSpace(output.String())
		if msg == "" {
			msg = fmt.Sprintf("%s exited with code %d", args[0], exitCode)
		}
		return "", errors.New(msg)
	}

	return output.String(), nil
}
