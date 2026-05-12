package main

import "testing"

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

func TestParsePRBodyCombinesRepeatedReleaseAndMigrationSections(t *testing.T) {
	body := `## Release note

First release note.

## Migration note

First migration note.

## Release notes

Second release note.

## Migration notes

Second migration note.
`

	sections := ParsePRBody(body)

	releaseNote := NormalizeNote(sections["release note"])
	wantReleaseNote := "First release note.\n\nSecond release note."
	if releaseNote != wantReleaseNote {
		t.Fatalf("release note = %q, want %q", releaseNote, wantReleaseNote)
	}

	migrationNote := NormalizeNote(sections["migration note"])
	wantMigrationNote := "First migration note.\n\nSecond migration note."
	if migrationNote != wantMigrationNote {
		t.Fatalf("migration note = %q, want %q", migrationNote, wantMigrationNote)
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

func TestParsePRBodyDropsMendralSummary(t *testing.T) {
	body := `## Migration note

None.

<!-- MENDRAL_SUMMARY -->
---

> [!NOTE]
> AI-generated summary.
>
> <sup>Written by [Mendral](https://mendral.com).</sup>
<!-- /MENDRAL_SUMMARY -->
`

	sections := ParsePRBody(body)
	if got := NormalizeNote(sections["migration note"]); got != "" {
		t.Fatalf("migration note = %q, want empty", got)
	}
}
