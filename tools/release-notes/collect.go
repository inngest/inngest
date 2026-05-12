package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/urfave/cli/v3"
)

func collectCommand() *cli.Command {
	return &cli.Command{
		Name:  "collect",
		Usage: "Collect release note metadata from PRs.",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "input", Usage: "Read PR JSON from file instead of GitHub."},
			&cli.StringFlag{Name: "output", Usage: "Write collected notes JSON to file."},
			&cli.StringFlag{Name: "cliff", Value: "cliff.toml", Usage: "git-cliff config for path exclusions."},
			&cli.StringFlag{Name: "range", Usage: "Git commit range to inspect, e.g. v1.2.3..HEAD."},
			&cli.StringFlag{Name: "repo", Usage: "GitHub repository, e.g. owner/name."},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			_ = ctx
			return collectNotesCommand(cmd.String("input"), cmd.String("output"), cmd.String("cliff"), cmd.String("range"), cmd.String("repo"))
		},
	}
}

func collectNotesCommand(inputPath, outputPath, cliffPath, commitRange, repo string) error {
	excludes, err := ParseCliffExcludePathsFile(cliffPath)
	if err != nil {
		return err
	}

	var prs []PullRequest
	if inputPath != "" {
		prs, err = readPRInput(inputPath)
	} else {
		prs, err = collectFromGitHub(commitRange, repo)
	}
	if err != nil {
		return err
	}

	for i := range prs {
		sections := ParsePRBody(prs[i].Body)
		prs[i].Description = sections["description"]
		prs[i].ReleaseNote = NormalizeNote(sections["release note"])
		prs[i].MigrationNote = NormalizeNote(sections["migration note"])
		prs[i].Excluded = ShouldExcludePR(prs[i], excludes)
	}

	out, err := json.MarshalIndent(NotesFile{PRs: prs}, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')

	if outputPath == "" {
		_, err = os.Stdout.Write(out)
		return err
	}
	return os.WriteFile(outputPath, out, 0o644)
}

func readPRInput(path string) ([]PullRequest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var wrapped NotesFile
	if err := json.Unmarshal(data, &wrapped); err == nil && wrapped.PRs != nil {
		return wrapped.PRs, nil
	}

	var prs []PullRequest
	if err := json.Unmarshal(data, &prs); err != nil {
		return nil, err
	}
	return prs, nil
}

func collectFromGitHub(commitRange, repo string) ([]PullRequest, error) {
	if commitRange == "" {
		return nil, errors.New("--range is required when --input is not used")
	}

	log, err := commandOutput("git", "log", "--first-parent", "--reverse", "--format=%s", commitRange)
	if err != nil {
		return nil, err
	}

	numbers := ExtractPRNumbers(log)
	prs := make([]PullRequest, 0, len(numbers))
	for _, number := range numbers {
		args := []string{"pr", "view", strconv.Itoa(number), "--json", "number,title,url,body,files,labels,mergeCommit"}
		if repo != "" {
			args = append(args, "--repo", repo)
		}

		out, err := commandOutput("gh", args...)
		if err != nil {
			return nil, err
		}

		pr, err := DecodeGHPR([]byte(out))
		if err != nil {
			return nil, fmt.Errorf("decode PR #%d: %w", number, err)
		}
		prs = append(prs, pr)
	}
	return prs, nil
}

func commandOutput(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("%s %s: %s", name, strings.Join(args, " "), msg)
	}
	return string(out), nil
}

type ghPR struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	URL    string `json:"url"`
	Body   string `json:"body"`
	Files  []struct {
		Path string `json:"path"`
	} `json:"files"`
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels"`
	MergeCommit struct {
		OID string `json:"oid"`
	} `json:"mergeCommit"`
}

func DecodeGHPR(data []byte) (PullRequest, error) {
	var raw ghPR
	if err := json.Unmarshal(data, &raw); err != nil {
		return PullRequest{}, err
	}

	pr := PullRequest{
		Number:      raw.Number,
		Title:       raw.Title,
		URL:         raw.URL,
		Body:        raw.Body,
		MergeCommit: raw.MergeCommit.OID,
	}
	for _, file := range raw.Files {
		if file.Path != "" {
			pr.Files = append(pr.Files, file.Path)
		}
	}
	for _, label := range raw.Labels {
		if label.Name != "" {
			pr.Labels = append(pr.Labels, label.Name)
		}
	}
	return pr, nil
}

var prNumberPatterns = []*regexp.Regexp{
	regexp.MustCompile(`\(#([0-9]+)\)`),
	regexp.MustCompile(`Merge pull request #([0-9]+)`),
}

func ExtractPRNumbers(log string) []int {
	seen := map[int]bool{}
	var numbers []int
	for _, line := range strings.Split(log, "\n") {
		for _, pattern := range prNumberPatterns {
			match := pattern.FindStringSubmatch(line)
			if len(match) != 2 {
				continue
			}
			number, err := strconv.Atoi(match[1])
			if err == nil && !seen[number] {
				seen[number] = true
				numbers = append(numbers, number)
			}
			break
		}
	}
	return numbers
}
