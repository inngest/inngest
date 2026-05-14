package cqrs

import (
	"context"

	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/oklog/ulid/v2"
)

// DeferStore writes parent-run → child-run defer linkages so the dev-server
// GraphQL can expose deferred runs after the parent's state has been deleted.
//
// deferID is the SHA1 hash of the user-supplied defer id (the join key with
// UpdateRunDeferChildRunID). userDeferID preserves the original string so the
// UI can show the id the SDK caller typed.
//
// The id parameter on each method is the **parent** run identifier. It carries
// the run's tenant (account, env, app) so downstream tenant-aware
// implementations (e.g. ClickHouse) can scope writes without each call site
// having to plumb separate accountID/envID parameters. Single-tenant
// implementations (the dev-server Postgres) just unpack id.RunID.
type DeferStore interface {
	InsertRunDefer(ctx context.Context, id sv2.ID, deferID, userDeferID, fnSlug string, status RunDeferStatus) error
	InsertRunDefers(ctx context.Context, defers []RunDeferInsert) error
	UpdateRunDeferChildRunID(ctx context.Context, id sv2.ID, deferID string, childRunID ulid.ULID) error
}

// RunDeferInsert is a single defer row to be persisted. ID is the parent
// run identifier (run id + tenant); the child run id is set later via
// UpdateRunDeferChildRunID once the deferred.schedule event fires.
type RunDeferInsert struct {
	ID          sv2.ID
	DeferID     string
	UserDeferID string
	FnSlug      string
	Status      RunDeferStatus
}

type RunDeferStatus string

const (
	RunDeferStatusScheduled RunDeferStatus = "SCHEDULED"
	RunDeferStatusAborted   RunDeferStatus = "ABORTED"
)

// RunDefer is a single defer attached to a parent function run. Parsed from
// the parent's history opcodes (DeferAdd/DeferAbort) and joined to the
// resulting child run when one exists. Run is nil if the defer was aborted,
// the schedule event has not yet been processed, or the child trace was
// pruned.
//
// ID is the SHA1-hashed defer id used as the join key. UserDeferID is the
// original SDK-caller-supplied id (the first arg to `defer("foo", ...)`) and
// is what the dev-server UI shows. The SDK always emits userland.id, so
// UserDeferID is expected to be non-empty for any defer this server records.
type RunDefer struct {
	ID          string
	UserDeferID string
	FnSlug      string
	Status      RunDeferStatus
	Run         *TraceRun
}

// RunDeferredFrom is the parent-run linkage of a deferred (child) run,
// parsed from the child's triggering inngest/deferred.schedule event metadata.
// ParentRun is nil if the parent trace_runs row has not been ingested yet,
// most commonly because the OTel pipeline that populates trace_runs lagged
// behind or dropped the parent span.
type RunDeferredFrom struct {
	ParentRunID ulid.ULID
	ParentRun   *TraceRun
}
