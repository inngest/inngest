package main

import (
	"os"
	"strings"
)

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
