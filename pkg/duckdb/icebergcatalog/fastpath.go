package icebergcatalog

import (
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/apache/iceberg-go"
	"github.com/apache/iceberg-go/catalog"
	sqlcat "github.com/apache/iceberg-go/catalog/sql"
	icebergio "github.com/apache/iceberg-go/io"
	"github.com/apache/iceberg-go/table"
	"github.com/google/uuid"
	"github.com/klauspost/compress/zstd"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/schema"
)

// CatalogDB is the database (and config) a sql catalog was created w/, for
// the load/commit fast path. See Options.CatalogDB.
type CatalogDB struct {
	DB *sql.DB
	// Dialect is the dialect the sql catalog was created w/. Only
	// sqlcat.SQLite and sqlcat.Postgres are supported, anything else leaves
	// the fast path off.
	Dialect sqlcat.SupportedDialect
	// Name is the sql catalog's name (its `catalog_name` column).
	Name string
	// Properties are the sql catalog's properties. metadata.json files are
	// written through the FileIO they load (plus the table's own
	// properties), same as the sql catalog does.
	Properties iceberg.Properties
}

// fastPath serves loads and commits w/o going through the sql catalog,
// which re-reads and re-parses the table's metadata.json on every LoadTable
// and CommitTable (CommitTable loads it again itself). Both grow w/ the
// table's snapshot count and dominate a small append.
//
// A load is served from the table cache once a one-row read of the table's
// `metadata_location` shows the cached version is still current. A commit
// whose cached base checks out the same way is applied to it w/
// table.UpdateTableMetadata, written to a new metadata.json and swapped in
// w/ the same compare-and-swap on `metadata_location` the sql catalog's
// CommitTable does, so a commit that raced another writer (another server
// sharing the catalog db) fails w/ a 409 instead of being lost. Anything
// else (no cached version, a stale one, creates) goes through the sql
// catalog as before.
//
// The queries match the sql catalog's own (catalog/sql/sql.go in iceberg-go
// v0.7.0): same `catalog_name`/`table_namespace` keys and the same
// `iceberg_type` handling on V0/V1 catalog schemas.
type fastPath struct {
	db    *bun.DB
	name  string
	props iceberg.Properties

	// mu guards the lazily probed catalog schema version.
	mu     sync.Mutex
	probed bool
	v0     bool

	// beforeSwap, when non-nil, runs right before the CAS (tests use it to
	// commit underneath an in-flight commit).
	beforeSwap func()
}

func newFastPath(c *CatalogDB) *fastPath {
	if c == nil || c.DB == nil {
		return nil
	}
	var d schema.Dialect
	switch c.Dialect {
	case sqlcat.SQLite:
		d = sqlitedialect.New()
	case sqlcat.Postgres:
		d = pgdialect.New()
	default:
		return nil
	}
	return &fastPath{db: bun.NewDB(c.DB, d), name: c.Name, props: c.Properties}
}

// icebergTableRow is the sql catalog's `iceberg_tables` row (its
// sqlIcebergTable isn't exported).
type icebergTableRow struct {
	bun.BaseModel `bun:"table:iceberg_tables"`

	CatalogName              string `bun:",pk"`
	TableNamespace           string `bun:",pk"`
	TableName                string `bun:",pk"`
	IcebergType              sql.NullString
	MetadataLocation         sql.NullString
	PreviousMetadataLocation sql.NullString
}

// isV0 reports whether the catalog db is on the legacy V0 schema (no
// `iceberg_type` column), probed the same way the sql catalog does on
// NewCatalog. Errors aren't cached, the next call probes again.
func (f *fastPath) isV0(ctx context.Context) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.probed {
		return f.v0, nil
	}
	var q *bun.SelectQuery
	switch f.db.Dialect().Name().String() {
	case "sqlite":
		q = f.db.NewSelect().TableExpr("pragma_table_info(?)", "iceberg_tables").
			Where("name = ?", "iceberg_type")
	default:
		q = f.db.NewSelect().TableExpr("information_schema.columns").
			Where("table_schema = current_schema()").
			Where("table_name = ?", "iceberg_tables").
			Where("column_name = ?", "iceberg_type")
	}
	exists, err := q.ColumnExpr("1").Limit(1).Exists(ctx)
	if err != nil {
		return false, fmt.Errorf("probing for iceberg_type column: %w", err)
	}
	f.probed, f.v0 = true, !exists
	return f.v0, nil
}

// encodedNamespacePrefix is the sql catalog's marker for namespaces that
// can't be stored as their dot-joined name.
const encodedNamespacePrefix = "__iceberg_namespace_v1__:"

// namespaceKey is ns's `table_namespace` value, the sql catalog's
// namespaceToString. ok is false when the sql catalog would have to look
// the namespace up to pick between two keys (a level containing the
// encoded marker, see its namespaceStorageKeys), which the fast path
// leaves to it.
func namespaceKey(ns table.Identifier) (string, bool) {
	legacy := strings.Join(ns, ".")
	dotted := slices.ContainsFunc(ns, func(p string) bool { return strings.Contains(p, ".") })
	if !dotted {
		return legacy, !strings.Contains(legacy, encodedNamespacePrefix)
	}
	b, err := json.Marshal(ns)
	if err != nil {
		return "", false
	}
	return encodedNamespacePrefix + string(b), true
}

// rowKey is a table's `iceberg_tables` key (w/o catalog_name) plus whether
// the catalog db is on the V0 schema.
type rowKey struct {
	ns, name string
	v0       bool
}

// rowKey is ident's rowKey, ok=false if it's not a table the fast path can
// find (see namespaceKey).
func (f *fastPath) rowKey(ctx context.Context, ident table.Identifier) (rowKey, bool, error) {
	if len(ident) < 2 {
		return rowKey{}, false, nil
	}
	nsKey, ok := namespaceKey(catalog.NamespaceFromIdent(ident))
	if !ok {
		return rowKey{}, false, nil
	}
	v0, err := f.isV0(ctx)
	if err != nil {
		return rowKey{}, false, err
	}
	return rowKey{ns: nsKey, name: catalog.TableNameFromIdent(ident), v0: v0}, true, nil
}

// currentLocation is ident's `metadata_location`, ok=false if it's not a
// table the fast path can find (missing, or see namespaceKey).
func (f *fastPath) currentLocation(ctx context.Context, ident table.Identifier) (string, bool, error) {
	k, ok, err := f.rowKey(ctx, ident)
	if err != nil || !ok {
		return "", false, err
	}
	sel := f.db.NewSelect().Model((*icebergTableRow)(nil)).
		Column("metadata_location").
		Where("catalog_name = ?", f.name).
		Where("table_namespace = ?", k.ns).
		Where("table_name = ?", k.name)
	if !k.v0 {
		sel = sel.Where("(iceberg_type = ? OR iceberg_type IS NULL)", sqlcat.TableType)
	}
	var loc sql.NullString
	if err := sel.Scan(ctx, &loc); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", false, nil
		}
		return "", false, err
	}
	return loc.String, loc.Valid, nil
}

// errNotCurrent is a cached base that's not the current version anymore.
var errNotCurrent = fmt.Errorf("%w: metadata-location moved underneath us", table.ErrCommitFailed)

// commit applies updates to base, the current version as of a
// currentLocation check, writes the result and swaps it in. A concurrent
// commit since that check fails it w/ table.ErrCommitFailed.
func (f *fastPath) commit(ctx context.Context, ident table.Identifier, base cachedTable, reqs []table.Requirement, updates []table.Update) (table.Metadata, string, error) {
	for _, r := range reqs {
		if err := r.Validate(base.md); err != nil {
			return nil, "", fmt.Errorf("%w: %w", table.ErrCommitFailed, err)
		}
	}
	md, err := table.UpdateTableMetadata(base.md, updates, base.loc)
	if err != nil {
		return nil, "", err
	}
	if md.Equals(base.md) {
		// Nothing changed, same as the sql catalog.
		return base.md, base.loc, nil
	}
	v := parseMetadataVersion(base.loc)
	if v < 0 {
		return nil, "", fmt.Errorf("invalid metadata location: %s", base.loc)
	}
	provider, err := table.LoadLocationProvider(md.Location(), md.Properties())
	if err != nil {
		return nil, "", err
	}
	loc, err := provider.NewTableMetadataFileLocation(v + 1)
	if err != nil {
		return nil, "", err
	}
	ioProps := maps.Clone(f.props)
	if ioProps == nil {
		ioProps = iceberg.Properties{}
	}
	maps.Copy(ioProps, md.Properties())
	fsys, err := icebergio.LoadFS(ctx, ioProps, loc)
	if err != nil {
		return nil, "", err
	}
	wfs, ok := fsys.(icebergio.WriteFileIO)
	if !ok {
		return nil, "", errors.New("filesystem IO does not support writing")
	}
	codec := md.Properties().Get(table.MetadataCompressionKey, table.MetadataCompressionDefault)
	if err := writeTableMetadata(md, wfs, loc, codec); err != nil {
		return nil, "", err
	}

	if f.beforeSwap != nil {
		f.beforeSwap()
	}
	err = f.swap(ctx, ident, base.loc, loc)
	if err != nil {
		// Nothing points at it. Best effort, same as the sql catalog.
		_ = wfs.Remove(loc)
		return nil, "", err
	}
	return md, loc, nil
}

// swap points ident at loc if it's still at prev.
func (f *fastPath) swap(ctx context.Context, ident table.Identifier, prev, loc string) error {
	k, ok, err := f.rowKey(ctx, ident)
	if err != nil {
		return err
	}
	if !ok {
		return errNotCurrent
	}
	upd := f.db.NewUpdate().Model(&icebergTableRow{
		CatalogName:              f.name,
		TableNamespace:           k.ns,
		TableName:                k.name,
		IcebergType:              sql.NullString{String: sqlcat.TableType, Valid: true},
		MetadataLocation:         sql.NullString{String: loc, Valid: true},
		PreviousMetadataLocation: sql.NullString{String: prev, Valid: true},
	}).WherePK().Where("metadata_location = ?", prev)
	if k.v0 {
		upd = upd.ExcludeColumn("iceberg_type")
	} else {
		// Same as the sql catalog: this also heals a legacy NULL.
		upd = upd.Where("(iceberg_type = ? OR iceberg_type IS NULL)", sqlcat.TableType)
	}
	res, err := upd.Exec(ctx)
	if err != nil {
		return fmt.Errorf("error updating table information: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error updating table information: %w", err)
	}
	if n == 0 {
		return errNotCurrent
	}
	return nil
}

// writeTableMetadata is iceberg-go's internal WriteTableMetadata.
func writeTableMetadata(md table.Metadata, fsys icebergio.WriteFileIO, loc, codec string) (err error) {
	switch codec {
	case table.MetadataCompressionCodecNone, table.MetadataCompressionCodecGzip, table.MetadataCompressionCodecZstd:
	default:
		return fmt.Errorf("unsupported write metadata compression codec: %s", codec)
	}
	out, err := fsys.Create(loc)
	if err != nil {
		return err
	}
	defer closeInto(out, &err)

	var w io.Writer = out
	switch codec {
	case table.MetadataCompressionCodecGzip:
		gz := gzip.NewWriter(out)
		defer closeInto(gz, &err)
		w = gz
	case table.MetadataCompressionCodecZstd:
		enc, err := zstd.NewWriter(out)
		if err != nil {
			return err
		}
		defer closeInto(enc, &err)
		w = enc
	}
	return json.NewEncoder(w).Encode(md)
}

func closeInto(c io.Closer, err *error) {
	if cerr := c.Close(); cerr != nil && *err == nil {
		*err = cerr
	}
}

// Same as iceberg-go's internal ParseMetadataVersion:
// `<version>-<uuid>[.<codec>].metadata.json`, or the legacy
// `v<version>[.gz].metadata.json`.
var (
	metadataFileNameRegex       = regexp.MustCompile(`^(\d+)-([\w-]{36})(?:\.\w+)?\.metadata\.json$`)
	legacyMetadataFileNameRegex = regexp.MustCompile(`^v(\d+)(?:\.gz)?\.metadata\.json$`)
)

// parseMetadataVersion is loc's metadata version, -1 if it doesn't parse.
func parseMetadataVersion(loc string) int {
	if strings.HasSuffix(loc, "/") {
		return -1
	}
	name := path.Base(loc)
	m := metadataFileNameRegex.FindStringSubmatch(name)
	if len(m) == 3 {
		if _, err := uuid.Parse(m[2]); err != nil {
			return -1
		}
	} else if m = legacyMetadataFileNameRegex.FindStringSubmatch(name); len(m) != 2 {
		return -1
	}
	v, err := strconv.Atoi(m[1])
	if err != nil {
		return -1
	}
	return v
}
