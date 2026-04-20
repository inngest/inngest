package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
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
