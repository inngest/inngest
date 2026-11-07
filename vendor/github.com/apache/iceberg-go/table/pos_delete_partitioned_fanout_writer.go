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
	"iter"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/iceberg-go"
	"golang.org/x/sync/errgroup"
)

// positionDeletePartitionedFanoutWriter distributes Arrow position delete records across multiple partitions based on
// a partition specification, writing data to separate delete files for each partition using
// a fanout pattern with configurable parallelism.
type positionDeletePartitionedFanoutWriter struct {
	partitionContextByFilePath map[string]partitionContext
	metadata                   Metadata
	itr                        iter.Seq2[arrow.RecordBatch, error]
	writerFactory              *writerFactory
}

// newPositionDeletePartitionedFanoutWriter creates a new PartitionedFanoutWriter with the specified
// metadata, partition context, and record iterator.
func newPositionDeletePartitionedFanoutWriter(metadata Metadata, partitionContextByFilePath map[string]partitionContext, itr iter.Seq2[arrow.RecordBatch, error], writerFactory *writerFactory) *positionDeletePartitionedFanoutWriter {
	return &positionDeletePartitionedFanoutWriter{
		partitionContextByFilePath: partitionContextByFilePath,
		metadata:                   metadata,
		itr:                        itr,
		writerFactory:              writerFactory,
	}
}

// Write writes the Arrow records to the specified location using a fanout pattern with
// the specified number of workers. The returned iterator yields the data files written
// by the fanout process.
func (p *positionDeletePartitionedFanoutWriter) Write(ctx context.Context, workers int) iter.Seq2[iceberg.DataFile, error] {
	inputRecordsCh := make(chan arrow.RecordBatch, workers)
	outputDataFilesCh := make(chan iceberg.DataFile, workers)

	fanoutBaseCtx, fanoutCancel := context.WithCancel(ctx)
	fanoutWorkers, fanoutCtx := errgroup.WithContext(fanoutBaseCtx)
	writerCtx, writerCancel := context.WithCancel(ctx)
	cancel := func() {
		fanoutCancel()
		writerCancel()
	}
	startRecordFeeder(fanoutCtx, p.itr, fanoutWorkers, inputRecordsCh)

	for range workers {
		fanoutWorkers.Go(func() error {
			return p.fanout(fanoutCtx, writerCtx, inputRecordsCh, outputDataFilesCh)
		})
	}

	return p.yieldDataFiles(fanoutWorkers, inputRecordsCh, outputDataFilesCh, cancel)
}

func (p *positionDeletePartitionedFanoutWriter) fanout(ctx context.Context, writerCtx context.Context, inputRecordsCh <-chan arrow.RecordBatch, dataFilesChannel chan<- iceberg.DataFile) error {
	for {
		select {
		case <-ctx.Done():
			return context.Cause(ctx)

		case record, ok := <-inputRecordsCh:
			if !ok {
				return nil
			}

			err := p.processBatch(ctx, writerCtx, record, dataFilesChannel)
			if err != nil {
				return err
			}
		}
	}
}

func (p *positionDeletePartitionedFanoutWriter) processBatch(ctx context.Context, writerCtx context.Context, batch arrow.RecordBatch, dataFilesChannel chan<- iceberg.DataFile) (err error) {
	defer batch.Release()

	select {
	case <-ctx.Done():
		return context.Cause(ctx)
	default:
	}

	if batch.NumRows() == 0 {
		return
	}

	columns := batch.Columns()
	filePathArray := columns[0].(*array.String)
	filePath := filePathArray.ValueStr(0)
	partitionContext, ok := p.partitionContextByFilePath[filePath]
	if !ok {
		return fmt.Errorf("unexpected missing partition context for path %s", filePath)
	}

	partitionPath, err := p.partitionPath(partitionContext)
	if err != nil {
		return err
	}
	rollingDataWriter, err := p.writerFactory.getOrCreateRollingDataWriter(writerCtx, partitionPath, partitionContext.partitionData, dataFilesChannel)
	if err != nil {
		return err
	}

	err = rollingDataWriter.Add(batch)
	if err != nil {
		return err
	}

	return nil
}

func (p *positionDeletePartitionedFanoutWriter) partitionPath(partitionContext partitionContext) (string, error) {
	spec := p.metadata.PartitionSpecByID(int(partitionContext.specID))
	if spec == nil {
		return "", fmt.Errorf("unexpected missing partition spec in metadata for spec id %d", partitionContext.specID)
	}

	// Resolve the schema at call time, not at construction. Unlike partitionedFanoutWriter
	// (which always writes against the current spec), this writer fans out across historic
	// specIDs from the target data files; callers must ensure those specIDs resolve against
	// fields still present in CurrentSchema. Dropped partition source columns are assumed to
	// be void-transformed per the Iceberg spec.
	schema := p.metadata.CurrentSchema()
	data := newPartitionRecord(partitionContext.partitionData, spec.PartitionType(schema))

	return spec.PartitionToPath(data, schema), nil
}

func (p *positionDeletePartitionedFanoutWriter) yieldDataFiles(fanoutWorkers *errgroup.Group, inputRecordsCh <-chan arrow.RecordBatch, outputDataFilesCh chan iceberg.DataFile, cancel context.CancelFunc) iter.Seq2[iceberg.DataFile, error] {
	return yieldDataFiles(
		p.writerFactory,
		fanoutWorkers,
		inputRecordsCh,
		outputDataFilesCh,
		p.writerFactory.closeAll,
		p.writerFactory.abortAll,
		cancel,
	)
}
