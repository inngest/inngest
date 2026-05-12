package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v3"
)

func prBodyCommand() *cli.Command {
	return &cli.Command{
		Name:  "pr-body",
		Usage: "Render a release PR body.",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "tag", Usage: "Release tag.", Required: true},
			&cli.StringFlag{Name: "compare-url", Usage: "GitHub compare URL since previous release."},
			&cli.StringFlag{Name: "compare-label", Usage: "Display label for the GitHub compare URL."},
			&cli.StringFlag{Name: "base", Value: "main", Usage: "Base branch."},
			&cli.StringFlag{Name: "head", Value: "release/next", Usage: "Release branch."},
			&cli.StringFlag{Name: "latest-tag", Usage: "Previous release tag."},
			&cli.StringFlag{Name: "preview", Usage: "Release notes preview markdown.", Required: true},
			&cli.StringFlag{Name: "existing-body", Usage: "Existing release PR body markdown."},
			&cli.StringFlag{Name: "output", Usage: "Write release PR body markdown to file."},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			_ = ctx
			return prBodyCommandAction(
				cmd.String("tag"),
				cmd.String("compare-url"),
				cmd.String("compare-label"),
				cmd.String("base"),
				cmd.String("head"),
				cmd.String("latest-tag"),
				cmd.String("preview"),
				cmd.String("existing-body"),
				cmd.String("output"),
			)
		},
	}
}

func prBodyCommandAction(tag, compareURL, compareLabel, base, head, latestTag, previewPath, existingBodyPath, outputPath string) error {
	if tag == "" {
		return errors.New("--tag is required")
	}
	if previewPath == "" {
		return errors.New("--preview is required")
	}

	preview, err := os.ReadFile(previewPath)
	if err != nil {
		return err
	}

	var existingBody string
	if existingBodyPath != "" {
		body, err := os.ReadFile(existingBodyPath)
		if err != nil {
			return err
		}
		existingBody = string(body)
	}

	rendered, err := RenderReleasePRBody(ReleasePRBodyInput{
		Tag:          tag,
		CompareURL:   compareURL,
		CompareLabel: compareLabel,
		Base:         base,
		Head:         head,
		LatestTag:    latestTag,
		Preview:      string(preview),
		ExistingBody: existingBody,
	})
	if err != nil {
		return err
	}

	if outputPath == "" {
		_, err = fmt.Fprint(os.Stdout, rendered)
		return err
	}
	return os.WriteFile(outputPath, []byte(rendered), 0o644)
}

type ReleasePRBodyInput struct {
	Tag          string
	CompareURL   string
	CompareLabel string
	Base         string
	Head         string
	LatestTag    string
	Preview      string
	ExistingBody string
}

func RenderReleasePRBody(input ReleasePRBodyInput) (string, error) {
	tag := strings.TrimSpace(input.Tag)
	if tag == "" {
		return "", errors.New("release tag is required")
	}

	preview := NormalizeWhitespace(input.Preview)
	if preview == "" {
		return "", errors.New("release preview is empty")
	}

	base := strings.TrimSpace(input.Base)
	if base == "" {
		base = "main"
	}
	head := strings.TrimSpace(input.Head)
	if head == "" {
		head = "release/next"
	}

	manualRelease := NormalizeNote(ExtractMarkerBlock(input.ExistingBody, "release-note:manual-start", "release-note:manual-end"))
	if manualRelease == "" {
		manualRelease = "None."
	}
	manualMigration := NormalizeNote(ExtractMarkerBlock(input.ExistingBody, "migration-note:manual-start", "migration-note:manual-end"))
	if manualMigration == "" {
		manualMigration = "None."
	}

	var b strings.Builder
	b.WriteString("<!-- auto-release-pr -->\n")
	b.WriteString("## Release\n\n")
	fmt.Fprintf(&b, "This PR prepares `%s`.\n\n", tag)
	if input.CompareURL != "" {
		compareLabel := strings.TrimSpace(input.CompareLabel)
		if compareLabel == "" && strings.TrimSpace(input.LatestTag) != "" {
			compareLabel = fmt.Sprintf("%s...%s", strings.TrimSpace(input.LatestTag), base)
		}
		if compareLabel == "" {
			compareLabel = "Compare changes"
		}
		fmt.Fprintf(&b, "- Code difference since last tag: [%s](%s)\n", compareLabel, input.CompareURL)
	}
	if input.LatestTag != "" {
		fmt.Fprintf(&b, "- Previous tag: `%s`\n", input.LatestTag)
	}
	fmt.Fprintf(&b, "- Source branch: `%s`\n", head)
	fmt.Fprintf(&b, "- Base branch: `%s`\n\n", base)

	b.WriteString("## Additional Release Notes\n")
	b.WriteString("<!-- release-note:manual-start -->\n")
	b.WriteString(manualRelease)
	b.WriteString("\n<!-- release-note:manual-end -->\n\n")

	b.WriteString("## Additional Migration Notes\n")
	b.WriteString("<!-- migration-note:manual-start -->\n")
	b.WriteString(manualMigration)
	b.WriteString("\n<!-- migration-note:manual-end -->\n\n")

	b.WriteString("## Notes Preview\n")
	b.WriteString("<!-- release-note:preview-start -->\n")
	b.WriteString(preview)
	b.WriteString("\n<!-- release-note:preview-end -->\n")

	return b.String(), nil
}
