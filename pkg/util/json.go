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
