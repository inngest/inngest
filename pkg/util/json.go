package util

import (
	"encoding/json"
)

func EnsureJSON(v json.RawMessage) json.RawMessage {
	if !json.Valid(v) {
		// Encode the output as a JSON string to make it valid JSON. Go's %q
		// can't be used here: it emits escapes such as \x1b that JSON rejects.
		byt, _ := json.Marshal(string(v))
		return byt
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
