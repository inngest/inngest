package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCollectCommandFromInput(t *testing.T) {
	dir := t.TempDir()
	inputPath := filepath.Join(dir, "prs.json")
	cliffPath := filepath.Join(dir, "cliff.toml")
	outputPath := filepath.Join(dir, "notes.json")

	prs := []PullRequest{
		{
			Number: 1,
			Title:  "feat: add thing",
			Body:   "## Description\n\nAdds a thing.\n\n## Release note\n\nThing is available.\n\n## Migration note\n\nNone.\n",
			Files:  []string{"pkg/execution/run.go"},
		},
	}
	data, err := json.Marshal(prs)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(inputPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cliffPath, []byte("exclude_paths = []\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := run([]string{"collect", "--input", inputPath, "--cliff", cliffPath, "--output", outputPath}); err != nil {
		t.Fatal(err)
	}

	notes, err := readNotesFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(notes.PRs) != 1 {
		t.Fatalf("PR count = %d", len(notes.PRs))
	}
	if notes.PRs[0].ReleaseNote != "Thing is available." {
		t.Fatalf("release note = %q", notes.PRs[0].ReleaseNote)
	}
	if notes.PRs[0].MigrationNote != "" {
		t.Fatalf("migration note = %q", notes.PRs[0].MigrationNote)
	}
}
