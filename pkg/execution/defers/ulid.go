package defers

import (
	"github.com/inngest/inngest/pkg/util"
	"github.com/oklog/ulid/v2"
)

// SpanKind tags one of the executor.defer spans we emit per defer. The tag
// makes each span's deterministic ID distinct so all rows survive in the
// linkage query — which filters by span name and never sees the EXTEND
// fragments UpdateSpan would produce. GetRunDefers collapses them by hashed
// ID.
type SpanKind string

const (
	SpanSchedule   SpanKind = "s"
	SpanChildRunID SpanKind = "c"
)

// SpanSeed returns the deterministic seed for an executor.defer span.
// Determinism keeps the span ID stable across retries.
func SpanSeed(parent ulid.ULID, hashedID string, kind SpanKind) []byte {
	return []byte(parent.String() + hashedID + string(kind))
}

// EventID returns the deterministic inngest/deferred.schedule event ID for a
// defer. Used as the runner's schedule idempotency key.
func EventID(parent ulid.ULID, hashedID string) ulid.ULID {
	id, _ := util.DeterministicULID(ulid.Time(parent.Time()), []byte(parent.String()+hashedID))
	return id
}
