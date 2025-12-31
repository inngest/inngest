package util

import (
	"strconv"
	"strings"
)

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

// StringToInt converts a string to an integer, handling both integer and float string representations.
// This is particularly useful when dealing with data from systems like Redis/Lua that may convert
// integers to floats during JSON serialization/deserialization (e.g., "5" or "5.0" both return 5).
//
// Examples:
//   - StringToInt("5") returns 5, nil
//   - StringToInt("5.0") returns 5, nil
//   - StringToInt("5.7") returns 5, nil (truncated)
//   - StringToInt("abc") returns 0, error
func StringToInt[T int | int64](s string) (T, error) {

	// parse it as a 64-bit integer, first (and convert to int or int64)
	if val, err := strconv.ParseInt(s, 10, 64); err == nil {
		return T(val), nil
	}

	// Try to parse as integer first (common case)
	if val, err := strconv.Atoi(s); err == nil {
		return T(val), nil
	}

	// If parsing as int fails, try parsing as float and convert to int
	// This handles cases where numbers are stored as float strings (e.g., "5.0")
	// This is useful for Lua versions that return floats
	if floatVal, err := strconv.ParseFloat(s, 64); err == nil {
		return T(floatVal), nil
	}

	// If both fail, return error with original string for context
	return 0, strconv.ErrSyntax
}
