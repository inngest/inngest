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
	"math"
	"strings"

	"github.com/apache/iceberg-go"
	"github.com/apache/iceberg-go/table/internal"
)

const (
	WriteDataPathKey                        = "write.data.path"
	WriteMetadataPathKey                    = "write.metadata.path"
	WriteMetadataLocationKey                = "write.metadata.location"
	WriteObjectStorePartitionedPathsKey     = "write.object-storage.partitioned-paths"
	WriteObjectStorePartitionedPathsDefault = true
	ObjectStoreEnabledKey                   = "write.object-storage.enabled"
	ObjectStoreEnabledDefault               = false

	// ReadSplitTargetSizeKey controls coalescing of safe row-group ranges during
	// local Parquet scan planning. Remote plans already contain catalog-owned
	// ranges and do not use this property.
	ReadSplitTargetSizeKey     = "read.split.target-size"
	ReadSplitTargetSizeDefault = 128 * 1024 * 1024 // 128 MiB

	// ReadManifestListCacheEnabledKey controls reuse of decoded snapshot
	// manifest lists across scans of a table handle. Disable it to avoid
	// retaining manifest descriptors between operations.
	ReadManifestListCacheEnabledKey     = "read.manifest-list-cache.enabled"
	ReadManifestListCacheEnabledDefault = true

	// These Java-compatible properties are reserved for future task-group
	// planning. They are exported so applications can share configuration
	// names across Iceberg implementations.
	ReadSplitPlanningLookbackKey     = "read.split.planning-lookback"
	ReadSplitPlanningLookbackDefault = 10
	ReadSplitOpenFileCostKey         = "read.split.open-file-cost"
	ReadSplitOpenFileCostDefault     = 4 * 1024 * 1024 // 4 MiB

	DefaultNameMappingKey = "schema.name-mapping.default"

	MetricsModeColumnConfPrefix    = "write.metadata.metrics.column"
	DefaultWriteMetricsModeKey     = "write.metadata.metrics.default"
	DefaultWriteMetricsModeDefault = "truncate(16)"

	ParquetRowGroupSizeBytesKey              = internal.ParquetRowGroupSizeBytesKey
	ParquetRowGroupSizeBytesDefault          = internal.ParquetRowGroupSizeBytesDefault
	ParquetRowGroupLimitKey                  = internal.ParquetRowGroupLimitKey
	ParquetRowGroupLimitDefault              = internal.ParquetRowGroupLimitDefault
	ParquetPageSizeBytesKey                  = internal.ParquetPageSizeBytesKey
	ParquetPageSizeBytesDefault              = internal.ParquetPageSizeBytesDefault
	ParquetPageRowLimitKey                   = internal.ParquetPageRowLimitKey
	ParquetPageRowLimitDefault               = internal.ParquetPageRowLimitDefault
	ParquetDictSizeBytesKey                  = internal.ParquetDictSizeBytesKey
	ParquetDictSizeBytesDefault              = internal.ParquetDictSizeBytesDefault
	ParquetPageVersionKey                    = internal.ParquetPageVersionKey
	ParquetPageVersionDefault                = internal.ParquetPageVersionDefault
	ParquetCompressionKey                    = internal.ParquetCompressionKey
	ParquetCompressionDefault                = internal.ParquetCompressionDefault
	ParquetCompressionLevelKey               = internal.ParquetCompressionLevelKey
	ParquetCompressionLevelDefault           = internal.ParquetCompressionLevelDefault
	ParquetBloomFilterMaxBytesKey            = internal.ParquetBloomFilterMaxBytesKey
	ParquetBloomFilterMaxBytesDefault        = internal.ParquetBloomFilterMaxBytesDefault
	ParquetBloomFilterColumnEnabledKeyPrefix = internal.ParquetBloomFilterColumnEnabledKeyPrefix

	ParquetBatchSizeKey     = internal.ParquetBatchSizeKey
	ParquetBatchSizeDefault = internal.ParquetBatchSizeDefault

	ManifestMergeEnabledKey     = "commit.manifest-merge.enabled"
	ManifestMergeEnabledDefault = false

	ManifestTargetSizeBytesKey     = "commit.manifest.target-size-bytes"
	ManifestTargetSizeBytesDefault = 8 * 1024 * 1024 // 8 MB

	ManifestMinMergeCountKey     = "commit.manifest.min-count-to-merge"
	ManifestMinMergeCountDefault = 100

	ManifestMergeMaxConcurrencyKey     = "commit.manifest-merge.max-concurrency"
	ManifestMergeMaxConcurrencyDefault = 0

	WritePartitionSummaryLimitKey     = "write.summary.partition-limit"
	WritePartitionSummaryLimitDefault = 0

	WriteDeleteModeKey     = "write.delete.mode"
	WriteDeleteModeDefault = WriteModeCopyOnWrite

	MetadataDeleteAfterCommitEnabledKey     = "write.metadata.delete-after-commit.enabled"
	MetadataDeleteAfterCommitEnabledDefault = false

	MetadataPreviousVersionsMaxKey     = "write.metadata.previous-versions-max"
	MetadataPreviousVersionsMaxDefault = 100

	MetadataCompressionKey     = "write.metadata.compression-codec"
	MetadataCompressionDefault = "none"

	WriteFormatDefaultKey     = "write.format.default"
	WriteFormatDefaultDefault = "parquet"

	WriteTargetFileSizeBytesKey     = "write.target-file-size-bytes"
	WriteTargetFileSizeBytesDefault = 512 * 1024 * 1024 // 512 MB

	ParquetShredVariantsKey         = internal.ParquetShredVariantsKey
	ParquetShredVariantsDefault     = internal.ParquetShredVariantsDefault
	ParquetVariantBufferSizeKey     = internal.ParquetVariantBufferSizeKey
	ParquetVariantBufferSizeDefault = internal.ParquetVariantBufferSizeDefault

	GCEnabledKey     = "gc.enabled"
	GCEnabledDefault = true

	MinSnapshotsToKeepKey     = "history.expire.min-snapshots-to-keep"
	MinSnapshotsToKeepDefault = 1

	MaxSnapshotAgeMsKey     = "history.expire.max-snapshot-age-ms"
	MaxSnapshotAgeMsDefault = int64(5 * 24 * 60 * 60 * 1000)

	MaxRefAgeMsKey     = "history.expire.max-ref-age-ms"
	MaxRefAgeMsDefault = int64(math.MaxInt64)

	legacyMinSnapshotsToKeepKey = "min-snapshots-to-keep"
	legacyMaxSnapshotAgeMsKey   = "max-snapshot-age-ms"
	legacyMaxRefAgeMsKey        = "max-ref-age-ms"

	// CommitNumRetriesKey is the number of commit retry attempts before
	// giving up on ErrCommitFailed from the catalog.
	//
	// The default is 0 (no retries). Each retry attempt reloads the
	// current catalog state and replays the update against it (see
	// doCommit's refresh-and-replay loop), so raising this can resolve
	// both transient catalog flakiness (dropped connections, brief 409
	// during leader election) and genuine OCC conflicts from concurrent
	// writers. It stays opt-in by default because a retry rebuilds the
	// snapshot's manifest list and adds latency.
	//
	// Commits that carry delete-file removals (e.g. the
	// deleteFilesToRemove argument of [Transaction.ReplaceFiles], or a
	// v3 merge-on-read delete superseding an existing deletion vector)
	// never replay regardless of this setting: removal identity is
	// snapshot-relative, so those commits fail with ErrCommitFailed on
	// the first conflict. The caller must reload the table and
	// re-resolve the removals against the current snapshot in a new
	// transaction; the failed transaction's staged removals cannot be
	// rebased.
	CommitNumRetriesKey     = "commit.retry.num-retries"
	CommitNumRetriesDefault = 0

	// CommitMinRetryWaitMsKey is the initial wait time in milliseconds
	// for exponential backoff between commit retry attempts. Default: 100ms.
	CommitMinRetryWaitMsKey     = "commit.retry.min-wait-ms"
	CommitMinRetryWaitMsDefault = 100

	// CommitMaxRetryWaitMsKey is the maximum wait time in milliseconds
	// between commit retry attempts. Default: 60s.
	CommitMaxRetryWaitMsKey     = "commit.retry.max-wait-ms"
	CommitMaxRetryWaitMsDefault = 60 * 1000

	// CommitTotalRetryTimeoutMsKey bounds the total time spent across all
	// retry attempts. Default: 30 minutes.
	CommitTotalRetryTimeoutMsKey     = "commit.retry.total-timeout-ms"
	CommitTotalRetryTimeoutMsDefault = 30 * 60 * 1000
)

func isGCEnabled(props iceberg.Properties) bool {
	value, ok := props[GCEnabledKey]
	if !ok {
		return GCEnabledDefault
	}

	return strings.EqualFold(value, "true")
}

// Reserved properties
const (
	PropertyFormatVersion            = "format-version"
	PropertyUuid                     = "uuid"
	PropertySnapshotCount            = "snapshot-count"
	PropertyCurrentSnapshotId        = "current-snapshot-id"
	PropertyCurrentSnapshotSummary   = "current-snapshot-summary"
	PropertyCurrentSnapshotTimestamp = "current-snapshot-timestamp"
	PropertyCurrentSchema            = "current-schema"
	PropertyDefaultPartitionSpec     = "default-partition-spec"
	PropertyDefaultSortOrder         = "default-sort-order"
)

var ReservedProperties = [9]string{
	PropertyFormatVersion,
	PropertyUuid,
	PropertySnapshotCount,
	PropertyCurrentSnapshotId,
	PropertyCurrentSnapshotSummary,
	PropertyCurrentSnapshotTimestamp,
	PropertyCurrentSchema,
	PropertyDefaultPartitionSpec,
	PropertyDefaultSortOrder,
}

// Metadata compression codecs
const (
	MetadataCompressionCodecNone = "none"
	MetadataCompressionCodecGzip = "gzip"
	MetadataCompressionCodecZstd = "zstd"
)

// Write modes
const (
	WriteModeCopyOnWrite = "copy-on-write"
	WriteModeMergeOnRead = "merge-on-read"
)
