// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package table

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"log/slog"
	"maps"
	"runtime"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/compute/exprs"
	"github.com/apache/iceberg-go"
	iceberginternal "github.com/apache/iceberg-go/internal"
	"github.com/apache/iceberg-go/io"
	"github.com/apache/iceberg-go/table/dv"
	tblutils "github.com/apache/iceberg-go/table/internal"
	"github.com/apache/iceberg-go/table/substrait"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

type snapshotUpdate struct {
	txn           *Transaction
	io            io.WriteFileIO
	snapshotProps iceberg.Properties
	operation     Operation
}

func (s snapshotUpdate) fastAppend() *snapshotProducer {
	return newFastAppendFilesProducer(OpAppend, s.txn, s.io, nil, s.snapshotProps)
}

// mergeOverwrite builds an overwrite producer. filter is the
// row-level predicate the caller declared (typically the same one
// used to plan the deletion); pass nil for a full-table overwrite.
// validate() reads it to decide between filter-bounded and full
// checks.
func (s snapshotUpdate) mergeOverwrite(commitUUID *uuid.UUID, filter iceberg.BooleanExpression) *snapshotProducer {
	op := s.operation
	if s.operation == OpOverwrite && s.txn.planningSnapshot(s.txn.meta) == nil {
		op = OpAppend
	}
	prod := newOverwriteFilesProducer(op, s.txn, s.io, commitUUID, s.snapshotProps)
	prod.setOverwriteFilter(filter)

	return prod
}

func (s snapshotUpdate) mergeAppend() *snapshotProducer {
	return newMergeAppendFilesProducer(OpAppend, s.txn, s.io, nil, s.snapshotProps)
}

type Transaction struct {
	tbl     *Table
	meta    *MetadataBuilder
	branch  string
	initErr error

	reqs []Requirement

	// validators collects per-producer conflict checks registered
	// during this transaction's lifetime. doCommit runs them against
	// the current catalog state before sending CommitTable so
	// producers can reject commits whose semantics are violated by
	// concurrent peers (partition-filter overlap, referenced-file
	// removal). Fast/merge-append producers do not register a
	// validator (they are safe under any isolation).
	validators []conflictValidatorFunc

	// noReplay marks the commit as non-replayable: on a CAS conflict
	// doCommit fails instead of refreshing and replaying. Set by
	// snapshot producers whose staged snapshot carries delete-file
	// removals; see the flag site in commitManifests.
	noReplay bool

	// pinnedRefs names branches with an explicit AssertRefSnapshotID
	// requirement. doCommit's refresh-and-replay must not rewrite their
	// assertions to the fresh branch head between retries: the caller
	// opted into compare-and-swap semantics, so a branch that has
	// changed fails the commit.
	pinnedRefs map[string]struct{}

	mx        sync.Mutex
	committed bool
}

func (t *Transaction) ensureInitialized() error {
	if t == nil {
		return fmt.Errorf("%w: transaction is nil", ErrInvalidMetadata)
	}
	if t.initErr != nil {
		return t.initErr
	}
	if t.meta == nil {
		return fmt.Errorf("%w: transaction metadata builder is not initialized", ErrInvalidMetadata)
	}

	return nil
}

// txnMeta returns the transaction's metadata builder after verifying the
// transaction is initialized. Entry points that need the builder should obtain
// it here so the nil-transaction / deferred-init / nil-builder checks are
// applied consistently in one place, rather than each caller having to remember
// to call ensureInitialized before touching t.meta.
func (t *Transaction) txnMeta() (*MetadataBuilder, error) {
	if err := t.ensureInitialized(); err != nil {
		return nil, err
	}

	return t.meta, nil
}

// checkNotNil rejects a nil transaction before any lock is taken. The
// mutex-guarded entry points that return an error (Commit, TableCommit, apply)
// must call this before t.mx.Lock(), which would panic on a nil receiver.
// txnMeta covers the same case, but only after the lock is already held.
func (t *Transaction) checkNotNil() error {
	if t == nil {
		return fmt.Errorf("%w: transaction is nil", ErrInvalidMetadata)
	}

	return nil
}

func (t *Transaction) apply(updates []Update, reqs []Requirement) error {
	if err := t.checkNotNil(); err != nil {
		return err
	}
	t.mx.Lock()
	defer t.mx.Unlock()

	meta, err := t.txnMeta()
	if err != nil {
		return err
	}

	if t.committed {
		return errors.New("transaction has already been committed")
	}

	stagedMeta := meta.clone()
	if stagedMeta == nil {
		return errors.New("cannot apply updates to nil metadata")
	}

	// Only new requirement validation needs the immutable metadata view.
	// Updates can be applied directly to the staged builder, and duplicate
	// requirements only need conflict checks.
	var current Metadata

	// Deduplicate requirements by semantic key, rejecting pairs that
	// share a key but cannot both hold (see checkRequirementConflict).
	// Only requirements actually appended are validated against the
	// staged metadata: a deduplicated twin re-asserts base state the
	// staged metadata has intentionally moved past, and its kept twin
	// was validated when first added.
	stagedReqs := t.reqs
	if len(reqs) > 0 {
		existing := make(map[string]Requirement, len(t.reqs))
		for _, r := range t.reqs {
			key, err := requirementSemanticKey(r)
			if err != nil {
				return err
			}

			existing[key] = r
		}

		copiedReqs := false
		for _, r := range reqs {
			key, err := requirementSemanticKey(r)
			if err != nil {
				return err
			}

			if prev, ok := existing[key]; ok {
				if err := checkRequirementConflict(prev, r); err != nil {
					return err
				}

				continue
			}
			if current == nil {
				built, err := stagedMeta.Build()
				if err != nil {
					return err
				}
				current = built
			}
			if err := r.Validate(current); err != nil {
				return err
			}
			if !copiedReqs {
				stagedReqs = make([]Requirement, len(t.reqs), len(t.reqs)+len(reqs))
				copy(stagedReqs, t.reqs)
				copiedReqs = true
			}
			existing[key] = r
			stagedReqs = append(stagedReqs, r)
		}
	}

	prevUpdates, prevLastUpdated := len(stagedMeta.updates), stagedMeta.lastUpdatedMS
	for _, u := range updates {
		if err := u.Apply(stagedMeta); err != nil {
			return err
		}
	}

	// u.Apply will add updates to t.meta.updates if they are not no-ops
	// and actually perform changes. So let's check if we actually had any
	// changes added and thus need to update the lastupdated value.
	if prevUpdates < len(stagedMeta.updates) {
		if prevLastUpdated == stagedMeta.lastUpdatedMS {
			stagedMeta.lastUpdatedMS = time.Now().UnixMilli()
		}
	}

	t.reqs = stagedReqs
	t.meta = stagedMeta

	return nil
}

// requirementSemanticKey computes the dedupe key for a requirement.
//
// For most requirement types the key is the canonical JSON marshaling, which
// keeps semantically distinct requirements (e.g. assertions on different refs)
// from collapsing into one another.
//
// assert-ref-snapshot-id is special-cased to key by requirement type + ref name
// only, deliberately ignoring the asserted snapshot id. Within a single
// transaction the builder mutates its own ref state across operations (e.g. the
// first append asserts main == nil, a later append asserts main == snapshot-1),
// which would otherwise produce multiple, mutually contradictory base-state
// assertions for the same ref against the pre-transaction metadata. Keying by
// ref name keeps only the first assertion for each ref while still letting
// assertions for different refs survive dedupe.
func requirementSemanticKey(r Requirement) (string, error) {
	if ref, ok := r.(*assertRefSnapshotID); ok {
		return fmt.Sprintf("%s\x00%s", reqAssertRefSnapshotID, ref.Ref), nil
	}

	data, err := json.Marshal(r)
	if err != nil {
		return "", fmt.Errorf("marshal requirement %q: %w", r.GetType(), err)
	}

	return string(data), nil
}

// checkRequirementConflict reports whether two requirements sharing a
// semantic key can be collapsed into one. Identical ref assertions dedupe
// to the first; two assert-ref-snapshot-id for the same ref requiring
// different snapshot ids can never both hold, so keeping either one
// would silently drop a check the caller registered — reject instead.
func checkRequirementConflict(prev, next Requirement) error {
	a, aok := prev.(*assertRefSnapshotID)
	b, bok := next.(*assertRefSnapshotID)
	if !aok || !bok {
		return nil
	}

	if (a.SnapshotID == nil) != (b.SnapshotID == nil) ||
		(a.SnapshotID != nil && *a.SnapshotID != *b.SnapshotID) {
		return fmt.Errorf("conflicting snapshot-id assertions for ref %q: %s vs %s",
			a.Ref, formatSnapshotID(a.SnapshotID), formatSnapshotID(b.SnapshotID))
	}

	return nil
}

// formatSnapshotID renders a required snapshot id for error messages; a
// nil id asserts the ref's absence.
func formatSnapshotID(id *int64) string {
	if id == nil {
		return "<ref absent>"
	}

	return strconv.FormatInt(*id, 10)
}

// baseRefSnapshotID returns the snapshot id the transaction's BASE
// table metadata records for branch, or nil when the branch does not
// exist on the base. Ref requirements must be built from this state —
// the catalog state the writer actually read, as in Java's
// UpdateRequirements — rather than the staged builder's refs, which
// move past it as producer commits stage intermediate snapshots the
// catalog has never seen.
func (t *Transaction) baseRefSnapshotID(branch string) *int64 {
	if t.tbl == nil || t.tbl.metadata == nil {
		return nil
	}
	for name, ref := range t.tbl.metadata.Refs() {
		if name == branch {
			id := ref.SnapshotID

			return &id
		}
	}

	return nil
}

// planningSnapshot must resolve the same head createSnapshotProducer parents on,
// or an operation plans against files the new snapshot does not inherit.
func (t *Transaction) planningSnapshot(meta *MetadataBuilder) *Snapshot {
	return meta.currentSnapshotForRef(t.branch)
}

// transactionRequirements returns the requirements a commit on branch must carry.
// The result is always a fresh allocation, never aliasing reqs, so callers may append to it
// while holding the transaction lock.
func transactionRequirements(reqs []Requirement, branch string, base Metadata) []Requirement {
	if branch == "" {
		branch = MainBranch
	}

	out := slices.Clone(reqs)
	for i, req := range out {
		if ref, ok := req.(*assertRefSnapshotID); ok && ref.Ref == branch {
			if ref.requireBranch {
				return out
			}

			out[i] = assertBranchRefSnapshotID(branch, ref.SnapshotID)

			return out
		}
	}

	var snapshotID *int64
	for name, ref := range base.Refs() {
		if name == branch {
			id := ref.SnapshotID
			snapshotID = &id

			break
		}
	}

	return append(out, assertBranchRefSnapshotID(branch, snapshotID))
}

// addValidator appends a conflict validator under t.mx. Producers
// that register validators from outside doCommit (RowDelta, RewriteFiles)
// must use this helper rather than mutating t.validators directly —
// the RewriteFiles type doc endorses fanout builders against a single
// transaction, so the append races with Transaction.Commit's
// validator read under t.mx.
func (t *Transaction) addValidator(v conflictValidatorFunc) {
	t.mx.Lock()
	defer t.mx.Unlock()
	t.validators = append(t.validators, v)
}

func (t *Transaction) appendSnapshotProducer(meta *MetadataBuilder, wfs io.WriteFileIO, props iceberg.Properties) *snapshotProducer {
	manifestMerge := meta.props.GetBool(ManifestMergeEnabledKey, ManifestMergeEnabledDefault)
	updateSnapshot := t.updateSnapshot(wfs, props, OpAppend)
	if manifestMerge {
		return updateSnapshot.mergeAppend()
	}

	return updateSnapshot.fastAppend()
}

func (t *Transaction) updateSnapshot(wfs io.WriteFileIO, props iceberg.Properties, operation Operation) snapshotUpdate {
	return snapshotUpdate{
		txn:           t,
		io:            wfs,
		snapshotProps: props,
		operation:     operation,
	}
}

func (t *Transaction) SetProperties(props iceberg.Properties) error {
	if _, err := t.txnMeta(); err != nil {
		return err
	}
	if len(props) > 0 {
		return t.apply([]Update{NewSetPropertiesUpdate(props)}, nil)
	}

	return nil
}

// AssertRefSnapshotID records a requirement that the given branch is
// unchanged from the transaction's base table metadata — the catalog
// state this transaction read: the branch must still reference the
// same snapshot id, or still not exist when the base has none. An
// empty branch means the transaction's target branch (main unless the
// transaction was created with NewTransactionOnBranch).
//
// It provides optimistic concurrency for metadata-only commits, such
// as SetProperties used for exactly-once bookkeeping. Commit normally
// retries against the fresh branch head; with this requirement it
// instead fails with ErrCommitFailed if any snapshot was committed to
// the branch after the transaction's read. Snapshot producers
// targeting the branch in the same transaction fail the same way
// rather than retry.
//
// The requirement is submitted together with the transaction's
// updates; a transaction with no updates never contacts the catalog,
// so the requirement alone does not force a commit.
func (t *Transaction) AssertRefSnapshotID(branch string) error {
	if _, err := t.txnMeta(); err != nil {
		return err
	}
	if branch == "" {
		branch = t.branch
	}
	if branch == "" {
		branch = MainBranch
	}

	id := t.baseRefSnapshotID(branch)

	// Record the pin before the requirement can become visible to a
	// concurrent Commit, which would otherwise rewrite the requirement
	// to the fresh head on retry and silently void the
	// compare-and-swap. The transient pin is harmless on its own and
	// is rolled back if the requirement fails to apply.
	t.mx.Lock()
	if t.pinnedRefs == nil {
		t.pinnedRefs = make(map[string]struct{})
	}
	_, alreadyPinned := t.pinnedRefs[branch]
	t.pinnedRefs[branch] = struct{}{}
	t.mx.Unlock()

	if err := t.apply(nil, []Requirement{AssertRefSnapshotID(branch, id)}); err != nil {
		if !alreadyPinned {
			t.mx.Lock()
			delete(t.pinnedRefs, branch)
			t.mx.Unlock()
		}

		return err
	}

	return nil
}

// AssertDefaultShape registers requirements pinning the table's default
// partition spec ID and default sort order ID at their values in the
// transaction's base metadata, so a concurrent change to either fails
// this commit instead of interleaving silently — the shape analogue of
// AssertRefSnapshotID's explicit branch fence. It gives commits that
// stage no shape change of their own (for example a properties-only
// revision marker) the same serialization against concurrent spec or
// sort-order evolutions that Java UpdateRequirements.forUpdateTable
// registers for SetDefaultPartitionSpec / SetDefaultSortOrder, and that
// UpdateSpec and ReplaceSortOrder already register for shape-changing
// commits; when both are present they pin the same base values and
// deduplicate.
//
// Register the fence BEFORE staging spec or sort-order changes:
// requirements are validated against the staged metadata at
// registration time, so a fence registered after the transaction moved
// a default fails immediately. Unlike ref assertions the fence is never
// rewritten by the retry loop's refresh-and-replay, and a transaction
// with no updates never contacts the catalog, so the fence alone does
// not force a commit.
func (t *Transaction) AssertDefaultShape() error {
	if _, err := t.txnMeta(); err != nil {
		return err
	}
	meta := t.tbl.Metadata()

	return t.apply(nil, []Requirement{
		AssertDefaultSpecID(meta.DefaultPartitionSpec()),
		AssertDefaultSortOrderID(meta.DefaultSortOrder()),
	})
}

// RemoveProperties removes the given property keys from the table. Keys that
// are not currently set are ignored.
func (t *Transaction) RemoveProperties(keys []string) error {
	if _, err := t.txnMeta(); err != nil {
		return err
	}
	if len(keys) > 0 {
		return t.apply([]Update{NewRemovePropertiesUpdate(keys)}, nil)
	}

	return nil
}

// UpgradeFormatVersion upgrades the table to the given format version. Downgrading
// is not allowed. If the table is already at the given version, this is a no-op.
func (t *Transaction) UpgradeFormatVersion(version int) error {
	if _, err := t.txnMeta(); err != nil {
		return err
	}

	return t.apply([]Update{NewUpgradeFormatVersionUpdate(version)}, nil)
}

func (t *Transaction) RollbackToSnapshot(snapshotID int64) error {
	meta, err := t.txnMeta()
	if err != nil {
		return err
	}

	branch := t.branch
	if branch == "" {
		branch = MainBranch
	}

	// Resolve the head from the target branch's OWN ref. This deliberately does
	// not reuse a resolver that falls back to main's head for an unknown ref:
	// rolling back a branch that does not exist would then rewind
	// main instead. An absent ref must fail here, never fall back.
	ref, ok := meta.refs[branch]
	if !ok {
		return fmt.Errorf("%w: cannot rollback: branch %q does not exist",
			iceberg.ErrInvalidArgument, branch)
	}

	if ref.SnapshotRefType != BranchRef {
		return fmt.Errorf("%w: cannot rollback: ref %q is a %s, not a branch",
			iceberg.ErrInvalidArgument, branch, ref.SnapshotRefType)
	}

	// Present but invalid is not the same as absent: metadata validation
	// (checkRefsExist) rejects dangling refs on load, so reaching this means
	// in-flight builder state is inconsistent. Fail closed and name the ref
	// rather than reporting the branch as empty.
	// Delibertely not wrapped in iceberg.ErrInvalidArgument:
	// table state is invalid; not caller input
	cs, err := meta.SnapshotByID(ref.SnapshotID)
	if err != nil {
		return fmt.Errorf("cannot rollback: branch %q references unknown snapshot %d: %w",
			branch, ref.SnapshotID, err)
	}

	lookup := func(id int64) *Snapshot {
		s, _ := meta.SnapshotByID(id)

		return s
	}

	if !IsAncestorOf(cs.SnapshotID, snapshotID, lookup) {
		return fmt.Errorf("%w: snapshot %d is not an ancestor of branch %q head snapshot %d",
			iceberg.ErrInvalidArgument, snapshotID, branch, cs.SnapshotID)
	}

	update := meta.NewRetainingSnapshotRefUpdate(branch, snapshotID, BranchRef)

	// Assert the base branch head so a concurrent head move fails the
	// commit instead of the rollback clobbering it. When the branch was
	// staged by this transaction (absent on the base), the update that
	// created it already carries its own base-state requirement.
	var reqs []Requirement
	if id := t.baseRefSnapshotID(branch); id != nil {
		reqs = append(reqs, AssertRefSnapshotID(branch, id))
	}

	return t.apply([]Update{update}, reqs)
}

func (t *Transaction) UpdateSpec(caseSensitive bool) *UpdateSpec {
	return NewUpdateSpec(t, caseSensitive)
}

// ReplaceSortOrder stages order as the table's new default sort order,
// the analogue of Java's Table.replaceSortOrder() / BaseReplaceSortOrder.
// The metadata builder reuses the ID of an equivalent existing order or
// assigns the next available one, so order's own ID is ignored. The
// commit is fenced by AssertDefaultSortOrderID, matching
// UpdateRequirements.forUpdateTable on SetDefaultSortOrder. Replacing
// the default with a field-identical order is a no-op.
func (t *Transaction) ReplaceSortOrder(order SortOrder) error {
	meta, err := t.txnMeta()
	if err != nil {
		return err
	}
	if cur, err := meta.GetSortOrderByID(meta.defaultSortOrderID); err == nil && sameSortFields(*cur, order) {
		return nil
	}

	return t.apply(
		[]Update{NewAddSortOrderUpdate(&order), NewSetDefaultSortOrderUpdate(-1)},
		[]Requirement{AssertDefaultSortOrderID(t.tbl.Metadata().DefaultSortOrder())},
	)
}

func sameSortFields(a, b SortOrder) bool {
	if a.Len() != b.Len() {
		return false
	}
	for i := range a.Len() {
		if !a.Field(i).Equals(b.Field(i)) {
			return false
		}
	}

	return true
}

// UpdateSchema creates a new UpdateSchema instance for managing schema changes
// within this transaction.
//
// Parameters:
//   - caseSensitive: If true, field name lookups are case-sensitive; if false,
//     field names are matched case-insensitively.
//   - allowIncompatibleChanges: If true, allows schema changes that would normally
//     be rejected for being incompatible (e.g., adding required fields without
//     default values, changing field types in non-promotable ways, or changing
//     column nullability from optional to required).
//   - opts: Optional configuration functions to customize the UpdateSchema behavior.
//
// Returns an UpdateSchema instance that can be used to build and apply schema changes.
func (t *Transaction) UpdateSchema(caseSensitive bool, allowIncompatibleChanges bool, opts ...UpdateSchemaOption) *UpdateSchema {
	return NewUpdateSchema(t, caseSensitive, allowIncompatibleChanges, opts...)
}

type expireSnapshotsCfg struct {
	minSnapshotsToKeep *int
	maxSnapshotAgeMs   *int64
	postCommit         bool
	validationErr      error
}

type ExpireSnapshotsOpt func(*expireSnapshotsCfg)

// WithRetainLast sets the minimum number of snapshots to keep per branch,
// regardless of age. It overrides the MinSnapshotsToKeepKey table property for
// this call. Snapshots beyond this count become eligible for expiry only once
// they are also older than the configured age (see WithOlderThan).
func WithRetainLast(n int) ExpireSnapshotsOpt {
	return func(cfg *expireSnapshotsCfg) {
		cfg.minSnapshotsToKeep = &n
		if n < 1 && cfg.validationErr == nil {
			cfg.validationErr = errors.New("retain-last must be at least 1")
		}
	}
}

// WithOlderThan expires snapshots older than the given duration, measured from
// the current time. It overrides the MaxSnapshotAgeMsKey table property for this
// call. The most recent snapshots are still retained up to the count set by
// WithRetainLast. This option does not expire branches or tags based on their
// age; configure MaxRefAgeMsKey on the ref or table for that.
func WithOlderThan(t time.Duration) ExpireSnapshotsOpt {
	return func(cfg *expireSnapshotsCfg) {
		n := t.Milliseconds()
		cfg.maxSnapshotAgeMs = &n
		if t < 0 && cfg.validationErr == nil {
			cfg.validationErr = errors.New("snapshot age must be non-negative")
		}
	}
}

// WithPostCommit controls whether orphaned files (manifests, manifest lists,
// data files) are deleted immediately after expiring snapshots. Defaults to true.
// Set to false to defer file deletion to a separate maintenance job, avoiding
// conflicts with in-flight queries that may still reference those files.
func WithPostCommit(postCommit bool) ExpireSnapshotsOpt {
	return func(cfg *expireSnapshotsCfg) {
		cfg.postCommit = postCommit
	}
}

func getRetentionInt(props iceberg.Properties, standardKey, legacyKey string, defaultValue int) int {
	if _, ok := props[standardKey]; ok {
		return props.GetInt(standardKey, defaultValue)
	}

	return props.GetInt(legacyKey, defaultValue)
}

func getRetentionInt64(props iceberg.Properties, standardKey, legacyKey string, defaultValue int64) int64 {
	if _, ok := props[standardKey]; ok {
		return props.GetInt64(standardKey, defaultValue)
	}

	return props.GetInt64(legacyKey, defaultValue)
}

// ExpireSnapshots removes expired snapshots from the table metadata, staging
// the changes on the transaction. Call [Transaction.Commit] to persist them.
//
// A snapshot is retained when it is referenced by a branch or tag, or when it
// is needed to satisfy the retention rules. Retention is resolved per ref,
// with reference age falling back through the ref's max-ref-age-ms, the table's
// MaxRefAgeMsKey property, and the specification default. Snapshot age falls
// back through the ref's own settings, the options passed here, and finally the
// table's MinSnapshotsToKeepKey and MaxSnapshotAgeMsKey properties. The snapshot
// a retained ref points at is always kept, for every branch and tag, so expiry
// can never leave a ref referencing a snapshot that no longer exists.
//
// By default the now-unreferenced manifests, manifest lists, and data files are
// deleted once the commit lands; pass WithPostCommit(false) to defer that
// cleanup to a separate maintenance job and avoid racing in-flight readers.
//
//	txn := tbl.NewTransaction()
//	err := txn.ExpireSnapshots(
//		table.WithOlderThan(7*24*time.Hour),
//		table.WithRetainLast(10),
//	)
//	if err != nil {
//		// ...
//	}
//	newTbl, err := txn.Commit(ctx)
//
// The "iceberg expire-snapshots" CLI command wraps the same operation.
func (t *Transaction) ExpireSnapshots(opts ...ExpireSnapshotsOpt) error {
	var (
		cfg         = expireSnapshotsCfg{postCommit: true}
		updates     []Update
		reqs        []Requirement
		snapsToKeep = make(map[int64]struct{})
		nowMs       = time.Now().UnixMilli()
	)

	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.validationErr != nil {
		return cfg.validationErr
	}
	meta, err := t.txnMeta()
	if err != nil {
		return err
	}
	if err := validateGCEnabledForSnapshotExpiration(meta.props); err != nil {
		return err
	}

	// Read table-level retention properties as the last-resort defaults,
	// mirroring the Java implementation. The unprefixed property names are
	// retained as compatibility fallbacks for tables created by older Go
	// versions.
	propMaxRefAgeMs := getRetentionInt64(
		meta.props, MaxRefAgeMsKey, legacyMaxRefAgeMsKey, MaxRefAgeMsDefault,
	)
	propMinSnapshotsToKeep := getRetentionInt(
		meta.props, MinSnapshotsToKeepKey, legacyMinSnapshotsToKeepKey, MinSnapshotsToKeepDefault,
	)
	propMaxSnapshotAgeMs := getRetentionInt64(
		meta.props, MaxSnapshotAgeMsKey, legacyMaxSnapshotAgeMsKey, MaxSnapshotAgeMsDefault,
	)
	defaultMaxSnapshotAgeMs := propMaxSnapshotAgeMs
	if cfg.maxSnapshotAgeMs != nil {
		defaultMaxSnapshotAgeMs = *cfg.maxSnapshotAgeMs
	}

	lookup := func(id int64) *Snapshot {
		s, _ := meta.SnapshotByID(id)

		return s
	}

	retainedRefs := make(map[string]SnapshotRef, len(meta.refs))
	for refName, ref := range meta.refs {
		// Assert the ref's base snapshot id so we don't accidentally
		// expire snapshots that are now referenced by concurrently
		// updated refs. Refs the transaction itself staged (absent on
		// the base) get no assertion here: the update that created
		// them carries its own base-state requirement.
		if id := t.baseRefSnapshotID(refName); id != nil {
			reqs = append(reqs, AssertRefSnapshotID(refName, id))
		}

		snap, err := meta.SnapshotByID(ref.SnapshotID)
		if err != nil {
			return err
		}

		maxRefAgeMs := cmp.Or(ref.MaxRefAgeMs, &propMaxRefAgeMs)

		refAge := nowMs - snap.TimestampMs
		if refAge > *maxRefAgeMs && refName != MainBranch {
			updates = append(updates, NewRemoveSnapshotRefUpdate(refName))

			continue
		}
		retainedRefs[refName] = ref

		// Retained regardless of age: the branch walk below can skip the head
		// when min-snapshots-to-keep is 0, and tags never run that walk.
		snapsToKeep[ref.SnapshotID] = struct{}{}

		if ref.SnapshotRefType != BranchRef {
			continue
		}

		var (
			minSnapshotsToKeep = cmp.Or(ref.MinSnapshotsToKeep, cfg.minSnapshotsToKeep, &propMinSnapshotsToKeep)
			maxSnapshotAgeMs   = cmp.Or(ref.MaxSnapshotAgeMs, cfg.maxSnapshotAgeMs, &propMaxSnapshotAgeMs)
		)

		for numSnapshots, snap := range AncestorsOf(ref.SnapshotID, lookup) {
			// Age against the single nowMs so this cutoff and the
			// unreferenced-snapshot cutoff below cannot disagree.
			snapAge := nowMs - snap.TimestampMs
			if (snapAge > *maxSnapshotAgeMs) && (numSnapshots >= *minSnapshotsToKeep) {
				break
			}

			snapsToKeep[snap.SnapshotID] = struct{}{}
		}
	}

	referencedSnapshots := make(map[int64]struct{})
	for _, ref := range retainedRefs {
		if ref.SnapshotRefType != BranchRef {
			referencedSnapshots[ref.SnapshotID] = struct{}{}

			continue
		}

		for _, snap := range AncestorsOf(ref.SnapshotID, lookup) {
			referencedSnapshots[snap.SnapshotID] = struct{}{}
		}
	}

	unreferencedSnapshotCutoff := nowMs - defaultMaxSnapshotAgeMs
	for _, snap := range meta.snapshotList {
		if _, referenced := referencedSnapshots[snap.SnapshotID]; referenced {
			continue
		}

		if snap.TimestampMs >= unreferencedSnapshotCutoff {
			snapsToKeep[snap.SnapshotID] = struct{}{}
		}
	}

	var snapsToDelete []int64

	for _, snap := range meta.snapshotList {
		if _, found := snapsToKeep[snap.SnapshotID]; !found {
			snapsToDelete = append(snapsToDelete, snap.SnapshotID)
		}
	}

	// Only add the update if there are actually snapshots to delete
	if len(snapsToDelete) > 0 {
		updates = append(updates, NewRemoveSnapshotsUpdate(snapsToDelete, cfg.postCommit))
	}

	return t.apply(updates, reqs)
}

func (t *Transaction) AppendTable(ctx context.Context, tbl arrow.Table, batchSize int64, snapshotProps iceberg.Properties) error {
	if _, err := t.txnMeta(); err != nil {
		return err
	}
	rdr := array.NewTableReader(tbl, batchSize)
	defer rdr.Release()

	return t.Append(ctx, rdr, snapshotProps)
}

func (t *Transaction) Append(ctx context.Context, rdr array.RecordReader, snapshotProps iceberg.Properties) error {
	meta, err := t.txnMeta()
	if err != nil {
		return err
	}
	fs, err := t.tbl.fsF(ctx)
	if err != nil {
		return err
	}
	wfs, err := requireWriteFileIO(fs)
	if err != nil {
		return err
	}
	appendFiles := t.appendSnapshotProducer(meta, wfs, snapshotProps)
	itr := recordsToDataFiles(ctx, t.tbl.Location(), meta, recordWritingArgs{
		sc:        rdr.Schema(),
		itr:       array.IterFromReader(rdr),
		fs:        wfs,
		writeUUID: &appendFiles.commitUuid,
	})

	for df, err := range itr {
		if err != nil {
			return err
		}
		appendFiles.appendDataFile(df)
	}

	updates, reqs, err := appendFiles.commit(ctx)
	if err != nil {
		return err
	}

	return t.apply(updates, reqs)
}

// ReplaceFiles is actually just an overwrite operation with multiple
// files deleted and added.
//
// TODO: technically, this could be a REPLACE operation but we aren't performing
// any validation here that there are no changes to the underlying data. A REPLACE
// operation is only valid if the data is exactly the same as the previous snapshot.
//
// For now, we'll keep using an overwrite operation.
func (t *Transaction) ReplaceDataFiles(ctx context.Context, filesToDelete, filesToAdd []string, snapshotProps iceberg.Properties) error {
	meta, err := t.txnMeta()
	if err != nil {
		return err
	}
	if len(filesToDelete) == 0 {
		if len(filesToAdd) > 0 {
			return t.AddFiles(ctx, filesToAdd, snapshotProps, false)
		}
	}

	var (
		setToDelete = make(map[string]struct{})
		setToAdd    = make(map[string]struct{})
	)

	for _, f := range filesToDelete {
		setToDelete[f] = struct{}{}
	}

	for _, f := range filesToAdd {
		setToAdd[f] = struct{}{}
	}

	if len(setToDelete) != len(filesToDelete) {
		return errors.New("delete file paths must be unique for ReplaceDataFiles")
	}

	if len(setToAdd) != len(filesToAdd) {
		return errors.New("add file paths must be unique for ReplaceDataFiles")
	}

	s := t.planningSnapshot(meta)
	if s == nil {
		return fmt.Errorf("%w: cannot replace files in a table without an existing snapshot", ErrInvalidOperation)
	}

	fs, err := t.tbl.fsF(ctx)
	if err != nil {
		return err
	}
	wfs, err := requireWriteFileIO(fs)
	if err != nil {
		return err
	}

	markedForDeletion := make([]iceberg.DataFile, 0, len(setToDelete))
	for df, err := range s.dataFiles(fs, nil) {
		if err != nil {
			return err
		}

		if _, ok := setToDelete[df.FilePath()]; ok {
			markedForDeletion = append(markedForDeletion, df)
		}

		if _, ok := setToAdd[df.FilePath()]; ok {
			return fmt.Errorf("cannot add files that are already referenced by table, files: %s", df.FilePath())
		}
	}

	if len(markedForDeletion) != len(setToDelete) {
		return errors.New("cannot delete files that do not belong to the table")
	}

	if meta.NameMapping() == nil {
		nameMapping := meta.CurrentSchema().NameMapping()
		mappingJson, err := json.Marshal(nameMapping)
		if err != nil {
			return err
		}
		err = t.SetProperties(iceberg.Properties{DefaultNameMappingKey: string(mappingJson)})
		if err != nil {
			return err
		}
	}

	commitUUID := uuid.New()
	updater := t.updateSnapshot(wfs, snapshotProps, OpOverwrite).mergeOverwrite(&commitUUID, nil)

	for _, df := range markedForDeletion {
		updater.deleteDataFile(df)
	}

	// SetProperties above may have staged a name mapping, rebuilding t.meta;
	// re-read the current builder through the accessor before parsing files.
	meta, err = t.txnMeta()
	if err != nil {
		return err
	}
	dataFiles, err := filesToDataFiles(ctx, fs, meta, filesToAdd, 1)
	if err != nil {
		return err
	}
	for _, dataFile := range dataFiles {
		updater.appendDataFile(dataFile)
	}

	updates, reqs, err := updater.commit(ctx)
	if err != nil {
		return err
	}

	return t.apply(updates, reqs)
}

type partitionValidationField struct {
	id   int
	name string
}

// partitionValidationPlan contains the metadata needed to validate
// a data file's partition tuple against one partition spec. The field slice
// preserves spec order for deterministic missing-field errors. The plan is
// immutable after construction and can be reused for concurrent validation.
type partitionValidationPlan struct {
	specID           int
	fields           []partitionValidationField
	expectedFieldIDs map[int]struct{}
}

func newPartitionValidationPlan(spec *iceberg.PartitionSpec) *partitionValidationPlan {
	plan := &partitionValidationPlan{
		specID:           spec.ID(),
		fields:           make([]partitionValidationField, 0, spec.NumFields()),
		expectedFieldIDs: make(map[int]struct{}, spec.NumFields()),
	}

	for _, field := range spec.Fields() {
		plan.expectedFieldIDs[field.FieldID] = struct{}{}
		plan.fields = append(plan.fields, partitionValidationField{
			id:   field.FieldID,
			name: field.Name,
		})
	}

	return plan
}

func (p *partitionValidationPlan) validate(df iceberg.DataFile) error {
	partitionData := dataFilePartition(df)

	for _, field := range p.fields {
		if _, ok := partitionData[field.id]; !ok {
			return fmt.Errorf("missing partition value for field id %d (%s)", field.id, field.name)
		}
	}

	if len(partitionData) == len(p.expectedFieldIDs) {
		return nil
	}

	for fieldID := range partitionData {
		if _, ok := p.expectedFieldIDs[fieldID]; !ok {
			return fmt.Errorf("unknown partition field id %d for spec id %d", fieldID, p.specID)
		}
	}

	return nil
}

// validateDataFilesToAdd performs metadata-only validation for caller-provided
// DataFiles and returns a set of paths that passed validation.
//
// requireFirstRowID gates the v3 first_row_id check: callers handing in
// externally-written parquet files (e.g. [Transaction.AddDataFiles]) must set
// first_row_id explicitly because the library cannot fabricate row IDs for
// files it did not write. Library-internal rewrite paths (compaction via
// [withRewriteSemantics]) pass false because the manifest-list writer assigns
// first_row_id via inheritance at write time and the output files carry
// explicit _row_id columns that win on read regardless.
func (t *Transaction) validateDataFilesToAdd(dataFiles []iceberg.DataFile, operation string, requireFirstRowID bool) (map[string]struct{}, error) {
	meta, err := t.txnMeta()
	if err != nil {
		return nil, err
	}

	currentSpec, err := meta.CurrentSpec()
	if err != nil {
		return nil, fmt.Errorf("could not get current partition spec: %w", err)
	}
	if currentSpec == nil {
		return nil, errors.New("could not get current partition spec: no current partition spec found")
	}
	// The spec forbids committing files against a spec with an unknown
	// transform, whoever computed the partition tuple.
	if err := checkNoUnknownTransform(currentSpec); err != nil {
		return nil, fmt.Errorf("%s: %w", operation, err)
	}

	// A data file may target any partition spec registered in the table
	// metadata, not just the default: its manifest is written with that
	// spec (see snapshotProducer.manifestProducer). Cache validation plans
	// since files typically share a handful of specs.
	validationPlansByID := make(map[int32]*partitionValidationPlan)

	setToAdd := make(map[string]struct{}, len(dataFiles))

	for i, df := range dataFiles {
		if df == nil {
			return nil, fmt.Errorf("nil data file at index %d for %s", i, operation)
		}

		path := df.FilePath()
		if path == "" {
			return nil, fmt.Errorf("data file path cannot be empty for %s", operation)
		}

		if _, ok := setToAdd[path]; ok {
			return nil, fmt.Errorf("add data file paths must be unique for %s", operation)
		}
		setToAdd[path] = struct{}{}

		if df.ContentType() != iceberg.EntryContentData {
			return nil, fmt.Errorf("adding files other than data files is not yet implemented: file %s has content type %s for %s", path, df.ContentType(), operation)
		}

		switch df.FileFormat() {
		case iceberg.ParquetFile, iceberg.OrcFile, iceberg.AvroFile:
		default:
			return nil, fmt.Errorf("data file %s has invalid file format %s for %s", path, df.FileFormat(), operation)
		}

		plan, ok := validationPlansByID[df.SpecID()]
		if !ok {
			spec, err := meta.GetSpecByID(int(df.SpecID()))
			if err != nil || spec == nil {
				return nil, fmt.Errorf("data file %s has unregistered partition spec id %d for %s",
					path, df.SpecID(), operation)
			}
			if err := checkNoUnknownTransform(spec); err != nil {
				return nil, fmt.Errorf("data file %s for %s: %w", path, operation, err)
			}
			plan = newPartitionValidationPlan(spec)
			validationPlansByID[df.SpecID()] = plan
		}

		if err := plan.validate(df); err != nil {
			return nil, fmt.Errorf("data file %s has invalid partition data for %s: %w", path, operation, err)
		}

		_, firstRowID, _, _, _ := iceberginternal.BorrowedDataFilePointers(df)
		if requireFirstRowID && meta.formatVersion >= 3 && firstRowID == nil {
			return nil, fmt.Errorf(
				"data file %s is missing first_row_id which is required for v3 tables for %s: use DataFileBuilder.FirstRowID() to set it explicitly",
				path, operation)
		}
	}

	return setToAdd, nil
}

type deletionVectorBlobKey struct {
	path   string
	offset int64
	length int64
}

type deleteFilesToAddSet struct {
	paths    map[string]struct{}
	dvPaths  map[string]struct{}
	dvBlobs  map[deletionVectorBlobKey]struct{}
	dvsByRef map[string]rewriteDeleteFileAddition
}

// validateDeletionVectorToAdd is safe on any DataFile and no-ops on non-DVs.
func validateDeletionVectorToAdd(df iceberg.DataFile, formatVersion int, operation string) error {
	if !IsDeletionVector(df) {
		return nil
	}
	path := df.FilePath()
	if formatVersion < 3 {
		return fmt.Errorf("deletion vector %s requires table format version >= 3 for %s", path, operation)
	}
	if ref := df.ReferencedDataFile(); ref == nil || *ref == "" {
		return fmt.Errorf("deletion vector %s is missing referenced_data_file for %s", path, operation)
	}
	offset := df.ContentOffset()
	if offset == nil {
		return fmt.Errorf("deletion vector %s is missing content_offset for %s", path, operation)
	}
	if *offset < 0 {
		return fmt.Errorf("deletion vector %s has invalid content_offset %d for %s", path, *offset, operation)
	}
	length := df.ContentSizeInBytes()
	if length == nil {
		return fmt.Errorf("deletion vector %s is missing content_size_in_bytes for %s", path, operation)
	}
	if *length <= 0 {
		return fmt.Errorf("deletion vector %s has invalid content_size_in_bytes %d for %s", path, *length, operation)
	}

	return nil
}

// validateDeleteFilesToAdd performs metadata-only validation for delete files
// supplied to a rewrite. Delete files may use an older partition spec, so the
// partition values are checked against the spec carried by each file rather
// than only against the table's current spec.
func (t *Transaction) validateDeleteFilesToAdd(deleteFiles []rewriteDeleteFileAddition, operation string) (*deleteFilesToAddSet, error) {
	meta, err := t.txnMeta()
	if err != nil {
		return nil, err
	}
	setToAdd := &deleteFilesToAddSet{
		paths: make(map[string]struct{}, len(deleteFiles)),
	}
	if len(deleteFiles) == 0 {
		return setToAdd, nil
	}
	if meta.formatVersion < 2 {
		return nil, fmt.Errorf("delete files require table format version >= 2, got v%d", meta.formatVersion)
	}
	validationPlansByID := make(map[int32]*partitionValidationPlan)

	for i, addition := range deleteFiles {
		df := addition.file
		if df == nil {
			return nil, fmt.Errorf("nil delete file at index %d for %s", i, operation)
		}
		if addition.dataSequenceNumber != nil && *addition.dataSequenceNumber < 0 {
			return nil, fmt.Errorf("delete file %s has invalid data sequence number %d for %s",
				df.FilePath(), *addition.dataSequenceNumber, operation)
		}
		path := df.FilePath()
		if path == "" {
			return nil, fmt.Errorf("delete file path cannot be empty for %s", operation)
		}
		_, pathAlreadyAdded := setToAdd.paths[path]
		setToAdd.paths[path] = struct{}{}

		switch df.ContentType() {
		case iceberg.EntryContentPosDeletes, iceberg.EntryContentEqDeletes:
		default:
			return nil, fmt.Errorf("adding non-delete file %s has content type %s for %s", path, df.ContentType(), operation)
		}

		plan, ok := validationPlansByID[df.SpecID()]
		if !ok {
			spec, err := meta.GetSpecByID(int(df.SpecID()))
			if err != nil {
				return nil, fmt.Errorf("delete file %s references unknown partition spec id %d for %s: %w", path, df.SpecID(), operation, err)
			}
			if spec == nil {
				return nil, fmt.Errorf("delete file %s references unknown partition spec id %d for %s", path, df.SpecID(), operation)
			}
			plan = newPartitionValidationPlan(spec)
			validationPlansByID[df.SpecID()] = plan
		}
		if err := plan.validate(df); err != nil {
			return nil, fmt.Errorf("delete file %s has invalid partition data for %s: %w", path, operation, err)
		}

		if df.ContentType() == iceberg.EntryContentEqDeletes {
			eqIDs := df.EqualityFieldIDs()
			if len(eqIDs) == 0 {
				return nil, fmt.Errorf("equality delete file must have non-empty EqualityFieldIDs: %s", path)
			}
			if err := validateRewriteEqualityFieldIDs(meta.schemaList, eqIDs); err != nil {
				return nil, fmt.Errorf("invalid equality delete file %s: %w", path, err)
			}
		}

		if !IsDeletionVector(df) {
			if meta.formatVersion >= 3 && df.ContentType() == iceberg.EntryContentPosDeletes {
				return nil, fmt.Errorf("position delete file %s must be a deletion vector for v%d table for %s",
					path, meta.formatVersion, operation)
			}
			if pathAlreadyAdded {
				if _, ok := setToAdd.dvPaths[path]; ok {
					return nil, fmt.Errorf("delete file path %s cannot identify both a deletion vector container and a regular delete file for %s",
						path, operation)
				}

				return nil, fmt.Errorf("add delete file paths must be unique for %s", operation)
			}

			continue
		}

		if err := validateDeletionVectorToAdd(df, meta.formatVersion, operation); err != nil {
			return nil, err
		}
		// non-nil: validateDeletionVectorToAdd checked all three.
		ref, offset, length := df.ReferencedDataFile(), df.ContentOffset(), df.ContentSizeInBytes()

		blob := deletionVectorBlobKey{path: path, offset: *offset, length: *length}
		if setToAdd.dvBlobs == nil {
			setToAdd.dvBlobs = make(map[deletionVectorBlobKey]struct{}, len(deleteFiles))
		}
		if _, ok := setToAdd.dvBlobs[blob]; ok {
			return nil, fmt.Errorf("deletion vector blob identity must be unique for %s: %s at offset %d with length %d",
				operation, path, *offset, *length)
		}
		if _, ok := setToAdd.dvsByRef[*ref]; ok {
			return nil, fmt.Errorf("deletion vectors to add must reference distinct data files for %s: %s",
				operation, *ref)
		}
		if pathAlreadyAdded {
			if _, ok := setToAdd.dvPaths[path]; !ok {
				return nil, fmt.Errorf("delete file path %s cannot identify both a deletion vector container and a regular delete file for %s",
					path, operation)
			}
		}
		if setToAdd.dvPaths == nil {
			setToAdd.dvPaths = make(map[string]struct{}, len(deleteFiles))
		}
		if setToAdd.dvsByRef == nil {
			setToAdd.dvsByRef = make(map[string]rewriteDeleteFileAddition, len(deleteFiles))
		}
		setToAdd.dvPaths[path] = struct{}{}
		setToAdd.dvBlobs[blob] = struct{}{}
		setToAdd.dvsByRef[*ref] = addition
	}

	return setToAdd, nil
}

// validateRewriteEqualityFieldIDs accepts fields retained only in schema
// history. Rewriting an existing equality delete must preserve its original
// key even after a key column has been dropped from the current schema.
func validateRewriteEqualityFieldIDs(schemas []*iceberg.Schema, fieldIDs []int) error {
	for _, id := range fieldIDs {
		var field iceberg.NestedField
		found := false
		for _, schema := range schemas {
			if candidate, ok := schema.FindFieldByID(id); ok {
				field = candidate
				found = true

				break
			}
		}
		if !found {
			return fmt.Errorf("%w: field ID %d not found in table schema history", iceberg.ErrInvalidSchema, id)
		}
		if isFloatingPointType(field.Type) {
			return fmt.Errorf("%w: equality field ID %d (%s) has unsupported floating-point type %s: floating-point columns cannot be used as equality delete keys",
				iceberg.ErrInvalidSchema, id, field.Name, field.Type)
		}
	}

	return nil
}

type rewriteFileState struct {
	file               iceberg.DataFile
	dataSequenceNumber int64
}

// validateAddedDeletionVectorTargets guarantees that each new DV has the same
// partition as its target, applies to the target by sequence number, and cannot
// suppress a surviving position-delete file that may still apply. File-scoped
// deletes are matched by referenced_data_file or file_path bounds;
// partition-scoped deletes use the same spec/partition tuple fallback as scan
// and rewrite conflict planning.
func validateAddedDeletionVectorTargets(
	dvsByRef map[string]rewriteDeleteFileAddition,
	liveDataFiles map[string]rewriteFileState,
	survivingPositionDeletes []rewriteFileState,
	defaultDeleteSequenceNumber int64,
) error {
	for ref, addition := range dvsByRef {
		dataState, ok := liveDataFiles[ref]
		if !ok {
			return fmt.Errorf("%w: deletion vector references data file that will not exist in the rewritten snapshot: %s",
				ErrInvalidOperation, ref)
		}
		dataFile := dataState.file

		partitionKey, err := canonicalPartitionKey(dataFile.SpecID(), dataFile.Partition())
		if err != nil {
			return fmt.Errorf("building partition key for deletion vector target %s (spec %d): %w",
				ref, dataFile.SpecID(), err)
		}
		dvPartitionKey, err := canonicalPartitionKey(addition.file.SpecID(), addition.file.Partition())
		if err != nil {
			return fmt.Errorf("building partition key for deletion vector %s (spec %d): %w",
				addition.file.FilePath(), addition.file.SpecID(), err)
		}
		if dvPartitionKey != partitionKey {
			return fmt.Errorf("%w: deletion vector %s has partition spec or values that do not match referenced data file %s",
				ErrInvalidOperation, addition.file.FilePath(), ref)
		}

		dvSequenceNumber := defaultDeleteSequenceNumber
		if addition.dataSequenceNumber != nil {
			dvSequenceNumber = *addition.dataSequenceNumber
		}
		if dataState.dataSequenceNumber < 0 {
			return fmt.Errorf("%w: referenced data file %s has no data sequence number",
				ErrInvalidOperation, ref)
		}
		if dataState.dataSequenceNumber > dvSequenceNumber {
			return fmt.Errorf("%w: deletion vector %s with data sequence number %d does not apply to referenced data file %s with data sequence number %d",
				ErrInvalidOperation, addition.file.FilePath(), dvSequenceNumber, ref, dataState.dataSequenceNumber)
		}

		for _, deleteState := range survivingPositionDeletes {
			if deleteState.dataSequenceNumber >= 0 && dataState.dataSequenceNumber > deleteState.dataSequenceNumber {
				continue
			}
			deleteFile := deleteState.file
			if target := referencedDataFilePath(deleteFile); target != "" {
				if target == ref {
					return fmt.Errorf("%w: adding a deletion vector for %s requires replacing all applicable position-delete files; %s would survive",
						ErrInvalidOperation, ref, deleteFile.FilePath())
				}

				continue
			}

			deletePartitionKey, err := canonicalPartitionKey(deleteFile.SpecID(), deleteFile.Partition())
			if err != nil {
				return fmt.Errorf("building partition key for surviving position-delete file %s (spec %d): %w",
					deleteFile.FilePath(), deleteFile.SpecID(), err)
			}
			if deletePartitionKey == partitionKey {
				return fmt.Errorf("%w: adding a deletion vector for %s requires replacing all applicable position-delete files; partition-scoped file %s would survive",
					ErrInvalidOperation, ref, deleteFile.FilePath())
			}
		}
	}

	return nil
}

// WriteOption is an option for methods that operate on pre-built DataFile objects.
type WriteOption func(*dataFileCfg)

type dataFileCfg struct {
	skipAutoNameMapping bool
	skipDuplicateCheck  bool
	rewriteSemantics    bool
	dataSequenceNumber  *int64
}

// withRewriteSemantics marks an overwrite/replace operation as a
// rewrite (compaction) rather than a user-facing overwrite. The
// overwrite producer's default pre-commit conflict validator is
// suppressed in favor of the rewrite-specific validator built by
// [rewriteValidator] and queued onto the transaction's validator list.
//
// Unexported: the only safe way to flip this flag is via the
// [RewriteFiles] builder ([Transaction.NewRewrite]), which always
// pairs the suppression with the matching validator. Direct callers
// of [Transaction.ReplaceFiles] cannot bypass overwrite isolation.
func withRewriteSemantics() WriteOption {
	return func(cfg *dataFileCfg) {
		cfg.rewriteSemantics = true
	}
}

func withDataSequenceNumber(seq int64) WriteOption {
	return func(cfg *dataFileCfg) {
		seqCopy := seq
		cfg.dataSequenceNumber = &seqCopy
	}
}

// WithoutAutoNameMapping disables the automatic setting of the schema name
// mapping in table properties. By default, methods like [Transaction.AddDataFiles]
// and [Transaction.ReplaceDataFilesWithDataFiles] will set the name mapping if
// one does not already exist. This option is useful when working with catalogs
// (such as Databricks Unity Catalog) that reject the name mapping property.
func WithoutAutoNameMapping() WriteOption {
	return func(cfg *dataFileCfg) {
		cfg.skipAutoNameMapping = true
	}
}

// WithoutDuplicateCheck disables the duplicate file path check against
// existing data files in the current snapshot. By default, [Transaction.AddDataFiles]
// scans all manifests to ensure no file being added already exists in the
// table. For tables with many manifests this scan can be expensive because
// each manifest must be read from storage. Use this option when the caller
// can guarantee that the files being added are not already in the table.
func WithoutDuplicateCheck() WriteOption {
	return func(cfg *dataFileCfg) {
		cfg.skipDuplicateCheck = true
	}
}

// ensureNameMapping sets the schema name mapping in table properties if one
// does not already exist. This is extracted as a helper so it can be called
// from any method that accepts WriteOption.
func (t *Transaction) ensureNameMapping() error {
	meta, err := t.txnMeta()
	if err != nil {
		return err
	}

	if meta.NameMapping() == nil {
		nameMapping := meta.CurrentSchema().NameMapping()
		mappingJson, err := json.Marshal(nameMapping)
		if err != nil {
			return err
		}

		return t.SetProperties(iceberg.Properties{DefaultNameMappingKey: string(mappingJson)})
	}

	return nil
}

// AddDataFiles adds pre-built DataFiles to the table without scanning them from storage.
// This is useful for clients who have already constructed DataFile objects with metadata,
// avoiding the need to read files to extract schema and statistics.
//
// Unlike AddFiles, this method does not read files from storage. It validates only metadata
// that can be checked without opening files (for example spec-id and partition field IDs).
//
// Each DataFile may target any partition spec registered in the table metadata,
// identified by its SpecID; files are grouped into one manifest per spec. This
// permits writers to continue producing data under a previous spec after a
// partition evolution, and rewrites that migrate files between specs.
// Unregistered spec ids are rejected.
//
// By default this method automatically sets the schema name mapping in table
// properties if one does not already exist. Pass [WithoutAutoNameMapping] to
// disable this behavior, for example when working with catalogs that reject
// the name mapping property.
//
// Callers are responsible for ensuring each DataFile is valid and consistent with the table.
// Supplying incorrect DataFile metadata can produce an invalid snapshot and break reads.
func (t *Transaction) AddDataFiles(ctx context.Context, dataFiles []iceberg.DataFile, snapshotProps iceberg.Properties, opts ...WriteOption) error {
	if _, err := t.txnMeta(); err != nil {
		return err
	}
	if len(dataFiles) == 0 {
		return nil
	}

	var cfg dataFileCfg
	for _, o := range opts {
		o(&cfg)
	}

	setToAdd, err := t.validateDataFilesToAdd(dataFiles, "AddDataFiles", true)
	if err != nil {
		return err
	}

	fs, err := t.tbl.fsF(ctx)
	if err != nil {
		return err
	}
	wfs, err := requireWriteFileIO(fs)
	if err != nil {
		return err
	}

	if !cfg.skipAutoNameMapping {
		if err := t.ensureNameMapping(); err != nil {
			return err
		}
	}

	// ensureNameMapping above may have staged a name mapping, rebuilding t.meta;
	// read the current builder through the accessor before inspecting the
	// snapshot and creating the producer.
	meta, err := t.txnMeta()
	if err != nil {
		return err
	}

	if !cfg.skipDuplicateCheck {
		if s := t.planningSnapshot(meta); s != nil {
			referenced := make([]string, 0)
			for df, err := range s.dataFiles(fs, nil) {
				if err != nil {
					return err
				}

				if _, ok := setToAdd[df.FilePath()]; ok {
					referenced = append(referenced, df.FilePath())
				}
			}

			if len(referenced) > 0 {
				return fmt.Errorf("cannot add files that are already referenced by table, files: %v", referenced)
			}
		}
	}

	appendFiles := t.appendSnapshotProducer(meta, wfs, snapshotProps)
	for _, df := range dataFiles {
		appendFiles.appendDataFile(df)
	}

	updates, reqs, err := appendFiles.commit(ctx)
	if err != nil {
		return err
	}

	return t.apply(updates, reqs)
}

// ReplaceDataFilesWithDataFiles replaces files using pre-built DataFile objects.
// This avoids scanning files to extract schema and statistics - the caller provides
// DataFile objects directly with all required metadata.
//
// For the files to add, use iceberg.NewDataFileBuilder to construct DataFile objects
// with the appropriate metadata (path, record count, file size, partition values).
//
// This method does not open files. It validates only metadata that can be checked
// without reading file contents.
//
// By default this method automatically sets the schema name mapping in table
// properties if one does not already exist. Pass [WithoutAutoNameMapping] to
// disable this behavior, for example when working with catalogs that reject
// the name mapping property.
//
// Callers are responsible for ensuring each DataFile is valid and consistent with the table.
// Supplying incorrect DataFile metadata can produce an invalid snapshot and break reads.
//
// This is useful when:
//   - Files are written via a separate I/O path and metadata is already known
//   - Avoiding file scanning improves performance or reliability
//   - Working with storage systems where immediate file reads may be unreliable
func (t *Transaction) ReplaceDataFilesWithDataFiles(ctx context.Context, filesToDelete, filesToAdd []iceberg.DataFile, snapshotProps iceberg.Properties, opts ...WriteOption) error {
	meta, err := t.txnMeta()
	if err != nil {
		return err
	}
	if len(filesToDelete) == 0 {
		if len(filesToAdd) > 0 {
			return t.AddDataFiles(ctx, filesToAdd, snapshotProps, opts...)
		}

		return nil
	}

	var cfg dataFileCfg
	for _, o := range opts {
		o(&cfg)
	}

	setToAdd, err := t.validateDataFilesToAdd(filesToAdd, "ReplaceDataFilesWithDataFiles", !cfg.rewriteSemantics)
	if err != nil {
		return err
	}

	setToDelete := make(map[string]struct{}, len(filesToDelete))
	for i, df := range filesToDelete {
		if df == nil {
			return fmt.Errorf("nil data file at index %d for ReplaceDataFilesWithDataFiles", i)
		}

		path := df.FilePath()
		if path == "" {
			return errors.New("delete data file paths must be non-empty for ReplaceDataFilesWithDataFiles")
		}

		if _, ok := setToDelete[path]; ok {
			return errors.New("delete data file paths must be unique for ReplaceDataFilesWithDataFiles")
		}
		setToDelete[path] = struct{}{}
	}

	s := t.planningSnapshot(meta)
	if s == nil {
		return fmt.Errorf("%w: cannot replace files in a table without an existing snapshot", ErrInvalidOperation)
	}

	fs, err := t.tbl.fsF(ctx)
	if err != nil {
		return err
	}
	wfs, err := requireWriteFileIO(fs)
	if err != nil {
		return err
	}

	markedForDeletion := make([]iceberg.DataFile, 0, len(setToDelete))
	for df, err := range s.dataFiles(fs, nil) {
		if err != nil {
			return err
		}

		if _, ok := setToDelete[df.FilePath()]; ok {
			markedForDeletion = append(markedForDeletion, df)
		}

		if _, ok := setToAdd[df.FilePath()]; ok {
			return fmt.Errorf("cannot add files that are already referenced by table, files: %s", df.FilePath())
		}
	}

	if len(markedForDeletion) != len(setToDelete) {
		return errors.New("cannot delete files that do not belong to the table")
	}

	if !cfg.skipAutoNameMapping {
		if err := t.ensureNameMapping(); err != nil {
			return err
		}
	}

	op := OpOverwrite
	if cfg.rewriteSemantics {
		op = OpReplace
	}

	commitUUID := uuid.New()
	updater := t.updateSnapshot(wfs, snapshotProps, op).mergeOverwrite(&commitUUID, nil)
	if cfg.rewriteSemantics {
		updater.setSkipDefaultValidator(true)
	}
	if cfg.dataSequenceNumber != nil {
		updater.setNewDataFilesDataSequenceNumber(*cfg.dataSequenceNumber)
	}

	for _, df := range markedForDeletion {
		updater.deleteDataFile(df)
	}

	for _, df := range filesToAdd {
		updater.appendDataFile(df)
	}

	updates, reqs, err := updater.commit(ctx)
	if err != nil {
		return err
	}

	return t.apply(updates, reqs)
}

// DeleteFileAddition is a delete file added by a rewrite. The data sequence
// number must be copied from the delete files being replaced so the new file
// has the same applicability boundary as the old files. When one rewrite
// contains multiple replacement groups, callers are responsible for assigning
// each output the maximum sequence number of the exact source files it replaces;
// the transaction can validate membership in the overall source sequence set,
// but the grouping is not represented by this API.
type DeleteFileAddition struct {
	File               iceberg.DataFile
	DataSequenceNumber int64
}

// ReplaceFiles atomically replaces data files and removes associated delete files
// in a single snapshot. This is the commit primitive for compaction: old data files
// are replaced with new (compacted) data files, and delete files that are fully
// applied are removed.
func (t *Transaction) ReplaceFiles(ctx context.Context, dataFilesToDelete, dataFilesToAdd, deleteFilesToRemove []iceberg.DataFile, snapshotProps iceberg.Properties, opts ...WriteOption) error {
	return t.replaceFiles(ctx, dataFilesToDelete, dataFilesToAdd, deleteFilesToRemove, nil, nil, snapshotProps, opts...)
}

// ReplaceFilesWithDeleteFiles atomically replaces data files, adds new delete
// files, and removes superseded delete files in one snapshot. Every added delete
// file must carry the data sequence number of the delete files it replaces.
// This is intended for rewrites that change delete-file representation, such as
// rewriting position deletes into deletion vectors.
func (t *Transaction) ReplaceFilesWithDeleteFiles(ctx context.Context, dataFilesToDelete, dataFilesToAdd, deleteFilesToRemove []iceberg.DataFile, deleteFilesToAdd []DeleteFileAddition, snapshotProps iceberg.Properties, opts ...WriteOption) error {
	additions := make([]rewriteDeleteFileAddition, len(deleteFilesToAdd))
	for i, addition := range deleteFilesToAdd {
		seq := addition.DataSequenceNumber
		additions[i] = rewriteDeleteFileAddition{
			file:               addition.File,
			dataSequenceNumber: &seq,
		}
	}

	return t.replaceFiles(ctx, dataFilesToDelete, dataFilesToAdd, deleteFilesToRemove, nil, additions, snapshotProps, opts...)
}

func (t *Transaction) replaceFiles(ctx context.Context, dataFilesToDelete, dataFilesToAdd, deleteFilesToRemove, autoDeleteFilesToRemove []iceberg.DataFile, deleteFilesToAdd []rewriteDeleteFileAddition, snapshotProps iceberg.Properties, opts ...WriteOption) error {
	meta, err := t.txnMeta()
	if err != nil {
		return err
	}
	var cfg dataFileCfg
	for _, o := range opts {
		o(&cfg)
	}

	nextSequenceNumber := meta.nextSequenceNumber()
	if cfg.dataSequenceNumber != nil && *cfg.dataSequenceNumber > nextSequenceNumber {
		return fmt.Errorf("%w: data sequence number %d cannot be greater than new snapshot sequence number %d",
			ErrInvalidOperation, *cfg.dataSequenceNumber, nextSequenceNumber)
	}

	// Delegate data file replacement to existing logic.
	if len(deleteFilesToRemove) == 0 && len(autoDeleteFilesToRemove) == 0 && len(deleteFilesToAdd) == 0 {
		return t.ReplaceDataFilesWithDataFiles(ctx, dataFilesToDelete, dataFilesToAdd, snapshotProps, opts...)
	}
	if len(deleteFilesToAdd) > 0 && len(deleteFilesToRemove) == 0 {
		return fmt.Errorf("%w: adding delete files requires replacing at least one existing delete file",
			ErrInvalidOperation)
	}

	setToAdd, err := t.validateDataFilesToAdd(dataFilesToAdd, "ReplaceFiles", !cfg.rewriteSemantics)
	if err != nil {
		return err
	}
	setDeleteFilesToAdd, err := t.validateDeleteFilesToAdd(deleteFilesToAdd, "ReplaceFiles")
	if err != nil {
		return err
	}
	for path := range setDeleteFilesToAdd.paths {
		if _, ok := setToAdd[path]; ok {
			return fmt.Errorf("data and delete file paths must be unique for ReplaceFiles: %s", path)
		}
	}

	setToDelete := make(map[string]struct{}, len(dataFilesToDelete))
	for i, df := range dataFilesToDelete {
		if df == nil {
			return fmt.Errorf("nil data file at index %d for ReplaceFiles", i)
		}
		path := df.FilePath()
		if path == "" {
			return errors.New("delete data file paths must be non-empty for ReplaceFiles")
		}
		if _, ok := setToDelete[path]; ok {
			return errors.New("delete data file paths must be unique for ReplaceFiles")
		}
		setToDelete[path] = struct{}{}
	}

	setDeleteFilesToRemove := make(map[string]struct{}, len(deleteFilesToRemove))
	dvsToRemoveByRef := make(map[string]iceberg.DataFile, len(deleteFilesToRemove))
	for i, df := range deleteFilesToRemove {
		if df == nil {
			return fmt.Errorf("nil delete file at index %d for ReplaceFiles", i)
		}
		path := df.FilePath()
		if path == "" {
			return errors.New("delete file paths must be non-empty for ReplaceFiles")
		}
		if IsDeletionVector(df) {
			ref := iceberginternal.BorrowedDataFileReferencedDataFile(df)
			if ref == nil {
				return errors.New("deletion vector to remove is missing referenced_data_file for ReplaceFiles")
			}
			if _, ok := dvsToRemoveByRef[*ref]; ok {
				return errors.New("deletion vectors to remove must reference distinct data files for ReplaceFiles")
			}
			dvsToRemoveByRef[*ref] = df

			continue
		}
		if _, ok := setDeleteFilesToRemove[path]; ok {
			return errors.New("delete file paths must be unique for ReplaceFiles")
		}
		setDeleteFilesToRemove[path] = struct{}{}
	}
	autoSetDeleteFilesToRemove := make(map[string]struct{}, len(autoDeleteFilesToRemove))
	autoDVsToRemoveByRef := make(map[string]iceberg.DataFile, len(autoDeleteFilesToRemove))
	for i, df := range autoDeleteFilesToRemove {
		if df == nil {
			return fmt.Errorf("nil automatic delete file at index %d for ReplaceFiles", i)
		}
		path := df.FilePath()
		if path == "" {
			return errors.New("automatic delete file paths must be non-empty for ReplaceFiles")
		}
		if IsDeletionVector(df) {
			ref := df.ReferencedDataFile()
			if ref == nil {
				return errors.New("automatic deletion vector to remove is missing referenced_data_file for ReplaceFiles")
			}
			if _, ok := dvsToRemoveByRef[*ref]; ok {
				return errors.New("delete vectors to remove must reference distinct data files for ReplaceFiles")
			}
			if _, ok := autoDVsToRemoveByRef[*ref]; ok {
				return errors.New("automatic deletion vectors to remove must reference distinct data files for ReplaceFiles")
			}
			autoDVsToRemoveByRef[*ref] = df

			continue
		}
		if _, ok := setDeleteFilesToRemove[path]; ok {
			return errors.New("delete file paths must be unique for ReplaceFiles")
		}
		if _, ok := autoSetDeleteFilesToRemove[path]; ok {
			return errors.New("automatic delete file paths must be unique for ReplaceFiles")
		}
		autoSetDeleteFilesToRemove[path] = struct{}{}
	}

	s := t.planningSnapshot(meta)
	if s == nil {
		return fmt.Errorf("%w: cannot replace files in a table without an existing snapshot", ErrInvalidOperation)
	}

	fs, err := t.tbl.fsF(ctx)
	if err != nil {
		return err
	}
	wfs, err := requireWriteFileIO(fs)
	if err != nil {
		return err
	}

	// Scan all entries (data + delete files) in a single pass to validate
	// that all files to delete/remove actually exist in the table.
	markedDataForDeletion := make([]iceberg.DataFile, 0, len(setToDelete))
	markedDeleteForRemoval := make([]iceberg.DataFile, 0, len(setDeleteFilesToRemove))
	markedAutoDeleteForRemoval := make([]iceberg.DataFile, 0, len(autoSetDeleteFilesToRemove))
	markedDVsForRemoval := make(map[string]iceberg.DataFile, len(dvsToRemoveByRef))
	removedDeleteSequenceNumbers := make([]int64, 0, len(deleteFilesToRemove))
	removedDeleteContents := make(map[iceberg.ManifestEntryContent]struct{})
	liveDataFiles := make(map[string]rewriteFileState)
	survivingPositionDeletes := make([]rewriteFileState, 0)
	for entry, err := range s.entries(fs, -1, false) {
		if err != nil {
			return err
		}
		df := entry.DataFile()
		path := df.FilePath()
		isData := df.ContentType() == iceberg.EntryContentData
		// A DELETED entry is a file already removed from the table. It must not
		// count as found when removing files, but it still blocks re-adding its path.
		isLive := entry.Status() != iceberg.EntryStatusDELETED
		if isData && isLive {
			liveDataFiles[path] = rewriteFileState{file: df, dataSequenceNumber: entry.SequenceNum()}
			if _, ok := setToDelete[path]; ok {
				markedDataForDeletion = append(markedDataForDeletion, df)
			}
		}
		if !isData && isLive {
			if _, ok := setDeleteFilesToRemove[path]; ok {
				markedDeleteForRemoval = append(markedDeleteForRemoval, df)
				if seq := entry.SequenceNum(); seq >= 0 {
					removedDeleteSequenceNumbers = append(removedDeleteSequenceNumbers, seq)
					removedDeleteContents[df.ContentType()] = struct{}{}
				} else {
					return fmt.Errorf("delete file %s has no data sequence number in the current snapshot", path)
				}
			} else if _, ok := autoSetDeleteFilesToRemove[path]; ok {
				markedAutoDeleteForRemoval = append(markedAutoDeleteForRemoval, df)
			} else if ref := iceberginternal.BorrowedDataFileReferencedDataFile(df); IsDeletionVector(df) && ref != nil {
				// Match the DV's path too, not only its data file. Otherwise an old DV that
				// was already replaced would match the newer DV, which holds more deletes.
				if want, ok := dvsToRemoveByRef[*ref]; ok && want.FilePath() == path {
					markedDVsForRemoval[*ref] = df
					if seq := entry.SequenceNum(); seq >= 0 {
						removedDeleteSequenceNumbers = append(removedDeleteSequenceNumbers, seq)
						removedDeleteContents[df.ContentType()] = struct{}{}
					} else {
						return fmt.Errorf("deletion vector %s has no data sequence number in the current snapshot", path)
					}
				} else if want, ok := autoDVsToRemoveByRef[*ref]; ok && want.FilePath() == path {
					markedDVsForRemoval[*ref] = df
				}
			}
		}
		if _, ok := setToAdd[path]; ok {
			return fmt.Errorf("cannot add files that are already referenced by table, files: %s", path)
		}
		if _, ok := setDeleteFilesToAdd.paths[path]; ok {
			if _, isDeletionVector := setDeleteFilesToAdd.dvPaths[path]; !isDeletionVector {
				return fmt.Errorf("cannot add files that are already referenced by table, files: %s", path)
			}
		}
		if _, ok := setDeleteFilesToAdd.dvPaths[path]; ok && isLive {
			if !IsDeletionVector(df) {
				return fmt.Errorf("cannot add deletion vector container path already referenced by a non-DV file: %s", path)
			}
			if offset, length := df.ContentOffset(), df.ContentSizeInBytes(); offset != nil && length != nil {
				blob := deletionVectorBlobKey{path: path, offset: *offset, length: *length}
				if _, duplicate := setDeleteFilesToAdd.dvBlobs[blob]; duplicate {
					return fmt.Errorf("cannot add deletion vector blob already referenced by table: %s at offset %d with length %d",
						path, *offset, *length)
				}
			}
		}

		if !isLive || isData {
			continue
		}
		if IsDeletionVector(df) {
			ref := df.ReferencedDataFile()
			if ref == nil {
				continue
			}
			if _, addingReplacement := setDeleteFilesToAdd.dvsByRef[*ref]; !addingReplacement {
				continue
			}
			if _, explicitlyRemoved := dvsToRemoveByRef[*ref]; explicitlyRemoved {
				continue
			}
			if _, automaticallyRemoved := autoDVsToRemoveByRef[*ref]; automaticallyRemoved {
				continue
			}

			return fmt.Errorf("%w: deletion vector for data file %s already exists and must be replaced",
				ErrInvalidOperation, *ref)
		}
		if df.ContentType() == iceberg.EntryContentPosDeletes {
			if _, removed := setDeleteFilesToRemove[path]; !removed {
				if _, automaticallyRemoved := autoSetDeleteFilesToRemove[path]; !automaticallyRemoved {
					survivingPositionDeletes = append(survivingPositionDeletes, rewriteFileState{
						file: df, dataSequenceNumber: entry.SequenceNum(),
					})
				}
			}
		}
	}

	addedDataSequenceNumber := nextSequenceNumber
	if cfg.dataSequenceNumber != nil {
		addedDataSequenceNumber = *cfg.dataSequenceNumber
	}
	for _, df := range dataFilesToAdd {
		liveDataFiles[df.FilePath()] = rewriteFileState{
			file: df, dataSequenceNumber: addedDataSequenceNumber,
		}
	}
	for path := range setToDelete {
		delete(liveDataFiles, path)
	}
	if len(markedDataForDeletion) != len(setToDelete) {
		return errors.New("cannot delete data files that do not belong to the table")
	}
	if len(markedDeleteForRemoval) != len(setDeleteFilesToRemove) {
		return errors.New("cannot remove delete files that do not belong to the table")
	}
	if len(markedAutoDeleteForRemoval) != len(autoSetDeleteFilesToRemove) {
		return errors.New("cannot remove automatic delete files that do not belong to the table")
	}
	// markedDVsForRemoval has one slot per data file, so equal counts mean
	// every requested DV was found live in the table.
	if len(markedDVsForRemoval) != len(dvsToRemoveByRef)+len(autoDVsToRemoveByRef) {
		return errors.New("cannot remove deletion vectors that do not belong to the table")
	}

	deleteSequenceNumber := int64(-1)
	if len(deleteFilesToAdd) > 0 {
		if len(removedDeleteSequenceNumbers) == 0 {
			return fmt.Errorf("%w: added delete files must replace existing delete files",
				ErrInvalidOperation)
		}
		if len(removedDeleteContents) != 1 {
			return fmt.Errorf("%w: delete-file rewrites cannot combine position and equality delete files",
				ErrInvalidOperation)
		}

		maxSequenceNumber := removedDeleteSequenceNumbers[0]
		var sourceContent iceberg.ManifestEntryContent
		for content := range removedDeleteContents {
			sourceContent = content
		}
		for _, seq := range removedDeleteSequenceNumbers[1:] {
			maxSequenceNumber = max(maxSequenceNumber, seq)
		}
		deleteSequenceNumber = maxSequenceNumber
		removedDeleteSeqs := make(map[int64]struct{}, len(removedDeleteSequenceNumbers))
		for _, seq := range removedDeleteSequenceNumbers {
			removedDeleteSeqs[seq] = struct{}{}
		}
		if sourceContent == iceberg.EntryContentEqDeletes && len(removedDeleteSeqs) != 1 {
			return fmt.Errorf("%w: equality delete files with different data sequence numbers cannot be merged",
				ErrInvalidOperation)
		}
		for _, addition := range deleteFilesToAdd {
			if addition.file.ContentType() != sourceContent {
				return fmt.Errorf("%w: cannot rewrite %s delete files as %s delete files",
					ErrInvalidOperation, sourceContent, addition.file.ContentType())
			}
			if addition.dataSequenceNumber != nil {
				if sourceContent == iceberg.EntryContentEqDeletes {
					if *addition.dataSequenceNumber != maxSequenceNumber {
						return fmt.Errorf("%w: equality delete file %s has data sequence number %d, expected %d",
							ErrInvalidOperation, addition.file.FilePath(), *addition.dataSequenceNumber, maxSequenceNumber)
					}
				} else if _, ok := removedDeleteSeqs[*addition.dataSequenceNumber]; !ok {
					return fmt.Errorf("%w: delete file %s has data sequence number %d, expected one of the replaced delete sequence numbers",
						ErrInvalidOperation, addition.file.FilePath(), *addition.dataSequenceNumber)
				}
			}
		}
	}
	if err := validateAddedDeletionVectorTargets(setDeleteFilesToAdd.dvsByRef, liveDataFiles,
		survivingPositionDeletes, deleteSequenceNumber); err != nil {
		return err
	}

	var rewriteValidationFiles []iceberg.DataFile
	var referencedDataFilePaths []string
	if cfg.rewriteSemantics && len(deleteFilesToAdd) > 0 {
		rewriteValidationFiles = slices.Clone(dataFilesToDelete)
		for ref := range setDeleteFilesToAdd.dvsByRef {
			if _, added := setToAdd[ref]; added {
				continue
			}

			state, ok := liveDataFiles[ref]
			if !ok {
				return fmt.Errorf("%w: deletion vector target %s is not live in the rewrite",
					ErrInvalidOperation, ref)
			}
			rewriteValidationFiles = append(rewriteValidationFiles, state.file)
			referencedDataFilePaths = append(referencedDataFilePaths, ref)
		}
		slices.Sort(referencedDataFilePaths)
	}

	if !cfg.skipAutoNameMapping {
		if err := t.ensureNameMapping(); err != nil {
			return err
		}
	}

	op := OpOverwrite
	if cfg.rewriteSemantics {
		op = OpReplace
	}

	commitUUID := uuid.New()
	updater := t.updateSnapshot(wfs, snapshotProps, op).mergeOverwrite(&commitUUID, nil)
	if cfg.rewriteSemantics {
		updater.setSkipDefaultValidator(true)
	}
	if cfg.dataSequenceNumber != nil {
		updater.setNewDataFilesDataSequenceNumber(*cfg.dataSequenceNumber)
	}

	for _, df := range markedDataForDeletion {
		updater.deleteDataFile(df)
	}
	for _, df := range dataFilesToAdd {
		updater.appendDataFile(df)
	}
	for _, addition := range deleteFilesToAdd {
		sequenceNumber := deleteSequenceNumber
		if addition.dataSequenceNumber != nil {
			sequenceNumber = *addition.dataSequenceNumber
		}
		updater.appendDeleteFileWithDataSequenceNumber(addition.file, sequenceNumber)
	}
	for _, df := range markedDeleteForRemoval {
		updater.removeDeleteFile(df)
	}
	for _, df := range markedAutoDeleteForRemoval {
		updater.removeDeleteFile(df)
	}
	for _, df := range markedDVsForRemoval {
		updater.removeDeletionVector(df)
	}

	updates, reqs, err := updater.commit(ctx)
	if err != nil {
		return err
	}

	if err := t.apply(updates, reqs); err != nil {
		return err
	}
	if len(rewriteValidationFiles) > 0 {
		t.addValidator(rewriteValidatorWithReferencedDataFiles(rewriteValidationFiles, referencedDataFilePaths))
	}

	return nil
}

type AddFilesOption func(addFilesOp *addFilesOperation)

type addFilesOperation struct {
	concurrency int
}

// WithAddFilesConcurrency overwrites the default concurrency for add files operation.
// Default: runtime.GOMAXPROCS(0)
func WithAddFilesConcurrency(concurrency int) AddFilesOption {
	return func(op *addFilesOperation) {
		if concurrency > 0 {
			op.concurrency = concurrency
		}
	}
}

func (t *Transaction) AddFiles(ctx context.Context, filePaths []string, snapshotProps iceberg.Properties, ignoreDuplicates bool, opts ...AddFilesOption) error {
	meta, err := t.txnMeta()
	if err != nil {
		return err
	}
	addFilesOp := addFilesOperation{
		concurrency: runtime.GOMAXPROCS(0),
	}
	for _, apply := range opts {
		apply(&addFilesOp)
	}

	set := make(map[string]struct{}, len(filePaths))
	for _, filePath := range filePaths {
		if _, ok := set[filePath]; ok {
			return errors.New("file paths must be unique for AddFiles")
		}
		set[filePath] = struct{}{}
	}

	fs, err := t.tbl.fsF(ctx)
	if err != nil {
		return err
	}
	wfs, err := requireWriteFileIO(fs)
	if err != nil {
		return err
	}

	if !ignoreDuplicates {
		if s := t.planningSnapshot(meta); s != nil {
			referenced := make([]string, 0)
			for df, err := range s.dataFiles(fs, nil) {
				if err != nil {
					return err
				}

				if _, ok := set[df.FilePath()]; ok {
					referenced = append(referenced, df.FilePath())
				}
			}
			if len(referenced) > 0 {
				return fmt.Errorf("cannot add files that are already referenced by table, files: %v", referenced)
			}
		}
	}

	if meta.NameMapping() == nil {
		nameMapping := meta.CurrentSchema().NameMapping()
		mappingJson, err := json.Marshal(nameMapping)
		if err != nil {
			return err
		}
		err = t.SetProperties(iceberg.Properties{DefaultNameMappingKey: string(mappingJson)})
		if err != nil {
			return err
		}
	}

	// SetProperties above may have staged a name mapping, rebuilding t.meta;
	// re-read the current builder through the accessor before creating the
	// producer or parsing files.
	meta, err = t.txnMeta()
	if err != nil {
		return err
	}
	updater := t.appendSnapshotProducer(meta, wfs, snapshotProps)

	dataFiles, err := filesToDataFiles(ctx, fs, meta, filePaths, addFilesOp.concurrency)
	if err != nil {
		return err
	}
	for _, dataFile := range dataFiles {
		updater.appendDataFile(dataFile)
	}

	updates, reqs, err := updater.commit(ctx)
	if err != nil {
		return err
	}

	return t.apply(updates, reqs)
}

// OverwriteTable overwrites the table data using an Arrow Table.
//
// An optional filter (see WithOverwriteFilter) determines which existing data to delete or rewrite:
//   - If filter is nil or AlwaysTrue, all existing data files are deleted and replaced with new data.
//   - If a filter is provided, it acts as a row-level predicate on existing data:
//   - Files where all rows match the filter (strict match) are completely deleted
//   - Files where some rows match and others don't (partial match) are rewritten to keep only non-matching rows
//   - Files where no rows match the filter are kept unchanged
//
// The filter uses both inclusive and strict metrics evaluators on file statistics to classify files:
//   - Inclusive evaluator identifies candidate files that may contain matching rows
//   - Strict evaluator determines if all rows in a file must match the filter
//   - Files that pass inclusive but not strict evaluation are rewritten with filtered data
//
// New data from the provided table is written to the table regardless of the filter.
//
// The batchSize parameter refers to the batch size for reading the input data, not the batch size for writes.
// The concurrency parameter controls the level of parallelism for manifest processing and file rewriting and
// can be overridden using the WithOverwriteConcurrency option.
// If concurrency <= 0, defaults to runtime.GOMAXPROCS(0).
func (t *Transaction) OverwriteTable(ctx context.Context, tbl arrow.Table, batchSize int64, snapshotProps iceberg.Properties, opts ...OverwriteOption) error {
	rdr := array.NewTableReader(tbl, batchSize)
	defer rdr.Release()

	return t.Overwrite(ctx, rdr, snapshotProps, opts...)
}

type overwriteOperation struct {
	concurrency   int
	filter        iceberg.BooleanExpression
	caseSensitive bool
}

// OverwriteOption applies options to overwrite operations
type OverwriteOption func(op *overwriteOperation)

// WithOverwriteConcurrency overwrites the default concurrency for overwrite operations.
// Default: runtime.GOMAXPROCS(0)
func WithOverwriteConcurrency(concurrency int) OverwriteOption {
	return func(op *overwriteOperation) {
		if concurrency <= 0 {
			op.concurrency = runtime.GOMAXPROCS(0)

			return
		}
		op.concurrency = concurrency
	}
}

// WithOverwriteFilter overwrites the default deletion filter on overwrite operations.
// Default: iceberg.AlwaysTrue
func WithOverwriteFilter(filter iceberg.BooleanExpression) OverwriteOption {
	return func(op *overwriteOperation) {
		op.filter = filter
	}
}

// WithOverwriteCaseInsensitive overwrites the default case sensitivity that applies on the binding of the filter.
// Default: case sensitive
// Note that the sensitivity only applies to the field name and not the evaluation of the literals on string fields.
func WithOverwriteCaseInsensitive() OverwriteOption {
	return func(op *overwriteOperation) {
		op.caseSensitive = false
	}
}

// Overwrite overwrites the table data using a RecordReader.
//
// An optional filter (see WithOverwriteFilter) determines which existing data to delete or rewrite:
//   - If filter is nil or AlwaysTrue, all existing data files are deleted and replaced with new data.
//   - If a filter is provided, it acts as a row-level predicate on existing data:
//   - Files where all rows match the filter (strict match) are completely deleted
//   - Files where some rows match and others don't (partial match) are rewritten to keep only non-matching rows
//   - Files where no rows match the filter are kept unchanged
//
// The filter uses both inclusive and strict metrics evaluators on file statistics to classify files:
//   - Inclusive evaluator identifies candidate files that may contain matching rows
//   - Strict evaluator determines if all rows in a file must match the filter
//   - Files that pass inclusive but not strict evaluation are rewritten with filtered data
//
// New data from the provided RecordReader is written to the table regardless of the filter.
//
// The concurrency parameter controls the level of parallelism for manifest processing and file rewriting and
// can be overridden using the WithOverwriteConcurrency option.
// If concurrency <= 0, defaults to runtime.GOMAXPROCS(0).
func (t *Transaction) Overwrite(ctx context.Context, rdr array.RecordReader, snapshotProps iceberg.Properties, opts ...OverwriteOption) error {
	if _, err := t.txnMeta(); err != nil {
		return err
	}
	overwrite := overwriteOperation{
		concurrency:   runtime.GOMAXPROCS(0),
		filter:        iceberg.AlwaysTrue{},
		caseSensitive: true,
	}
	for _, apply := range opts {
		apply(&overwrite)
	}

	updater, wfs, err := t.performCopyOnWriteDeletion(ctx, OpOverwrite, snapshotProps, overwrite.filter, overwrite.caseSensitive, overwrite.concurrency)
	if err != nil {
		return err
	}

	// Re-read through the accessor: performCopyOnWriteDeletion may have staged a
	// name mapping (which rebuilds t.meta), and the records must be written
	// against that current builder.
	meta, err := t.txnMeta()
	if err != nil {
		return err
	}

	itr := recordsToDataFiles(ctx, t.tbl.Location(), meta, recordWritingArgs{
		sc:        rdr.Schema(),
		itr:       array.IterFromReader(rdr),
		fs:        wfs,
		writeUUID: &updater.commitUuid,
	})

	for df, err := range itr {
		if err != nil {
			return err
		}
		updater.appendDataFile(df)
	}

	// Diagnostic: warn when an overwrite writes fewer rows than it deletes.
	// This is not always a bug (the caller may intentionally reduce data),
	// but it surfaces silent row loss from broken RecordReader adapters
	// or iterator issues that would otherwise go unnoticed (#860).
	var addedRows, deletedRows int64
	for _, df := range updater.addedFiles {
		addedRows += df.Count()
	}
	for _, df := range updater.deletedFiles {
		deletedRows += df.Count()
	}
	if deletedRows > 0 && addedRows < deletedRows {
		slog.Warn("Overwrite produced fewer rows than deleted",
			"added_rows", addedRows,
			"deleted_rows", deletedRows,
			"delta", addedRows-deletedRows)
	}

	updates, reqs, err := updater.commit(ctx)
	if err != nil {
		return err
	}

	return t.apply(updates, reqs)
}

func (t *Transaction) performCopyOnWriteDeletion(ctx context.Context, operation Operation, snapshotProps iceberg.Properties, filter iceberg.BooleanExpression, caseSensitive bool, concurrency int) (*snapshotProducer, io.WriteFileIO, error) {
	meta, err := t.txnMeta()
	if err != nil {
		return nil, nil, err
	}
	fs, err := t.tbl.fsF(ctx)
	if err != nil {
		return nil, nil, err
	}
	wfs, err := requireWriteFileIO(fs)
	if err != nil {
		return nil, nil, err
	}

	if meta.NameMapping() == nil {
		nameMapping := meta.CurrentSchema().NameMapping()
		mappingJson, err := json.Marshal(nameMapping)
		if err != nil {
			return nil, nil, err
		}
		err = t.SetProperties(iceberg.Properties{DefaultNameMappingKey: string(mappingJson)})
		if err != nil {
			return nil, nil, err
		}
	}

	commitUUID := uuid.New()
	updater := t.updateSnapshot(wfs, snapshotProps, operation).mergeOverwrite(&commitUUID, filter)
	updater.setManifestConcurrency(concurrency)

	filesToDelete, filesToRewrite, fileSeqByPath, err := t.classifyFilesForDeletions(ctx, fs, filter, caseSensitive, concurrency)
	if err != nil {
		return nil, nil, err
	}

	for _, df := range filesToDelete {
		updater.deleteDataFile(df)
	}

	if len(filesToRewrite) > 0 {
		if err := t.rewriteFilesWithFilter(ctx, fs, updater, filesToRewrite, fileSeqByPath, filter, caseSensitive, concurrency); err != nil {
			return nil, nil, err
		}
	}

	return updater, wfs, nil
}

func (t *Transaction) performMergeOnReadDeletion(ctx context.Context, snapshotProps iceberg.Properties, filter iceberg.BooleanExpression, caseSensitive bool, concurrency int) (*snapshotProducer, error) {
	meta, err := t.txnMeta()
	if err != nil {
		return nil, err
	}
	fs, err := t.tbl.fsF(ctx)
	if err != nil {
		return nil, err
	}
	wfs, err := requireWriteFileIO(fs)
	if err != nil {
		return nil, err
	}

	if meta.NameMapping() == nil {
		nameMapping := meta.CurrentSchema().NameMapping()
		mappingJson, err := json.Marshal(nameMapping)
		if err != nil {
			return nil, err
		}
		err = t.SetProperties(iceberg.Properties{DefaultNameMappingKey: string(mappingJson)})
		if err != nil {
			return nil, err
		}
	}

	commitUUID := uuid.New()
	updater := t.updateSnapshot(wfs, snapshotProps, OpDelete).mergeOverwrite(&commitUUID, filter)
	updater.setManifestConcurrency(concurrency)

	filesToDelete, withPartialDeletions, _, err := t.classifyFilesForDeletions(ctx, fs, filter, caseSensitive, concurrency)
	if err != nil {
		return nil, err
	}

	for _, df := range filesToDelete {
		updater.deleteDataFile(df)
	}

	if len(withPartialDeletions) > 0 {
		referenced, err := t.writePositionDeletesForFiles(ctx, fs, updater, withPartialDeletions, filter, caseSensitive, concurrency, commitUUID)
		if err != nil {
			return nil, err
		}

		// Verify the data files these position deletes target still exist on
		// the branch head at commit time. The overwriteFiles producer only
		// validates partition-filter overlap; without this check a concurrent
		// commit that removed a referenced data file would silently orphan the
		// deletes (they would apply to rewritten data or dangle). Mirrors
		// RowDelta.validate: collect referenced data-file paths and run
		// validateDataFilesExist unconditionally (no isolation gating).
		if len(referenced) > 0 {
			t.addValidator(func(cc *conflictContext) error {
				return validateDataFilesExist(cc, referenced)
			})
		}
	}

	return updater, nil
}

type DeleteOption func(deleteOp *deleteOperation)

type deleteOperation struct {
	caseSensitive bool
	concurrency   int
}

// WithDeleteConcurrency overwrites the default concurrency for delete operations.
// Default: runtime.GOMAXPROCS(0)
func WithDeleteConcurrency(concurrency int) DeleteOption {
	return func(op *deleteOperation) {
		if concurrency <= 0 {
			op.concurrency = runtime.GOMAXPROCS(0)

			return
		}
		op.concurrency = concurrency
	}
}

// WithDeleteCaseInsensitive changes the binding of the filter to be case insensitive instead of the
// Default: case sensitive
// Note that the sensitivity only applies to the field name and not the evaluation of the literals on string fields.
func WithDeleteCaseInsensitive() DeleteOption {
	return func(deleteOp *deleteOperation) {
		deleteOp.caseSensitive = false
	}
}

// Delete deletes records matching the provided filter.
//
// The provided filter acts as a row-level predicate on existing data:
//   - Files where all rows match the filter (strict match) are completely deleted
//   - Files where some rows match and others don't (partial match) are rewritten to keep only non-matching rows
//   - Files where no rows match the filter are kept unchanged
//
// The filter uses both inclusive and strict metrics evaluators on file statistics to classify files:
//   - Inclusive evaluator identifies candidate files that may contain matching rows
//   - Strict evaluator determines if all rows in a file must match the filter
//   - Files that pass inclusive but not strict evaluation are rewritten with filtered data
//
// The concurrency parameter controls the level of parallelism for manifest processing and file rewriting and
// can be overridden using the WithOverwriteConcurrency option. Defaults to runtime.GOMAXPROCS(0).
func (t *Transaction) Delete(ctx context.Context, filter iceberg.BooleanExpression, snapshotProps iceberg.Properties, opts ...DeleteOption) (err error) {
	meta, err := t.txnMeta()
	if err != nil {
		return err
	}
	deleteOp := deleteOperation{
		concurrency:   runtime.GOMAXPROCS(0),
		caseSensitive: true,
	}
	for _, apply := range opts {
		apply(&deleteOp)
	}

	var updater *snapshotProducer
	writeDeleteMode := WriteDeleteModeDefault
	// Only copy on write is supported on v1 so we ignore any override to the write delete mode unless the version is
	// 2 and up
	if meta.formatVersion > 1 {
		writeDeleteMode = meta.props.Get(WriteDeleteModeKey, WriteDeleteModeDefault)
	}
	switch writeDeleteMode {
	case WriteModeCopyOnWrite:
		updater, _, err = t.performCopyOnWriteDeletion(ctx, OpDelete, snapshotProps, filter, deleteOp.caseSensitive, deleteOp.concurrency)
		if err != nil {
			return err
		}
	case WriteModeMergeOnRead:
		updater, err = t.performMergeOnReadDeletion(ctx, snapshotProps, filter, deleteOp.caseSensitive, deleteOp.concurrency)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported write mode: '%s'", writeDeleteMode)
	}

	updates, reqs, err := updater.commit(ctx)
	if err != nil {
		return err
	}

	return t.apply(updates, reqs)
}

// classifyFilesForDeletions classifies existing data files based on the provided filter.
// Returns files to delete completely, files to rewrite partially, the
// per-file file_sequence_number map for rewrite candidates (used to
// synthesize row-lineage columns when reading), and any error.
func (t *Transaction) classifyFilesForDeletions(ctx context.Context, fs io.IO, filter iceberg.BooleanExpression, caseSensitive bool, concurrency int) (filesToDelete, filesWithPartialDeletions []iceberg.DataFile, fileSeqByPath map[string]*int64, err error) {
	meta, err := t.txnMeta()
	if err != nil {
		return nil, nil, nil, err
	}

	s := t.planningSnapshot(meta)
	if s == nil {
		return nil, nil, nil, nil
	}

	if filter == nil || filter.Equals(iceberg.AlwaysTrue{}) {
		for df, err := range s.dataFiles(fs, nil) {
			if err != nil {
				return nil, nil, nil, err
			}
			if df.ContentType() == iceberg.EntryContentData {
				filesToDelete = append(filesToDelete, df)
			}
		}

		return filesToDelete, filesWithPartialDeletions, nil, nil
	}

	return t.classifyFilesForFilteredDeletions(ctx, fs, filter, caseSensitive, concurrency)
}

type fileClassificationTask struct {
	meta             Metadata
	partitionFilters *keyDefaultMapErr[int, iceberg.BooleanExpression]
	caseSensitive    bool
	rowFilter        iceberg.BooleanExpression
}

func newFileClassificationTask(meta Metadata, rowFilter iceberg.BooleanExpression, caseSensitive bool) *fileClassificationTask {
	classificationTask := &fileClassificationTask{
		meta:          meta,
		caseSensitive: caseSensitive,
		rowFilter:     rowFilter,
	}
	classificationTask.partitionFilters = newKeyDefaultMapWrapErr(classificationTask.buildPartitionProjection)

	return classificationTask
}

func (t *fileClassificationTask) buildManifestEvaluator(specID int) (func(iceberg.ManifestFile) (bool, error), error) {
	return buildManifestEvaluator(specID, t.meta, t.meta.CurrentSchema(), t.partitionFilters, t.caseSensitive)
}

func (t *fileClassificationTask) buildPartitionProjection(specID int) (iceberg.BooleanExpression, error) {
	return buildPartitionProjection(specID, t.meta, t.meta.CurrentSchema(), t.rowFilter, t.caseSensitive)
}

// classifyFilesForFilteredDeletions classifies files for filtered overwrite operations.
// Returns files to delete completely, files to rewrite partially, the
// per-file file_sequence_number map for rewrite candidates, and any error.
func (t *Transaction) classifyFilesForFilteredDeletions(ctx context.Context, fs io.IO, filter iceberg.BooleanExpression, caseSensitive bool, concurrency int) (filesToDelete, filesWithPartialDeletes []iceberg.DataFile, fileSeqByPath map[string]*int64, err error) {
	meta, err := t.txnMeta()
	if err != nil {
		return nil, nil, nil, err
	}

	schema := meta.CurrentSchema()
	builtMeta, err := meta.Build()
	if err != nil {
		return nil, nil, nil, err
	}

	inclusiveEvaluator, err := newInclusiveMetricsEvaluator(schema, filter, caseSensitive, false)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create inclusive metrics evaluator: %w", err)
	}

	strictEvaluator, err := newStrictMetricsEvaluator(schema, filter, caseSensitive, false)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create strict metrics evaluator: %w", err)
	}

	classificationTask := newFileClassificationTask(builtMeta, filter, caseSensitive)
	manifestEvaluators := newKeyDefaultMapWrapErr(classificationTask.buildManifestEvaluator)

	s := t.planningSnapshot(meta)
	var manifests []iceberg.ManifestFile
	if s != nil {
		manifests, err = s.Manifests(fs)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to get manifests: %w", err)
		}
	}

	var mu sync.Mutex
	fileSeqByPath = make(map[string]*int64)

	g, _ := errgroup.WithContext(ctx)
	g.SetLimit(min(concurrency, len(manifests)))

	for _, manifest := range manifests {
		g.Go(func() error {
			manifestEval, err := manifestEvaluators.Get(int(manifest.PartitionSpecID()))
			if err != nil {
				return fmt.Errorf("failed to build manifest evaluator for spec %d: %w", manifest.PartitionSpecID(), err)
			}
			match, err := manifestEval(manifest)
			if err != nil {
				return fmt.Errorf("failed to evaluate manifest %s: %w", manifest.FilePath(), err)
			}
			if !match {
				return nil
			}

			localDelete := make([]iceberg.DataFile, 0)
			localRewrite := make([]iceberg.DataFile, 0)
			localSeqByPath := make(map[string]*int64)

			for entry, err := range manifest.Entries(fs, false) {
				if err != nil {
					return fmt.Errorf("failed to fetch manifest entries: %w", err)
				}
				if entry.Status() == iceberg.EntryStatusDELETED {
					continue
				}

				df := entry.DataFile()
				if df.ContentType() != iceberg.EntryContentData {
					continue
				}

				inclusive, err := inclusiveEvaluator(df)
				if err != nil {
					return fmt.Errorf("failed to evaluate data file %s with inclusive evaluator: %w", df.FilePath(), err)
				}

				if !inclusive {
					continue
				}

				strict, err := strictEvaluator(df)
				if err != nil {
					return fmt.Errorf("failed to evaluate data file %s with strict evaluator: %w", df.FilePath(), err)
				}

				if strict {
					localDelete = append(localDelete, df)
				} else {
					localRewrite = append(localRewrite, df)
					// Capture the entry's data sequence number so the
					// rewrite path can synthesize
					// _last_updated_sequence_number for source rows that
					// have a null value (or no column) in the file. Per
					// spec the synthesized value is the manifest entry's
					// sequence_number (field id 3, the data sequence
					// number) — not file_sequence_number (field id 4) —
					// so back-dated EXISTING entries assign the correct
					// value to surviving rows.
					if seq := entry.SequenceNum(); seq >= 0 {
						s := seq
						localSeqByPath[df.FilePath()] = &s
					}
				}
			}

			if len(localDelete) > 0 || len(localRewrite) > 0 {
				mu.Lock()
				filesToDelete = append(filesToDelete, localDelete...)
				filesWithPartialDeletes = append(filesWithPartialDeletes, localRewrite...)
				maps.Copy(fileSeqByPath, localSeqByPath)
				mu.Unlock()
			}

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, nil, nil, err
	}

	return filesToDelete, filesWithPartialDeletes, fileSeqByPath, nil
}

// rewriteFilesWithFilter rewrites data files by preserving only rows that do NOT match the filter.
// fileSeqByPath maps each file's path to its file_sequence_number from the source manifest entry,
// used to synthesize _last_updated_sequence_number when the source row's value is null. The
// filter binding and substrait conversion are computed once here and reused across every file.
func (t *Transaction) rewriteFilesWithFilter(ctx context.Context, fs io.IO, updater *snapshotProducer, files []iceberg.DataFile, fileSeqByPath map[string]*int64, filter iceberg.BooleanExpression, caseSensitive bool, concurrency int) error {
	meta, err := t.txnMeta()
	if err != nil {
		return err
	}

	complementFilter := iceberg.NewNot(filter)

	// Bind + convert the complement filter once for the whole rewrite. The
	// per-batch filter function is reused across every record batch from
	// every file in this rewrite — substrait conversion in particular is
	// non-trivial so doing it inside the iterator loop wastes work.
	postFilter, err := prepareBatchFilter(complementFilter, meta.CurrentSchema(), caseSensitive)
	if err != nil {
		return err
	}

	for _, originalFile := range files {
		// Use a separate UUID for rewrite operations to avoid filename collisions with new data files
		rewriteUUID := uuid.New()
		args := rewriteSingleFileArgs{
			fs:            fs,
			originalFile:  originalFile,
			fileSeqNum:    fileSeqByPath[originalFile.FilePath()],
			filter:        complementFilter,
			postFilter:    postFilter,
			caseSensitive: caseSensitive,
			commitUUID:    rewriteUUID,
			concurrency:   concurrency,
		}
		rewrittenFiles, err := t.rewriteSingleFile(ctx, args)
		if err != nil {
			return fmt.Errorf("failed to rewrite file %s: %w", originalFile.FilePath(), err)
		}

		updater.deleteDataFile(originalFile)
		for _, rewrittenFile := range rewrittenFiles {
			updater.appendDataFile(rewrittenFile)
		}
	}

	return nil
}

// rewriteSingleFileArgs bundles the parameters for [Transaction.rewriteSingleFile]
// — there are several same-typed fields (two filter values, plus a UUID and an
// int) so positional ordering offers no compile-time order protection.
type rewriteSingleFileArgs struct {
	fs           io.IO
	originalFile iceberg.DataFile
	// fileSeqNum is the source file's data sequence number from its
	// manifest entry; required to synthesize _last_updated_sequence_number
	// for rows whose value is null in the source file.
	fileSeqNum *int64
	// filter is the per-row complement (rows that survive the rewrite).
	filter iceberg.BooleanExpression
	// postFilter is the pre-bound, pre-substrait-converted version of
	// filter. When preserveRowLineage applies it is the only place the
	// filter is evaluated; the scanner's row filter is bypassed so
	// _row_id positions stay correct.
	postFilter    func(context.Context, arrow.RecordBatch) (arrow.RecordBatch, error)
	caseSensitive bool
	commitUUID    uuid.UUID
	concurrency   int
}

// rewriteSingleFile reads a single data file, applies the filter, and writes
// new files with filtered data. See [rewriteSingleFileArgs] for parameter
// semantics.
func (t *Transaction) rewriteSingleFile(ctx context.Context, args rewriteSingleFileArgs) ([]iceberg.DataFile, error) {
	meta, err := t.txnMeta()
	if err != nil {
		return nil, err
	}

	scanTask := &FileScanTask{
		File:   args.originalFile,
		Start:  0,
		Length: args.originalFile.FileSizeBytes(),
	}

	// Preserve row lineage for v3 tables: include _row_id and
	// _last_updated_sequence_number in the scan projection so they are read
	// from the source file (or synthesized from file metadata when null) and
	// written explicitly into the output Parquet. The Iceberg v3 spec says
	// _last_updated_sequence_number on a data file row is the sequence
	// number of the snapshot that last *logically* updated the row — for a
	// pure CoW rewrite the surviving rows are unchanged, so they must keep
	// their original value rather than inheriting the new commit's seq.
	//
	// When preserving lineage, the row filter must NOT be applied inside the
	// scanner because _row_id synthesis depends on original file position. The
	// filter is applied post-synthesis in the record iterator instead.
	//
	// Keep the real filter available to Parquet row-group pruning while the
	// scanner's row filter stays AlwaysTrue. Pruning happens before rows are
	// materialized, so _row_id synthesis still sees the original positions of
	// every surviving row group and the post-synthesis filter remains correct.
	_, originalFirstRowID, _, _, _ := iceberginternal.BorrowedDataFilePointers(args.originalFile)
	preserveRowLineage := meta.formatVersion >= 3 && originalFirstRowID != nil
	projectedSchema := meta.CurrentSchema()
	var factoryOpts []writerFactoryOption
	var rowGroupFilter iceberg.BooleanExpression
	if preserveRowLineage {
		rowGroupFilter, err = iceberg.BindExpr(meta.CurrentSchema(), args.filter, args.caseSensitive)
		if err != nil {
			return nil, fmt.Errorf("failed to bind row-group filter: %w", err)
		}

		projectedSchema = iceberg.SchemaWithRowLineage(projectedSchema)
		factoryOpts = append(factoryOpts, withFactoryFileSchema(projectedSchema))
		scanTask.FirstRowID = args.originalFile.FirstRowID()
		// fileSeqNum drives _last_updated_sequence_number synthesis for
		// source rows whose value is null in the file (or for source
		// files written before row lineage existed). When the column
		// is non-null the existing per-row value wins, preserving the
		// original update sequence across the rewrite.
		scanTask.DataSequenceNumber = args.fileSeqNum
	}

	// When preserving row lineage, the bound scan-time filter is replaced
	// with AlwaysTrue and the rebound version inside prepareBatchFilter is
	// the one actually applied. Skip the wasted BindExpr in that case.
	scanFilter := iceberg.BooleanExpression(iceberg.AlwaysTrue{})
	if !preserveRowLineage {
		boundFilter, err := iceberg.BindExpr(meta.CurrentSchema(), args.filter, args.caseSensitive)
		if err != nil {
			return nil, fmt.Errorf("failed to bind filter: %w", err)
		}
		scanFilter = boundFilter
	}

	builtMeta, err := meta.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build metadata: %w", err)
	}

	scanner := &arrowScan{
		metadata:        builtMeta,
		fs:              args.fs,
		scanSchema:      meta.CurrentSchema(),
		projectedSchema: projectedSchema,
		boundRowFilter:  scanFilter,
		rowGroupFilter:  rowGroupFilter,
		caseSensitive:   args.caseSensitive,
		rowLimit:        -1, // No limit
		concurrency:     args.concurrency,
	}

	arrowSchema, recordIter, err := scanner.GetRecords(ctx, []FileScanTask{*scanTask})
	if err != nil {
		return nil, fmt.Errorf("failed to get records from original file: %w", err)
	}

	// When preserving row lineage, use an Arrow schema with field IDs so that
	// recordsToDataFiles can resolve _row_id via field ID rather than the name
	// mapping (which doesn't include metadata columns).
	if preserveRowLineage {
		arrowSchema, err = SchemaToArrowSchemaWithOptions(projectedSchema, ArrowSchemaOptions{
			IncludeFieldIDs: true,
			TableProperties: builtMeta.Properties(),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to build arrow schema with field IDs: %w", err)
		}
	}

	// Wrap the iterator to release records after consumption.
	// When preserving row lineage, also apply the (pre-bound) filter here
	// post-synthesis so _row_id positions stay correct.
	releaseIter := func(yield func(arrow.RecordBatch, error) bool) {
		for rec, err := range recordIter {
			if err != nil {
				yield(nil, err)

				return
			}

			if preserveRowLineage {
				rec, err = args.postFilter(ctx, rec)
				if err != nil {
					yield(nil, err)

					return
				}
				if rec.NumRows() == 0 {
					rec.Release()

					continue
				}
			}

			if !yield(rec, nil) {
				rec.Release()

				return
			}
			rec.Release()
		}
	}

	wfs, err := requireWriteFileIO(args.fs)
	if err != nil {
		return nil, err
	}

	var result []iceberg.DataFile
	itr := recordsToDataFiles(ctx, t.tbl.Location(), meta, recordWritingArgs{
		sc:          arrowSchema,
		itr:         releaseIter,
		fs:          wfs,
		writeUUID:   &args.commitUUID,
		factoryOpts: factoryOpts,
	})

	for df, err := range itr {
		if err != nil {
			return nil, err
		}
		result = append(result, df)
	}

	return result, nil
}

// writePositionDeletesForFiles writes position-delete files for the rows in
// files that match filter, appends them to updater, and returns the distinct
// data-file paths those deletes reference. A position-delete that does not
// record its referenced data file is omitted from the returned paths (it
// cannot be existence-checked), matching RowDelta.validate and Java.
func (t *Transaction) writePositionDeletesForFiles(ctx context.Context, fs io.IO, updater *snapshotProducer, files []iceberg.DataFile, filter iceberg.BooleanExpression, caseSensitive bool, concurrency int, commitUUID uuid.UUID) ([]string, error) {
	meta, err := t.txnMeta()
	if err != nil {
		return nil, err
	}

	posDeleteRecIter, err := t.makePositionDeleteRecordsForFilter(ctx, fs, files, filter, caseSensitive, concurrency)
	if err != nil {
		return nil, err
	}
	wfs, err := requireWriteFileIO(fs)
	if err != nil {
		return nil, err
	}

	partitionContextByFilePath := make(map[string]partitionContext, len(files))
	for _, df := range files {
		partitionContextByFilePath[df.FilePath()] = partitionContext{partitionData: df.Partition(), specID: df.SpecID()}
	}

	// V3 tables store deletes as deletion vectors, and the spec allows at most
	// one DV per data file. A data file being partially deleted may already
	// carry a DV from an earlier delete; those existing positions are folded
	// into the new DV (the incremental deletes) and the superseded DVs are
	// removed. Without this a second delete on the same file would write a
	// second live DV, corrupting the table (buildDVIndex rejects it).
	var (
		existingDVs       map[string]iceberg.DataFile
		existingDVBitmaps map[string]*dv.RoaringPositionBitmap
	)
	if meta.formatVersion >= 3 {
		existingDVs, err = t.collectExistingDVs(fs, files)
		if err != nil {
			return nil, err
		}

		existingDVBitmaps = make(map[string]*dv.RoaringPositionBitmap, len(existingDVs))
		for ref, dvFile := range existingDVs {
			bitmap, err := dv.ReadDV(fs, dvFile)
			if err != nil {
				return nil, fmt.Errorf("failed to read existing deletion vector for %s: %w", ref, err)
			}
			existingDVBitmaps[ref] = bitmap
		}
	}

	posDeleteFiles := positionDeleteRecordsToDataFiles(ctx, t.tbl.Location(), meta, partitionContextByFilePath, recordWritingArgs{
		sc:          PositionalDeleteArrowSchema,
		itr:         posDeleteRecIter,
		writeUUID:   &commitUUID,
		fs:          wfs,
		existingDVs: existingDVBitmaps,
	})

	var referenced []string
	seen := make(map[string]struct{})
	for f, err := range posDeleteFiles {
		if err != nil {
			return nil, err
		}
		updater.appendDeleteFile(f)
		if ref := iceberginternal.BorrowedDataFileReferencedDataFile(f); ref != nil && *ref != "" {
			if _, ok := seen[*ref]; !ok {
				seen[*ref] = struct{}{}
				referenced = append(referenced, *ref)
			}
		}
	}

	// Supersede the old DVs only after the merged replacements are staged, so a
	// write failure above leaves the existing DVs untouched.
	for _, dvFile := range existingDVs {
		updater.removeDeletionVector(dvFile)
	}

	return referenced, nil
}

// collectExistingDVs returns the live deletion vectors that reference any of
// files, keyed by the referenced data file path. Used by the v3 merge-on-read
// delete path to fold prior deletes into a new DV and remove the superseded
// entries, preserving the spec's one-DV-per-data-file invariant.
func (t *Transaction) collectExistingDVs(fs io.IO, files []iceberg.DataFile) (map[string]iceberg.DataFile, error) {
	meta, err := t.txnMeta()
	if err != nil {
		return nil, err
	}

	s := t.planningSnapshot(meta)
	if s == nil {
		return nil, nil
	}

	wanted := make(map[string]struct{}, len(files))
	for _, df := range files {
		wanted[df.FilePath()] = struct{}{}
	}

	result := make(map[string]iceberg.DataFile)
	// Skip DELETED entries: an old, replaced DV stays behind as one. Reading it
	// would build the new DV from old data and bring back deleted rows. See #1372.
	for entry, err := range s.entries(fs, iceberg.ManifestContentDeletes, true) {
		if err != nil {
			return nil, fmt.Errorf("scanning existing deletion vectors: %w", err)
		}
		df := entry.DataFile()
		if !IsDeletionVector(df) {
			continue
		}
		ref := iceberginternal.BorrowedDataFileReferencedDataFile(df)
		if ref == nil {
			continue
		}
		if _, ok := wanted[*ref]; ok {
			result[*ref] = df
		}
	}

	return result, nil
}

func (t *Transaction) makePositionDeleteRecordsForFilter(ctx context.Context, fs io.IO, files []iceberg.DataFile, filter iceberg.BooleanExpression, caseSensitive bool, concurrency int) (seq2 iter.Seq2[arrow.RecordBatch, error], err error) {
	meta, err := t.txnMeta()
	if err != nil {
		return nil, err
	}

	tasks := make([]FileScanTask, 0, len(files))
	for _, f := range files {
		tasks = append(tasks, FileScanTask{
			File:   f,
			Start:  0,
			Length: f.FileSizeBytes(),
		})
	}

	boundFilter, err := iceberg.BindExpr(meta.CurrentSchema(), filter, caseSensitive)
	if err != nil {
		return nil, fmt.Errorf("failed to bind filter: %w", err)
	}

	builtMeta, err := meta.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build metadata: %w", err)
	}

	scanner := &arrowScan{
		metadata:        builtMeta,
		fs:              fs,
		scanSchema:      meta.CurrentSchema(),
		projectedSchema: meta.CurrentSchema(),
		boundRowFilter:  boundFilter,
		caseSensitive:   caseSensitive,
		rowLimit:        -1, // No limit
		concurrency:     concurrency,
	}
	invariants, err := scanner.scanInvariants(builtMeta.Properties())
	if err != nil {
		return nil, err
	}

	deletesPerFile, err := readAllDeleteFiles(ctx, fs, tasks, concurrency)
	if err != nil {
		// readAllDeleteFiles can return a partially-populated map alongside
		// the error if some goroutines completed before the failure.
		releasePerFilePosDeletes(deletesPerFile)

		return nil, err
	}

	extSet := substrait.NewExtensionSet()

	ctx, cancel := context.WithCancelCause(exprs.WithExtensionIDSet(ctx, extSet))
	taskChan := make(chan tblutils.Enumerated[FileScanTask], len(tasks))

	numWorkers := min(concurrency, len(tasks))
	records := make(chan enumeratedRecord, numWorkers)

	var wg sync.WaitGroup
	wg.Add(numWorkers)
	for range numWorkers {
		go func() {
			defer wg.Done()
			sink := newRecordSink(records)
			for {
				if err := sink.reserve(ctx); err != nil {
					return
				}
				select {
				case <-ctx.Done():
					return
				case task, ok := <-taskChan:
					if !ok {
						return
					}

					if err := scanner.producePosDeletesFromTask(ctx, task, deletesPerFile[task.Value.File.FilePath()], sink, invariants); err != nil {
						cancel(err)

						return
					}
				}
			}
		}()
	}

	go func() {
		for i, t := range tasks {
			taskChan <- tblutils.Enumerated[FileScanTask]{
				Value: t, Index: i, Last: i == len(tasks)-1,
			}
		}
		close(taskChan)

		wg.Wait()
		close(records)
	}()

	return createIterator(ctx, uint(numWorkers), records, deletesPerFile, cancel, scanner.rowLimit), nil
}

func (t *Transaction) Scan(opts ...ScanOption) (*Scan, error) {
	meta, err := t.txnMeta()
	if err != nil {
		return nil, err
	}
	updatedMeta, err := meta.Build()
	if err != nil {
		return nil, err
	}

	s := &Scan{
		identifier:       slices.Clone(t.tbl.identifier),
		metadata:         updatedMeta,
		metadataLocation: t.tbl.metadataLocation,
		ioF:              t.tbl.fsF,
		manifestCache:    newSnapshotManifestCacheForMetadata(updatedMeta),
		// Catalog planners can only see committed table state, not metadata
		// staged inside this transaction. Keep transaction scans local so auto
		// mode cannot silently return stale tasks.
		planner: nil,
		// TODO(#1178 Phase 6): resolve scan-planning-mode table properties here.
		planningMode:   ScanPlanningLocal,
		rowFilter:      iceberg.AlwaysTrue{},
		selectedFields: []string{"*"},
		caseSensitive:  true,
		limit:          ScanNoLimit,
		concurrency:    runtime.GOMAXPROCS(0),
		reporter:       t.tbl.MetricsReporter(),
	}

	for _, opt := range opts {
		opt(s)
	}
	if s.selectorErr != nil {
		return nil, s.selectorErr
	}

	return s, nil
}

func (t *Transaction) StagedTable() (*StagedTable, error) {
	meta, err := t.txnMeta()
	if err != nil {
		return nil, err
	}
	updatedMeta, err := meta.Build()
	if err != nil {
		return nil, err
	}

	return &StagedTable{
		Table: New(
			t.tbl.identifier,
			updatedMeta,
			updatedMeta.Location(),
			t.tbl.fsF,
			t.tbl.cat,
			withReporterState(t.tbl.reporter, t.tbl.reporterSet),
			WithScanPlanningIOProperties(t.tbl.scanPlanningIOProps),
		),
	}, nil
}

// Commit persists the transaction's staged changes. Metadata validation may be
// deferred when applying an operation adds no new requirement to the transaction.
// The staged metadata is validated when a later transaction operation needs an
// immutable metadata view, or while the catalog applies the commit.
func (t *Transaction) Commit(ctx context.Context) (*Table, error) {
	if err := t.checkNotNil(); err != nil {
		return nil, err
	}
	t.mx.Lock()
	defer t.mx.Unlock()

	meta, err := t.txnMeta()
	if err != nil {
		return nil, err
	}

	if t.committed {
		return nil, errors.New("transaction has already been committed")
	}

	if len(meta.updates) > 0 {
		reqs := append(transactionRequirements(t.reqs, t.branch, t.tbl.metadata), AssertTableUUID(meta.uuid))
		tbl, err := t.tbl.doCommit(ctx, meta.updates, reqs,
			withCommitBranch(t.branch),
			withCommitValidators(t.validators...),
			withCommitNoReplay(t.noReplay),
			withCommitPinnedRefs(t.pinnedRefs),
		)
		if err != nil {
			// A clean conflict (ErrCommitFailed) committed nothing and stays
			// retriable. Any other failure leaves the commit state unknown
			// (the catalog may have accepted it), so mark it terminal to
			// avoid a double-apply on retry.
			if !errors.Is(err, ErrCommitFailed) {
				t.committed = true
			}

			return tbl, err
		}

		// Mark committed after the catalog accepts but before PostCommit runs.
		// A PostCommit failure must not leave the transaction retriable;
		// otherwise a retry would re-run the catalog commit.
		t.committed = true

		for _, u := range meta.updates {
			if perr := u.PostCommit(ctx, t.tbl, tbl); perr != nil {
				err = errors.Join(err, perr)
			}
		}

		return tbl, err
	}

	t.committed = true

	return t.tbl, nil
}

func validateGCEnabledForSnapshotExpiration(props iceberg.Properties) error {
	if !isGCEnabled(props) {
		return errors.New("cannot expire snapshots: GC is disabled (deleting files may corrupt other tables)")
	}

	return nil
}

type StagedTable struct {
	*Table
}

func (s *StagedTable) Refresh(ctx context.Context) (*Table, error) {
	return nil, fmt.Errorf("%w: cannot refresh a staged table", ErrInvalidOperation)
}

func (s *StagedTable) Scan(opts ...ScanOption) *Scan {
	panic(fmt.Errorf("%w: cannot scan a staged table", ErrInvalidOperation))
}

// prepareBatchFilter binds the given Iceberg filter against schema and converts
// it to substrait once, returning a per-batch filter function that can be
// reused across every record batch. The setup work (BindExpr, ConvertExpr) is
// independent of the batch and is the most expensive part of filter-eval, so
// hoisting it out of the iterator loop is a measurable win on rewrites that
// produce many batches.
//
// The returned function takes a per-call ctx so allocator/deadline values from
// the call-site context propagate into compute (the extension-ID set is
// attached internally on each call rather than baked into a captured ctx).
//
// Ownership: the returned function takes ownership of the input batch — the
// AlwaysFalse fast-path and the bound-filter path both Release rec internally.
// Callers must not Release the input batch separately. The function returns a
// possibly-new batch with one reference owned by the caller; the caller is
// responsible for releasing it.
//
// The bound substrait filter is evaluated against the input record's column
// indices, so callers must pass the bound filter's schema verbatim — extra
// trailing columns (e.g. row-lineage metadata) are tolerated only because
// they live at indices past the filter's referenced fields.
func prepareBatchFilter(filter iceberg.BooleanExpression, schema *iceberg.Schema, caseSensitive bool) (func(context.Context, arrow.RecordBatch) (arrow.RecordBatch, error), error) {
	if filter == nil || filter.Equals(iceberg.AlwaysTrue{}) {
		return func(_ context.Context, rec arrow.RecordBatch) (arrow.RecordBatch, error) {
			return rec, nil
		}, nil
	}

	bound, err := iceberg.BindExpr(schema, filter, caseSensitive)
	if err != nil {
		return nil, fmt.Errorf("prepareBatchFilter: bind expression: %w", err)
	}

	if bound == nil {
		return func(_ context.Context, rec arrow.RecordBatch) (arrow.RecordBatch, error) {
			return rec, nil
		}, nil
	}

	if bound.Equals(iceberg.AlwaysFalse{}) {
		// Return a record with the same schema and zero rows. NewSlice(0, 0)
		// preserves the per-field arrays so downstream code that calls
		// rec.Column(i) does not panic on the empty result.
		return func(_ context.Context, rec arrow.RecordBatch) (arrow.RecordBatch, error) {
			defer rec.Release()

			return rec.NewSlice(0, 0), nil
		}, nil
	}

	extSet, substraitFilter, err := substrait.ConvertExpr(schema, bound, caseSensitive)
	if err != nil {
		return nil, fmt.Errorf("prepareBatchFilter: convert expression: %w", err)
	}

	return func(ctx context.Context, rec arrow.RecordBatch) (arrow.RecordBatch, error) {
		// Attach a fresh extension-ID set to the call-site ctx so allocator
		// and deadline values flow through to compute, while the (immutable)
		// extSet computed during prepare is reused across every batch.
		ctx = exprs.WithExtensionIDSet(ctx, exprs.NewExtensionSetDefault(*extSet))

		return filterRecords(ctx, substraitFilter)(rec)
	}, nil
}
