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
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/apache/iceberg-go"
	iceio "github.com/apache/iceberg-go/io"
	lru "github.com/hashicorp/golang-lru/v2"
)

// snapshotManifestCacheSize bounds the number of decoded manifest lists held
// by a table. A table normally reuses its current snapshot, while a bounded
// history is enough to retain useful locality for repeated historical scans.
const snapshotManifestCacheSize = 64

// snapshotManifestCacheManifestLimit bounds the total number of manifest
// descriptors retained by the cache. Memory per descriptor also depends on
// paths, partition summaries, and key metadata; this is not a byte limit.
const snapshotManifestCacheManifestLimit = 32 * 1024

// snapshotManifestSet keeps the complete manifest list and data partition
// together. Manifest descriptors are immutable for an Iceberg snapshot, so a
// table can safely share this decoded result across scans.
type snapshotManifestSet struct {
	all  []iceberg.ManifestFile
	data []iceberg.ManifestFile
}

func newSnapshotManifestSet(manifests []iceberg.ManifestFile) snapshotManifestSet {
	set := snapshotManifestSet{all: manifests}
	dataCount := 0
	for _, manifest := range manifests {
		if manifest.ManifestContent() == iceberg.ManifestContentData {
			dataCount++
		}
	}
	if dataCount == 0 {
		return set
	}

	set.data = make([]iceberg.ManifestFile, 0, dataCount)
	for _, manifest := range manifests {
		if manifest.ManifestContent() == iceberg.ManifestContentData {
			set.data = append(set.data, manifest)
		}
	}

	return set
}

func (s snapshotManifestSet) allManifests() []iceberg.ManifestFile {
	return slices.Clone(s.all)
}

func (s snapshotManifestSet) dataManifests() []iceberg.ManifestFile {
	return slices.Clone(s.data)
}

func snapshotManifestSetSize(set snapshotManifestSet) int {
	return len(set.all)
}

type snapshotManifestCacheKey struct {
	snapshotID              int64
	manifestList            string
	hasEmbeddedSources      bool
	embeddedManifestSources string
}

func snapshotManifestCacheKeyFor(snapshot Snapshot) snapshotManifestCacheKey {
	var (
		hasEmbeddedSources      bool
		embeddedManifestSources string
	)
	if snapshot.ManifestList == "" {
		hasEmbeddedSources = snapshot.ManifestLocations != nil
		// Valid file locations cannot contain a literal NUL, so this join is
		// collision-free while preserving nil versus empty locations.
		embeddedManifestSources = strings.Join(snapshot.ManifestLocations, "\x00")
	}

	return snapshotManifestCacheKey{
		snapshotID:              snapshot.SnapshotID,
		manifestList:            snapshot.ManifestList,
		hasEmbeddedSources:      hasEmbeddedSources,
		embeddedManifestSources: embeddedManifestSources,
	}
}

type snapshotManifestCacheEntry struct {
	ready     chan struct{}
	readyOnce sync.Once
	manifests snapshotManifestSet
	err       error
	canceled  bool
}

type snapshotManifestLoader func(context.Context) (snapshotManifestSet, error)

// snapshotManifestCache memoizes successful manifest-list reads and shares an
// in-flight read with concurrent scans. Completed reads use a bounded LRU so a
// historical scan cannot retain every snapshot for the lifetime of a table.
// Failed reads are removed so a transient object-store error does not poison
// the cache.
type snapshotManifestCache struct {
	mu                    sync.Mutex
	entries               map[snapshotManifestCacheKey]*snapshotManifestCacheEntry
	complete              *lru.Cache[snapshotManifestCacheKey, snapshotManifestSet]
	completeManifestCount int
}

func newSnapshotManifestCache() *snapshotManifestCache {
	cache := &snapshotManifestCache{
		entries: make(map[snapshotManifestCacheKey]*snapshotManifestCacheEntry),
	}
	complete, err := lru.NewWithEvict(
		snapshotManifestCacheSize,
		func(_ snapshotManifestCacheKey, value snapshotManifestSet) {
			cache.completeManifestCount -= snapshotManifestSetSize(value)
		},
	)
	if err != nil {
		panic(err)
	}
	cache.complete = complete

	return cache
}

func newSnapshotManifestCacheForMetadata(meta Metadata) *snapshotManifestCache {
	if meta != nil && !meta.Properties().GetBool(ReadManifestListCacheEnabledKey, ReadManifestListCacheEnabledDefault) {
		return nil
	}

	return newSnapshotManifestCache()
}

func (c *snapshotManifestCache) get(
	ctx context.Context,
	snapshot Snapshot,
	load snapshotManifestLoader,
) (snapshotManifestSet, error) {
	if c == nil {
		return load(ctx)
	}
	key := snapshotManifestCacheKeyFor(snapshot)
	for {
		if err := ctx.Err(); err != nil {
			return snapshotManifestSet{}, err
		}

		c.mu.Lock()
		if manifests, ok := c.complete.Get(key); ok {
			c.mu.Unlock()

			return manifests, nil
		}
		if entry, ok := c.entries[key]; ok {
			c.mu.Unlock()

			select {
			case <-entry.ready:
			default:
				select {
				case <-entry.ready:
				case <-ctx.Done():
					return snapshotManifestSet{}, ctx.Err()
				}
			}
			// A failed read owned by a canceled caller must not fail an
			// independent scan. Retry with this caller's context and FileIO.
			if entry.canceled {
				continue
			}

			return entry.manifests, entry.err
		}

		entry := &snapshotManifestCacheEntry{ready: make(chan struct{})}
		c.entries[key] = entry
		c.mu.Unlock()

		return c.load(ctx, snapshot, key, entry, load)
	}
}

func (c *snapshotManifestCache) load(
	ctx context.Context,
	snapshot Snapshot,
	key snapshotManifestCacheKey,
	entry *snapshotManifestCacheEntry,
	load snapshotManifestLoader,
) (snapshotManifestSet, error) {
	var (
		value snapshotManifestSet
		err   error
	)
	defer func() {
		if recovered := recover(); recovered != nil {
			c.finish(key, entry, snapshotManifestSet{}, fmt.Errorf(
				"panic while reading snapshot %d manifest list: %v", snapshot.SnapshotID, recovered))
			panic(recovered)
		}

		entry.canceled = err != nil && ctx.Err() != nil
		c.finish(key, entry, value, err)
	}()

	// Both factory resolution and context-bound IO must stop with the caller.
	// Healthy waiters retry a canceled load instead of inheriting its error.
	value, err = load(ctx)

	return value, err
}

func (c *snapshotManifestCache) finish(
	key snapshotManifestCacheKey,
	entry *snapshotManifestCacheEntry,
	value snapshotManifestSet,
	err error,
) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if current, ok := c.entries[key]; ok && current == entry {
		delete(c.entries, key)
		if err == nil {
			c.complete.Add(key, value)
			c.completeManifestCount += snapshotManifestSetSize(value)
			for c.completeManifestCount > snapshotManifestCacheManifestLimit {
				if _, _, ok := c.complete.RemoveOldest(); !ok {
					break
				}
			}
		}
	}

	entry.manifests = value
	entry.err = err
	entry.readyOnce.Do(func() { close(entry.ready) })
}

func readSnapshotManifestSet(
	ctx context.Context,
	snapshot Snapshot,
	fsF FSysF,
) (snapshotManifestSet, error) {
	if err := ctx.Err(); err != nil {
		return snapshotManifestSet{}, err
	}
	if fsF == nil {
		return snapshotManifestSet{}, fmt.Errorf("%w: table file IO is not configured", ErrInvalidOperation)
	}

	// Keep factory resolution and the resulting FileIO on the same cancellable
	// context, including backends that capture it for subsequent reads.
	fio, err := fsF(ctx)
	if err != nil {
		return snapshotManifestSet{}, fmt.Errorf("get file IO: %w", err)
	}
	manifests, err := snapshot.Manifests(fio)
	if err != nil {
		return snapshotManifestSet{}, err
	}

	return newSnapshotManifestSet(manifests), nil
}

func sharedSnapshotManifestFSF(fsF FSysF) FSysF {
	var (
		once sync.Once
		fio  iceio.IO
		err  error
	)

	return func(ctx context.Context) (iceio.IO, error) {
		once.Do(func() {
			if fsF == nil {
				err = fmt.Errorf("%w: table file IO is not configured", ErrInvalidOperation)

				return
			}

			fio, err = fsF(ctx)
		})

		return fio, err
	}
}
