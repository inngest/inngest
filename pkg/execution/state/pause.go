package state

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/oklog/ulid/v2"
)

var tsSuffix = regexp.MustCompile(`\s*&&\s*\(\s*async.ts\s+==\s*null\s*\|\|\s*async.ts\s*>\s*\d*\)\s*$`)

var ErrConsumePauseKeyMissing = fmt.Errorf("no idempotency key provided for consuming pauses")

// PauseMutater manages creating, leasing, and consuming pauses from a backend implementation.
type PauseMutater interface {
	// SavePause indicates that the traversal of an edge is paused until some future time.
	//
	// This returns the number of pauses in the current pause.Index.
	SavePause(ctx context.Context, p Pause) (int64, error)

	// LeasePause allows us to lease the pause until the next step is enqueued, at which point
	// we can 'consume' the pause to remove it.
	//
	// This prevents a failure mode in which we consume the pause but enqueueing the next
	// action fails (eg. due to power loss).
	//
	// If the given pause has been leased within LeasePauseDuration, this should return an
	// ErrPauseLeased error.
	//
	// See https://github.com/inngest/inngest/issues/123 for more info
	LeasePause(ctx context.Context, id uuid.UUID) error

	// ConsumePause consumes a pause by its ID such that it can't be used again and
	// will not be returned from any query.
	//
	// Any data passed when consuming a pause will be stored within function run state
	// for future reference using the pause's DataKey.
	ConsumePause(ctx context.Context, p Pause, opts ConsumePauseOpts) (ConsumePauseResult, func() error, error)

	// DeletePause permanently deletes a pause.
	DeletePause(ctx context.Context, p Pause, opts ...DeletePauseOpt) error

	// DeletePauseByID removes a pause by its ID.
	DeletePauseByID(ctx context.Context, pauseID uuid.UUID, workspaceID uuid.UUID) error

	// DeleteRunPauseSet deletes the set tracking pauses for a run
	DeleteRunPauseSet(ctx context.Context, runID ulid.ULID) error
}

// PauseGetter allows a runner to return all existing pauses by event or by outgoing ID.  This
// is required to fetch pauses to automatically continue workflows.
type PauseGetter interface {
	// PausesByEvent returns all pauses for a given event, in a given workspace.
	PausesByEvent(ctx context.Context, workspaceID uuid.UUID, eventName string) (PauseIterator, error)

	// PauseLen returns the number of pauses for a given workspace ID, eventName combo in
	// the conneted datastore.
	PauseLen(ctx context.Context, workspaceID uuid.UUID, eventName string) (int64, error)

	PausesByEventSince(ctx context.Context, workspaceID uuid.UUID, event string, since time.Time) (PauseIterator, error)

	// PausesByEventSinceWithCreatedAt returns all pauses for a given event within a workspace since a given time,
	// with createdAt timestamps populated from Redis sorted set scores.
	PausesByEventSinceWithCreatedAt(ctx context.Context, workspaceID uuid.UUID, event string, since time.Time, limit int64) (PauseIterator, error)

	// EventHasPauses returns whether the event has pauses stored.
	EventHasPauses(ctx context.Context, workspaceID uuid.UUID, eventName string) (bool, error)

	// PauseByID returns a given pause by pause ID.  This must return expired pauses
	// that have not yet been consumed in order to properly handle timeouts.
	//
	// This should not return consumed pauses.
	PauseByID(ctx context.Context, pauseID uuid.UUID) (*Pause, error)

	// PauseByInvokeCorrelationID returns a given pause by the correlation ID.
	// This must return expired invoke pauses that have not yet been consumed in order to properly handle timeouts.
	//
	// This should not return consumed pauses.
	PauseByInvokeCorrelationID(ctx context.Context, wsID uuid.UUID, correlationID string) (*Pause, error)

	// PauseBySignalCorrelationID returns a given pause by the correlation ID.
	PauseBySignalID(ctx context.Context, wsID uuid.UUID, signalID string) (*Pause, error)

	// PauseCreatedAt returns the timestamp a pause was created, using the given
	// workspace <> event Index.
	PauseCreatedAt(ctx context.Context, workspaceID uuid.UUID, event string, pauseID uuid.UUID) (time.Time, error)

	// GetRunPauseIDs returns all pause IDs for a given run
	GetRunPauseIDs(ctx context.Context, runID ulid.ULID) ([]string, error)
}

// ConsumePauseOpts are the options to be passed in for consuming a pause
type ConsumePauseOpts struct {
	IdempotencyKey string
	Data           any
}

type ConsumePauseResult struct {
	// DidConsume indicates whether the pause was consumed.
	DidConsume bool

	// HasPendingSteps indicates whether the run still has pending steps.
	HasPendingSteps bool
}

// BlockIndex contains the block ID and event name for pause block indexing
type BlockIndex struct {
	BlockID   string `json:"b"`
	EventName string `json:"e"`
}

// DeletePauseOpts are the options to be passed in for deleting a pause
type DeletePauseOpts struct {
	// WriteBlockIndex is the block information to create a block index so the pause can still be
	// retrieved by its ID after deletion. Empty struct means no block index.
	WriteBlockIndex BlockIndex
}

type DeletePauseOpt func(*DeletePauseOpts)

func WithWriteBlockIndex(blockID string, eventName string) DeletePauseOpt {
	return func(opts *DeletePauseOpts) {
		opts.WriteBlockIndex = BlockIndex{
			BlockID:   blockID,
			EventName: eventName,
		}
	}
}

// PauseIterator allows the runner to iterate over all pauses returned by a PauseGetter.  This
// ensures that, at scale, all pauses do not need to be loaded into memory.
type PauseIterator interface {
	// Count returns the count of the pause iteration at the time of querying.
	//
	// Due to concurrent processing, the total number of iterated fields may not match
	// this count;  the count is a snapshot in time.
	Count() int

	// Next advances the iterator, returning an erorr or context.Canceled if the iteration
	// is complete.
	//
	// Next should be called prior to any call to the iterator's Val method, after
	// the iterator has been created.
	//
	// The order of the iterator is unspecified.
	Next(ctx context.Context) bool

	// Error returns the error returned during iteration, if any.  Use this to check
	// for errors during iteration when Next() returns false.
	Error() error

	// Val returns the current Pause from the iterator.
	Val(context.Context) *Pause

	// Index shows how far the iterator has progressed
	Index() int64
}

// PauseManager manages mutating and fetching pauses from a backend implementation.
type PauseManager interface {
	PauseMutater
	PauseGetter
}

// PauseIdentifier is a minimal identifier for a pause.  This exists and is used instead of
// eg. state.ID or state.Identifier for historical reasons:  pauses were added before state.ID
// existed, and this implements all things backcompat with the bare minimum fields needed for
// pauses and state to work.
type PauseIdentifier struct {
	// RunID is the ID of the run.
	RunID ulid.ULID `json:"runID"`
	// FunctionID tracks the internal ID of the function, and is used when saving
	// step responses.
	FunctionID uuid.UUID `json:"wID"`
	// AccountID represents the account ID for this run
	AccountID uuid.UUID `json:"aID"`
	// NOTE:
	// - Workspace ID is in the pause.
	// - App ID is not necessary to load fn state in the identifier.
}

// Pause allows steps of a function to be paused until some condition in the future.
//
// It pauses a specific workflow run via an Identifier, at a specific step in
// the function as specified by Target.
type Pause struct {
	// ID is a pause ID.  This should be a V7 UUID (incl. timestamp).
	ID uuid.UUID `json:"id"`
	// WorkspaceID scopes the pause to a specific workspace.
	WorkspaceID uuid.UUID `json:"wsID"`
	// Identifier is the specific workflow run to resume.  This is required.
	// This includes the minimum number of fields required to reload function
	// state.
	Identifier PauseIdentifier `json:"identifier"`
	// Outgoing is the parent step for the pause.
	Outgoing string `json:"outgoing"`
	// Incoming is the step to run after the pause completes.
	Incoming string `json:"incoming"`
	// StepName is the readable step name of the step to save within history.
	StepName string `json:"stepName"`
	// Opcode is the opcode of the step to save within history.  This is needed because
	// a pause can belong to `WaitForEvent` or `Invoke`;  it's used in both methods.
	Opcode *string `json:"opcode,omitempty"`
	// Expires is a time at which the pause can no longer be consumed.  This
	// gives each pause of a function a TTL.  This is required.
	//
	// NOTE: the pause should remain within the backing state store for
	// some period after the expiry time for checking timeout branches:
	//
	// If this pause has its OnTimeout flag set to true, we only traverse
	// the edge if the event *has not* been received.  In order to check
	// this, we enqueue a job that executes on the pause timeout:  if the
	// pause has not yet been consumed we can safely assume the event was
	// not received.  Therefore, we must be able to load the pause for some
	// time after timeout.
	Expires Time `json:"expires"`
	// Event is an optional event that can resume the pause automatically,
	// often paired with an expression.
	Event *string `json:"event,omitempty"`
	// Expression is an optional expression that must match for the pause
	// to be resumed.
	Expression *string `json:"expression,omitempty"`
	// InvokeCorrelationID is the correlation ID for the invoke pause.
	InvokeCorrelationID *string `json:"icID,omitempty"`
	// InvokeTargetFnID is the target function ID for the invoke pause.
	// This is used to be able to accurately reconstruct the entire invocation
	// span.
	InvokeTargetFnID *string `json:"itFnID,omitempty"`
	// SignalID is the ID of the signal that is responsible for this pause.
	SignalID *string `json:"signalID,omitempty"`
	// ReplaceSignalOnConflict indicates whether we should supersede the
	// signal if a wait already exists for the signal ID.
	ReplaceSignalOnConflict bool `json:"-"`
	// OnTimeout indicates that this incoming edge should only be ran
	// when the pause times out, if set to true.
	OnTimeout bool `json:"onTimeout,omitempty,omitzero"`
	// DataKey is the name of the step to use when adding data to the function
	// run's state after consuming this step.
	//
	// This allows us to create arbitrary "step" names for storing async event
	// data from matching events in async edges, eg. `waitForEvent`.
	//
	// If DataKey is empty and data is provided when consuming a pause, no
	// data will be saved in the function state.
	DataKey string `json:"dataKey,omitempty,omitzero"`
	// Cancellation indicates whether this pause exists as a cancellation
	// clause for a function.
	//
	// If so, when the matching pause is returned after processing an event
	// the function's status is set to cancelled, preventing any future work.
	Cancel bool `json:"cancel,omitempty,omitzero"`
	// MaxAttempts is the maximum number of attempts we can retry.  This is
	// included in the pause to allow the executor to set the correct maximum
	// number of retries when enqueuing next steps.
	MaxAttempts *int `json:"maxAtts,omitempty"`
	// GroupID stores the group ID for this step and history, allowing us to correlate
	// event receives with other history items.
	GroupID string `json:"groupID"`
	// TriggeringEventID is the event that triggered the original run.  This allows us
	// to exclude the original event ID when considering triggers.
	TriggeringEventID *string `json:"tID,omitempty"`
	// Metadata is additional metadata that should be stored with the pause
	Metadata map[string]any

	// ParallelMode controls discovery step scheduling after a parallel step
	// ends
	ParallelMode enums.ParallelMode `json:"pm,omitempty"`

	// CreatedAt is the timestamp when the pause was saved. This field may
	// be empty for older pauses created before this field was added. It's used to
	// determine which time-based storage blocks contain this pause, as block
	// timeframes are based on this timestamp (previously only available as the
	// sort score in pause indexes).
	CreatedAt time.Time `json:"ca"`
}

func (p Pause) GetOpcode() enums.Opcode {
	if p.Opcode == nil {
		return enums.OpcodeNone
	}
	switch *p.Opcode {
	case enums.OpcodeWaitForEvent.String():
		return enums.OpcodeWaitForEvent
	case enums.OpcodeWaitForSignal.String():
		return enums.OpcodeWaitForSignal
	case enums.OpcodeInvokeFunction.String():
		return enums.OpcodeInvokeFunction
	}
	return enums.OpcodeNone
}

func (p Pause) GetID() uuid.UUID {
	return p.ID
}

func (p Pause) GetExpression() string {
	if p.Expression == nil {
		return ""
	}

	// If this is a cancellation, ensure it doesn't have our `event.ts` suffix
	// added, eg:
	//  && (async.ts == null || async.ts > 1731035030976)
	if p.Cancel {
		return tsSuffix.ReplaceAllString(*p.Expression, "")
	}

	return *p.Expression
}

func (p Pause) GetEvent() *string {
	return p.Event
}

func (p Pause) GetWorkspaceID() uuid.UUID {
	return p.WorkspaceID
}

func (p Pause) Edge() inngest.Edge {
	return inngest.Edge{
		Outgoing: p.Outgoing,
		Incoming: p.Incoming,
	}
}

func (p Pause) IsWaitForEvent() bool {
	return p.Opcode != nil && *p.Opcode == enums.OpcodeWaitForEvent.String()
}

func (p Pause) IsInvoke() bool {
	return p.Opcode != nil && *p.Opcode == enums.OpcodeInvokeFunction.String()
}

func (p Pause) IsSignal() bool {
	return p.Opcode != nil && *p.Opcode == enums.OpcodeWaitForSignal.String()
}

type ResumeData struct {
	// If non-nil, RunID is the ID of the run that completed to cause this
	// resume.
	RunID    *ulid.ULID
	With     map[string]any
	StepName string
}

// Given an event, this returns data used to resume an execution.
func (p Pause) GetResumeData(evt event.Event) ResumeData {
	ret := ResumeData{
		With:     evt.Map(),
		StepName: p.StepName,
	}

	// Function invocations are resumed using an event, but we want to unwrap the event from this
	// data and return only what the function returned. We do this here by unpacking the function
	// finished event to pull out the correct data to place in state.
	isInvokeFunctionOpcode := p.Opcode != nil && *p.Opcode == enums.OpcodeInvokeFunction.String()
	if isInvokeFunctionOpcode && evt.IsFinishedEvent() {
		if retRunID, ok := evt.Data["run_id"].(string); ok {
			if ulidRunID, _ := ulid.Parse(retRunID); !ulidRunID.IsZero() {
				ret.RunID = &ulidRunID
			}
		}

		if errorData, errorExists := evt.Data["error"]; errorExists {
			ret.With = map[string]any{"error": errorData}
		} else if resultData, resultExists := evt.Data["result"]; resultExists {
			ret.With = map[string]any{"data": resultData}
		}
	}

	return ret
}
