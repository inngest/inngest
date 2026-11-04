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

package iceberg

// Labels is catalog-provided enrichment returned with a table or view in a REST
// load response. Labels are transient: generated per request and never persisted
// to metadata, so a client may ignore them and two catalogs may return different
// labels for the same object. It maps to the "labels" property of
// LoadTableResult and LoadViewResult in the REST catalog spec.
type Labels struct {
	ObjectLabels Properties   `json:"object-labels,omitempty"`
	Fields       []FieldLabel `json:"fields,omitempty"`
}

// FieldLabel holds the labels for a single schema field. FieldID may reference a
// column that has since been dropped, so callers must tolerate unresolved IDs.
type FieldLabel struct {
	FieldID int        `json:"field-id"`
	Labels  Properties `json:"labels,omitempty"`
}

// IsEmpty reports whether there are neither object-level nor field-level labels.
// It is nil-safe.
func (l *Labels) IsEmpty() bool {
	return l == nil || (len(l.ObjectLabels) == 0 && len(l.Fields) == 0)
}

// Object returns the object-level labels, or nil if there are none.
func (l *Labels) Object() Properties {
	if l == nil {
		return nil
	}

	return l.ObjectLabels
}

// Field returns the labels for the given field ID, or nil if it carries none
// (including a dropped field). The first match wins if IDs repeat.
func (l *Labels) Field(fieldID int) Properties {
	if l == nil {
		return nil
	}

	for i := range l.Fields {
		if l.Fields[i].FieldID == fieldID {
			return l.Fields[i].Labels
		}
	}

	return nil
}
