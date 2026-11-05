package icebergcatalog

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ---- strict member decoding ----

// members wraps a duplicate-key-free JSON object and tracks which keys were
// read, so leftovers (unknown fields, `Action`/`ACTION` case variants,
// legacy names like `updated`) are rejected instead of silently ignored.
type members struct {
	what string
	obj  jsonObject
	used map[string]bool
}

func newMembers(what string, raw json.RawMessage) (*members, error) {
	obj, err := decodeObject(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", errBadRequest, what, err)
	}
	return &members{what: what, obj: obj, used: map[string]bool{}}, nil
}

// get decodes key into v. Missing keys are an error unless optional.
func (m *members) get(key string, v any, optional bool) (bool, error) {
	m.used[key] = true
	raw, ok := m.obj[key]
	if !ok {
		if optional {
			return false, nil
		}
		return false, fmt.Errorf("%w: %s: missing %q", errBadRequest, m.what, key)
	}
	if err := json.Unmarshal(raw, v); err != nil {
		return false, fmt.Errorf("%w: %s: %q: %v", errBadRequest, m.what, key, err)
	}
	return true, nil
}

func (m *members) done() error {
	for k := range m.obj {
		if !m.used[k] {
			return fmt.Errorf("%w: %s: unknown field %q", errBadRequest, m.what, k)
		}
	}
	return nil
}

// ---- requirements ----

type requirement struct {
	typ        string
	uuid       string
	ref        string
	snapshotID *int64 // assert-ref-snapshot-id; nil asserts the ref doesn't exist
	id         int    // the int the other assert-* types carry
}

// intRequirements maps each int-valued requirement to its field.
var intRequirements = map[string]string{
	"assert-last-assigned-field-id":     "last-assigned-field-id",
	"assert-current-schema-id":          "current-schema-id",
	"assert-last-assigned-partition-id": "last-assigned-partition-id",
	"assert-default-spec-id":            "default-spec-id",
	"assert-default-sort-order-id":      "default-sort-order-id",
}

func parseRequirement(raw json.RawMessage) (requirement, error) {
	m, err := newMembers("requirement", raw)
	if err != nil {
		return requirement{}, err
	}
	var r requirement
	if _, err := m.get("type", &r.typ, false); err != nil {
		return r, err
	}
	switch r.typ {
	case "assert-create":
	case "assert-table-uuid":
		_, err = m.get("uuid", &r.uuid, false)
	case "assert-ref-snapshot-id":
		if _, err = m.get("ref", &r.ref, false); err == nil {
			_, err = m.get("snapshot-id", &r.snapshotID, true)
		}
	default:
		field, ok := intRequirements[r.typ]
		if !ok {
			return r, fmt.Errorf("%w: unknown requirement type %q", errBadRequest, r.typ)
		}
		_, err = m.get(field, &r.id, false)
	}
	if err != nil {
		return r, err
	}
	return r, m.done()
}

// check returns an errCommitFailed error if md (nil when the table doesn't
// exist) doesn't satisfy r.
func (r requirement) check(md *tableMetadata) error {
	fail := func(format string, args ...any) error {
		return fmt.Errorf("%w: requirement failed: %s", errCommitFailed, fmt.Sprintf(format, args...))
	}
	if r.typ == "assert-create" {
		if md != nil {
			return fail("table already exists")
		}
		return nil
	}
	if md == nil {
		return fail("%s: table does not exist", r.typ)
	}
	var have int
	switch r.typ {
	case "assert-table-uuid":
		if !strings.EqualFold(md.TableUUID, r.uuid) {
			return fail("table uuid %s != %s", md.TableUUID, r.uuid)
		}
		return nil
	case "assert-ref-snapshot-id":
		ref, ok := md.Refs[r.ref]
		switch {
		case r.snapshotID == nil && ok:
			return fail("ref %q was created concurrently", r.ref)
		case r.snapshotID != nil && !ok:
			return fail("ref %q is missing, expected snapshot %d", r.ref, *r.snapshotID)
		case r.snapshotID != nil && ref.SnapshotID != *r.snapshotID:
			return fail("ref %q is at snapshot %d, expected %d", r.ref, ref.SnapshotID, *r.snapshotID)
		}
		return nil
	case "assert-last-assigned-field-id":
		have = md.LastColumnID
	case "assert-current-schema-id":
		have = md.CurrentSchemaID
	case "assert-last-assigned-partition-id":
		have = md.LastPartitionID
	case "assert-default-spec-id":
		have = md.DefaultSpecID
	case "assert-default-sort-order-id":
		have = md.DefaultSortOrderID
	}
	if have != r.id {
		return fail("%s: %d != %d", r.typ, have, r.id)
	}
	return nil
}

// ---- updates ----

// update is one decoded table update. Only the fields for its action are
// set.
type update struct {
	action        string
	uuid          string
	formatVersion int
	obj           jsonObject // add-schema/add-spec/add-sort-order
	id            int        // set-current-schema/set-default-spec/set-default-sort-order
	location      string
	snapshot      *snapshot
	refName       string
	ref           snapshotRef
	snapshotIDs   []int64
	snapshotID    int64
	props         map[string]string
	removals      []string
	stats         *statisticsFile
	partStats     *partitionStatisticsFile
}

// allowedUpdates are the commit update actions we apply: what duckdb sends
// (see the smoke test) plus what a spec-compliant client managing snapshots,
// properties and statistics would. Anything else (encryption keys, removing
// schemas/specs) is rejected rather than reasoned about.
var allowedUpdates = map[string]bool{
	"assign-uuid":                 true,
	"upgrade-format-version":      true,
	"add-schema":                  true,
	"set-current-schema":          true,
	"add-spec":                    true,
	"set-default-spec":            true,
	"add-sort-order":              true,
	"set-default-sort-order":      true,
	"set-location":                true,
	"add-snapshot":                true,
	"set-snapshot-ref":            true,
	"remove-snapshots":            true,
	"remove-snapshot-ref":         true,
	"set-properties":              true,
	"remove-properties":           true,
	"set-statistics":              true,
	"remove-statistics":           true,
	"set-partition-statistics":    true,
	"remove-partition-statistics": true,
}

func parseUpdate(raw json.RawMessage) (update, error) {
	m, err := newMembers("update", raw)
	if err != nil {
		return update{}, err
	}
	var u update
	if _, err := m.get("action", &u.action, false); err != nil {
		return u, err
	}
	if !allowedUpdates[u.action] {
		return u, fmt.Errorf("%w: update action %q is not supported", errBadRequest, u.action)
	}
	get := func(key string, v any) {
		if err == nil {
			_, err = m.get(key, v, false)
		}
	}
	opt := func(key string, v any) {
		if err == nil {
			_, err = m.get(key, v, true)
		}
	}
	switch u.action {
	case "assign-uuid":
		get("uuid", &u.uuid)
	case "upgrade-format-version":
		get("format-version", &u.formatVersion)
	case "add-schema":
		var ignored *int // deprecated, we derive last-column-id
		get("schema", &u.obj)
		opt("last-column-id", &ignored)
	case "set-current-schema":
		get("schema-id", &u.id)
	case "add-spec":
		get("spec", &u.obj)
	case "set-default-spec":
		get("spec-id", &u.id)
	case "add-sort-order":
		get("sort-order", &u.obj)
	case "set-default-sort-order":
		get("sort-order-id", &u.id)
	case "set-location":
		get("location", &u.location)
	case "add-snapshot":
		get("snapshot", &u.snapshot)
	case "set-snapshot-ref":
		get("ref-name", &u.refName)
		get("snapshot-id", &u.ref.SnapshotID)
		get("type", &u.ref.Type)
		opt("min-snapshots-to-keep", &u.ref.MinSnapshotsToKeep)
		opt("max-snapshot-age-ms", &u.ref.MaxSnapshotAgeMS)
		opt("max-ref-age-ms", &u.ref.MaxRefAgeMS)
	case "remove-snapshots":
		get("snapshot-ids", &u.snapshotIDs)
	case "remove-snapshot-ref":
		get("ref-name", &u.refName)
	case "set-properties":
		get("updates", &u.props)
	case "remove-properties":
		get("removals", &u.removals)
	case "set-statistics":
		var deprecated *int64
		opt("snapshot-id", &deprecated)
		get("statistics", &u.stats)
	case "remove-statistics", "remove-partition-statistics":
		get("snapshot-id", &u.snapshotID)
	case "set-partition-statistics":
		get("partition-statistics", &u.partStats)
	}
	if err != nil {
		return u, err
	}
	if (u.action == "add-snapshot" && u.snapshot == nil) ||
		(u.action == "set-statistics" && u.stats == nil) ||
		(u.action == "set-partition-statistics" && u.partStats == nil) ||
		(strings.HasPrefix(u.action, "add-s") && u.action != "add-snapshot" && u.obj == nil) {
		return u, fmt.Errorf("%w: %s: null payload", errBadRequest, u.action)
	}
	return u, m.done()
}

// ---- applying updates ----

// builder applies updates to a copy of a table's metadata, following the
// Java/iceberg-go TableMetadata.Builder rules for id assignment and
// bookkeeping.
type builder struct {
	md                *tableMetadata
	create            bool
	lastAddedSchema   *int
	lastAddedSpec     *int
	lastAddedOrder    *int
	addedSnapshots    map[int64]bool
	mainSets          []int64
	removedSnapshots  bool
	lastUpdatedMS     int64
	changed           bool
	now               time.Time
	prevMetadataFile  string
	prevLastUpdatedMS int64
}

// emptyMetadata is the base a create commit (assert-create) applies its
// updates to: nothing but defaults, so the client's own add-schema etc.
// become schema/spec/order 0 (IE no placeholder schema like iceberg-go's
// catalog/sql creates).
func emptyMetadata() *tableMetadata {
	return &tableMetadata{
		FormatVersion:      2,
		LastPartitionID:    partitionDataIDStart - 1,
		CurrentSchemaID:    -1,
		DefaultSpecID:      -1,
		DefaultSortOrderID: -1,
		Properties:         map[string]string{},
		Refs:               map[string]snapshotRef{},
	}
}

func newBuilder(base *tableMetadata, metadataLoc string, now time.Time) *builder {
	b := &builder{addedSnapshots: map[int64]bool{}, now: now}
	if base == nil {
		b.create = true
		b.md = emptyMetadata()
		return b
	}
	b.md = base.clone()
	b.prevMetadataFile = metadataLoc
	b.prevLastUpdatedMS = base.LastUpdatedMS
	return b
}

func badf(format string, args ...any) error {
	return fmt.Errorf("%w: %s", errBadRequest, fmt.Sprintf(format, args...))
}

func (b *builder) apply(u update) error {
	md := b.md
	b.changed = true
	switch u.action {
	case "assign-uuid":
		if _, err := uuid.Parse(u.uuid); err != nil {
			return badf("invalid uuid %q", u.uuid)
		}
		// NOTE: Java/iceberg-go let a commit change an existing table's uuid,
		// we only allow it on create (or a no-op).
		if !b.create && !strings.EqualFold(md.TableUUID, u.uuid) {
			return badf("can't change the uuid of an existing table")
		}
		md.TableUUID = u.uuid
	case "upgrade-format-version":
		if u.formatVersion < 2 || u.formatVersion > maxFormatVersion {
			return badf("unsupported format-version %d (want 2..%d)", u.formatVersion, maxFormatVersion)
		}
		if !b.create && u.formatVersion < md.FormatVersion {
			return badf("can't downgrade format-version %d -> %d", md.FormatVersion, u.formatVersion)
		}
		md.FormatVersion = u.formatVersion
		if md.FormatVersion >= minRowLineageVersion && md.NextRowID == nil {
			zero := int64(0)
			md.NextRowID = &zero
		}
	case "add-schema":
		return b.addSchema(u.obj)
	case "set-current-schema":
		id, err := b.resolveLast(u.id, b.lastAddedSchema, "schema")
		if err != nil {
			return err
		}
		if !findByID(md.Schemas, "schema-id", id) {
			return badf("unknown schema %d", id)
		}
		md.CurrentSchemaID = id
	case "add-spec":
		return b.addSpec(u.obj)
	case "set-default-spec":
		id, err := b.resolveLast(u.id, b.lastAddedSpec, "spec")
		if err != nil {
			return err
		}
		if !findByID(md.PartitionSpecs, "spec-id", id) {
			return badf("unknown partition spec %d", id)
		}
		md.DefaultSpecID = id
	case "add-sort-order":
		return b.addSortOrder(u.obj)
	case "set-default-sort-order":
		id, err := b.resolveLast(u.id, b.lastAddedOrder, "sort order")
		if err != nil {
			return err
		}
		if !findByID(md.SortOrders, "order-id", id) {
			return badf("unknown sort order %d", id)
		}
		md.DefaultSortOrderID = id
	case "set-location":
		md.Location = strings.TrimSuffix(u.location, "/")
	case "add-snapshot":
		return b.addSnapshot(*u.snapshot)
	case "set-snapshot-ref":
		return b.setRef(u.refName, u.ref)
	case "remove-snapshots":
		return b.removeSnapshots(u.snapshotIDs)
	case "remove-snapshot-ref":
		// Java ignores unknown refs, iceberg-go errors. We follow Java.
		delete(md.Refs, u.refName)
		if u.refName == mainBranch {
			md.CurrentSnapshotID = nil
		}
	case "set-properties":
		for k, v := range u.props {
			md.Properties[k] = v
		}
	case "remove-properties":
		for _, k := range u.removals {
			delete(md.Properties, k)
		}
	case "set-statistics":
		md.Statistics = slices.DeleteFunc(md.Statistics, func(s statisticsFile) bool { return s.SnapshotID == u.stats.SnapshotID })
		if u.stats.BlobMetadata == nil || string(u.stats.BlobMetadata) == "null" {
			u.stats.BlobMetadata = json.RawMessage("[]")
		}
		md.Statistics = append(md.Statistics, *u.stats)
	case "remove-statistics":
		md.Statistics = slices.DeleteFunc(md.Statistics, func(s statisticsFile) bool { return s.SnapshotID == u.snapshotID })
	case "set-partition-statistics":
		md.PartitionStatistics = slices.DeleteFunc(md.PartitionStatistics, func(s partitionStatisticsFile) bool { return s.SnapshotID == u.partStats.SnapshotID })
		md.PartitionStatistics = append(md.PartitionStatistics, *u.partStats)
	case "remove-partition-statistics":
		md.PartitionStatistics = slices.DeleteFunc(md.PartitionStatistics, func(s partitionStatisticsFile) bool { return s.SnapshotID == u.snapshotID })
	default:
		return badf("update action %q is not supported", u.action)
	}
	return nil
}

// resolveLast resolves the spec's `-1` ("the one added last in this
// commit").
func (b *builder) resolveLast(id int, last *int, what string) (int, error) {
	if id != -1 {
		return id, nil
	}
	if last == nil {
		return 0, badf("no %s added in this commit to resolve -1 against", what)
	}
	return *last, nil
}

// reuseOrNewID returns the id of an existing object equal to obj (ignoring
// the id key, or comparing only cmpKey when set), or max(existing ids)+1 (at
// least floor).
func reuseOrNewID(existing []jsonObject, obj jsonObject, idKey, cmpKey string, floor int) (int, bool) {
	key := func(o jsonObject) string {
		if cmpKey != "" {
			return jsonObject{cmpKey: o[cmpKey]}.canonical()
		}
		return o.canonical(idKey)
	}
	want := key(obj)
	next := floor
	for _, e := range existing {
		id, _ := e.int(idKey)
		if key(e) == want {
			return id, true
		}
		next = max(next, id+1)
	}
	return next, false
}

func (b *builder) addSchema(schema jsonObject) error {
	schema = schema.clone()
	var typ string
	_ = json.Unmarshal(schema["type"], &typ)
	if typ != "struct" {
		return badf("schema must be a struct")
	}
	highest, err := highestFieldID(schema)
	if err != nil {
		return badf("invalid schema: %v", err)
	}
	id, _ := schema.int("schema-id")
	// Java/iceberg-go start new schema ids at the requested one, bumped
	// past any existing id.
	newID, reused := reuseOrNewID(b.md.Schemas, schema, "schema-id", "", id)
	b.lastAddedSchema = &newID
	if reused {
		return nil
	}
	schema.setInt("schema-id", newID)
	b.md.Schemas = append(b.md.Schemas, schema)
	b.md.LastColumnID = max(b.md.LastColumnID, highest)
	return nil
}

func (b *builder) addSpec(spec jsonObject) error {
	spec = spec.clone()
	fields, err := spec.objects("fields")
	if err != nil || spec["fields"] == nil {
		return badf("invalid partition spec")
	}
	// Assign field ids the client left out, above everything assigned so
	// far.
	floor := b.md.LastPartitionID
	for _, s := range b.md.PartitionSpecs {
		fs, _ := s.objects("fields")
		for _, f := range fs {
			id, _ := f.int("field-id")
			floor = max(floor, id)
		}
	}
	maxID := floor
	for _, f := range fields {
		if _, ok := f.int("field-id"); !ok {
			floor++
			f.setInt("field-id", floor)
		}
		id, _ := f.int("field-id")
		maxID = max(maxID, id)
	}
	spec.setObjects("fields", fields)
	newID, reused := reuseOrNewID(b.md.PartitionSpecs, spec, "spec-id", "fields", 0)
	b.lastAddedSpec = &newID
	if reused {
		return nil
	}
	spec = jsonObject{"fields": spec["fields"]}
	spec.setInt("spec-id", newID)
	b.md.PartitionSpecs = append(b.md.PartitionSpecs, spec)
	b.md.LastPartitionID = max(b.md.LastPartitionID, maxID)
	return nil
}

func (b *builder) addSortOrder(order jsonObject) error {
	fields, err := order.objects("fields")
	if err != nil || order["fields"] == nil {
		return badf("invalid sort order")
	}
	order = jsonObject{"fields": order["fields"]}
	var newID int
	reused := false
	if len(fields) == 0 {
		// Unsorted is always order 0.
		newID, reused = 0, findByID(b.md.SortOrders, "order-id", 0)
	} else {
		newID, reused = reuseOrNewID(b.md.SortOrders, order, "order-id", "fields", 1)
	}
	b.lastAddedOrder = &newID
	if reused {
		return nil
	}
	order.setInt("order-id", newID)
	b.md.SortOrders = append(b.md.SortOrders, order)
	return nil
}

const oneMinuteMS = 60_000

func (b *builder) addSnapshot(s snapshot) error {
	md := b.md
	switch {
	case len(md.Schemas) == 0:
		return badf("can't add a snapshot w/o a schema")
	case len(md.PartitionSpecs) == 0:
		return badf("can't add a snapshot w/o a partition spec")
	case md.snapshot(s.SnapshotID) != nil:
		return badf("snapshot %d already exists", s.SnapshotID)
	case s.ParentSnapshotID != nil && s.SequenceNumber <= md.LastSequenceNumber:
		return badf("snapshot sequence number %d must be > last sequence number %d", s.SequenceNumber, md.LastSequenceNumber)
	case s.ManifestList == "":
		return badf("snapshot %d has no manifest-list", s.SnapshotID)
	case s.Summary["operation"] == "":
		return badf("snapshot %d summary has no operation", s.SnapshotID)
	}
	if n := len(md.SnapshotLog); n > 0 && s.TimestampMS-md.SnapshotLog[n-1].TimestampMS < -oneMinuteMS {
		return badf("snapshot timestamp %d is before the last snapshot's", s.TimestampMS)
	}
	if s.TimestampMS-max(b.lastUpdatedMS, b.prevLastUpdatedMS) < -oneMinuteMS {
		return badf("snapshot timestamp %d is before the table's last update", s.TimestampMS)
	}
	if md.FormatVersion >= minRowLineageVersion {
		next := int64(0)
		if md.NextRowID != nil {
			next = *md.NextRowID
		}
		switch {
		case s.FirstRowID == nil || s.AddedRows == nil:
			return badf("v3 snapshots need first-row-id and added-rows")
		case *s.FirstRowID < next:
			return badf("first-row-id %d is behind next-row-id %d", *s.FirstRowID, next)
		case *s.AddedRows < 0:
			return badf("negative added-rows")
		}
		next += *s.AddedRows
		md.NextRowID = &next
	}
	md.Snapshots = append(md.Snapshots, s)
	md.LastSequenceNumber = s.SequenceNumber
	b.lastUpdatedMS = s.TimestampMS
	b.addedSnapshots[s.SnapshotID] = true
	return nil
}

func (b *builder) setRef(name string, ref snapshotRef) error {
	md := b.md
	if ref.Type != "branch" && ref.Type != "tag" {
		return badf("invalid ref type %q", ref.Type)
	}
	if name == "" {
		return badf("empty ref name")
	}
	s := md.snapshot(ref.SnapshotID)
	if s == nil {
		return badf("can't set ref %q to unknown snapshot %d", name, ref.SnapshotID)
	}
	if b.addedSnapshots[ref.SnapshotID] {
		b.lastUpdatedMS = s.TimestampMS
	}
	if name == mainBranch {
		if ref.Type != "branch" {
			return badf("main must be a branch")
		}
		ts := b.lastUpdatedMS
		if !b.addedSnapshots[ref.SnapshotID] {
			ts = b.now.UnixMilli()
			b.lastUpdatedMS = ts
		}
		id := ref.SnapshotID
		md.CurrentSnapshotID = &id
		md.SnapshotLog = append(md.SnapshotLog, snapshotLogEntry{SnapshotID: id, TimestampMS: ts})
		b.mainSets = append(b.mainSets, id)
	}
	md.Refs[name] = ref
	return nil
}

func (b *builder) removeSnapshots(ids []int64) error {
	md := b.md
	if md.CurrentSnapshotID != nil && slices.Contains(ids, *md.CurrentSnapshotID) {
		return badf("can't remove the current snapshot")
	}
	md.Snapshots = slices.DeleteFunc(md.Snapshots, func(s snapshot) bool { return slices.Contains(ids, s.SnapshotID) })
	for name, ref := range md.Refs {
		if md.snapshot(ref.SnapshotID) == nil {
			delete(md.Refs, name)
		}
	}
	md.Statistics = slices.DeleteFunc(md.Statistics, func(s statisticsFile) bool { return slices.Contains(ids, s.SnapshotID) })
	md.PartitionStatistics = slices.DeleteFunc(md.PartitionStatistics, func(s partitionStatisticsFile) bool { return slices.Contains(ids, s.SnapshotID) })
	b.removedSnapshots = true
	return nil
}

// finish does the end-of-commit bookkeeping (snapshot/metadata logs,
// last-updated-ms, defaults a create didn't set) and validates the result.
func (b *builder) finish(defaultLocation func() (string, error)) (*tableMetadata, error) {
	md := b.md
	if b.create {
		if md.TableUUID == "" {
			md.TableUUID = uuid.NewString()
		}
		if md.Location == "" {
			loc, err := defaultLocation()
			if err != nil {
				return nil, err
			}
			md.Location = loc
		}
		if md.DefaultSpecID == -1 {
			if err := b.addSpec(jsonObject{"fields": json.RawMessage("[]")}); err != nil {
				return nil, err
			}
			md.DefaultSpecID = *b.lastAddedSpec
		}
		if md.DefaultSortOrderID == -1 {
			if err := b.addSortOrder(jsonObject{"fields": json.RawMessage("[]")}); err != nil {
				return nil, err
			}
			md.DefaultSortOrderID = *b.lastAddedOrder
		}
	}
	b.updateSnapshotLog()

	if b.lastUpdatedMS == 0 {
		b.lastUpdatedMS = b.now.UnixMilli()
	}
	md.LastUpdatedMS = b.lastUpdatedMS
	if b.prevMetadataFile != "" {
		md.MetadataLog = append(md.MetadataLog, metadataLogEntry{MetadataFile: b.prevMetadataFile, TimestampMS: b.prevLastUpdatedMS})
		maxEntries := 100
		if v, err := strconv.Atoi(md.Properties["write.metadata.previous-versions-max"]); err == nil && v >= 1 {
			maxEntries = v
		}
		if n := len(md.MetadataLog); n > maxEntries {
			md.MetadataLog = md.MetadataLog[n-maxEntries:]
		}
	}
	return md, validateMetadata(md)
}

// updateSnapshotLog drops main-branch log entries for snapshots that were
// added and then superseded within this commit, and (after removals) only
// keeps the log suffix after the last removed snapshot, like iceberg-go's
// builder.
func (b *builder) updateSnapshotLog() {
	md := b.md
	intermediate := map[int64]bool{}
	for _, id := range b.mainSets {
		if b.addedSnapshots[id] && (md.CurrentSnapshotID == nil || *md.CurrentSnapshotID != id) {
			intermediate[id] = true
		}
	}
	if len(intermediate) == 0 && !b.removedSnapshots {
		return
	}
	out := make([]snapshotLogEntry, 0, len(md.SnapshotLog))
	for _, e := range md.SnapshotLog {
		switch {
		case md.snapshot(e.SnapshotID) == nil:
			if b.removedSnapshots {
				out = out[:0]
			}
		case !intermediate[e.SnapshotID]:
			out = append(out, e)
		}
	}
	md.SnapshotLog = out
}

// validateMetadata checks the invariants every table we write must hold.
func validateMetadata(md *tableMetadata) error {
	switch {
	case md.FormatVersion < 2 || md.FormatVersion > maxFormatVersion:
		return badf("unsupported format-version %d", md.FormatVersion)
	case md.TableUUID == "":
		return badf("table has no uuid")
	case md.Location == "":
		return badf("table has no location")
	case !findByID(md.Schemas, "schema-id", md.CurrentSchemaID):
		return badf("current schema %d doesn't exist", md.CurrentSchemaID)
	case !findByID(md.PartitionSpecs, "spec-id", md.DefaultSpecID):
		return badf("default partition spec %d doesn't exist", md.DefaultSpecID)
	case !findByID(md.SortOrders, "order-id", md.DefaultSortOrderID):
		return badf("default sort order %d doesn't exist", md.DefaultSortOrderID)
	case md.FormatVersion >= minRowLineageVersion && md.NextRowID == nil:
		return badf("v3 table has no next-row-id")
	}
	if md.FormatVersion < minRowLineageVersion {
		for _, s := range md.Schemas {
			for _, t := range schemaTypes(mustMarshal(s)) {
				if v3Types[t] {
					return badf("type %s requires format-version 3", t)
				}
			}
		}
	}
	for name, ref := range md.Refs {
		if md.snapshot(ref.SnapshotID) == nil {
			return badf("ref %q points at unknown snapshot %d", name, ref.SnapshotID)
		}
	}
	return nil
}

// errCommitFailed marks a failed requirement or a lost CAS (409).
var errCommitFailed = errors.New("commit failed")
