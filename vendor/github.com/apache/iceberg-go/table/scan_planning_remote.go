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
	"slices"

	"github.com/apache/iceberg-go"
	iceberginternal "github.com/apache/iceberg-go/internal"
)

// supportsAutomaticRemotePlanning is deliberately more conservative than
// explicit remote mode when a planner exposes a split capability surface. A
// REST server that only advertises the initial /plan endpoint may complete an
// inline plan, but auto mode cannot know that before submitting and has no safe
// local fallback once the server responds with a continuation handle.
func supportsAutomaticRemotePlanning(planner ScanPlanner) bool {
	if planner == nil {
		return false
	}
	if full, ok := planner.(FullRemoteScanPlanner); ok {
		return full.SupportsFullRemoteScanPlanning()
	}

	return planner.SupportsRemoteScanPlanning()
}

// remotePlanningSelectedFields returns the fully qualified physical field names
// sent for a wildcard REST scan projection. It mirrors Java's
// TypeUtil.getProjectedIds + Schema.findColumnName behavior. Java includes
// struct field IDs as well as primitive/variant field IDs. List element IDs are
// included only for primitive elements, and map key/value IDs are included only
// when the value is primitive. Explicit projections keep their user-provided
// names unchanged.
func remotePlanningSelectedFields(scan *Scan, schema *iceberg.Schema) ([]string, error) {
	if schema == nil || !slices.Contains(scan.selectedFields, "*") {
		return scan.remoteSelectedFields(schema), nil
	}

	ids := make([]int, 0, schema.NumFields())
	for _, field := range schema.FieldsRef(iceberginternal.SchemaRef{}) {
		appendRemoteProjectedFieldIDs(&ids, field)
	}
	slices.Sort(ids)

	selected := make([]string, 0, len(ids))
	for _, id := range ids {
		if name, ok := schema.FindColumnName(id); ok {
			selected = append(selected, name)
		}
	}

	return selected, nil
}

func appendRemoteProjectedFieldIDs(ids *[]int, field iceberg.NestedField) {
	appendRemoteProjectedTypeIDs(ids, field.Type, field.ID, true)
}

func appendRemoteProjectedTypeIDs(ids *[]int, typ iceberg.Type, fieldID int, includeFieldID bool) {
	switch typ := typ.(type) {
	case *iceberg.StructType:
		if includeFieldID {
			*ids = append(*ids, fieldID)
		}
		for _, field := range typ.Fields() {
			appendRemoteProjectedFieldIDs(ids, field)
		}
	case *iceberg.ListType:
		if remoteProjectedLeaf(typ.Element) {
			*ids = append(*ids, typ.ElementID)
		} else {
			appendRemoteProjectedTypeIDs(ids, typ.Element, typ.ElementID, false)
		}
	case *iceberg.MapType:
		if remoteProjectedLeaf(typ.ValueType) {
			*ids = append(*ids, typ.KeyID, typ.ValueID)
		} else {
			appendRemoteProjectedTypeIDs(ids, typ.ValueType, typ.ValueID, false)
			appendRemoteProjectedTypeIDs(ids, typ.KeyType, typ.KeyID, false)
		}
	default:
		if includeFieldID {
			*ids = append(*ids, fieldID)
		}
	}
}

func remoteProjectedLeaf(typ iceberg.Type) bool {
	switch typ.(type) {
	case *iceberg.StructType, *iceberg.ListType, *iceberg.MapType:
		return false
	default:
		// Variant types are leaves for TypeUtil.getProjectedIds as well.
		return true
	}
}
