package util

import "strings"

func StrPtr(s string) *string {
	return &s
}

func LogReplace(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\n", ""), "\r", "")
}
