package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/urfave/cli/v3"
)

type PrereleaseComment struct {
	Matched bool   `json:"matched"`
	Channel string `json:"channel,omitempty"`
	Version string `json:"version,omitempty"`
	DryRun  bool   `json:"dry_run,omitempty"`
}

func prereleaseCommandCommand() *cli.Command {
	return &cli.Command{
		Name:  "prerelease-command",
		Usage: "Parse a PR prerelease comment command.",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "comment-file", Usage: "File containing the PR comment body.", Required: true},
			&cli.StringFlag{Name: "output", Usage: "Write GitHub Actions output values to this file."},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			_ = ctx
			body, err := os.ReadFile(cmd.String("comment-file"))
			if err != nil {
				return err
			}

			parsed, err := ParsePrereleaseComment(string(body))
			if err != nil {
				return err
			}

			if output := cmd.String("output"); output != "" {
				return WriteGitHubOutputs(output, map[string]string{
					"matched": strconv.FormatBool(parsed.Matched),
					"channel": parsed.Channel,
					"version": parsed.Version,
					"dry_run": strconv.FormatBool(parsed.DryRun),
				})
			}

			encoded, err := json.MarshalIndent(parsed, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(encoded))
			return nil
		},
	}
}

func prereleaseVersionCommand() *cli.Command {
	return &cli.Command{
		Name:  "prerelease-version",
		Usage: "Select the next prerelease version for a channel.",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "channel", Usage: "Prerelease channel: alpha, beta, or rc.", Required: true},
			&cli.StringFlag{Name: "stable-version", Usage: "Stable SemVer base, for example v1.18.0.", Required: true},
			&cli.StringFlag{Name: "existing-tags-file", Usage: "File containing existing prerelease tags, one per line."},
			&cli.StringFlag{Name: "output", Usage: "Write GitHub Actions output values to this file."},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			_ = ctx
			var existing []string
			if path := cmd.String("existing-tags-file"); path != "" {
				data, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				existing = strings.Fields(string(data))
			}

			version, err := NextPrereleaseVersion(cmd.String("stable-version"), cmd.String("channel"), existing)
			if err != nil {
				return err
			}

			if output := cmd.String("output"); output != "" {
				return WriteGitHubOutputs(output, map[string]string{"version": version})
			}

			fmt.Println(version)
			return nil
		},
	}
}

func ParsePrereleaseComment(body string) (PrereleaseComment, error) {
	line := firstNonEmptyLine(body)
	if line == "" || !strings.HasPrefix(line, "/prerelease") {
		return PrereleaseComment{Matched: false}, nil
	}

	fields := strings.Fields(line)
	if len(fields) < 2 || fields[0] != "/prerelease" {
		return PrereleaseComment{}, fmt.Errorf("invalid prerelease command: %q", line)
	}

	parsed := PrereleaseComment{
		Matched: true,
		Channel: fields[1],
	}
	if !ValidPrereleaseChannel(parsed.Channel) {
		return PrereleaseComment{}, fmt.Errorf("invalid prerelease channel %q", parsed.Channel)
	}

	for _, field := range fields[2:] {
		switch {
		case field == "--dry-run":
			parsed.DryRun = true
		case strings.HasPrefix(field, "-"):
			return PrereleaseComment{}, fmt.Errorf("unknown prerelease option %q", field)
		case parsed.Version == "":
			if err := ValidatePrereleaseVersion(field, parsed.Channel); err != nil {
				return PrereleaseComment{}, err
			}
			parsed.Version = field
		default:
			return PrereleaseComment{}, fmt.Errorf("unexpected prerelease argument %q", field)
		}
	}

	return parsed, nil
}

func firstNonEmptyLine(body string) string {
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}

func ValidPrereleaseChannel(channel string) bool {
	switch channel {
	case "alpha", "beta", "rc":
		return true
	default:
		return false
	}
}

var prereleaseVersionPattern = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+-(alpha|beta|rc)\.[0-9]+$`)

func ValidatePrereleaseVersion(version, channel string) error {
	if !prereleaseVersionPattern.MatchString(version) {
		return fmt.Errorf("invalid prerelease version %q", version)
	}
	if !strings.Contains(version, "-"+channel+".") {
		return fmt.Errorf("version %q does not match channel %q", version, channel)
	}
	return nil
}

var stableVersionPattern = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+$`)

func NextPrereleaseVersion(stableVersion, channel string, existingTags []string) (string, error) {
	stableVersion = NormalizeStableVersion(stableVersion)
	if !stableVersionPattern.MatchString(stableVersion) {
		return "", fmt.Errorf("invalid stable version %q", stableVersion)
	}
	if !ValidPrereleaseChannel(channel) {
		return "", fmt.Errorf("invalid prerelease channel %q", channel)
	}

	prefix := stableVersion + "-" + channel + "."
	next := 1
	for _, tag := range existingTags {
		if !strings.HasPrefix(tag, prefix) {
			continue
		}
		suffix := strings.TrimPrefix(tag, prefix)
		index, err := strconv.Atoi(suffix)
		if err != nil {
			continue
		}
		if index >= next {
			next = index + 1
		}
	}

	return fmt.Sprintf("%s%d", prefix, next), nil
}

func NormalizeStableVersion(version string) string {
	version = strings.TrimSpace(version)
	if version == "" {
		return ""
	}
	if strings.HasPrefix(version, "v") {
		return version
	}
	return "v" + version
}

func WriteGitHubOutputs(path string, values map[string]string) error {
	if path == "" {
		return errors.New("output path is required")
	}

	var b strings.Builder
	for _, key := range sortedKeys(values) {
		fmt.Fprintf(&b, "%s=%s\n", key, values[key])
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}
