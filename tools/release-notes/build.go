package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/urfave/cli/v3"
)

func buildCommand() *cli.Command {
	return &cli.Command{
		Name:  "build",
		Usage: "Build release notes markdown.",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "notes", Usage: "Collected notes JSON file.", Required: true},
			&cli.StringFlag{Name: "changelog", Value: "CHANGELOG.md", Usage: "Changelog file."},
			&cli.StringFlag{Name: "tag", Usage: "Release tag.", Required: true},
			&cli.StringFlag{Name: "release-pr-body", Usage: "Release PR body markdown."},
			&cli.StringFlag{Name: "output", Usage: "Write release notes markdown to file."},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			_ = ctx
			return buildNotesCommand(
				cmd.String("notes"),
				cmd.String("changelog"),
				cmd.String("tag"),
				cmd.String("release-pr-body"),
				cmd.String("output"),
			)
		},
	}
}

func buildNotesCommand(notesPath, changelogPath, tag, releasePRBodyPath, outputPath string) error {
	if notesPath == "" {
		return errors.New("--notes is required")
	}
	if tag == "" {
		return errors.New("--tag is required")
	}

	notes, err := readNotesFile(notesPath)
	if err != nil {
		return err
	}

	changelog, err := os.ReadFile(changelogPath)
	if err != nil {
		return err
	}
	changelogSection, err := ExtractChangelogSection(string(changelog), tag)
	if err != nil {
		return err
	}

	var releasePRBody string
	if releasePRBodyPath != "" {
		body, err := os.ReadFile(releasePRBodyPath)
		if err != nil {
			return err
		}
		releasePRBody = string(body)
	}

	rendered, err := BuildReleaseNotes(notes, changelogSection, releasePRBody)
	if err != nil {
		return err
	}

	if outputPath == "" {
		_, err = fmt.Fprint(os.Stdout, rendered)
		return err
	}
	return os.WriteFile(outputPath, []byte(rendered), 0o644)
}

func readNotesFile(path string) (NotesFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return NotesFile{}, err
	}

	var notes NotesFile
	if err := json.Unmarshal(data, &notes); err != nil {
		return NotesFile{}, err
	}
	return notes, nil
}

func ExtractChangelogSection(changelog, tag string) (string, error) {
	tag = strings.TrimPrefix(strings.TrimSpace(tag), "v")
	versionPatterns := []string{
		fmt.Sprintf("## [v%s]", tag),
		fmt.Sprintf("## [%s]", tag),
		fmt.Sprintf("## v%s", tag),
		fmt.Sprintf("## %s", tag),
	}

	lines := strings.Split(changelog, "\n")
	start := -1
	for i, line := range lines {
		for _, pattern := range versionPatterns {
			if strings.HasPrefix(line, pattern) {
				start = i + 1
				break
			}
		}
		if start >= 0 {
			break
		}
	}
	if start < 0 {
		return "", fmt.Errorf("changelog section for v%s not found", tag)
	}

	end := len(lines)
	for i := start; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "## ") {
			end = i
			break
		}
	}
	return NormalizeWhitespace(strings.Join(lines[start:end], "\n")), nil
}

func BuildReleaseNotes(notes NotesFile, changelogSection, releasePRBody string) (string, error) {
	changelogSection = NormalizeWhitespace(changelogSection)
	if changelogSection == "" {
		return "", errors.New("changelog section is empty")
	}

	releaseNotes := collectNotes(notes.PRs, func(pr PullRequest) string { return pr.ReleaseNote })
	migrationNotes := collectNotes(notes.PRs, func(pr PullRequest) string { return pr.MigrationNote })

	manualRelease := NormalizeNote(ExtractMarkerBlock(releasePRBody, "release-note:manual-start", "release-note:manual-end"))
	manualMigration := NormalizeNote(ExtractMarkerBlock(releasePRBody, "migration-note:manual-start", "migration-note:manual-end"))

	var b strings.Builder
	if manualRelease != "" || len(releaseNotes) > 0 {
		b.WriteString("## Release Notes\n\n")
		if manualRelease != "" {
			b.WriteString(manualRelease)
			b.WriteString("\n\n")
		}
		writePRNotes(&b, releaseNotes)
	}

	if manualMigration != "" || len(migrationNotes) > 0 {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString("## Migration Notes\n\n")
		if manualMigration != "" {
			b.WriteString(manualMigration)
			b.WriteString("\n\n")
		}
		writePRNotes(&b, migrationNotes)
	}

	if b.Len() > 0 {
		b.WriteString("\n")
	}
	b.WriteString("## Changelog\n\n")
	b.WriteString(changelogSection)
	b.WriteString("\n")

	return b.String(), nil
}

type prNote struct {
	Number int
	Title  string
	URL    string
	Note   string
}

func collectNotes(prs []PullRequest, pick func(PullRequest) string) []prNote {
	out := make([]prNote, 0, len(prs))
	for _, pr := range prs {
		if pr.Excluded {
			continue
		}
		note := NormalizeNote(pick(pr))
		if note == "" {
			continue
		}
		out = append(out, prNote{
			Number: pr.Number,
			Title:  pr.Title,
			URL:    pr.URL,
			Note:   note,
		})
	}

	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Number < out[j].Number
	})
	return out
}

func writePRNotes(b *strings.Builder, notes []prNote) {
	for i, note := range notes {
		if i > 0 {
			b.WriteString("\n")
		}
		title := note.Title
		if title == "" {
			title = fmt.Sprintf("PR #%d", note.Number)
		}
		if note.URL != "" && note.Number > 0 {
			fmt.Fprintf(b, "- [#%d](%s) %s\n\n", note.Number, note.URL, title)
		} else if note.Number > 0 {
			fmt.Fprintf(b, "- #%d %s\n\n", note.Number, title)
		} else {
			fmt.Fprintf(b, "- %s\n\n", title)
		}
		for _, line := range strings.Split(note.Note, "\n") {
			if line == "" {
				b.WriteString("\n")
			} else {
				b.WriteString("  ")
				b.WriteString(line)
				b.WriteString("\n")
			}
		}
	}
}
