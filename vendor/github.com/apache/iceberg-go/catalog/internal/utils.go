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
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/apache/iceberg-go"
	"github.com/apache/iceberg-go/catalog"
	"github.com/apache/iceberg-go/internal"
	icebergio "github.com/apache/iceberg-go/io"
	"github.com/apache/iceberg-go/table"
	"github.com/google/uuid"
	"github.com/klauspost/compress/zstd"
)

func WriteTableMetadata(metadata table.Metadata, fs icebergio.WriteFileIO, loc string, compression string) (err error) {
	switch compression {
	case table.MetadataCompressionCodecNone, table.MetadataCompressionCodecGzip, table.MetadataCompressionCodecZstd:
		// supported codecs
	default:
		return fmt.Errorf("unsupported write metadata compression codec: %s", compression)
	}

	out, err := fs.Create(loc)
	if err != nil {
		return err
	}
	defer internal.CheckedClose(out, &err)

	var writer io.Writer = out
	switch compression {
	case table.MetadataCompressionCodecGzip:
		gzw := gzip.NewWriter(out)
		writer = gzw
		defer internal.CheckedClose(gzw, &err)
	case table.MetadataCompressionCodecZstd:
		enc, zErr := zstd.NewWriter(out)
		if zErr != nil {
			return zErr
		}
		writer = enc
		defer internal.CheckedClose(enc, &err)
	}

	err = json.NewEncoder(writer).Encode(metadata)

	return
}

func WriteMetadata(ctx context.Context, tbl *table.Table) error {
	fs, err := tbl.FS(ctx)
	if err != nil {
		return err
	}

	wfs, ok := fs.(icebergio.WriteFileIO)
	if !ok {
		return errors.New("filesystem IO does not support writing")
	}

	compression := tbl.Properties().Get(table.MetadataCompressionKey, table.MetadataCompressionDefault)

	return WriteTableMetadata(tbl.Metadata(), wfs, tbl.MetadataLocation(), compression)
}

func UpdateTableMetadata(base table.Metadata, updates []table.Update, metadataLoc string) (table.Metadata, error) {
	return table.UpdateTableMetadata(base, updates, metadataLoc)
}

func CreateStagedTable(ctx context.Context, catprops iceberg.Properties, nspropsFn GetNamespacePropsFn, ident table.Identifier, sc *iceberg.Schema, opts ...catalog.CreateTableOpt) (table.StagedTable, error) {
	return createStagedTable(ctx, catprops, nspropsFn, ident, "", sc, opts...)
}

// CreateStagedTableWithNamespaceKey creates a staged table using namespaceKey
// in its default warehouse location while retaining ident's namespace
// components for namespace property lookup.
func CreateStagedTableWithNamespaceKey(
	ctx context.Context,
	catprops iceberg.Properties,
	nspropsFn GetNamespacePropsFn,
	ident table.Identifier,
	namespaceKey string,
	sc *iceberg.Schema,
	opts ...catalog.CreateTableOpt,
) (table.StagedTable, error) {
	return createStagedTable(ctx, catprops, nspropsFn, ident, namespaceKey, sc, opts...)
}

func createStagedTable(
	ctx context.Context,
	catprops iceberg.Properties,
	nspropsFn GetNamespacePropsFn,
	ident table.Identifier,
	namespaceKey string,
	sc *iceberg.Schema,
	opts ...catalog.CreateTableOpt,
) (table.StagedTable, error) {
	if len(ident) < 2 {
		return table.StagedTable{}, fmt.Errorf("%w: missing namespace or invalid identifier %v", catalog.ErrNoSuchNamespace, ident)
	}
	if err := catalog.ValidateTableIdentifier(ident); err != nil {
		return table.StagedTable{}, err
	}

	var cfg catalog.CreateTableCfg
	for _, opt := range opts {
		opt(&cfg)
	}

	dbIdent := catalog.NamespaceFromIdent(ident)
	tblname := catalog.TableNameFromIdent(ident)
	if namespaceKey == "" {
		namespaceKey = strings.Join(dbIdent, ".")
	}

	loc, err := ResolveTableLocationWithNamespace(
		ctx, cfg.Location, dbIdent, namespaceKey, tblname, catprops, nspropsFn)
	if err != nil {
		return table.StagedTable{}, err
	}

	provider, err := table.LoadLocationProvider(loc, cfg.Properties)
	if err != nil {
		return table.StagedTable{}, err
	}

	metadataLoc, err := provider.NewTableMetadataFileLocation(0)
	if err != nil {
		return table.StagedTable{}, err
	}

	metadata, err := table.NewMetadata(sc, cfg.PartitionSpec, cfg.SortOrder, loc, cfg.Properties)
	if err != nil {
		return table.StagedTable{}, err
	}

	ioProps := maps.Clone(catprops)
	if ioProps == nil {
		ioProps = iceberg.Properties{}
	}
	maps.Copy(ioProps, cfg.Properties)

	return table.StagedTable{
		Table: table.New(
			ident,
			metadata,
			metadataLoc,
			icebergio.LoadFSFunc(ioProps, metadataLoc),
			nil,
		),
	}, nil
}

type GetNamespacePropsFn func(context.Context, table.Identifier) (iceberg.Properties, error)

func ResolveTableLocation(
	ctx context.Context,
	loc, dbname, tablename string,
	catprops iceberg.Properties,
	nsprops GetNamespacePropsFn,
) (string, error) {
	return ResolveTableLocationWithNamespace(
		ctx, loc, strings.Split(dbname, "."), dbname, tablename, catprops, nsprops)
}

// ResolveTableLocationWithNamespace keeps namespace property lookup component-aware
// while allowing catalogs to provide a distinct key for the default warehouse path.
func ResolveTableLocationWithNamespace(
	ctx context.Context,
	loc string,
	namespace table.Identifier,
	namespaceKey, tablename string,
	catprops iceberg.Properties,
	nsprops GetNamespacePropsFn,
) (string, error) {
	if len(loc) == 0 {
		dbprops, err := nsprops(ctx, namespace)
		if err != nil {
			return "", err
		}

		return getDefaultWarehouseLocation(namespaceKey, tablename, dbprops, catprops)
	}

	return strings.TrimSuffix(loc, "/"), nil
}

func getDefaultWarehouseLocation(namespaceKey, tablename string, nsprops, catprops iceberg.Properties) (string, error) {
	if dblocation := nsprops.Get("location", ""); dblocation != "" {
		return url.JoinPath(dblocation, tablename)
	}

	if warehousepath := catprops.Get("warehouse", ""); warehousepath != "" {
		return url.JoinPath(warehousepath, namespaceKey+".db", tablename)
	}

	return "", errors.New("no default path set, please specify a location when creating a table")
}

// (\d+)            -> version number
// -                -> separator
// ([\w-]{36})      -> UUID (36 characters, including hyphens)
// (?:\.\w+)?       -> optional codec name
// \.metadata\.json -> file extension at the end of the filename
var (
	tableMetadataFileNameRegex       = regexp.MustCompile(`^(\d+)-([\w-]{36})(?:\.\w+)?\.metadata\.json$`)
	legacyTableMetadataFileNameRegex = regexp.MustCompile(`^v(\d+)(?:\.gz)?\.metadata\.json$`)
)

func ParseMetadataVersion(location string) int {
	if strings.HasSuffix(location, "/") {
		return -1
	}

	fileName := path.Base(location)
	matches := tableMetadataFileNameRegex.FindStringSubmatch(fileName)

	if len(matches) == 3 {
		if _, err := uuid.Parse(matches[2]); err != nil {
			return -1
		}
	} else {
		matches = legacyTableMetadataFileNameRegex.FindStringSubmatch(fileName)
		if len(matches) != 2 {
			return -1
		}
	}

	v, err := strconv.Atoi(matches[1])
	if err != nil {
		return -1
	}

	return v
}

func UpdateAndStageTable(ctx context.Context, catprops iceberg.Properties, current *table.Table, ident table.Identifier, reqs []table.Requirement, updates []table.Update, cat table.CatalogIO) (*table.StagedTable, error) {
	var (
		baseMeta        table.Metadata
		currentMetadata table.Metadata
		metadataLoc     string
	)

	if current != nil {
		currentMetadata = current.Metadata()
		baseMeta = currentMetadata
		metadataLoc = current.MetadataLocation()
	} else {
		var err error
		baseMeta, err = table.NewMetadata(iceberg.NewSchema(0), nil, table.UnsortedSortOrder, "", nil)
		if err != nil {
			return nil, err
		}
	}
	newVersion := 0
	if current != nil {
		metadataVersion := ParseMetadataVersion(metadataLoc)
		if metadataVersion < 0 {
			return nil, fmt.Errorf("invalid metadata location: %s", metadataLoc)
		}
		newVersion = metadataVersion + 1
	}
	for _, r := range reqs {
		if err := r.Validate(currentMetadata); err != nil {
			return nil, err
		}
	}

	updated, err := UpdateTableMetadata(baseMeta, updates, metadataLoc)
	if err != nil {
		return nil, err
	}

	provider, err := table.LoadLocationProvider(updated.Location(), updated.Properties())
	if err != nil {
		return nil, err
	}

	newLocation, err := provider.NewTableMetadataFileLocation(newVersion)
	if err != nil {
		return nil, err
	}

	ioProps := maps.Clone(catprops)
	if ioProps == nil {
		ioProps = iceberg.Properties{}
	}
	maps.Copy(ioProps, updated.Properties())

	return &table.StagedTable{
		Table: table.New(
			ident,
			updated,
			newLocation,
			icebergio.LoadFSFunc(ioProps, newLocation),
			cat,
		),
	}, nil
}
