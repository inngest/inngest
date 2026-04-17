package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/inngest/inngest/pkg/cqrs/base_cqrs"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	_ "modernc.org/sqlite"
)

const (
	defaultSQLiteOutput   = "pkg/cqrs/base_cqrs/sqlc/sqlite/schema.sql"
	defaultPostgresOutput = "pkg/cqrs/base_cqrs/sqlc/postgres/schema.sql"
	defaultPostgresImage  = "postgres:16-alpine"
	defaultPostgresDBName = "inngest_schema_dump"
	defaultPostgresUser   = "postgres"
)

type config struct {
	dialect        string
	sqliteOutput   string
	postgresOutput string
	postgresImage  string
	postgresWait   time.Duration
}

func main() {
	cfg := config{}
	flag.StringVar(&cfg.dialect, "dialect", "all", "Which schema to dump: sqlite, postgres, or all")
	flag.StringVar(&cfg.sqliteOutput, "sqlite-output", defaultSQLiteOutput, "Path to write the SQLite schema dump")
	flag.StringVar(&cfg.postgresOutput, "postgres-output", defaultPostgresOutput, "Path to write the Postgres schema dump")
	flag.StringVar(&cfg.postgresImage, "postgres-image", defaultPostgresImage, "Docker image to use for the ephemeral Postgres instance")
	flag.DurationVar(&cfg.postgresWait, "postgres-wait", 30*time.Second, "How long to wait for the Postgres container to accept connections")
	flag.Parse()

	if err := run(context.Background(), cfg); err != nil {
		fmt.Fprintf(os.Stderr, "schema dump failed: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, cfg config) error {
	switch cfg.dialect {
	case "sqlite":
		return dumpSQLiteToFile(ctx, cfg.sqliteOutput)
	case "postgres":
		return dumpPostgresToFile(ctx, cfg.postgresOutput, cfg.postgresImage, cfg.postgresWait)
	case "all":
		if err := dumpSQLiteToFile(ctx, cfg.sqliteOutput); err != nil {
			return err
		}
		return dumpPostgresToFile(ctx, cfg.postgresOutput, cfg.postgresImage, cfg.postgresWait)
	default:
		return fmt.Errorf("unsupported dialect %q", cfg.dialect)
	}
}

func dumpSQLiteToFile(ctx context.Context, outputPath string) error {
	schema, err := dumpSQLiteSchema(ctx)
	if err != nil {
		return err
	}

	if err := writeSchema(outputPath, schema); err != nil {
		return err
	}

	fmt.Printf("wrote sqlite schema to %s\n", outputPath)
	return nil
}

func dumpPostgresToFile(ctx context.Context, outputPath, image string, wait time.Duration) error {
	schema, err := dumpPostgresSchema(ctx, image, wait)
	if err != nil {
		return err
	}

	if err := writeSchema(outputPath, schema); err != nil {
		return err
	}

	fmt.Printf("wrote postgres schema to %s\n", outputPath)
	return nil
}

func dumpSQLiteSchema(ctx context.Context) (string, error) {
	tempDir, err := os.MkdirTemp("", "inngest-schema-sqlite-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "schema.db")
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?cache=shared", dbPath))
	if err != nil {
		return "", err
	}
	defer db.Close()

	if err := migrateSQLite(ctx, db); err != nil {
		return "", err
	}

	const query = `
SELECT sql
FROM sqlite_schema
WHERE sql IS NOT NULL
  AND type IN ('table', 'index')
  AND name NOT LIKE 'sqlite_%'
  AND name != 'migrations'
ORDER BY
  CASE type WHEN 'table' THEN 0 ELSE 1 END,
  rowid
`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	statements := []string{}
	for rows.Next() {
		var stmt string
		if err := rows.Scan(&stmt); err != nil {
			return "", err
		}

		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if !strings.HasSuffix(stmt, ";") {
			stmt += ";"
		}
		statements = append(statements, stmt)
	}

	if err := rows.Err(); err != nil {
		return "", err
	}

	if len(statements) == 0 {
		return "", errors.New("sqlite schema dump was empty")
	}

	return strings.Join(statements, "\n\n") + "\n", nil
}

func dumpPostgresSchema(ctx context.Context, image string, wait time.Duration) (string, error) {
	password, err := randomHex(12)
	if err != nil {
		return "", err
	}

	container, dsn, err := startPostgresContainer(ctx, image, password, wait)
	if err != nil {
		return "", err
	}
	defer func() { _ = container.Terminate(context.Background()) }()

	db, err := waitForPostgres(ctx, dsn, wait)
	if err != nil {
		return "", err
	}
	defer db.Close()

	if err := migratePostgres(ctx, db, dsn); err != nil {
		return "", err
	}

	dump, err := runContainerCommand(
		ctx,
		container,
		"pg_dump",
		"--schema-only",
		"--no-owner",
		"--no-privileges",
		"--exclude-table=migrations",
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

func migrateSQLite(ctx context.Context, db *sql.DB) error {
	src, err := iofs.New(base_cqrs.FS, path.Join("migrations", "sqlite"))
	if err != nil {
		return err
	}

	driver, err := sqlite.WithInstance(db, &sqlite.Config{
		MigrationsTable: "migrations",
		NoTxWrap:        true,
	})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance("iofs", src, "sqlite", driver)
	if err != nil {
		return err
	}

	err = m.Up()
	if err == migrate.ErrNoChange {
		return nil
	}
	return err
}

func migratePostgres(ctx context.Context, db *sql.DB, dsn string) error {
	src, err := iofs.New(base_cqrs.FS, path.Join("migrations", "postgres"))
	if err != nil {
		return err
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{
		MigrationsTable: "migrations",
		DatabaseName:    defaultPostgresDBName,
	})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance("iofs", src, dsn, driver)
	if err != nil {
		return err
	}

	err = m.Up()
	if err == migrate.ErrNoChange {
		return nil
	}
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

func writeSchema(outputPath, contents string) error {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o750); err != nil {
		return err
	}
	return os.WriteFile(outputPath, []byte(contents), 0o644)
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
		case strings.HasPrefix(trimmed, "--"):
			continue
		case strings.HasPrefix(trimmed, "SET "):
			continue
		case strings.HasPrefix(trimmed, "SELECT pg_catalog.set_config"):
			continue
		case strings.HasPrefix(trimmed, "\\restrict "):
			continue
		case strings.HasPrefix(trimmed, "\\unrestrict "):
			continue
		case strings.HasPrefix(trimmed, "CREATE SCHEMA public"):
			continue
		case strings.HasPrefix(trimmed, "COMMENT ON SCHEMA public"):
			continue
		case strings.HasPrefix(trimmed, "ALTER SCHEMA public OWNER TO"):
			continue
		case strings.HasPrefix(trimmed, "ALTER TABLE ") && strings.Contains(trimmed, " OWNER TO "):
			continue
		case strings.HasPrefix(trimmed, "ALTER INDEX ") && strings.Contains(trimmed, " OWNER TO "):
			continue
		case strings.HasPrefix(trimmed, "REVOKE "):
			continue
		case strings.HasPrefix(trimmed, "GRANT "):
			continue
		}

		line = strings.ReplaceAll(line, "public.", "")

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
	exitCode, reader, err := container.Exec(ctx, args)
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
