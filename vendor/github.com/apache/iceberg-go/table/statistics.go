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
	"encoding/json"
	"fmt"
)

// BlobType is the type of blob in a Puffin file.
type BlobType string

const (
	BlobTypeApacheDatasketchesThetaV1 BlobType = "apache-datasketches-theta-v1"
	BlobTypeDeletionVectorV1          BlobType = "deletion-vector-v1"
)

// IsKnown reports whether bt is one of the standard blob types understood by
// this package. Unknown non-empty wire values may still be preserved for
// forward compatibility.
func (bt BlobType) IsKnown() bool {
	switch bt {
	case BlobTypeApacheDatasketchesThetaV1, BlobTypeDeletionVectorV1:
		return true
	default:
		return false
	}
}

// IsValid reports whether bt is non-nil and names a standard blob type.
// Unknown non-empty values can be decoded and preserved, but are not valid for
// operations that require a known blob type.
func (bt *BlobType) IsValid() bool {
	return bt != nil && bt.IsKnown()
}

func (bt *BlobType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	*bt = BlobType(s)
	if s == "" {
		return fmt.Errorf("invalid blob type: %s", s)
	}

	return nil
}

// StatisticsFile represents a statistics file in the Puffin format, that can be used to read table data more
// efficiently.
//
// Statistics are informational. A reader can choose to ignore statistics information.
// Statistics support is not required to read the table correctly.
type StatisticsFile struct {
	SnapshotID            int64          `json:"snapshot-id"`
	StatisticsPath        string         `json:"statistics-path"`
	FileSizeInBytes       int64          `json:"file-size-in-bytes"`
	FileFooterSizeInBytes int64          `json:"file-footer-size-in-bytes"`
	KeyMetadata           *string        `json:"key-metadata,omitempty"`
	BlobMetadata          []BlobMetadata `json:"blob-metadata"`
}

// BlobMetadata is the metadata of a statistics or indices blob.
type BlobMetadata struct {
	Type           BlobType          `json:"type"`
	SnapshotID     int64             `json:"snapshot-id"`
	SequenceNumber int64             `json:"sequence-number"`
	Fields         []int32           `json:"fields"`
	Properties     map[string]string `json:"properties"`
}

// PartitionStatisticsFile represents a partition statistics file that can be used to read table data more efficiently.
//
// Statistics are informational. A reader can choose to ignore statistics information. Statistics
// support is not required to read the table correctly.
type PartitionStatisticsFile struct {
	SnapshotID      int64  `json:"snapshot-id"`
	StatisticsPath  string `json:"statistics-path"`
	FileSizeInBytes int64  `json:"file-size-in-bytes"`
}
