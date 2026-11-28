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
	"errors"
	"fmt"
	"reflect"

	"github.com/apache/iceberg-go"
)

const MainBranch = "main"

// RefType will be either a BranchRef or a TagRef
type RefType string

const (
	BranchRef RefType = "branch"
	TagRef    RefType = "tag"
)

var ErrInvalidRefType = errors.New("invalid snapshot ref type, should be 'branch' or 'tag'")

// SnapshotRef represents the reference information for a specific snapshot
type SnapshotRef struct {
	SnapshotID         int64   `json:"snapshot-id"`
	SnapshotRefType    RefType `json:"type"`
	MinSnapshotsToKeep *int    `json:"min-snapshots-to-keep,omitempty"`
	MaxSnapshotAgeMs   *int64  `json:"max-snapshot-age-ms,omitempty"`
	MaxRefAgeMs        *int64  `json:"max-ref-age-ms,omitempty"`
}

func (s *SnapshotRef) Equals(rhs SnapshotRef) bool {
	return reflect.DeepEqual(s, &rhs)
}

func (s *SnapshotRef) UnmarshalJSON(b []byte) error {
	aux := struct {
		SnapshotID         *int64   `json:"snapshot-id"`
		SnapshotRefType    *RefType `json:"type"`
		MinSnapshotsToKeep *int     `json:"min-snapshots-to-keep,omitempty"`
		MaxSnapshotAgeMs   *int64   `json:"max-snapshot-age-ms,omitempty"`
		MaxRefAgeMs        *int64   `json:"max-ref-age-ms,omitempty"`
	}{}
	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}

	if aux.SnapshotID == nil {
		return fmt.Errorf("%w: snapshot ref is missing required snapshot-id", iceberg.ErrInvalidArgument)
	}
	if aux.SnapshotRefType == nil {
		return fmt.Errorf("%w: snapshot ref is missing required type", iceberg.ErrInvalidArgument)
	}

	ref := SnapshotRef{
		SnapshotID:         *aux.SnapshotID,
		SnapshotRefType:    *aux.SnapshotRefType,
		MinSnapshotsToKeep: aux.MinSnapshotsToKeep,
		MaxSnapshotAgeMs:   aux.MaxSnapshotAgeMs,
		MaxRefAgeMs:        aux.MaxRefAgeMs,
	}
	if err := ref.validate(); err != nil {
		return err
	}

	*s = ref

	return nil
}

func (s SnapshotRef) validate() error {
	switch s.SnapshotRefType {
	case BranchRef, TagRef:
	default:
		return ErrInvalidRefType
	}

	if s.MinSnapshotsToKeep != nil && *s.MinSnapshotsToKeep <= 0 {
		return fmt.Errorf("%w: min snapshots to keep must be greater than 0", iceberg.ErrInvalidArgument)
	}
	if s.MaxSnapshotAgeMs != nil && *s.MaxSnapshotAgeMs <= 0 {
		return fmt.Errorf("%w: max snapshot age must be greater than 0 ms", iceberg.ErrInvalidArgument)
	}
	if s.MaxRefAgeMs != nil && *s.MaxRefAgeMs <= 0 {
		return fmt.Errorf("%w: max reference age must be greater than 0 ms", iceberg.ErrInvalidArgument)
	}
	if s.SnapshotRefType == TagRef {
		if s.MinSnapshotsToKeep != nil {
			return fmt.Errorf("%w: tags do not support setting min snapshots to keep", iceberg.ErrInvalidArgument)
		}
		if s.MaxSnapshotAgeMs != nil {
			return fmt.Errorf("%w: tags do not support setting max snapshot age", iceberg.ErrInvalidArgument)
		}
	}

	return nil
}
