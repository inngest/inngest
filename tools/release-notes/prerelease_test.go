package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParsePrereleaseComment(t *testing.T) {
	tests := []struct {
		name string
		body string
		want PrereleaseComment
	}{
		{
			name: "not a command",
			body: "Looks good",
			want: PrereleaseComment{Matched: false},
		},
		{
			name: "channel only",
			body: "/prerelease beta",
			want: PrereleaseComment{Matched: true, Channel: "beta"},
		},
		{
			name: "explicit version",
			body: "/prerelease alpha v1.18.0-alpha.1",
			want: PrereleaseComment{Matched: true, Channel: "alpha", Version: "v1.18.0-alpha.1"},
		},
		{
			name: "dry run",
			body: "/prerelease rc --dry-run",
			want: PrereleaseComment{Matched: true, Channel: "rc", DryRun: true},
		},
		{
			name: "leading blank lines",
			body: "\n\n/prerelease beta --dry-run\nextra text",
			want: PrereleaseComment{Matched: true, Channel: "beta", DryRun: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePrereleaseComment(tt.body)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("ParsePrereleaseComment() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestParsePrereleaseCommentErrors(t *testing.T) {
	tests := []string{
		"/prerelease",
		"/prerelease stable",
		"/prerelease beta v1.18.0-alpha.1",
		"/prerelease beta nope",
		"/prerelease beta --force",
		"/prerelease beta v1.18.0-beta.1 extra",
	}

	for _, body := range tests {
		t.Run(body, func(t *testing.T) {
			if _, err := ParsePrereleaseComment(body); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestNextPrereleaseVersion(t *testing.T) {
	got, err := NextPrereleaseVersion("1.18.0", "beta", []string{
		"v1.18.0-beta.1",
		"v1.18.0-beta.3",
		"v1.18.0-alpha.9",
		"v1.17.0-beta.7",
		"not-a-version",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != "v1.18.0-beta.4" {
		t.Fatalf("version = %q", got)
	}
}

func TestPrereleaseCommandWritesOutputs(t *testing.T) {
	dir := t.TempDir()
	commentPath := filepath.Join(dir, "comment.md")
	outputPath := filepath.Join(dir, "github-output")

	if err := os.WriteFile(commentPath, []byte("/prerelease beta v1.18.0-beta.2 --dry-run\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := run([]string{
		"prerelease-command",
		"--comment-file", commentPath,
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
	assertContains(t, got, "channel=beta")
	assertContains(t, got, "dry_run=true")
	assertContains(t, got, "matched=true")
	assertContains(t, got, "version=v1.18.0-beta.2")
}

func TestPrereleaseVersionCommandWritesOutput(t *testing.T) {
	dir := t.TempDir()
	tagsPath := filepath.Join(dir, "tags")
	outputPath := filepath.Join(dir, "github-output")

	if err := os.WriteFile(tagsPath, []byte("v1.18.0-rc.1\nv1.18.0-rc.2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := run([]string{
		"prerelease-version",
		"--channel", "rc",
		"--stable-version", "v1.18.0",
		"--existing-tags-file", tagsPath,
		"--output", outputPath,
	})
	if err != nil {
		t.Fatal(err)
	}

	output, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	assertContains(t, string(output), "version=v1.18.0-rc.3")
}
