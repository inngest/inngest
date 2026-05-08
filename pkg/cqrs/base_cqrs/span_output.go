package base_cqrs

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/inngest/inngest/pkg/logger"
)

// unwrapSpanOutput processes raw output bytes by extracting the wrapped
// "data" or "error" payload. waitForEvent output is left as-is because
// it is not wrapped.
func unwrapSpanOutput(ctx context.Context, raw json.RawMessage, spanID string) (data json.RawMessage, isError bool) {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil || m == nil {
		return raw, false
	}

	if isWaitForEventOutput(m) {
		return raw, false
	}

	if errData, ok := m["error"]; ok {
		marshaled, _ := json.Marshal(errData)
		return marshaled, true
	}
	if successData, ok := m["data"]; ok {
		marshaled, _ := json.Marshal(successData)
		return marshaled, false
	}

	sanitizedSpanID := strings.ReplaceAll(spanID, "\n", "")
	sanitizedSpanID = strings.ReplaceAll(sanitizedSpanID, "\r", "")
	logger.StdlibLogger(ctx).Error("span output is not keyed, assuming success", "spanID", sanitizedSpanID)

	return raw, false
}

func isWaitForEventOutput(o map[string]any) bool {
	_, name := o["name"]
	_, data := o["data"]
	_, ts := o["ts"]
	return name && data && ts
}
