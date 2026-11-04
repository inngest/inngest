package icebergcatalog

import (
	"encoding/json"
	"strings"
	"sync"

	"github.com/apache/iceberg-go/table"
)

// tableCache keeps each table's latest metadata this server has seen (from a
// load or its own commit) along w/ its JSON encoding, keyed by table and
// checked against the metadata location so a stale entry is never served.
//
// It exists because duckdb loads a table before every write and the commit
// that follows needs the same metadata as its base: w/o it each write
// parses the table's metadata 3 times (load, commit pre-validation,
// iceberg-go's own CommitTable load) and encodes it twice, all of which
// grow w/ the table's snapshot count.
type tableCache struct {
	mu sync.Mutex
	m  map[string]cachedTable
}

type cachedTable struct {
	loc  string
	md   table.Metadata
	body json.RawMessage
}

func tableKey(ident table.Identifier) string { return strings.Join(ident, "\x00") }

// get returns ident's cached metadata, if any. It's only the latest version
// this server saw, so callers that must not act on a stale version check
// loc against the catalog (see body) or tolerate it (see commitTable).
func (c *tableCache) get(ident table.Identifier) (cachedTable, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	t, ok := c.m[tableKey(ident)]
	return t, ok
}

// body returns md's JSON encoding, reusing the cached one when loc is the
// cached location and caching it otherwise.
func (c *tableCache) body(ident table.Identifier, loc string, md table.Metadata) (json.RawMessage, error) {
	if t, ok := c.get(ident); ok && t.loc == loc {
		return t.body, nil
	}
	return c.put(ident, loc, md)
}

// put encodes md and caches it as ident's latest version.
func (c *tableCache) put(ident table.Identifier, loc string, md table.Metadata) (json.RawMessage, error) {
	body, err := json.Marshal(md)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.m == nil {
		c.m = map[string]cachedTable{}
	}
	c.m[tableKey(ident)] = cachedTable{loc: loc, md: md, body: body}
	return body, nil
}

func (c *tableCache) drop(idents ...table.Identifier) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, ident := range idents {
		delete(c.m, tableKey(ident))
	}
}
