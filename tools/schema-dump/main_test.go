package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDumpSQLiteSchema(t *testing.T) {
	t.Parallel()

	schema, err := dumpSQLiteSchema(context.Background())
	if err != nil {
		t.Fatalf("dumpSQLiteSchema returned error: %v", err)
	}

	for _, want := range []string{
		"CREATE TABLE apps",
		"CREATE TABLE spans",
		"CREATE INDEX idx_spans_run_id ON spans(run_id);",
	} {
		if !strings.Contains(schema, want) {
			t.Fatalf("sqlite schema missing %q:\n%s", want, schema)
		}
	}
}

func TestNormalizePostgresDump(t *testing.T) {
	t.Parallel()

	raw := `
--
-- PostgreSQL database dump
--

SET statement_timeout = 0;
SET client_encoding = 'UTF8';
SELECT pg_catalog.set_config('search_path', '', false);
CREATE SCHEMA public;
COMMENT ON SCHEMA public IS 'standard public schema';

CREATE TABLE public.apps (
    id uuid NOT NULL
);

ALTER TABLE ONLY public.apps
    ADD CONSTRAINT apps_pkey PRIMARY KEY (id);

ALTER TABLE apps OWNER TO postgres;

CREATE INDEX idx_apps_id ON public.apps (id);
\unrestrict abc123
`

	got := normalizePostgresDump(raw)

	for _, unwanted := range []string{
		"\\unrestrict",
	} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("normalized dump should not contain %q:\n%s", unwanted, got)
		}
	}

	for _, want := range []string{
		"SET statement_timeout = 0;",
		"SELECT pg_catalog.set_config('search_path', '', false);",
		"CREATE SCHEMA public;",
		"COMMENT ON SCHEMA public IS 'standard public schema';",
		"CREATE TABLE public.apps (",
		"ALTER TABLE ONLY public.apps",
		"ALTER TABLE apps OWNER TO postgres;",
		"CREATE INDEX idx_apps_id ON public.apps (id);",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("normalized dump missing %q:\n%s", want, got)
		}
	}
}

func TestRunSQLiteWritesRequestedOutput(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "nested", "sqlite-schema.sql")

	err := run(context.Background(), config{
		dialect:      "sqlite",
		sqliteOutput: outputPath,
	})
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	got, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("reading output file: %v", err)
	}

	for _, want := range []string{
		"CREATE TABLE apps",
		"CREATE TABLE spans",
	} {
		if !strings.Contains(string(got), want) {
			t.Fatalf("sqlite output missing %q:\n%s", want, string(got))
		}
	}
}
