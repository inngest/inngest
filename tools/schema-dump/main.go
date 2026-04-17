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
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	dbpostgres "github.com/inngest/inngest/pkg/db/postgres"
	dbsqlite "github.com/inngest/inngest/pkg/db/sqlite"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	tcexec "github.com/testcontainers/testcontainers-go/exec"
	"github.com/testcontainers/testcontainers-go/wait"
	_ "modernc.org/sqlite"
)

const (
	defaultSQLiteOutputs   = "pkg/cqrs/base_cqrs/sqlc/sqlite/schema.sql,pkg/db/sqlite/schema.sql"
	defaultPostgresOutputs = "pkg/cqrs/base_cqrs/sqlc/postgres/schema.sql,pkg/db/postgres/schema.sql"
	defaultPostgresImage   = "postgres:16-alpine"
	defaultPostgresDBName  = "inngest_schema_dump"
	defaultPostgresUser    = "postgres"
)

type config struct {
	dialect         string
	sqliteOutputs   []string
	postgresOutputs []string
	postgresImage   string
	postgresWait    time.Duration
}

func main() {
	cfg := config{}
	var sqliteOutputs string
	var postgresOutputs string
	flag.StringVar(&cfg.dialect, "dialect", "all", "Which schema to dump: sqlite, postgres, or all")
	flag.StringVar(&sqliteOutputs, "sqlite-output", defaultSQLiteOutputs, "Comma-separated paths to write the SQLite schema dump")
	flag.StringVar(&postgresOutputs, "postgres-output", defaultPostgresOutputs, "Comma-separated paths to write the Postgres schema dump")
	flag.StringVar(&cfg.postgresImage, "postgres-image", defaultPostgresImage, "Docker image to use for the ephemeral Postgres instance")
	flag.DurationVar(&cfg.postgresWait, "postgres-wait", 30*time.Second, "How long to wait for the Postgres container to accept connections")
	flag.Parse()
	cfg.sqliteOutputs = splitOutputPaths(sqliteOutputs)
	cfg.postgresOutputs = splitOutputPaths(postgresOutputs)

	if err := run(context.Background(), cfg); err != nil {
		fmt.Fprintf(os.Stderr, "schema dump failed: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, cfg config) error {
	switch cfg.dialect {
	case "sqlite":
		return dumpSQLiteToFiles(ctx, cfg.sqliteOutputs)
	case "postgres":
		return dumpPostgresToFiles(ctx, cfg.postgresOutputs, cfg.postgresImage, cfg.postgresWait)
	case "all":
		if err := dumpSQLiteToFiles(ctx, cfg.sqliteOutputs); err != nil {
			return err
		}
		return dumpPostgresToFiles(ctx, cfg.postgresOutputs, cfg.postgresImage, cfg.postgresWait)
	default:
		return fmt.Errorf("unsupported dialect %q", cfg.dialect)
	}
}

func dumpSQLiteToFiles(ctx context.Context, outputPaths []string) error {
	schema, err := dumpSQLiteSchema(ctx)
	if err != nil {
		return err
	}

	if err := writeSchemaFiles(outputPaths, schema); err != nil {
		return err
	}

	fmt.Printf("wrote sqlite schema to %s\n", strings.Join(outputPaths, ", "))
	return nil
}

func dumpPostgresToFiles(ctx context.Context, outputPaths []string, image string, wait time.Duration) error {
	schema, err := dumpPostgresSchema(ctx, image, wait)
	if err != nil {
		return err
	}

	if err := writeSchemaFiles(outputPaths, schema); err != nil {
		return err
	}

	fmt.Printf("wrote postgres schema to %s\n", strings.Join(outputPaths, ", "))
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

	if _, err := db.ExecContext(ctx, `DROP TABLE IF EXISTS goose_db_version`); err != nil {
		return "", fmt.Errorf("dropping sqlite goose metadata table: %w", err)
	}

	// Use the sqlite3 shell's .schema output instead of reconstructing DDL from
	// sqlite_schema rows ourselves. This stays closer to the SQL text SQLite
	// emits for humans, which is a better fit for the checked-in sqlc schema.
	dump, err := runCommand(ctx, "sqlite3", dbPath, ".schema")
	if err != nil {
		return "", fmt.Errorf("dumping sqlite schema: %w", err)
	}

	if len(dump) == 0 {
		return "", errors.New("sqlite schema dump was empty")
	}

	return normalizeSQLiteDump(dump), nil
}

func dumpPostgresSchema(ctx context.Context, image string, wait time.Duration) (string, error) {
	password, err := randomHex(12)
	if err != nil {
		return "", err
	}

	// Use an ephemeral Postgres container so the schema dump always reflects the
	// checked-in migrations instead of any locally running database state.
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

	if err := migratePostgres(ctx, db); err != nil {
		return "", err
	}

	if _, err := db.ExecContext(ctx, `DROP TABLE IF EXISTS goose_db_version`); err != nil {
		return "", fmt.Errorf("dropping postgres goose metadata table: %w", err)
	}

	// Run pg_dump inside the container that already has the migrated schema. This
	// gives us the database's own canonical DDL, which we normalize below before
	// writing it into the sqlc schema file.
	dump, err := runContainerCommand(
		ctx,
		container,
		"pg_dump",
		"--schema-only",
		"--no-owner",
		"--no-privileges",
		// "--exclude-table=migrations",
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
	// Match the test setup used elsewhere in the repo: start a throwaway
	// container, wait for it to accept connections, then derive a DSN for
	// running migrations and pg_dump.
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
	migrationsFS, err := fs.Sub(dbsqlite.MigrationsFS, "migrations")
	if err != nil {
		return err
	}

	provider, err := goose.NewProvider(goose.DialectSQLite3, db, migrationsFS)
	if err != nil {
		return err
	}

	_, err = provider.Up(ctx)
	return err
}

func migratePostgres(ctx context.Context, db *sql.DB) error {
	// Apply the embedded Postgres migrations into the ephemeral container, then
	// use pg_dump to extract the final DDL from the database itself.
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

func writeSchemaFiles(outputPaths []string, contents string) error {
	if len(outputPaths) == 0 {
		return errors.New("no output paths configured")
	}

	for _, outputPath := range outputPaths {
		if err := os.MkdirAll(filepath.Dir(outputPath), 0o750); err != nil {
			return err
		}
		if err := os.WriteFile(outputPath, []byte(contents), 0o644); err != nil {
			return err
		}
	}

	return nil
}

func splitOutputPaths(raw string) []string {
	if raw == "" {
		return nil
	}

	seen := map[string]struct{}{}
	outputs := make([]string, 0, 2)
	for _, item := range strings.Split(raw, ",") {
		path := strings.TrimSpace(item)
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		outputs = append(outputs, path)
	}

	return outputs
}

func normalizeSQLiteDump(raw string) string {
	lines := strings.Split(raw, "\n")
	filtered := make([]string, 0, len(lines))
	blankPending := false
	skipStatement := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if skipStatement {
			if strings.HasSuffix(trimmed, ";") {
				skipStatement = false
			}
			continue
		}

		switch {
		case trimmed == "":
			blankPending = len(filtered) > 0
			continue
		case strings.Contains(trimmed, "goose_db_version"):
			skipStatement = !strings.HasSuffix(trimmed, ";")
			continue
		case strings.Contains(trimmed, "sqlite_sequence"):
			skipStatement = !strings.HasSuffix(trimmed, ";")
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

func normalizePostgresDump(raw string) string {
	lines := strings.Split(raw, "\n")
	filtered := make([]string, 0, len(lines))
	blankPending := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// pg_dump includes session setup, ownership, schema qualifiers, and other
		// environment-specific statements that are not useful to sqlc. Strip those
		// so the checked-in schema stays portable and focused on DDL.
		switch {
		case trimmed == "":
			blankPending = len(filtered) > 0
			continue
		// case strings.HasPrefix(trimmed, "SET "):
		// 	continue
		// case strings.HasPrefix(trimmed, "SELECT pg_catalog.set_config"):
		// 	continue
		case strings.HasPrefix(trimmed, "\\restrict "):
			continue
		case strings.HasPrefix(trimmed, "\\unrestrict "):
			continue
			// case strings.HasPrefix(trimmed, "CREATE SCHEMA public"):
			// 	continue
			// case strings.HasPrefix(trimmed, "COMMENT ON SCHEMA public"):
			// 	continue
			// case strings.HasPrefix(trimmed, "ALTER SCHEMA public OWNER TO"):
			// 	continue
			// case strings.HasPrefix(trimmed, "ALTER TABLE ") && strings.Contains(trimmed, " OWNER TO "):
			// 	continue
			// case strings.HasPrefix(trimmed, "ALTER INDEX ") && strings.Contains(trimmed, " OWNER TO "):
			// 	continue
			// case strings.HasPrefix(trimmed, "REVOKE "):
			// 	continue
			// case strings.HasPrefix(trimmed, "GRANT "):
			// 	continue
		}

		// line = strings.ReplaceAll(line, "public.", "")

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
	// Exec output is multiplexed by default; request the demultiplexed stream so
	// pg_dump output is plain SQL text rather than Docker-framed bytes.
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

func runCommand(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}
		if msg == "" {
			msg = err.Error()
		}
		return "", errors.New(msg)
	}

	return stdout.String(), nil
}
