package main

import (
	"regexp"
	"strings"
)

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

var (
	htmlCommentPattern    = regexp.MustCompile(`(?s)<!--.*?-->`)
	mendralSummaryPattern = regexp.MustCompile(`(?s)<!--\s*MENDRAL_SUMMARY\s*-->.*?<!--\s*/MENDRAL_SUMMARY\s*-->`)
)

func CleanMarkdownSection(value string) string {
	value = mendralSummaryPattern.ReplaceAllString(value, "")
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
