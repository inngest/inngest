package icebergcatalog

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
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
		return checkLocalLocation(localPath(s.opts.Warehouse, wh), loc)
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

func checkLocalLocation(warehouse, loc string) (string, error) {
	u, err := url.Parse(loc)
	if err != nil || !isLocal(u) {
		return "", fmt.Errorf("%w: location %q is outside the warehouse", errBadRequest, loc)
	}
	p := localPath(loc, u)
	if !filepath.IsAbs(p) {
		return "", fmt.Errorf("%w: location %q must be absolute", errBadRequest, loc)
	}
	root, err := resolveExisting(filepath.Clean(warehouse))
	if err != nil {
		return "", err
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
func (s *Server) checkLocationProps(props map[string]string) error {
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

// checkMetadata checks every path table metadata carries: the table
// location and location-ish properties have to be under the warehouse, and
// manifest lists, v1 manifests and (partition) statistics files under the
// table location or one of those properties. These are what the catalog
// writes metadata to and what a purge would delete.
func (s *Server) checkMetadata(md *tableMetadata) error {
	loc, err := s.checkLocation(md.Location)
	if err != nil {
		return fmt.Errorf("table location: %w", err)
	}
	roots := []string{loc}
	props := md.Properties
	for _, k := range locationProperties {
		v, ok := props[k]
		if !ok {
			continue
		}
		clean, err := s.checkLocation(v)
		if err != nil {
			return fmt.Errorf("property %q: %w", k, err)
		}
		roots = append(roots, clean)
	}

	var paths []string
	for _, snap := range md.Snapshots {
		if snap.ManifestList != "" {
			paths = append(paths, snap.ManifestList)
		}
		paths = append(paths, snap.Manifests...)
	}
	for _, st := range md.Statistics {
		paths = append(paths, st.StatisticsPath)
	}
	for _, st := range md.PartitionStatistics {
		paths = append(paths, st.StatisticsPath)
	}
	for _, p := range paths {
		if err := s.checkUnder(p, roots); err != nil {
			return err
		}
	}
	return nil
}

// checkUnder checks p is under the warehouse and one of roots (each already
// normalized by checkLocation).
func (s *Server) checkUnder(p string, roots []string) error {
	clean, err := s.checkLocation(p)
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
// metadata md: its location, if that's strictly inside the (local)
// warehouse, never the warehouse itself.
func (s *Server) purgeLocation(md *tableMetadata) (string, error) {
	if err := s.checkMetadata(md); err != nil {
		return "", err
	}
	wh, err := url.Parse(s.opts.Warehouse)
	if err != nil {
		return "", err
	}
	if !isLocal(wh) {
		return "", fmt.Errorf("%w: purging a remote warehouse isn't supported", errBadRequest)
	}
	loc, err := s.checkLocation(md.Location)
	if err != nil {
		return "", err
	}
	root := filepath.Clean(localPath(s.opts.Warehouse, wh))
	if loc == root || !underRoot(loc, root, string(filepath.Separator)) {
		return "", fmt.Errorf("%w: refusing to purge %q", errBadRequest, loc)
	}
	return loc, nil
}
