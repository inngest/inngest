package icebergcatalog

import (
	"cmp"
	"errors"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/apache/iceberg-go"
	"github.com/apache/iceberg-go/table"
)

const (
	// DefaultMinSnapshotsToKeep is Retention.MinSnapshotsToKeep's default.
	DefaultMinSnapshotsToKeep = 20
	// DefaultMaxSnapshotAge is Retention.MaxSnapshotAge's default. It has
	// to outlive anything that reads a table at an older snapshot: a
	// federated query pinned to a snapshot id (`AT (VERSION => ...)`, see
	// federate.LakeSource) or a duckdb transaction that started before the
	// table changed. Both only bind the snapshot at the start of a
	// statement, so this is way more than they need.
	DefaultMaxSnapshotAge = 5 * time.Minute
	// DefaultMaxSnapshots is Retention.MaxSnapshots's default. Every
	// duckdb write is a snapshot, so a table taking a steady stream of
	// small appends piles up way more than MinSnapshotsToKeep inside
	// MaxSnapshotAge, and each commit rewrites (and each load parses) all
	// of them (~1.3MB of metadata.json at 1000-run batches w/o the cap vs
	// ~111KB w/ it).
	DefaultMaxSnapshots = 100

	// MaxSnapshotsKey is the table property that overrides
	// Retention.MaxSnapshots. It's ours, not a standard Iceberg property
	// (Java only has the min-keep/max-age pair), but named like them.
	MaxSnapshotsKey = "history.expire.max-snapshots"
)

// Retention configures the snapshot expiry and metadata file cleanup the
// server does on every commit. iceberg-go's sql catalog does neither (it
// ignores `write.metadata.delete-after-commit.enabled`, and
// `write.metadata.previous-versions-max` only trims the metadata log), and
// duckdb can't expire snapshots, so w/o this every commit rewrites (and
// every load parses) a metadata.json that grows w/ each snapshot, and the
// old ones pile up forever.
//
// A snapshot is expired if it's older than MaxSnapshotAge AND not one of
// the newest MinSnapshotsToKeep, or if it's past the newest MaxSnapshots
// regardless of age. A snapshot a ref points at is never expired. Tables
// override these w/ the standard `history.expire.min-snapshots-to-keep` and
// `history.expire.max-snapshot-age-ms` properties and our own
// `history.expire.max-snapshots` (MaxSnapshotsKey).
//
// Only snapshots are dropped from the metadata, never data files or
// manifests (the lake is append-only so an expired snapshot's files are
// still live anyways). Orphan file cleanup is TODO(maintenance).
type Retention struct {
	// Disabled turns off snapshot expiry.
	Disabled bool
	// MinSnapshotsToKeep is how many of the newest snapshots are always
	// kept, 0 -> DefaultMinSnapshotsToKeep.
	MinSnapshotsToKeep int
	// MaxSnapshotAge is how old a snapshot has to be before it can be
	// expired, 0 -> DefaultMaxSnapshotAge.
	MaxSnapshotAge time.Duration
	// MaxSnapshots caps how many snapshots a table keeps: past the newest
	// MaxSnapshots (counting ref targets, which are still kept) snapshots
	// are expired regardless of MaxSnapshotAge. 0 -> DefaultMaxSnapshots,
	// negative -> no cap. It never goes below MinSnapshotsToKeep.
	//
	// NOTE: this can expire a snapshot younger than MaxSnapshotAge, so a
	// reader pinned to it (see DefaultMaxSnapshotAge) fails if the table
	// takes more than MaxSnapshots commits in between.
	MaxSnapshots int
	// KeepOldMetadata turns off deleting metadata.json files that fell out
	// of the metadata log (a table can also opt out w/
	// `write.metadata.delete-after-commit.enabled=false`).
	KeepOldMetadata bool
}

// retentionPolicy is a table's effective Retention.
type retentionPolicy struct {
	keep   int
	maxAge time.Duration
	// maxSnapshots is the count cap, 0 for none.
	maxSnapshots int
}

// policy is r's policy for a table w/ props.
func (r Retention) policy(props iceberg.Properties) retentionPolicy {
	p := retentionPolicy{
		keep:         cmp.Or(r.MinSnapshotsToKeep, DefaultMinSnapshotsToKeep),
		maxAge:       cmp.Or(r.MaxSnapshotAge, DefaultMaxSnapshotAge),
		maxSnapshots: cmp.Or(r.MaxSnapshots, DefaultMaxSnapshots),
	}
	if n, err := strconv.Atoi(props[table.MinSnapshotsToKeepKey]); err == nil && n > 0 {
		p.keep = n
	}
	if ms, err := strconv.ParseInt(props[table.MaxSnapshotAgeMsKey], 10, 64); err == nil && ms >= 0 {
		p.maxAge = time.Duration(ms) * time.Millisecond
	}
	if n, err := strconv.Atoi(props[MaxSnapshotsKey]); err == nil && n != 0 {
		p.maxSnapshots = n
	}
	p.keep = max(p.keep, 1)
	if p.maxSnapshots < 0 {
		p.maxSnapshots = 0
	} else {
		p.maxSnapshots = max(p.maxSnapshots, p.keep)
	}
	return p
}

// expireSnapshots picks the snapshots to drop from md, the metadata a
// commit produces (the client's updates already applied, so its new
// snapshot and ref moves count). Newest first, the first p.keep snapshots
// and any younger than p.maxAge are kept up to p.maxSnapshots, plus every
// ref target regardless.
func expireSnapshots(md table.Metadata, p retentionPolicy, now time.Time) []int64 {
	snaps := slices.Clone(md.Snapshots())
	if len(snaps) <= p.keep {
		return nil
	}
	refs := map[int64]bool{}
	for _, ref := range md.Refs() {
		refs[ref.SnapshotID] = true
	}
	slices.SortStableFunc(snaps, func(a, b table.Snapshot) int {
		return cmp.Or(cmp.Compare(b.SequenceNumber, a.SequenceNumber), cmp.Compare(b.TimestampMs, a.TimestampMs))
	})
	cutoff := now.Add(-p.maxAge).UnixMilli()
	var ids []int64
	for i, snap := range snaps {
		if refs[snap.SnapshotID] {
			continue
		}
		capped := p.maxSnapshots > 0 && i >= p.maxSnapshots
		if !capped && (i < p.keep || snap.TimestampMs >= cutoff) {
			continue
		}
		ids = append(ids, snap.SnapshotID)
	}
	return ids
}

// retain appends a remove-snapshots for what md (base w/ the client's
// updates applied) can expire to the commit, plus an assert-ref-snapshot-id
// per ref on base so a concurrent ref move can't land on a snapshot we
// drop. iceberg-go would silently drop that ref instead.
//
// NOTE: a ref *created* concurrently on an old snapshot (by another writer
// sharing the catalog db) between our base and CommitTable's isn't caught,
// there's no requirement for "no other refs". Nothing we run creates refs
// besides main.
func (s *Server) retain(base, md table.Metadata, client table.Updates, req *commitTableRequest, now time.Time) {
	if s.opts.Retention.Disabled || base == nil {
		return
	}
	for _, u := range client {
		// The client is managing snapshots itself, stay out of its way.
		switch u.Action() {
		case table.UpdateRemoveSnapshots, table.UpdateRemoveSnapshotRef:
			return
		}
	}
	ids := expireSnapshots(md, s.opts.Retention.policy(md.Properties()), now)
	if len(ids) == 0 {
		return
	}
	for name, ref := range base.Refs() {
		id := ref.SnapshotID
		req.Requirements = append(req.Requirements, table.AssertRefSnapshotID(name, &id))
	}
	req.Updates = append(req.Updates, table.NewRemoveSnapshotsUpdate(ids, false))
}

// deleteOldMetadata deletes the metadata.json files on prev's metadata log
// that aren't on next's (IE what iceberg-go just trimmed per
// `write.metadata.previous-versions-max`), which is what Java's
// `write.metadata.delete-after-commit.enabled` does.
//
// This doesn't take anything away from readers: duckdb only finds an older
// metadata.json (for a transaction that started before the table changed)
// through the log, so a file that fell out of it is unreachable anyways.
// Only files directly under the table's `metadata/` dir in a local
// warehouse named like the catalog names them are touched.
func (s *Server) deleteOldMetadata(prev, next table.Metadata, nextLoc string) {
	if s.opts.Retention.KeepOldMetadata || prev == nil {
		return
	}
	if v, ok := next.Properties()[table.MetadataDeleteAfterCommitEnabledKey]; ok {
		if on, err := strconv.ParseBool(v); err == nil && !on {
			return
		}
	}
	wh, err := url.Parse(s.opts.Warehouse)
	if err != nil {
		return
	}
	if !isLocal(wh) {
		// TODO(maintenance): delete from remote warehouses too (same as
		// purge).
		return
	}
	keep := map[string]bool{nextLoc: true}
	for e := range next.PreviousFiles() {
		keep[e.MetadataFile] = true
	}
	var dropped []string
	for e := range prev.PreviousFiles() {
		if !keep[e.MetadataFile] {
			dropped = append(dropped, e.MetadataFile)
		}
	}
	if len(dropped) == 0 {
		return
	}

	c := s.locations()
	loc, err := c.check(next.Location())
	if err != nil {
		return
	}
	dir := filepath.Join(loc, "metadata")
	for _, f := range dropped {
		p, err := c.check(f)
		if err != nil || filepath.Dir(p) != dir || !strings.HasSuffix(p, ".metadata.json") {
			continue
		}
		if err := os.Remove(p); err != nil && !errors.Is(err, fs.ErrNotExist) && s.opts.Logger != nil {
			s.opts.Logger.Warn("failed to delete old iceberg metadata file", "path", p, "error", err)
		}
	}
}
