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

package udf

import (
	"slices"

	"github.com/apache/iceberg-go"
	"github.com/apache/iceberg-go/table"
)

// UDF is a function (SQL UDF) loaded from a catalog: its catalog identifier,
// its metadata, and the location the metadata was loaded from.
type UDF struct {
	identifier       table.Identifier
	metadata         Metadata
	metadataLocation string
}

// New creates a UDF from its catalog identifier, metadata and metadata
// location.
func New(ident table.Identifier, meta Metadata, metadataLocation string) *UDF {
	return &UDF{
		identifier:       slices.Clone(ident),
		metadata:         meta,
		metadataLocation: metadataLocation,
	}
}

// Equals reports whether two UDFs have the same identifier, metadata
// location and metadata. The value receiver and argument are intentional,
// matching view.View: a UDF is a cheap value of references, so callers
// holding the *UDF from New compare with fn.Equals(*other).
func (u UDF) Equals(other UDF) bool {
	return slices.Equal(u.identifier, other.identifier) &&
		u.metadataLocation == other.metadataLocation &&
		u.metadata.Equals(other.metadata)
}

func (u UDF) Identifier() table.Identifier   { return slices.Clone(u.identifier) }
func (u UDF) Metadata() Metadata             { return u.metadata }
func (u UDF) MetadataLocation() string       { return u.metadataLocation }
func (u UDF) Definitions() []*Definition     { return u.metadata.Definitions() }
func (u UDF) Properties() iceberg.Properties { return u.metadata.Properties() }
func (u UDF) Location() string               { return u.metadata.Location() }
