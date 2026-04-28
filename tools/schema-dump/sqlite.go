package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	dbsqlite "github.com/inngest/inngest/pkg/db/sqlite"
	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

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

	dump, err := runCommand(ctx, "sqlite3", dbPath, ".schema")
	if err != nil {
		return "", fmt.Errorf("dumping sqlite schema: %w", err)
	}

	if len(dump) == 0 {
		return "", errors.New("sqlite schema dump was empty")
	}

	return normalizeSQLiteDump(dump), nil
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
