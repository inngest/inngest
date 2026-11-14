// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.
package puffin

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"math"
	"slices"
	"strconv"

	"github.com/apache/iceberg-go"
)

// Writer writes blobs and metadata to a Puffin file.
//
// Usage:
//
//	w, err := puffin.NewWriter(file)
//	if err != nil {
//	    return err
//	}
//	_, err = w.AddBlob(puffin.BlobMetadataInput{
//	    Type:       puffin.BlobTypeDataSketchesTheta,
//	    SnapshotID: 123,
//	    Fields:     []int32{1},
//	}, sketchBytes)
//	if err != nil {
//	    return err
//	}
//	return w.Finish()
//
// Writer cannot be reused after a physical write fails. Once poisoned, all
// subsequent operations return the original write error without changing state.
type Writer struct {
	w         io.Writer
	offset    int64
	blobs     []BlobMetadata
	props     map[string]string
	done      bool
	failed    error
	createdBy string
}

// BlobMetadataInput contains fields the caller provides when adding a blob.
// Offset, Length, and CompressionCodec are set by the writer.
type BlobMetadataInput struct {
	Type           BlobType
	SnapshotID     int64
	SequenceNumber int64
	Fields         []int32
	Properties     map[string]string
}

// NewWriter creates a new Writer and writes the file header magic.
// The caller is responsible for closing the underlying writer after Finish returns.
func NewWriter(w io.Writer) (*Writer, error) {
	if w == nil {
		return nil, errors.New("puffin: writer is nil")
	}

	// Write header magic bytes
	if err := writeAll(w, magic[:]); err != nil {
		return nil, fmt.Errorf("puffin: write header magic: %w", err)
	}

	return &Writer{
		w:         w,
		offset:    MagicSize,
		props:     make(map[string]string),
		createdBy: "iceberg-go " + iceberg.Version(),
	}, nil
}

// SetProperties merges the provided properties into the file-level properties
// written to the footer. Can be called multiple times before Finish.
func (w *Writer) AddProperties(props map[string]string) error {
	if w.done {
		return errors.New("puffin: cannot set properties: writer already finalized")
	}
	if w.failed != nil {
		return fmt.Errorf("puffin: cannot set properties: writer previously failed: %w", w.failed)
	}
	maps.Copy(w.props, props)

	return nil
}

// ClearProperties removes all file-level properties.
func (w *Writer) ClearProperties() error {
	if w.done {
		return errors.New("puffin: cannot clear properties: writer already finalized")
	}
	if w.failed != nil {
		return fmt.Errorf("puffin: cannot clear properties: writer previously failed: %w", w.failed)
	}
	w.props = make(map[string]string)

	return nil
}

// SetCreatedBy overrides the default "created-by" property written to the footer.
// The default value is "iceberg-go". Example: "MyApp version 1.2.3".
func (w *Writer) SetCreatedBy(createdBy string) error {
	if w.done {
		return errors.New("puffin: cannot set created-by: writer already finalized")
	}
	if w.failed != nil {
		return fmt.Errorf("puffin: cannot set created-by: writer previously failed: %w", w.failed)
	}
	if createdBy == "" {
		return errors.New("puffin: cannot set created-by: value cannot be empty")
	}
	w.createdBy = createdBy

	return nil
}

// AddBlob writes blob data and records its metadata for the footer.
// Returns the complete BlobMetadata including the computed Offset and Length.
// The input.Type is required; use constants like ApacheDataSketchesThetaV1.
func (w *Writer) AddBlob(input BlobMetadataInput, data []byte) (BlobMetadata, error) {
	if w.done {
		return BlobMetadata{}, errors.New("puffin: cannot add blob: writer already finalized")
	}
	if w.failed != nil {
		return BlobMetadata{}, fmt.Errorf("puffin: cannot add blob: writer previously failed: %w", w.failed)
	}
	if input.Type == "" {
		return BlobMetadata{}, errors.New("puffin: cannot add blob: type is required")
	}
	if input.Fields == nil {
		return BlobMetadata{}, errors.New("puffin: cannot add blob: fields is required")
	}
	fields := slices.Clone(input.Fields)
	properties := maps.Clone(input.Properties)

	// Deletion vectors have special requirements per spec. Format matches
	// the reader-side validateBlobs check so both sides surface the same
	// "<type> requires X to be -1, got N" phrasing for log grep.
	if input.Type == BlobTypeDeletionVector {
		if input.SnapshotID != -1 {
			return BlobMetadata{}, fmt.Errorf("puffin: %s requires snapshot-id to be -1, got %d",
				BlobTypeDeletionVector, input.SnapshotID)
		}
		if input.SequenceNumber != -1 {
			return BlobMetadata{}, fmt.Errorf("puffin: %s requires sequence-number to be -1, got %d",
				BlobTypeDeletionVector, input.SequenceNumber)
		}
		// Properties cardinality and referenced-data-file are spec-mandated
		// for deletion-vector-v1; the reader in table/dv does not enforce
		// them today, so we hold the line here to keep Go-emitted files
		// always conformant.
		if properties["cardinality"] == "" {
			return BlobMetadata{}, errors.New("puffin: deletion-vector-v1 requires a cardinality property")
		}
		// Parse the same signed range used by deletion-vector readers and
		// manifest record counts so every emitted value is readable.
		cardinality, err := strconv.ParseInt(properties["cardinality"], 10, 64)
		if err != nil {
			return BlobMetadata{}, fmt.Errorf("puffin: deletion-vector-v1 cardinality property %q is not a valid non-negative integer: %w",
				properties["cardinality"], err)
		}
		if cardinality < 0 {
			return BlobMetadata{}, fmt.Errorf("puffin: deletion-vector-v1 cardinality property %q is not a valid non-negative integer",
				properties["cardinality"])
		}
		if properties["referenced-data-file"] == "" {
			return BlobMetadata{}, errors.New("puffin: deletion-vector-v1 requires a referenced-data-file property")
		}
	}

	meta := BlobMetadata{
		Type:           input.Type,
		SnapshotID:     input.SnapshotID,
		SequenceNumber: input.SequenceNumber,
		Fields:         fields,
		Offset:         w.offset,
		Length:         int64(len(data)),
		Properties:     properties,
	}

	if err := w.write(data); err != nil {
		return BlobMetadata{}, fmt.Errorf("puffin: write blob: %w", err)
	}

	w.offset += meta.Length
	w.blobs = append(w.blobs, meta)

	meta.Fields = slices.Clone(meta.Fields)
	meta.Properties = maps.Clone(meta.Properties)

	return meta, nil
}

// Finish writes the footer and completes the Puffin file structure.
// Must be called exactly once after all blobs are written.
// After Finish returns, no further operations are allowed on the writer.
func (w *Writer) Finish() error {
	if w.done {
		return errors.New("puffin: cannot finish: writer already finalized")
	}
	if w.failed != nil {
		return fmt.Errorf("puffin: cannot finish: writer previously failed: %w", w.failed)
	}

	// Build footer
	blobs := w.blobs
	if blobs == nil {
		blobs = []BlobMetadata{}
	}

	// Only include properties in the footer if there are any to write.
	// Add created-by first, then decide whether to include properties.
	if w.createdBy != "" {
		w.props[CreatedBy] = w.createdBy
	}
	var props map[string]string
	if len(w.props) > 0 {
		props = w.props
	}

	footer := Footer{
		Blobs:      blobs,
		Properties: props,
	}

	payload, err := json.Marshal(footer)
	if err != nil {
		return fmt.Errorf("puffin: marshal footer: %w", err)
	}

	// Check footer size fits in int32
	if len(payload) > math.MaxInt32 {
		return fmt.Errorf("puffin: footer too large: %d bytes exceeds 2GB limit", len(payload))
	}

	// Write footer start magic
	if err := w.write(magic[:]); err != nil {
		return fmt.Errorf("puffin: write footer magic: %w", err)
	}

	// Write footer payload
	if err := w.write(payload); err != nil {
		return fmt.Errorf("puffin: write footer payload: %w", err)
	}

	// Write trailer: PayloadSize(4) + Flags(4) + Magic(4)
	var trailer [footerTrailerSize]byte
	binary.LittleEndian.PutUint32(trailer[0:4], uint32(len(payload)))
	binary.LittleEndian.PutUint32(trailer[4:8], 0) // flags = 0 (uncompressed)
	copy(trailer[8:12], magic[:])

	if err := w.write(trailer[:]); err != nil {
		return fmt.Errorf("puffin: write footer trailer: %w", err)
	}

	w.done = true

	return nil
}

func (w *Writer) write(data []byte) error {
	if err := writeAll(w.w, data); err != nil {
		w.failed = err

		return err
	}

	return nil
}

// writeAll writes all bytes to w or returns an error.
// Handles the io.Writer contract where Write can return n < len(data) without error.
func writeAll(w io.Writer, data []byte) error {
	n, err := w.Write(data)
	if err != nil {
		return err
	}
	if n != len(data) {
		return fmt.Errorf("short write: wrote %d of %d bytes", n, len(data))
	}

	return nil
}
