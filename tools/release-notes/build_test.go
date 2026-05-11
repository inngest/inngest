package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractChangelogSection(t *testing.T) {
	changelog := `# Changelog

## [v1.2.3] - 2026-05-08

### Features

- Add thing

## [v1.2.2] - 2026-05-01

- Old thing
`

	got, err := ExtractChangelogSection(changelog, "v1.2.3")
	if err != nil {
		t.Fatal(err)
	}

	want := "### Features\n\n- Add thing"
	if got != want {
		t.Fatalf("section = %q, want %q", got, want)
	}
}

func TestExtractChangelogSectionHeadingFormats(t *testing.T) {
	tests := []struct {
		name      string
		changelog string
	}{
		{
			name: "unprefixed bracket",
			changelog: `## [1.2.3] - 2026-05-08

- Entry
`,
		},
		{
			name: "prefixed plain",
			changelog: `## v1.2.3

- Entry
`,
		},
		{
			name: "unprefixed plain",
			changelog: `## 1.2.3

- Entry
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractChangelogSection(tt.changelog, "v1.2.3")
			if err != nil {
				t.Fatal(err)
			}
			if got != "- Entry" {
				t.Fatalf("section = %q", got)
			}
		})
	}
}

func TestExtractChangelogSectionMissing(t *testing.T) {
	_, err := ExtractChangelogSection("## [v1.2.2]\n\n- Old", "v1.2.3")
	if err == nil {
		t.Fatal("expected missing changelog section error")
	}
}

func TestBuildReleaseNotesDeterministic(t *testing.T) {
	notes := NotesFile{PRs: []PullRequest{
		{
			Number:        20,
			Title:         "feat: add second thing",
			URL:           "https://github.com/inngest/inngest/pull/20",
			ReleaseNote:   "Second thing is available.",
			MigrationNote: "Set `SECOND_THING=true` before enabling it.",
		},
		{
			Number:      10,
			Title:       "fix: repair first thing",
			URL:         "https://github.com/inngest/inngest/pull/10",
			ReleaseNote: "First thing no longer fails.",
		},
		{
			Number:      30,
			Title:       "cloud: dashboard only",
			ReleaseNote: "Should not appear.",
			Excluded:    true,
		},
	}}
	releasePRBody := `## Additional Release Notes
<!-- release-note:manual-start -->
Manual release context.
<!-- release-note:manual-end -->

## Additional Migration Notes
<!-- migration-note:manual-start -->
Manual migration context.
<!-- migration-note:manual-end -->`

	got, err := BuildReleaseNotes(notes, "### Features\n\n- Add thing", releasePRBody)
	if err != nil {
		t.Fatal(err)
	}
	gotAgain, err := BuildReleaseNotes(notes, "### Features\n\n- Add thing", releasePRBody)
	if err != nil {
		t.Fatal(err)
	}
	if got != gotAgain {
		t.Fatal("BuildReleaseNotes output is not deterministic")
	}

	assertContains(t, got, "## Release Notes")
	assertContains(t, got, "Manual release context.")
	assertContains(t, got, "[#10](https://github.com/inngest/inngest/pull/10) fix: repair first thing")
	assertContains(t, got, "[#20](https://github.com/inngest/inngest/pull/20) feat: add second thing")
	assertContains(t, got, "## Migration Notes")
	assertContains(t, got, "Manual migration context.")
	assertContains(t, got, "Set `SECOND_THING=true` before enabling it.")
	assertContains(t, got, "## Changelog")
	assertContains(t, got, "### Features")

	if strings.Contains(got, "Should not appear") {
		t.Fatalf("excluded note rendered:\n%s", got)
	}
	if strings.Index(got, "#10") > strings.Index(got, "#20") {
		t.Fatalf("PR notes not sorted deterministically:\n%s", got)
	}
}

func TestBuildReleaseNotesOmitsEmptySections(t *testing.T) {
	got, err := BuildReleaseNotes(NotesFile{}, "### Bug Fixes\n\n- Fix thing", "")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "## Release Notes") {
		t.Fatalf("unexpected release notes section:\n%s", got)
	}
	if strings.Contains(got, "## Migration Notes") {
		t.Fatalf("unexpected migration notes section:\n%s", got)
	}
	assertContains(t, got, "## Changelog")
	assertContains(t, got, "### Bug Fixes")
}

func TestBuildReleaseNotesManualPlaceholdersAreIgnored(t *testing.T) {
	releasePRBody := `## Additional Release Notes
<!-- release-note:manual-start -->
None.
<!-- release-note:manual-end -->

## Additional Migration Notes
<!-- migration-note:manual-start -->
N/A
<!-- migration-note:manual-end -->`

	got, err := BuildReleaseNotes(NotesFile{}, "### Bug Fixes\n\n- Fix thing", releasePRBody)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "## Release Notes") || strings.Contains(got, "## Migration Notes") {
		t.Fatalf("placeholder manual notes should not render:\n%s", got)
	}
}

func TestBuildCommandWritesOutput(t *testing.T) {
	dir := t.TempDir()
	notesPath := filepath.Join(dir, "notes.json")
	changelogPath := filepath.Join(dir, "CHANGELOG.md")
	outputPath := filepath.Join(dir, "RELEASE_NOTES.md")

	notes := NotesFile{PRs: []PullRequest{
		{
			Number:      2,
			Title:       "fix: repair thing",
			ReleaseNote: "Thing is repaired.",
		},
	}}
	data, err := json.Marshal(notes)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(notesPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(changelogPath, []byte("## [v1.2.3] - 2026-05-08\n\n### Bug Fixes\n\n- Repair thing\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err = run([]string{
		"build",
		"--notes", notesPath,
		"--changelog", changelogPath,
		"--tag", "v1.2.3",
		"--output", outputPath,
	})
	if err != nil {
		t.Fatal(err)
	}

	output, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	got := string(output)
	assertContains(t, got, "Thing is repaired.")
	assertContains(t, got, "### Bug Fixes")
}
