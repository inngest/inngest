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

package view

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"slices"

	"github.com/apache/iceberg-go"
	"github.com/apache/iceberg-go/internal"
	"github.com/apache/iceberg-go/io"
	"github.com/apache/iceberg-go/table"
	"github.com/google/uuid"
)

type View struct {
	identifier       table.Identifier
	metadata         Metadata
	metadataLocation string
}

func (t View) Equals(other View) bool {
	return slices.Equal(t.identifier, other.identifier) &&
		t.metadataLocation == other.metadataLocation &&
		t.metadata.Equals(other.metadata)
}

func (t View) Identifier() table.Identifier     { return slices.Clone(t.identifier) }
func (t View) Metadata() Metadata               { return t.metadata }
func (t View) MetadataLocation() string         { return t.metadataLocation }
func (t View) CurrentVersion() *Version         { return t.metadata.CurrentVersion() }
func (t View) CurrentSchema() *iceberg.Schema   { return t.metadata.CurrentSchema() }
func (t View) Properties() iceberg.Properties   { return t.metadata.Properties() }
func (t View) Location() string                 { return t.metadata.Location() }
func (t View) Versions() []*Version             { return t.metadata.Versions() }
func (t View) Schemas() map[int]*iceberg.Schema { return t.metadata.SchemasByID() }

func New(ident table.Identifier, meta Metadata, metadataLocation string) *View {
	return &View{
		identifier:       slices.Clone(ident),
		metadata:         meta,
		metadataLocation: metadataLocation,
	}
}

func NewFromLocation(
	ctx context.Context,
	ident table.Identifier,
	metalocation string,
	fsysF table.FSysF,
) (*View, error) {
	var meta Metadata

	fsys, err := fsysF(ctx)
	if err != nil {
		return nil, err
	}
	if rf, ok := fsys.(io.ReadFileIO); ok {
		data, err := rf.ReadFile(metalocation)
		if err != nil {
			return nil, err
		}

		if meta, err = ParseMetadataBytes(data); err != nil {
			return nil, err
		}
	} else {
		f, err := fsys.Open(metalocation)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		if meta, err = ParseMetadata(f); err != nil {
			return nil, err
		}
	}

	return New(ident, meta, metalocation), nil
}

// CreateView creates a new view metadata file and writes it to storage.
//
// Returns a new View, or an error if creation fails.
//
// Note: This function only supports creating new views with format version 1.
// It does not support updating existing view metadata.
func CreateView(
	ctx context.Context,
	catalogName string,
	viewIdent table.Identifier,
	schema *iceberg.Schema,
	viewSQL string,
	defaultNS table.Identifier,
	loc string,
	props iceberg.Properties,
) (*View, error) {
	return createView(ctx, catalogName, viewIdent, schema, viewSQL, defaultNS, loc, props, props)
}

// CreateViewWithIOProperties creates a view using ioProps to configure the
// filesystem and metadataProps as the properties persisted in view metadata.
func CreateViewWithIOProperties(
	ctx context.Context,
	catalogName string,
	viewIdent table.Identifier,
	schema *iceberg.Schema,
	viewSQL string,
	defaultNS table.Identifier,
	loc string,
	ioProps iceberg.Properties,
	metadataProps iceberg.Properties,
) (*View, error) {
	return createView(ctx, catalogName, viewIdent, schema, viewSQL, defaultNS, loc, ioProps, metadataProps)
}

func createView(
	ctx context.Context,
	catalogName string,
	viewIdent table.Identifier,
	schema *iceberg.Schema,
	viewSQL string,
	defaultNS table.Identifier,
	loc string,
	ioProps iceberg.Properties,
	metadataProps iceberg.Properties,
) (*View, error) {
	versionId := int64(1)

	builder, err := NewMetadataBuilder()
	if err != nil {
		return nil, err
	}

	viewVersion, err := NewVersionFromSQL(
		versionId,
		schema.ID,
		viewSQL,
		defaultNS,
		WithDefaultViewCatalog(catalogName),
	)
	if err != nil {
		return nil, err
	}
	viewMD, err := builder.
		SetLoc(loc).
		SetCurrentVersion(
			viewVersion,
			schema,
		).
		SetProperties(metadataProps).
		Build()
	if err != nil {
		return nil, err
	}

	metadataLocation, err := url.JoinPath(loc, "metadata", "view-"+uuid.New().String()+".metadata.json")
	if err != nil {
		return nil, fmt.Errorf("failed to build view metadata location: %w", err)
	}

	viewMetadataBytes, err := json.Marshal(viewMD)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal view metadata: %w", err)
	}

	fs, err := io.LoadFS(ctx, ioProps, metadataLocation)
	if err != nil {
		return nil, fmt.Errorf("failed to load filesystem for view metadata: %w", err)
	}

	wfs, ok := fs.(io.WriteFileIO)
	if !ok {
		return nil, errors.New("filesystem IO does not support writing")
	}

	if err := writeViewMetadata(wfs, metadataLocation, viewMetadataBytes); err != nil {
		return nil, err
	}

	return New(viewIdent, viewMD, metadataLocation), nil
}

func writeViewMetadata(wfs io.WriteFileIO, metadataLocation string, data []byte) (err error) {
	out, err := wfs.Create(metadataLocation)
	if err != nil {
		return fmt.Errorf("failed to create view metadata file: %w", err)
	}
	defer internal.CheckedClose(out, &err)

	if _, err := out.Write(data); err != nil {
		return fmt.Errorf("failed to write view metadata: %w", err)
	}

	return nil
}
