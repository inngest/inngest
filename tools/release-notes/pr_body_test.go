package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderReleasePRBodyPreservesManualBlocks(t *testing.T) {
	existing := `<!-- auto-release-pr -->
## Release

Old generated text.

## Additional Release Notes
<!-- release-note:manual-start -->
Keep this release context.
<!-- release-note:manual-end -->

## Additional Migration Notes
<!-- migration-note:manual-start -->
Keep this migration context.
<!-- migration-note:manual-end -->

## Notes Preview
<!-- release-note:preview-start -->
Old preview.
<!-- release-note:preview-end -->
`

	got, err := RenderReleasePRBody(ReleasePRBodyInput{
		Tag:          "v1.2.3",
		Ahead:        "12",
		Base:         "main",
		Head:         "release/next",
		LatestTag:    "v1.2.2",
		Preview:      "## Release Notes\n\nNew preview.",
		ExistingBody: existing,
	})
	if err != nil {
		t.Fatal(err)
	}

	assertContains(t, got, "This PR prepares `v1.2.3`.")
	assertContains(t, got, "- Commits since last tag: 12")
	assertContains(t, got, "- Previous tag: `v1.2.2`")
	assertContains(t, got, "Keep this release context.")
	assertContains(t, got, "Keep this migration context.")
	assertContains(t, got, "New preview.")

	if strings.Contains(got, "Old preview.") {
		t.Fatalf("old preview was preserved:\n%s", got)
	}
}

func TestRenderReleasePRBodyDefaultsManualBlocks(t *testing.T) {
	got, err := RenderReleasePRBody(ReleasePRBodyInput{
		Tag:     "v1.2.3",
		Preview: "## Changelog\n\n- Entry",
	})
	if err != nil {
		t.Fatal(err)
	}

	assertContains(t, got, "<!-- release-note:manual-start -->\nNone.\n<!-- release-note:manual-end -->")
	assertContains(t, got, "<!-- migration-note:manual-start -->\nNone.\n<!-- migration-note:manual-end -->")
	assertContains(t, got, "- Source branch: `release/next`")
	assertContains(t, got, "- Base branch: `main`")
}

func TestPRBodyCommandWritesOutput(t *testing.T) {
	dir := t.TempDir()
	previewPath := filepath.Join(dir, "preview.md")
	existingPath := filepath.Join(dir, "existing.md")
	outputPath := filepath.Join(dir, "body.md")

	if err := os.WriteFile(previewPath, []byte("## Changelog\n\n- Entry\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(existingPath, []byte(`<!-- release-note:manual-start -->
Manual context.
<!-- release-note:manual-end -->`), 0o644); err != nil {
		t.Fatal(err)
	}

	err := run([]string{
		"pr-body",
		"--tag", "v1.2.3",
		"--ahead", "4",
		"--preview", previewPath,
		"--existing-body", existingPath,
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
	assertContains(t, got, "Manual context.")
	assertContains(t, got, "## Changelog")
}
