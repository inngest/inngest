package util

import "strings"

func StrPtr(s string) *string {
	return &s
}

var logSanitizer = strings.NewReplacer(
	"\r", "",
	"\n", "",
)

// SanitizeLogField removes carriage returns and newlines from log fields to prevent log forging.
func SanitizeLogField(s string) string {
	return logSanitizer.Replace(s)
}
