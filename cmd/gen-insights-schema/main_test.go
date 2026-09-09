package main

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTableDumpsMatchesRegistry(t *testing.T) {
	tables := tableDumps()
	require.Len(t, tables, 6)

	names := make(map[string]bool, len(tables))
	for _, tbl := range tables {
		names[tbl.Name] = true
		require.NotEmptyf(t, tbl.Description, "table %q has no description", tbl.Name)
		require.NotEmptyf(t, tbl.Columns, "table %q has no columns", tbl.Name)
		for _, col := range tbl.Columns {
			require.NotEmptyf(t, col.Name, "table %q has an unnamed column", tbl.Name)
			require.NotEmptyf(t, col.Type, "table %q column %q has no type", tbl.Name, col.Name)
			require.NotEmptyf(t, col.Description, "table %q column %q has no description", tbl.Name, col.Name)
		}
	}
	require.Equal(t, map[string]bool{
		"runs": true, "events": true, "metadata": true,
		"extended_trace_spans": true, "steps": true, "step_attempts": true,
	}, names)

	// A hinted column (runs.run_id) reports its hint; an unhinted one
	// (events.name) omits the field entirely rather than serializing "".
	var runsRunID, eventsName *columnDump
	for _, tbl := range tables {
		for i, col := range tbl.Columns {
			if tbl.Name == "runs" && col.Name == "run_id" {
				runsRunID = &tbl.Columns[i]
			}
			if tbl.Name == "events" && col.Name == "name" {
				eventsName = &tbl.Columns[i]
			}
		}
	}
	require.NotNil(t, runsRunID)
	require.NotNil(t, runsRunID.Hint)
	require.Equal(t, "RUN_ID", *runsRunID.Hint)
	require.NotNil(t, eventsName)
	require.Nil(t, eventsName.Hint)
}

func TestFunctionDumpsMatchesRegistry(t *testing.T) {
	fns := functionDumps()
	require.Len(t, fns, 175)
	for _, fn := range fns {
		require.NotEmptyf(t, fn.Name, "a function has no name")
		require.NotEmptyf(t, fn.Description, "function %q has no description", fn.Name)
		require.NotEmptyf(t, fn.DocsURL, "function %q has no docsUrl", fn.Name)
	}
}

// TestSchemaDumpIsValidJSON proves the whole dump round-trips through the
// exact marshaling main() uses -- catching e.g. a cyclic or unexported
// field mistake that unit tests on the individual helpers wouldn't.
func TestSchemaDumpIsValidJSON(t *testing.T) {
	dump := schemaDump{Tables: tableDumps(), Functions: functionDumps()}
	b, err := json.MarshalIndent(dump, "", "  ")
	require.NoError(t, err)

	var roundTripped schemaDump
	require.NoError(t, json.Unmarshal(b, &roundTripped))
	require.Equal(t, dump, roundTripped)
}
