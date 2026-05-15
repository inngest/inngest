package cqrs

import (
	"context"

	"github.com/inngest/inngest/pkg/enums"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/oklog/ulid/v2"
)

// The id parameter on each method is the **parent** run identifier. It carries
// the run's tenant (account, env, app) so downstream tenant-aware
// implementations (e.g. ClickHouse) can scope writes without each call site
// having to plumb separate accountID/envID parameters. Single-tenant
// implementations (the dev-server Postgres) just unpack id.RunID.
type DeferStore interface {
	InsertRunDefer(ctx context.Context, id sv2.ID, rd RunDeferInsert) error
	InsertRunDefers(ctx context.Context, id sv2.ID, defers []RunDeferInsert) error
	UpdateRunDeferChildRunID(ctx context.Context, id sv2.ID, upd RunDeferUpdate) error
}

// RunDeferInsert is the per-row payload for inserting a defer. The parent
// run identifier is passed alongside (see DeferStore); the child run id is
// set later via UpdateRunDeferChildRunID once the deferred.schedule event
// fires.
type RunDeferInsert struct {
	HashedDeferID string
	UserDeferID   string
	FnSlug        string
	Status        enums.DeferStatus
}

// RunDeferUpdate is the payload for UpdateRunDeferChildRunID. DeferID is the
// SHA1-hashed defer id (join key); ChildRunID is the run id of the deferred
// child run, parsed from the inngest/deferred.schedule event the runner just
// processed.
type RunDeferUpdate struct {
	HashedDeferID string
	ChildRunID    ulid.ULID
}

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
	HashedDeferID string
	UserDeferID   string
	FnSlug        string
	Status        enums.DeferStatus
	Run           *TraceRun
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
