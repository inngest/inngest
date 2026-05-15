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

type RunDeferInsert struct {
	HashedDeferID string // SHA-1 hashed UserDeferID
	UserDeferID   string // ID provided by the user, ie `defer("foo", ...)` => "foo"
	FnSlug        string // Slug of the deferred function, "foo-defer"
	Status        enums.DeferStatus
}

type RunDeferUpdate struct {
	HashedDeferID string
	ChildRunID    ulid.ULID // RunID of the deferred child run
}

// RunDefer is a single defer attached to a parent function run. Parsed from
// the parent's history opcodes (DeferAdd/DeferAbort) and joined to the
// resulting child run when one exists. Run is nil if the defer was aborted,
// the schedule event has not yet been processed, or the child trace was
// pruned.
type RunDefer struct {
	HashedDeferID string            // SHA1-hashed UserDeferID
	UserDeferID   string            // ID provided by the user, ie `defer("foo", ...)` => "foo"
	FnSlug        string            // Slug of the deferred function, "foo-defer"
	Status        enums.DeferStatus // status of the defer itself, not the potentially associated run
	Run           *TraceRun
}

type RunDeferredFrom struct {
	ParentRunID ulid.ULID
	ParentRun   *TraceRun
}
