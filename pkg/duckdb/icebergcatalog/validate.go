package icebergcatalog

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/apache/iceberg-go"
	"github.com/apache/iceberg-go/table"
)

// errBadRequest marks validation failures, which classify maps to a 400.
var errBadRequest = errors.New("bad request")

// locationProperties are the table/namespace properties that point the
// catalog or a writer at storage. Each one has to stay under the warehouse
// like a table location does, or a client could redirect metadata/data
// files anywhere the server (or duckdb) can write.
var locationProperties = []string{
	"location",
	"write.data.path",
	"write.metadata.path",
	"write.folder-storage.path",
	"write.object-storage.path",
}

// validateName rejects namespace levels and table names that could escape
// the warehouse once they're joined into a default location
// (`<warehouse>/<ns>.db/<table>`).
func validateName(name string) error {
	if name == "" {
		return fmt.Errorf("%w: empty name", errBadRequest)
	}
	if strings.ContainsAny(name, `/\`) || strings.Contains(name, "..") {
		return fmt.Errorf("%w: invalid name %q", errBadRequest, name)
	}
	for _, r := range name {
		if r < 0x20 || r == 0x7f {
			return fmt.Errorf("%w: invalid name %q", errBadRequest, name)
		}
	}
	return nil
}

func validateNames(names []string) error {
	if len(names) == 0 {
		return fmt.Errorf("%w: empty identifier", errBadRequest)
	}
	for _, n := range names {
		if err := validateName(n); err != nil {
			return err
		}
	}
	return nil
}

// checkLocation normalizes loc and errors unless it's the warehouse or under
// it. Remote warehouses (s3://...) need the same scheme + host and a cleaned
// path under the warehouse's. Local ones compare cleaned absolute paths w/
// symlinks resolved (for whatever part of loc already exists), so a symlink
// inside the warehouse can't point a table outside it.
func (s *Server) checkLocation(loc string) (string, error) {
	return s.locations().check(loc)
}

// locationChecker is checkLocation w/ the local warehouse root's symlinks
// resolved once, for checking a batch of paths (checkMetadata).
type locationChecker struct {
	s    *Server
	root string
}

func (s *Server) locations() *locationChecker { return &locationChecker{s: s} }

func (c *locationChecker) check(loc string) (string, error) {
	s := c.s
	if s.opts.Warehouse == "" {
		return "", fmt.Errorf("%w: no warehouse configured, explicit locations aren't allowed", errBadRequest)
	}
	if strings.ContainsRune(loc, 0) || hasDotSegment(loc) {
		return "", fmt.Errorf("%w: invalid location %q", errBadRequest, loc)
	}

	wh, err := url.Parse(s.opts.Warehouse)
	if err != nil {
		return "", fmt.Errorf("parsing warehouse: %w", err)
	}
	if isLocal(wh) {
		if c.root == "" {
			if c.root, err = resolveExisting(filepath.Clean(localPath(s.opts.Warehouse, wh))); err != nil {
				return "", err
			}
		}
		return checkLocalLocation(c.root, loc)
	}

	u, err := url.Parse(loc)
	if err != nil {
		return "", fmt.Errorf("%w: invalid location %q: %v", errBadRequest, loc, err)
	}
	if u.Scheme != wh.Scheme || u.Host != wh.Host || u.RawQuery != "" || u.Fragment != "" {
		return "", fmt.Errorf("%w: location %q is outside the warehouse", errBadRequest, loc)
	}
	root := path.Clean("/" + wh.Path)
	p := path.Clean("/" + u.Path)
	if !underRoot(p, root, "/") {
		return "", fmt.Errorf("%w: location %q is outside the warehouse", errBadRequest, loc)
	}
	u.Path = p
	return u.String(), nil
}

// hasDotSegment reports a `.` or `..` path segment. These are rejected
// rather than cleaned: object stores don't resolve them, so a cleaned check
// and the key actually written could disagree.
func hasDotSegment(loc string) bool {
	for _, seg := range strings.FieldsFunc(loc, func(r rune) bool { return r == '/' || r == '\\' }) {
		if seg == "." || seg == ".." {
			return true
		}
	}
	return false
}

func isLocal(u *url.URL) bool {
	return u.Scheme == "" || u.Scheme == "file"
}

// localPath turns a plain path or file:// URL into a filesystem path.
func localPath(raw string, u *url.URL) string {
	if u.Scheme == "file" {
		return u.Path
	}
	return raw
}

// checkLocalLocation checks loc against root, the warehouse w/ its symlinks
// already resolved.
func checkLocalLocation(root, loc string) (string, error) {
	u, err := url.Parse(loc)
	if err != nil || !isLocal(u) {
		return "", fmt.Errorf("%w: location %q is outside the warehouse", errBadRequest, loc)
	}
	p := localPath(loc, u)
	if !filepath.IsAbs(p) {
		return "", fmt.Errorf("%w: location %q must be absolute", errBadRequest, loc)
	}
	clean := filepath.Clean(p)
	resolved, err := resolveExisting(clean)
	if err != nil {
		return "", err
	}
	if !underRoot(resolved, root, string(filepath.Separator)) {
		return "", fmt.Errorf("%w: location %q is outside the warehouse", errBadRequest, loc)
	}
	return clean, nil
}

// resolveExisting resolves symlinks in the longest existing prefix of p and
// re-appends the rest (which can't contain symlinks since it doesn't exist).
func resolveExisting(p string) (string, error) {
	rest := ""
	cur := p
	for {
		r, err := filepath.EvalSymlinks(cur)
		if err == nil {
			return filepath.Join(r, rest), nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("resolving %q: %w", cur, err)
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return p, nil
		}
		rest = filepath.Join(filepath.Base(cur), rest)
		cur = parent
	}
}

// underRoot is a path-prefix check w/ a trailing separator, so `/wh-evil`
// isn't under `/wh`.
func underRoot(p, root, sep string) bool {
	if p == root {
		return true
	}
	return strings.HasPrefix(p, strings.TrimSuffix(root, sep)+sep)
}

// checkLocationProps validates (and normalizes in place) every location-ish
// property in props.
func (s *Server) checkLocationProps(props iceberg.Properties) error {
	for _, k := range locationProperties {
		v, ok := props[k]
		if !ok {
			continue
		}
		clean, err := s.checkLocation(v)
		if err != nil {
			return fmt.Errorf("property %q: %w", k, err)
		}
		props[k] = clean
	}
	return nil
}

// allowedUpdates are the commit update actions we apply: what duckdb sends
// (see the smoke test) plus what a spec-compliant client managing snapshots,
// properties and statistics would. Anything else (encryption keys, removing
// schemas/specs) is rejected rather than reasoned about.
var allowedUpdates = map[string]bool{
	table.UpdateAssignUUID:                true,
	table.UpdateUpgradeFormatVersion:      true,
	table.UpdateAddSchema:                 true,
	table.UpdateSetCurrentSchema:          true,
	table.UpdateAddSpec:                   true,
	table.UpdateSetDefaultSpec:            true,
	table.UpdateAddSortOrder:              true,
	table.UpdateSetDefaultSortOrder:       true,
	table.UpdateSetLocation:               true,
	table.UpdateAddSnapshot:               true,
	table.UpdateSetSnapshotRef:            true,
	table.UpdateRemoveSnapshots:           true,
	table.UpdateRemoveSnapshotRef:         true,
	table.UpdateSetProperties:             true,
	table.UpdateRemoveProperties:          true,
	table.UpdateSetStatistics:             true,
	table.UpdateRemoveStatistics:          true,
	table.UpdateSetPartitionStatistics:    true,
	table.UpdateRemovePartitionStatistics: true,
}

// checkCommit validates a commit against what iceberg-go will actually
// apply: updates is iceberg-go's own decode of the request (so a duplicate
// or mixed-case `action` key resolves exactly the way CommitTable sees it),
// every action has to be allowlisted, and the metadata the updates produce
// from base (nil for a create) has to keep every path in it inside the
// warehouse. updates is applied to a throwaway builder, so pass a separate
// decode from the one handed to CommitTable. It returns the metadata the
// updates produce.
func (s *Server) checkCommit(base table.Metadata, updates table.Updates) (table.Metadata, error) {
	for _, u := range updates {
		if !allowedUpdates[u.Action()] {
			return nil, fmt.Errorf("%w: update action %q is not supported", errBadRequest, u.Action())
		}
	}
	if base == nil {
		// Same placeholder base iceberg-go's CommitTable creates from.
		var err error
		base, err = table.NewMetadata(iceberg.NewSchema(0), nil, table.UnsortedSortOrder, "", nil)
		if err != nil {
			return nil, err
		}
	}
	md, err := table.UpdateTableMetadata(base, updates, "")
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errBadRequest, err)
	}
	return md, s.checkMetadata(md, base)
}

// checkMetadata checks every path table metadata carries: the table
// location and location-ish properties have to be under the warehouse, and
// manifest lists, v1 manifests and (partition) statistics files under the
// table location or one of those properties. These are what the catalog
// writes metadata to and what a purge would delete.
//
// prev is the metadata md was built from (nil to check everything). If md
// keeps prev's location + location properties, paths prev already carried
// were checked when prev was committed, so only the ones md adds are
// (re)checked. Otherwise every commit would re-resolve every manifest list
// in the table's history, which grows w/ each snapshot.
func (s *Server) checkMetadata(md, prev table.Metadata) error {
	c := s.locations()
	loc, err := c.check(md.Location())
	if err != nil {
		return fmt.Errorf("table location: %w", err)
	}
	roots := []string{loc}
	props := md.Properties()
	for _, k := range locationProperties {
		v, ok := props[k]
		if !ok {
			continue
		}
		clean, err := c.check(v)
		if err != nil {
			return fmt.Errorf("property %q: %w", k, err)
		}
		roots = append(roots, clean)
	}

	var known map[string]bool
	if prev != nil && sameRoots(md, prev) {
		known = map[string]bool{}
		for _, p := range metadataPaths(prev) {
			known[p] = true
		}
	}
	for _, p := range metadataPaths(md) {
		if known[p] {
			continue
		}
		if err := c.checkUnder(p, roots); err != nil {
			return err
		}
	}
	return nil
}

// metadataPaths is every file path md references besides its location.
func metadataPaths(md table.Metadata) []string {
	var paths []string
	for _, snap := range md.Snapshots() {
		if snap.ManifestList != "" {
			paths = append(paths, snap.ManifestList)
		}
		paths = append(paths, snap.ManifestLocations...)
	}
	for st := range md.Statistics() {
		paths = append(paths, st.StatisticsPath)
	}
	for st := range md.PartitionStatistics() {
		paths = append(paths, st.StatisticsPath)
	}
	return paths
}

// sameRoots reports whether a and b have the same location and
// location-ish properties, IE the same roots checkMetadata checks paths
// against.
func sameRoots(a, b table.Metadata) bool {
	if a.Location() != b.Location() {
		return false
	}
	ap, bp := a.Properties(), b.Properties()
	for _, k := range locationProperties {
		av, aok := ap[k]
		bv, bok := bp[k]
		if aok != bok || av != bv {
			return false
		}
	}
	return true
}

// checkUnder checks p is under the warehouse and one of roots (each already
// normalized by check).
func (c *locationChecker) checkUnder(p string, roots []string) error {
	clean, err := c.check(p)
	if err != nil {
		return err
	}
	sep := "/"
	if !strings.Contains(clean, "://") {
		sep = string(filepath.Separator)
	}
	for _, root := range roots {
		if underRoot(clean, root, sep) {
			return nil
		}
	}
	return fmt.Errorf("%w: %q is outside the table location", errBadRequest, p)
}

// purgeLocation is the local directory a purge may delete for a table w/
// metadata md: its location, if that's strictly inside a local warehouse
// (never the warehouse itself). It's empty for a remote warehouse, which
// purge doesn't delete from (yet).
func (s *Server) purgeLocation(md table.Metadata) (string, error) {
	if err := s.checkMetadata(md, nil); err != nil {
		return "", err
	}
	wh, err := url.Parse(s.opts.Warehouse)
	if err != nil {
		return "", err
	}
	if !isLocal(wh) {
		return "", nil
	}
	loc, err := s.checkLocation(md.Location())
	if err != nil {
		return "", err
	}
	root := filepath.Clean(localPath(s.opts.Warehouse, wh))
	if loc == root || !underRoot(loc, root, string(filepath.Separator)) {
		return "", fmt.Errorf("%w: refusing to purge %q", errBadRequest, loc)
	}
	return loc, nil
}
