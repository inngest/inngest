package events

import (
	"encoding/json"
	"time"
)

// CloudWatchEvent is the outer structure of an event sent via EventBridge serverless service.
type CloudWatchEvent struct {
	Version    string          `json:"version"`
	ID         string          `json:"id"`
	DetailType string          `json:"detail-type"`
	Source     string          `json:"source"`
	AccountID  string          `json:"account"`
	Time       time.Time       `json:"time"`
	Region     string          `json:"region"`
	Resources  []string        `json:"resources"`
	Detail     json.RawMessage `json:"detail"`
}

type EventBridgeEvent = CloudWatchEvent
