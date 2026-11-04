package icebergcatalog

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/apache/iceberg-go"
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
	if strings.ContainsRune(loc, 0) {
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

// checkCommitLocations validates `set-location` and location-ish
// `set-properties` updates in a raw commit body, rewriting them to their
// normalized form.
func (s *Server) checkCommitLocations(raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) == 0 {
		return raw, nil
	}
	var ups []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &ups); err != nil {
		return nil, fmt.Errorf("%w: malformed updates: %v", errBadRequest, err)
	}
	for _, u := range ups {
		var action string
		_ = json.Unmarshal(u["action"], &action)
		switch action {
		case "set-location":
			var loc string
			if err := json.Unmarshal(u["location"], &loc); err != nil {
				return nil, fmt.Errorf("%w: malformed set-location: %v", errBadRequest, err)
			}
			clean, err := s.checkLocation(loc)
			if err != nil {
				return nil, err
			}
			u["location"], _ = json.Marshal(clean)
		case "set-properties":
			var props iceberg.Properties
			if err := json.Unmarshal(u["updates"], &props); err != nil {
				return nil, fmt.Errorf("%w: malformed set-properties: %v", errBadRequest, err)
			}
			if err := s.checkLocationProps(props); err != nil {
				return nil, err
			}
			u["updates"], _ = json.Marshal(props)
		}
	}
	return json.Marshal(ups)
}
