package icebergcatalog

import (
	"context"
	"encoding/json"
	"strings"
	"sync"

	icebergio "github.com/apache/iceberg-go/io"
	"github.com/apache/iceberg-go/table"
)

const maxTableCacheEntries = 4096

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
	mu         sync.Mutex
	m          map[string]cachedTable
	maxEntries int
}

type cachedTable struct {
	loc  string
	md   table.Metadata
	body json.RawMessage
	// fsF is the loaded table's FS factory (bound to the catalog's trusted
	// properties), kept so remote metadata cleanup works even w/o CatalogDB
	// or StorageProperties.
	fsF func(context.Context) (icebergio.IO, error)
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
	return c.bodyFS(ident, loc, md, nil)
}

func (c *tableCache) bodyFS(ident table.Identifier, loc string, md table.Metadata, fsF func(context.Context) (icebergio.IO, error)) (json.RawMessage, error) {
	if t, ok := c.get(ident); ok && t.loc == loc {
		if t.fsF == nil && fsF != nil {
			c.mu.Lock()
			if cur, exists := c.m[tableKey(ident)]; exists && cur.loc == loc && cur.fsF == nil {
				cur.fsF = fsF
				c.m[tableKey(ident)] = cur
			}
			c.mu.Unlock()
		}
		return t.body, nil
	}
	return c.putFS(ident, loc, md, fsF)
}

// put encodes md and caches it as ident's latest version, refusing to
// regress an existing entry for the same table UUID to an older metadata
// version and bounding total cache size.
func (c *tableCache) put(ident table.Identifier, loc string, md table.Metadata) (json.RawMessage, error) {
	return c.putFS(ident, loc, md, nil)
}

func (c *tableCache) putFS(ident table.Identifier, loc string, md table.Metadata, fsF func(context.Context) (icebergio.IO, error)) (json.RawMessage, error) {
	body, err := json.Marshal(md)
	if err != nil {
		return nil, err
	}
	key := tableKey(ident)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.m == nil {
		c.m = map[string]cachedTable{}
	}
	if existing, ok := c.m[key]; ok {
		if existing.md != nil && md != nil && existing.md.TableUUID() == md.TableUUID() {
			if fsF == nil {
				fsF = existing.fsF
			}
			oldV := parseMetadataVersion(existing.loc)
			newV := parseMetadataVersion(loc)
			if oldV >= 0 && newV >= 0 && newV < oldV {
				return body, nil
			}
		}
	} else {
		limit := c.maxEntries
		if limit <= 0 {
			limit = maxTableCacheEntries
		}
		if len(c.m) >= limit {
			for k := range c.m {
				delete(c.m, k)
				break
			}
		}
	}
	c.m[key] = cachedTable{loc: loc, md: md, body: body, fsF: fsF}
	return body, nil
}

func (c *tableCache) drop(idents ...table.Identifier) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, ident := range idents {
		delete(c.m, tableKey(ident))
	}
}
