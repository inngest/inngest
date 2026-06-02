package util

import (
	"encoding/json"
	"fmt"
)

func EnsureJSON(v json.RawMessage) json.RawMessage {
	if !json.Valid(v) {
		// Wrap the output in quotes to make it valid JSON.
		return json.RawMessage(fmt.Sprintf("%q", v))
	}
	return v
}

// IsJSONObject reports whether it's a JSON object. This is a best effort check
// which assumes valid JSON.
func IsJSONObject(r json.RawMessage) bool {
	for _, b := range r {
		if b == ' ' || b == '\t' || b == '\r' || b == '\n' {
			continue
		}
		return b == '{'
	}
	return false
}
