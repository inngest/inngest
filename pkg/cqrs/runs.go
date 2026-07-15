package cqrs

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/oklog/ulid/v2"
)

// a shared contract for all by run list consumers
// across oss and cloud, rest api  and graphql
type RunListItem struct {
	AccountID    uuid.UUID       `json:"account_id"`
	WorkspaceID  uuid.UUID       `json:"workspace_id"`
	AppID        uuid.UUID       `json:"app_id"`
	FunctionID   uuid.UUID       `json:"function_id"`
	TraceID      string          `json:"trace_id"`
	RunID        string          `json:"run_id"`
	QueuedAt     time.Time       `json:"queued_at"`
	StartedAt    time.Time       `json:"started_at,omitempty"`
	EndedAt      time.Time       `json:"ended_at,omitempty"`
	Duration     time.Duration   `json:"duration"`
	SourceID     string          `json:"source_id,omitempty"`
	TriggerIDs   []string        `json:"trigger_ids"`
	Triggers     [][]byte        `json:"triggers"`
	Output       []byte          `json:"output,omitempty"`
	Status       enums.RunStatus `json:"status"`
	IsDeferred   bool            `json:"is_deferred"`
	IsBatch      bool            `json:"is_batch"`
	IsDebounce   bool            `json:"is_debounce"`
	HasAI        bool            `json:"has_ai"`
	BatchID      *ulid.ULID      `json:"batch_id,omitempty"`
	CronSchedule *string         `json:"cron_schedule,omitempty"`
	Cursor       string          `json:"cursor"`
}

type RunReader interface {
	ListRuns(ctx context.Context, opts ListRunsOptions) ([]*RunListItem, error)
}

type ListRunsOptions struct {
	Filter        RunListFilter
	Order         []RunListOrder
	Cursor        string
	Items         uint
	IncludeOutput bool
}

type RunListTimeField = enums.TraceRunTime

const (
	RunListTimeFieldQueuedAt  = enums.TraceRunTimeQueuedAt
	RunListTimeFieldStartedAt = enums.TraceRunTimeStartedAt
	RunListTimeFieldEndedAt   = enums.TraceRunTimeEndedAt
)

type RunListOrderDirection = enums.TraceRunOrder

const (
	RunListOrderAsc  = enums.TraceRunOrderAsc
	RunListOrderDesc = enums.TraceRunOrderDesc
)

type RunListFilter struct {
	AccountID   uuid.UUID
	WorkspaceID uuid.UUID
	AppID       []uuid.UUID
	FunctionID  []uuid.UUID
	EventID     []ulid.ULID
	TimeField   RunListTimeField
	From        time.Time
	Until       time.Time
	Status      []enums.RunStatus
	CEL         string
	IsDeferred  *bool
}

type RunListOrder struct {
	Field     RunListTimeField
	Direction RunListOrderDirection
}

// RunPageCursor is the composite cursor used for stable run pagination.
type RunPageCursor struct {
	ID      string               `json:"id"`
	Cursors map[string]RunCursor `json:"c"`
}

func (c *RunPageCursor) IsEmpty() bool {
	return len(c.Cursors) == 0
}

func (c *RunPageCursor) Find(field string) *RunCursor {
	if c.IsEmpty() {
		return nil
	}

	f := strings.ToLower(field)
	if v, ok := c.Cursors[f]; ok {
		return &v
	}
	return nil
}

func (c *RunPageCursor) Add(field string) {
	f := strings.ToLower(field)
	if _, ok := c.Cursors[f]; !ok {
		c.Cursors[f] = RunCursor{Field: f}
	}
}

func (c *RunPageCursor) Encode() (string, error) {
	byt, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(byt), nil
}

func (c *RunPageCursor) Decode(val string) error {
	if c.Cursors == nil {
		c.Cursors = map[string]RunCursor{}
	}
	byt, err := base64.StdEncoding.DecodeString(val)
	if err != nil {
		return err
	}
	return json.Unmarshal(byt, c)
}

type RunCursor struct {
	Field string `json:"f"`
	Value int64  `json:"v"`
}
