package icebergcatalog

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

// createUpdates turns a create-table request into the updates of an
// equivalent create commit, w/ fresh ids assigned the way Java/iceberg-go
// do on create: schema field ids renumbered from 1 (AssignFreshIds order),
// partition field ids from 1000 and spec/order source ids remapped to
// match. The `format-version` property picks the format version (default
// 2) and isn't stored as a property, like Java's reserved properties.
func createUpdates(req createTableRequest) ([]update, error) {
	formatVersion := 2
	props := map[string]string{}
	for k, v := range req.Properties {
		props[k] = v
	}
	if v, ok := props["format-version"]; ok {
		n, err := strconv.Atoi(v)
		if err != nil {
			return nil, badf("invalid format-version %q", v)
		}
		formatVersion = n
		delete(props, "format-version")
	}

	next := 0
	ids := map[int]int{}
	schema, err := walkSchemaIDs(req.Schema, func(old int) int {
		next++
		ids[old] = next
		return next
	})
	if err != nil {
		return nil, badf("invalid schema: %v", err)
	}
	schema.setInt("schema-id", 0)
	if raw, ok := schema["identifier-field-ids"]; ok {
		var old []int
		if err := json.Unmarshal(raw, &old); err != nil {
			return nil, badf("invalid identifier-field-ids: %v", err)
		}
		fresh := make([]int, len(old))
		for i, id := range old {
			if fresh[i], ok = ids[id]; !ok {
				return nil, badf("identifier field %d isn't in the schema", id)
			}
		}
		schema["identifier-field-ids"] = mustMarshal(fresh)
	}

	remapSources := func(what string, f jsonObject) error {
		if id, ok := f.int("source-id"); ok {
			fresh, ok := ids[id]
			if !ok {
				return badf("%s source field %d isn't in the schema", what, id)
			}
			f.setInt("source-id", fresh)
		}
		if raw, ok := f["source-ids"]; ok {
			var old []int
			if err := json.Unmarshal(raw, &old); err != nil {
				return badf("invalid %s source-ids: %v", what, err)
			}
			for i, id := range old {
				if old[i], ok = ids[id]; !ok {
					return badf("%s source field %d isn't in the schema", what, id)
				}
			}
			f["source-ids"] = mustMarshal(old)
		}
		return nil
	}
	freshFields := func(what string, obj jsonObject, partition bool) (jsonObject, error) {
		fields, err := obj.objects("fields")
		if err != nil {
			return nil, badf("invalid %s: %v", what, err)
		}
		for i, f := range fields {
			if err := remapSources(what, f); err != nil {
				return nil, err
			}
			if partition {
				f.setInt("field-id", partitionDataIDStart+i)
			}
		}
		out := jsonObject{}
		out.setObjects("fields", fields)
		return out, nil
	}
	spec, err := freshFields("partition spec", req.PartitionSpec, true)
	if err != nil {
		return nil, err
	}
	order, err := freshFields("sort order", req.WriteOrder, false)
	if err != nil {
		return nil, err
	}

	updates := []update{
		{action: "assign-uuid", uuid: uuid.NewString()},
		{action: "upgrade-format-version", formatVersion: formatVersion},
		{action: "add-schema", obj: schema},
		{action: "set-current-schema", id: -1},
		{action: "add-spec", obj: spec},
		{action: "set-default-spec", id: -1},
		{action: "add-sort-order", obj: order},
		{action: "set-default-sort-order", id: -1},
		{action: "set-properties", props: props},
	}
	if req.Location != "" {
		updates = append(updates, update{action: "set-location", location: req.Location})
	}
	return updates, nil
}

// defaultLocation is `<namespace location>/<table>` or
// `<warehouse>/<ns>.db/<table>`, iceberg-go/Java's default.
func (s *Server) defaultLocation(ctx context.Context, id identifier) (string, error) {
	props, err := s.store.loadNamespace(ctx, id.Namespace)
	if err != nil {
		return "", err
	}
	var loc string
	switch {
	case props["location"] != "":
		loc = strings.TrimSuffix(props["location"], "/") + "/" + id.Name
	case s.opts.Warehouse == "":
		return "", badf("no warehouse configured and no location given")
	default:
		loc = filepath.Join(s.opts.Warehouse, strings.Join(id.Namespace, ".")+".db", id.Name)
	}
	// Names and the namespace location are validated on the way in, this
	// just makes sure nothing slipped through (IE a pre-existing property).
	return s.checkLocation(loc)
}

// metadataFileRe matches `<version>-<uuid>.metadata.json` (and the legacy
// `v<version>.metadata.json`).
var metadataFileRe = regexp.MustCompile(`^(?:(\d+)-[0-9a-fA-F-]{36}|v(\d+))(?:\.gz)?\.metadata\.json$`)

// writeMetadata writes md as the next metadata file (`<metadata dir>/
// <version>-<uuid>.metadata.json`, version = prev's + 1) and returns its
// path. The metadata dir is `write.metadata.path` if set (checkMetadata
// keeps that under the warehouse) or `<location>/metadata`.
func (s *Server) writeMetadata(md *tableMetadata, prevLoc string) (string, error) {
	version := 0
	if prevLoc != "" {
		m := metadataFileRe.FindStringSubmatch(path.Base(prevLoc))
		if m == nil {
			return "", fmt.Errorf("can't parse metadata version from %q", prevLoc)
		}
		v, _ := strconv.Atoi(m[1] + m[2])
		version = v + 1
	}
	dir := md.Properties["write.metadata.path"]
	if dir == "" {
		dir = md.Location + "/metadata"
	}
	dir, err := s.checkLocation(dir)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	loc := filepath.Join(dir, fmt.Sprintf("%05d-%s.metadata.json", version, uuid.NewString()))
	b, err := json.Marshal(md)
	if err != nil {
		return "", err
	}
	f, err := os.OpenFile(loc, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return "", err
	}
	if _, err := f.Write(b); err != nil {
		_ = f.Close()
		_ = os.Remove(loc)
		return "", err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		_ = os.Remove(loc)
		return "", err
	}
	return loc, f.Close()
}

// readMetadata reads a metadata file the catalog points at. Locations come
// from our own db, so they're trusted.
func readMetadata(loc string) (*tableMetadata, error) {
	p := strings.TrimPrefix(loc, "file://")
	if strings.HasSuffix(p, ".gz.metadata.json") {
		return nil, fmt.Errorf("compressed metadata isn't supported: %s", loc)
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	return parseMetadata(b)
}
