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
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/apache/iceberg-go"
	"github.com/google/uuid"
)

// BuildPartitionMatchPredicate constructs a BooleanExpression matching all rows
// that belong to any of the given partitions. It is the core of dynamic
// partition overwrite (see https://github.com/apache/iceberg-go/issues/1216):
// the returned expression selects the existing data to delete before the new
// data is appended.
//
// partitions holds partition tuples as returned by DataFile.Partition(), each
// keyed by partition-field ID and carrying the field's (transformed) value. The
// result is an OR across distinct partitions, each clause an AND across the
// spec's fields:
//
//	source == value   when the partition value is present
//	IsNaN(source)      when the value is a floating-point NaN (x == NaN is never true)
//	IsNull(source)     when the partition value is absent or nil
//
// Duplicate tuples collapse to a single clause, and an empty input yields
// AlwaysFalse (matching nothing). Callers are expected to pass a partitioned
// spec; dynamic partition overwrite rejects unpartitioned tables upstream.
//
// Because only identity transforms are accepted (see below), the partition
// value equals the source-column value, so "source == value" selects exactly
// the rows in that partition. Non-identity transforms (bucket, truncate, the
// time transforms) cannot be matched by a source-column predicate and need
// partition-level matching instead; they are rejected here and tracked as a
// follow-up under issue #1215.
func BuildPartitionMatchPredicate(spec iceberg.PartitionSpec, schema *iceberg.Schema, partitions []map[int]any) (iceberg.BooleanExpression, error) {
	type fieldRef struct {
		id   int
		name string
	}

	var fields []fieldRef
	for _, f := range spec.Fields() {
		if _, ok := f.Transform.(iceberg.IdentityTransform); !ok {
			return nil, fmt.Errorf("%w: dynamic partition overwrite supports identity-transform partition fields only, got %s on %q (tracked in https://github.com/apache/iceberg-go/issues/1215)",
				iceberg.ErrNotImplemented, f.Transform, f.Name)
		}

		// Identity transforms always have exactly one source column.
		if len(f.SourceIDs) != 1 {
			return nil, fmt.Errorf("%w: identity partition field %q must have exactly one source id, got %d",
				iceberg.ErrInvalidArgument, f.Name, len(f.SourceIDs))
		}

		src, ok := schema.FindFieldByID(f.SourceIDs[0])
		if !ok {
			return nil, fmt.Errorf("%w: partition field %q references unknown source id %d",
				iceberg.ErrInvalidArgument, f.Name, f.SourceIDs[0])
		}

		fields = append(fields, fieldRef{id: f.FieldID, name: src.Name})
	}

	// NewAnd/NewOr fold away the AlwaysTrue/AlwaysFalse seeds, so a single-field,
	// single-partition input returns the bare leaf predicate rather than a
	// wrapped tree.
	var result iceberg.BooleanExpression = iceberg.AlwaysFalse{}
	seen := make(map[string]struct{}, len(partitions))

	for _, part := range partitions {
		var clause iceberg.BooleanExpression = iceberg.AlwaysTrue{}

		// sigParts encodes (field id, canonical value bytes) per field so that
		// distinct tuples can never collide into one dedup key, regardless of
		// the value's content (e.g. strings containing the separators).
		sigParts := make([]string, 0, len(fields))

		for _, fr := range fields {
			ref := iceberg.Reference(fr.name)

			// A missing key (!ok) is treated like an explicit nil. In practice
			// DataFile.Partition() stores a null partition value as a nil-valued
			// entry rather than an absent key; !ok is a defensive guard.
			val, ok := part[fr.id]
			if !ok || val == nil {
				clause = iceberg.NewAnd(clause, iceberg.IsNull(ref))
				sigParts = append(sigParts, strconv.Itoa(fr.id)+":null")

				continue
			}

			if isNaN(val) {
				// x == NaN is never true (IEEE 754), so a NaN partition value must
				// match via IsNaN. This intentionally diverges from PyIceberg's
				// _build_partition_predicate, which emits EqualTo for every value:
				// IsNaN is both correct as a row filter and lets the strict-metrics
				// evaluator delete NaN-only files outright instead of rewriting them.
				//
				// NaN has many valid bit patterns, so the dedup signature uses a
				// fixed sentinel (not MarshalBinary) to keep duplicate NaN
				// partitions collapsing to a single clause.
				clause = iceberg.NewAnd(clause, iceberg.IsNaN(ref))
				sigParts = append(sigParts, strconv.Itoa(fr.id)+":nan")

				continue
			}

			lit, err := LiteralForPartitionValue(val)
			if err != nil {
				return nil, fmt.Errorf("partition field %q: %w", fr.name, err)
			}

			clause = iceberg.NewAnd(clause, iceberg.LiteralPredicate(iceberg.OpEQ, ref, lit))

			enc, err := lit.MarshalBinary()
			if err != nil {
				return nil, fmt.Errorf("partition field %q: encoding value for dedup: %w", fr.name, err)
			}
			sigParts = append(sigParts, strconv.Itoa(fr.id)+":v"+hex.EncodeToString(enc))
		}

		sig := strings.Join(sigParts, "|")
		if _, dup := seen[sig]; dup {
			continue
		}
		seen[sig] = struct{}{}

		result = iceberg.NewOr(result, clause)
	}

	return result, nil
}

// isNaN reports whether v is a floating-point NaN.
func isNaN(v any) bool {
	switch f := v.(type) {
	case float32:
		return math.IsNaN(float64(f))
	case float64:
		return math.IsNaN(f)
	default:
		return false
	}
}

// LiteralForPartitionValue converts a partition value (as stored on a DataFile)
// into a typed Literal so the resulting predicate binds against the source
// field with the correct type, rather than relying on a string rendering.
//
// DataFile.Partition() yields either a Literal (e.g. DecimalLiteral for decimal
// fields, decoded in manifest.go) or a raw Go value; both are handled.
func LiteralForPartitionValue(v any) (iceberg.Literal, error) {
	// Decoded partition values are sometimes already Literals (decimal fields).
	if lit, ok := v.(iceberg.Literal); ok {
		return lit, nil
	}

	// The remaining cases cover the raw Go values manifest decoding emits
	// (int32/int64/float/string/[]byte and the Date/Time/Timestamp aliases).
	// The int and iceberg.Decimal cases are accepted defensively for
	// hand-constructed inputs; the decoder itself yields int32/int64 and, for
	// decimals, a DecimalLiteral caught by the passthrough above.
	switch val := v.(type) {
	case bool:
		return iceberg.NewLiteral(val), nil
	case int32:
		return iceberg.NewLiteral(val), nil
	case int64:
		return iceberg.NewLiteral(val), nil
	case int:
		return iceberg.NewLiteral(int64(val)), nil
	case float32:
		return iceberg.NewLiteral(val), nil
	case float64:
		return iceberg.NewLiteral(val), nil
	case string:
		return iceberg.NewLiteral(val), nil
	case []byte:
		return iceberg.NewLiteral(val), nil
	case iceberg.Date:
		return iceberg.NewLiteral(val), nil
	case iceberg.Time:
		return iceberg.NewLiteral(val), nil
	case iceberg.Timestamp:
		return iceberg.NewLiteral(val), nil
	case iceberg.TimestampNano:
		return iceberg.NewLiteral(val), nil
	case uuid.UUID:
		return iceberg.NewLiteral(val), nil
	case iceberg.Decimal:
		return iceberg.NewLiteral(val), nil
	default:
		return nil, fmt.Errorf("%w: unsupported partition value type %T", iceberg.ErrInvalidArgument, v)
	}
}
