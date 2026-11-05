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
	"cmp"
	"context"
	"errors"
	"fmt"
	"iter"
	"slices"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/iceberg-go"
)

// clusteredPartitionedWrite writes records to partitioned data files,
// keeping at most one partition writer open at a time. When the
// partition changes, the current writer is closed before opening a
// new one. This is the memory-efficient write path for pre-clustered
// input (e.g. compaction reads where each source file belongs to a
// single partition).
//
// The input must be clustered by partition across batches: once a
// partition's writer has been closed, encountering further records
// for that partition returns an error. Within a single batch the
// writer reclusters rows by partition. Use the fanout writer if the
// input is not clustered across batches.
//
// Breaking out of the returned iterator early cancels the producer
// so it stops opening new writers; in-flight writers finish cleanly.
func clusteredPartitionedWrite(
	ctx context.Context,
	spec iceberg.PartitionSpec,
	schema *iceberg.Schema,
	factory *writerFactory,
	records iter.Seq2[arrow.RecordBatch, error],
) iter.Seq2[iceberg.DataFile, error] {
	ctx, cancel := context.WithCancel(ctx)

	outputCh := make(chan iceberg.DataFile, 16)
	errCh := make(chan error, 1)

	go func() {
		defer close(outputCh)
		defer close(errCh)
		defer factory.stopCount()

		var (
			currentRec          partitionRecord
			currentWriter       *RollingDataWriter
			completedPartitions = make(closedPartitionSet)
			extractionPlan      *partitionExtractionPlan
		)

		closeCurrentWriter := func() error {
			if currentWriter == nil {
				return nil
			}
			w := currentWriter
			currentWriter = nil
			completedPartitions.add(currentRec)
			close(w.recordCh)
			w.wg.Wait()

			// stream's deferred close(errorCh) runs before its
			// deferred wg.Done, so by the time Wait returns the
			// channel is closed; this read never blocks and yields
			// either the buffered error or nil.
			return <-w.errorCh
		}

		sendErr := func(err error) {
			select {
			case errCh <- err:
			default:
			}
		}

		fail := func(err error) {
			sendErr(errors.Join(err, closeCurrentWriter()))
		}

		// Recover any panic so the consumer is not left blocking on
		// errCh forever. Declared last so it runs first on goroutine
		// exit, before the close(errCh) and close(outputCh) defers.
		defer func() {
			if r := recover(); r != nil {
				fail(fmt.Errorf("clustered write panic: %v", r))
			}
		}()

		takeFn := partitionBatchByKey(ctx)

		processPart := func(rec arrow.RecordBatch, part *partitionInfo) error {
			subBatch, err := takeFn(rec, part.rows)
			if err != nil {
				return err
			}
			defer subBatch.Release()

			if currentWriter == nil || !partitionRecordsEqual(currentRec, part.partitionRec) {
				if err := closeCurrentWriter(); err != nil {
					return err
				}
				if completedPartitions.contains(part.partitionRec) {
					partitionPath := extractionPlan.pathPlan.format(part.partitionRec)

					return fmt.Errorf("clustered write: incoming records violate the clustering assumption; "+
						"partition %q has records arriving after its writer was already closed", partitionPath)
				}
				partitionPath := extractionPlan.pathPlan.format(part.partitionRec)
				currentWriter = factory.newRollingDataWriter(
					ctx, partitionPath, part.partitionValues, outputCh)
				currentRec = part.partitionRec
			}

			return currentWriter.Add(subBatch)
		}

		for rec, err := range records {
			if err != nil {
				fail(err)

				return
			}
			if ctxErr := ctx.Err(); ctxErr != nil {
				fail(context.Cause(ctx))

				return
			}

			if extractionPlan == nil {
				extractionPlan, err = newPartitionExtractionPlan(spec, schema, rec.Schema())
				if err != nil {
					fail(err)

					return
				}
			}

			partitions, err := extractionPlan.getRecordPartitions(rec)
			if err != nil {
				fail(err)

				return
			}

			// Process partitions in input row order so the revisit
			// check is deterministic; getRecordPartitions returns
			// them in arbitrary (Go map iteration) order.
			slices.SortFunc(partitions, func(a, b *partitionInfo) int {
				return cmp.Compare(a.rows[0], b.rows[0])
			})

			for _, part := range partitions {
				select {
				case <-ctx.Done():
					fail(context.Cause(ctx))

					return
				default:
				}

				if err := processPart(rec, part); err != nil {
					fail(err)

					return
				}
			}
		}

		if err := closeCurrentWriter(); err != nil {
			sendErr(err)
		}
	}()

	return func(yield func(iceberg.DataFile, error) bool) {
		// LIFO defer order matters: cancel signals the producer first
		// (synchronous, instant), then the drain pulls outputCh so
		// any in-flight stream send can complete and the producer's
		// closeCurrentWriter / wg.Wait paths unblock.
		defer func() {
			for range outputCh {
			}
		}()
		defer cancel()

		for df := range outputCh {
			if !yield(df, nil) {
				return
			}
		}

		if err := <-errCh; err != nil {
			yield(nil, err)
		}
	}
}

// closedPartitionSet tracks already-closed partitions by walking a
// tree keyed on each partition field's value, mirroring the layout of
// partitionMapNode. Go's any-equality at each level distinguishes SQL
// NULL from the literal string "null" — the same property that
// PartitionToPath drops via Transform.ToHumanStr.
type closedPartitionSet map[any]closedPartitionSet

func (s closedPartitionSet) add(rec partitionRecord) {
	node := s
	for _, part := range rec {
		key := comparablePartitionKey(part)
		next, ok := node[key]
		if !ok {
			next = make(closedPartitionSet)
			node[key] = next
		}
		node = next
	}
}

func (s closedPartitionSet) contains(rec partitionRecord) bool {
	node := s
	for _, part := range rec {
		next, ok := node[comparablePartitionKey(part)]
		if !ok {
			return false
		}
		node = next
	}

	return true
}
