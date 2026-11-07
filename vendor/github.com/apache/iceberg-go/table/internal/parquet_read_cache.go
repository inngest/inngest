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

package internal

import (
	"context"
	"sync"

	"github.com/apache/arrow-go/v18/parquet/metadata"
)

type parquetMetadataCacheContextKey struct{}

type parquetMetadataCacheKey struct {
	path string
	size int64
}

type parquetMetadataCacheEntry struct {
	once          sync.Once
	fileMetadata  *metadata.FileMetaData
	rowGroupInfos []parquetRowGroupInfo
	err           error
}

type parquetMetadataCache struct {
	mu      sync.Mutex
	entries map[parquetMetadataCacheKey]*parquetMetadataCacheEntry
}

// WithParquetMetadataCache adds a per-operation cache for immutable Parquet
// file metadata. Readers and input handles are still created per task, so
// parallel split tasks do not share mutable reader state.
func WithParquetMetadataCache(ctx context.Context) context.Context {
	if _, ok := ctx.Value(parquetMetadataCacheContextKey{}).(*parquetMetadataCache); ok {
		return ctx
	}

	return context.WithValue(ctx, parquetMetadataCacheContextKey{}, &parquetMetadataCache{})
}

func parquetMetadataCacheFromContext(ctx context.Context) *parquetMetadataCache {
	cache, _ := ctx.Value(parquetMetadataCacheContextKey{}).(*parquetMetadataCache)

	return cache
}

func (c *parquetMetadataCache) entry(key parquetMetadataCacheKey) *parquetMetadataCacheEntry {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.entries == nil {
		c.entries = make(map[parquetMetadataCacheKey]*parquetMetadataCacheEntry)
	}
	if entry, ok := c.entries[key]; ok {
		return entry
	}

	entry := &parquetMetadataCacheEntry{}
	c.entries[key] = entry

	return entry
}

type parquetRowGroupInfo struct {
	splitOffset int64
	numRows     int64
}
