package state

import (
	"time"

	"github.com/inngest/inngest/pkg/enums"
)

type History struct {
	Type       enums.HistoryType `json:"type"`
	Identifier Identifier        `json:"id"`
	CreatedAt  time.Time         `json:"createdAt"`
	Data       any               `json:"data"`
}

type HistoryFunctionCancelled struct {
	Type enums.CancellationType `json:"type"`
	Data any                    `json:"data"`
}

// TODO Add tracking of the parent steps so that we can create a visual DAG
type HistoryStep struct {
	Name    string `json:"name"`
	Attempt int    `json:"attempt"`
	Data    any    `json:"data"`
}

type HistoryStepWaitingForEvent struct {
	Name       string    `json:"name"`
	EventName  string    `json:"eventName"`
	Expression string    `json:"expression"`
	ExpiryTime time.Time `json:"expiry"`
}

type HistoryStepSleepingUntil struct {
	Name   string    `json:"name"`
	WakeAt time.Time `json:"wakeAt"`
}
