package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParsePRBody(t *testing.T) {
	body := `## Description

<!-- template comment -->
Adds the new thing.


## Release note

Customers can now use the new thing.

## Migration note

None.
`

	sections := ParsePRBody(body)
	if got := sections["description"]; got != "Adds the new thing." {
		t.Fatalf("description = %q", got)
	}
	if got := NormalizeNote(sections["release note"]); got != "Customers can now use the new thing." {
		t.Fatalf("release note = %q", got)
	}
	if got := NormalizeNote(sections["migration note"]); got != "" {
		t.Fatalf("migration note = %q", got)
	}
}

func TestParsePRBodyMultilineAndRepeatedSections(t *testing.T) {
	body := `## Release note

First paragraph.

Second paragraph.

## Release notes

Additional note.
`

	sections := ParsePRBody(body)
	got := NormalizeNote(sections["release note"])
	want := "First paragraph.\n\nSecond paragraph.\n\nAdditional note."
	if got != want {
		t.Fatalf("release note = %q, want %q", got, want)
	}
}

func TestParsePRBodyKeepsNestedHeadingsInsideSection(t *testing.T) {
	body := `## Release note

Users can enable the new behavior.

### Details

- Works for serve functions.
- Works for connect functions.

## Migration note

Set the new flag before rollout.
`

	sections := ParsePRBody(body)
	got := NormalizeNote(sections["release note"])
	want := "Users can enable the new behavior.\n\n### Details\n\n- Works for serve functions.\n- Works for connect functions."
	if got != want {
		t.Fatalf("release note = %q, want %q", got, want)
	}
	if got := NormalizeNote(sections["migration note"]); got != "Set the new flag before rollout." {
		t.Fatalf("migration note = %q", got)
	}
}

func TestNormalizeNotePlaceholders(t *testing.T) {
	for _, input := range []string{"", "None", "None.", "N/A", "n/a.", " NA "} {
		t.Run(input, func(t *testing.T) {
			if got := NormalizeNote(input); got != "" {
				t.Fatalf("NormalizeNote(%q) = %q, want empty", input, got)
			}
		})
	}
}

func TestParseCliffExcludePaths(t *testing.T) {
	config := `
exclude_paths = [
  # Cloud dashboard
  "ui/apps/dashboard/",
  "pkg/debugapi/", # inline comment
  # "pkg/constraintapi/",
  "cmd/debug/"
]
`

	got := ParseCliffExcludePaths(config)
	want := []string{"ui/apps/dashboard/", "pkg/debugapi/", "cmd/debug/"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("path[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestShouldExcludePR(t *testing.T) {
	excludes := []string{"ui/apps/dashboard/", "pkg/debugapi/"}

	tests := []struct {
		name string
		pr   PullRequest
		want bool
	}{
		{
			name: "all paths excluded",
			pr: PullRequest{
				Title: "feat: dashboard tweak",
				Files: []string{"ui/apps/dashboard/page.tsx", "pkg/debugapi/api.go"},
			},
			want: true,
		},
		{
			name: "mixed paths included",
			pr: PullRequest{
				Title: "feat: runtime tweak",
				Files: []string{"ui/apps/dashboard/page.tsx", "pkg/execution/run.go"},
			},
			want: false,
		},
		{
			name: "non release prefix",
			pr: PullRequest{
				Title: "internal: update tooling",
				Files: []string{"pkg/execution/run.go"},
			},
			want: true,
		},
		{
			name: "scoped non release prefix",
			pr: PullRequest{
				Title: "cloud(dashboard): update card",
				Files: []string{"pkg/execution/run.go"},
			},
			want: true,
		},
		{
			name: "similar prefix is not excluded",
			pr: PullRequest{
				Title: "internalize: expose helper",
				Files: []string{"pkg/execution/run.go"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ShouldExcludePR(tt.pr, excludes); got != tt.want {
				t.Fatalf("ShouldExcludePR() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPathExcluded(t *testing.T) {
	excludes := []string{"pkg/debugapi/", "cmd/debug", "ui/apps/dashboard/"}

	tests := []struct {
		path string
		want bool
	}{
		{path: "pkg/debugapi/api.go", want: true},
		{path: "./pkg/debugapi/api.go", want: true},
		{path: "pkg/debugapi_extra/api.go", want: false},
		{path: "cmd/debug", want: true},
		{path: "cmd/debug/main.go", want: true},
		{path: "cmd/debugger/main.go", want: false},
		{path: "ui/apps/dashboard", want: false},
		{path: "ui/apps/dashboard/page.tsx", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := PathExcluded(tt.path, excludes); got != tt.want {
				t.Fatalf("PathExcluded(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

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

func assertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Fatalf("expected output to contain %q:\n%s", needle, haystack)
	}
}
