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
//
// Package puffin provides reading and writing of Puffin files.
//
// Puffin is a file format designed to store statistics and indexes
// for Iceberg tables. A Puffin file contains blobs (opaque byte sequences)
// with associated metadata, such as Apache DataSketches or deletion vectors.
//
// File structure:
//
//	[Magic] [Blob]* [Magic] [Footer Payload] [Footer Payload Size] [Flags] [Magic]
//
// See the specification at https://iceberg.apache.org/puffin-spec/
package puffin

var magic = [4]byte{'P', 'F', 'A', '1'}

const (
	//[Magic] [FooterPayload] [FooterPayloadSize] [Flags] [Magic]
	// MagicSize is the number of bytes in the magic marker.
	MagicSize = 4

	// footerTrailerSize accounts for footer length (4)+ flags (4) + trailing magic (4).
	footerTrailerSize = 12

	// FooterFlagCompressed indicates that the footer payload uses the Puffin
	// format's LZ4 frame compression.
	FooterFlagCompressed = 1 // bit 0

	// Prevents OOM
	// DefaultMaxBlobSize is the maximum blob size allowed when reading (256 MB).
	// Override with WithMaxBlobSize when creating a reader.
	DefaultMaxBlobSize = 256 << 20

	// DefaultMaxFooterSize is the maximum footer payload size allowed when
	// reading (64 MB). For compressed footer payloads, the limit applies after
	// decompression. Override with WithMaxFooterSize when creating a reader.
	DefaultMaxFooterSize = 64 << 20

	// CreatedBy is a human-readable identification of the application writing the file, along with its version.
	// Example: "Trino version 381".
	CreatedBy = "created-by"
)

type BlobMetadata struct {
	Type             BlobType          `json:"type"`
	Fields           []int32           `json:"fields"`
	SnapshotID       int64             `json:"snapshot-id"`
	SequenceNumber   int64             `json:"sequence-number"`
	Offset           int64             `json:"offset"`
	Length           int64             `json:"length"`
	CompressionCodec *string           `json:"compression-codec,omitempty"`
	Properties       map[string]string `json:"properties,omitempty"`
}

// Footer describes the blobs and file-level properties stored in a Puffin file.
type Footer struct {
	Blobs      []BlobMetadata    `json:"blobs"`
	Properties map[string]string `json:"properties,omitempty"`
}

type BlobType string

const (
	// BlobTypeDataSketchesTheta is a serialized compact Theta sketch
	// produced by the Apache DataSketches library.
	BlobTypeDataSketchesTheta BlobType = "apache-datasketches-theta-v1"

	// BlobTypeDeletionVector is a serialized deletion vector per the
	// Iceberg spec. Requires snapshot-id and sequence-number to be -1.
	BlobTypeDeletionVector BlobType = "deletion-vector-v1"
)
