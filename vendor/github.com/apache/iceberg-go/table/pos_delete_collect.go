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

	"github.com/apache/iceberg-go"
	iceio "github.com/apache/iceberg-go/io"
)

// CollectDeadPositionDeletes walks the given snapshot's manifests and returns
// the classic position-delete files made dead by a rewrite that removes
// rewrittenPaths. A returned delete references only data files the rewrite is
// removing, so it is safe to expunge in the same commit.
//
// A file-scoped delete is resolved through referenced_data_file or equal
// file_path bounds. A partition-scoped delete is retained when a surviving
// data file in the same (specID, partition) has a sequence number to which the
// delete still applies.
//
// Deletion vectors are intentionally excluded: they are one-to-one with their
// referenced data file and are removed from a rewrite using the task result
// that identified the rewritten file.
func CollectDeadPositionDeletes(
	ctx context.Context,
	fs iceio.IO,
	snap *Snapshot,
	rewrittenPaths map[string]struct{},
) ([]iceberg.DataFile, error) {
	if snap == nil || len(rewrittenPaths) == 0 {
		return nil, nil
	}

	manifests, err := snap.Manifests(fs)
	if err != nil {
		return nil, err
	}

	candidates, hasPartitionScoped, err := collectPositionDeleteCandidates(ctx, fs, manifests)
	if err != nil {
		return nil, err
	}

	var minSurvivorSeq map[string]int64
	if hasPartitionScoped {
		minSurvivorSeq, err = minSurvivorSeqByPartition(ctx, fs, manifests, rewrittenPaths)
		if err != nil {
			return nil, err
		}
	}

	return decideDeadPositionDeletes(candidates, rewrittenPaths, minSurvivorSeq), nil
}

type positionDeleteCandidate struct {
	df               iceberg.DataFile
	fileScopedTarget string
	partitionKey     string
	seq              int64
}

func collectPositionDeleteCandidates(ctx context.Context, fs iceio.IO, manifests []iceberg.ManifestFile) (candidates []positionDeleteCandidate, hasPartitionScoped bool, err error) {
	seen := make(map[string]struct{})
	for _, m := range manifests {
		if cerr := ctx.Err(); cerr != nil {
			return nil, false, cerr
		}
		if m.ManifestContent() != iceberg.ManifestContentDeletes {
			continue
		}
		for e, err := range m.Entries(fs, true) {
			if err != nil {
				return nil, false, err
			}
			df := e.DataFile()
			if df.ContentType() != iceberg.EntryContentPosDeletes || IsDeletionVector(df) {
				continue
			}
			path := df.FilePath()
			if _, ok := seen[path]; ok {
				continue
			}
			seen[path] = struct{}{}

			if target := referencedDataFilePath(df); target != "" {
				candidates = append(candidates, positionDeleteCandidate{df: df, fileScopedTarget: target})

				continue
			}

			// An unset delete sequence cannot be compared safely. Retain it.
			if seq := e.SequenceNum(); seq >= 0 {
				partitionKey, err := canonicalPartitionKey(df.SpecID(), dataFilePartition(df))
				if err != nil {
					return nil, false, fmt.Errorf("building partition key for position delete %s (spec %d): %w", df.FilePath(), df.SpecID(), err)
				}
				hasPartitionScoped = true
				candidates = append(candidates, positionDeleteCandidate{
					df:           df,
					partitionKey: partitionKey,
					seq:          seq,
				})
			}
		}
	}

	return candidates, hasPartitionScoped, nil
}

func minSurvivorSeqByPartition(ctx context.Context, fs iceio.IO, manifests []iceberg.ManifestFile, rewrittenPaths map[string]struct{}) (map[string]int64, error) {
	minSeq := make(map[string]int64)
	for _, m := range manifests {
		if cerr := ctx.Err(); cerr != nil {
			return nil, cerr
		}
		if m.ManifestContent() != iceberg.ManifestContentData {
			continue
		}
		for e, err := range m.Entries(fs, true) {
			if err != nil {
				return nil, err
			}
			df := e.DataFile()
			if df.ContentType() != iceberg.EntryContentData {
				continue
			}
			if _, rewritten := rewrittenPaths[df.FilePath()]; rewritten {
				continue
			}

			// Unknown survivor sequence numbers are treated as old as possible,
			// retaining every delete whose applicability is uncertain.
			seq := max(e.SequenceNum(), 0)
			key, err := canonicalPartitionKey(df.SpecID(), dataFilePartition(df))
			if err != nil {
				return nil, fmt.Errorf("building partition key for surviving data file %s (spec %d): %w", df.FilePath(), df.SpecID(), err)
			}
			if cur, ok := minSeq[key]; ok {
				seq = min(seq, cur)
			}
			minSeq[key] = seq
		}
	}

	return minSeq, nil
}

func decideDeadPositionDeletes(candidates []positionDeleteCandidate, rewrittenPaths map[string]struct{}, minSurvivorSeq map[string]int64) []iceberg.DataFile {
	dead := make([]iceberg.DataFile, 0, len(candidates))
	for _, c := range candidates {
		if c.fileScopedTarget != "" {
			if _, rewritten := rewrittenPaths[c.fileScopedTarget]; rewritten {
				dead = append(dead, c.df)
			}

			continue
		}
		if cur, ok := minSurvivorSeq[c.partitionKey]; !ok || cur > c.seq {
			dead = append(dead, c.df)
		}
	}

	return dead
}
