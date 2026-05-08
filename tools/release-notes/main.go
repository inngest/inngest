package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type PullRequest struct {
	Number        int      `json:"number"`
	Title         string   `json:"title"`
	URL           string   `json:"url,omitempty"`
	Body          string   `json:"body,omitempty"`
	Files         []string `json:"files,omitempty"`
	Labels        []string `json:"labels,omitempty"`
	MergeCommit   string   `json:"merge_commit,omitempty"`
	Description   string   `json:"description,omitempty"`
	ReleaseNote   string   `json:"release_note,omitempty"`
	MigrationNote string   `json:"migration_note,omitempty"`
	Excluded      bool     `json:"excluded,omitempty"`
}

type NotesFile struct {
	PRs []PullRequest `json:"prs"`
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "release-notes: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: release-notes <collect|build> [args...]")
	}

	switch args[0] {
	case "collect":
		return runCollect(args[1:])
	case "build":
		return runBuild(args[1:])
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runCollect(args []string) error {
	fs := flag.NewFlagSet("collect", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	var inputPath string
	var outputPath string
	var cliffPath string
	var commitRange string
	var repo string
	fs.StringVar(&inputPath, "input", "", "read PR JSON from file instead of GitHub")
	fs.StringVar(&outputPath, "output", "", "write collected notes JSON to file")
	fs.StringVar(&cliffPath, "cliff", "cliff.toml", "git-cliff config for path exclusions")
	fs.StringVar(&commitRange, "range", "", "git commit range to inspect, e.g. v1.2.3..HEAD")
	fs.StringVar(&repo, "repo", "", "GitHub repository, e.g. owner/name")

	if err := fs.Parse(args); err != nil {
		return err
	}

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

func runBuild(args []string) error {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	var notesPath string
	var changelogPath string
	var tag string
	var releasePRBodyPath string
	var outputPath string
	fs.StringVar(&notesPath, "notes", "", "collected notes JSON file")
	fs.StringVar(&changelogPath, "changelog", "CHANGELOG.md", "changelog file")
	fs.StringVar(&tag, "tag", "", "release tag")
	fs.StringVar(&releasePRBodyPath, "release-pr-body", "", "release PR body markdown")
	fs.StringVar(&outputPath, "output", "", "write release notes markdown to file")

	if err := fs.Parse(args); err != nil {
		return err
	}
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

func ParsePRBody(body string) map[string]string {
	sections := map[string]string{}
	var current string
	var lines []string

	flush := func() {
		if current == "" {
			return
		}
		value := CleanMarkdownSection(strings.Join(lines, "\n"))
		if sections[current] != "" && value != "" {
			sections[current] = sections[current] + "\n\n" + value
		} else if value != "" {
			sections[current] = value
		}
		lines = nil
	}

	for _, line := range strings.Split(body, "\n") {
		if title, ok := parseHeading(line); ok {
			flush()
			current = title
			continue
		}
		if current != "" {
			lines = append(lines, line)
		}
	}
	flush()
	return sections
}

func parseHeading(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "#") {
		return "", false
	}

	i := 0
	for i < len(trimmed) && trimmed[i] == '#' {
		i++
	}
	if i == 0 || i >= len(trimmed) || trimmed[i] != ' ' {
		return "", false
	}

	title := strings.TrimSpace(trimmed[i:])
	title = strings.Trim(title, "# ")
	title = strings.ToLower(title)
	switch title {
	case "description", "release note", "release notes", "migration note", "migration notes":
		return strings.TrimSuffix(title, "s"), true
	default:
		return "", false
	}
}

var htmlCommentPattern = regexp.MustCompile(`(?s)<!--.*?-->`)

func CleanMarkdownSection(value string) string {
	value = htmlCommentPattern.ReplaceAllString(value, "")
	return NormalizeWhitespace(value)
}

func NormalizeNote(value string) string {
	value = NormalizeWhitespace(value)
	placeholder := strings.Trim(strings.ToLower(value), ". ")
	switch placeholder {
	case "", "none", "n/a", "na":
		return ""
	default:
		return value
	}
}

func NormalizeWhitespace(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")

	lines := strings.Split(value, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
	}
	value = strings.Join(lines, "\n")
	value = strings.TrimSpace(value)

	blankLines := regexp.MustCompile(`\n{3,}`)
	return blankLines.ReplaceAllString(value, "\n\n")
}

func ParseCliffExcludePathsFile(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseCliffExcludePaths(string(data)), nil
}

func ParseCliffExcludePaths(config string) []string {
	var paths []string
	inExcludePaths := false
	for _, line := range strings.Split(config, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "exclude_paths") && strings.Contains(trimmed, "[") {
			inExcludePaths = true
			continue
		}
		if !inExcludePaths {
			continue
		}
		if strings.HasPrefix(trimmed, "]") {
			break
		}
		if strings.HasPrefix(trimmed, "#") || trimmed == "" {
			continue
		}
		if idx := strings.Index(trimmed, "#"); idx >= 0 {
			trimmed = strings.TrimSpace(trimmed[:idx])
		}
		trimmed = strings.TrimSuffix(trimmed, ",")
		trimmed = strings.Trim(trimmed, `"`)
		if trimmed != "" {
			paths = append(paths, trimmed)
		}
	}
	return paths
}

func ShouldExcludePR(pr PullRequest, excludes []string) bool {
	if isNonReleaseTitle(pr.Title) {
		return true
	}
	if len(pr.Files) == 0 || len(excludes) == 0 {
		return false
	}
	for _, file := range pr.Files {
		if !PathExcluded(file, excludes) {
			return false
		}
	}
	return true
}

func isNonReleaseTitle(title string) bool {
	lower := strings.ToLower(strings.TrimSpace(title))
	return strings.HasPrefix(lower, "cloud:") ||
		strings.HasPrefix(lower, "cloud(") ||
		strings.HasPrefix(lower, "internal:") ||
		strings.HasPrefix(lower, "internal(") ||
		strings.HasPrefix(lower, "noop:") ||
		strings.HasPrefix(lower, "noop(")
}

func PathExcluded(path string, excludes []string) bool {
	path = strings.TrimPrefix(path, "./")
	for _, exclude := range excludes {
		exclude = strings.TrimPrefix(exclude, "./")
		if strings.HasSuffix(exclude, "/") {
			if strings.HasPrefix(path, exclude) {
				return true
			}
			continue
		}
		if path == exclude || strings.HasPrefix(path, exclude+"/") {
			return true
		}
	}
	return false
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

func ExtractMarkerBlock(body, startMarker, endMarker string) string {
	if body == "" {
		return ""
	}
	start := strings.Index(body, "<!-- "+startMarker+" -->")
	if start < 0 {
		return ""
	}
	start += len("<!-- " + startMarker + " -->")
	end := strings.Index(body[start:], "<!-- "+endMarker+" -->")
	if end < 0 {
		return ""
	}
	return CleanMarkdownSection(body[start : start+end])
}
