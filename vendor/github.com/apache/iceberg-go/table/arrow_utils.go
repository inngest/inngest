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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"log/slog"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/bitutil"
	"github.com/apache/arrow-go/v18/arrow/compute"
	"github.com/apache/arrow-go/v18/arrow/decimal128"
	"github.com/apache/arrow-go/v18/arrow/extensions"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/apache/arrow-go/v18/arrow/scalar"
	"github.com/apache/iceberg-go"
	"github.com/apache/iceberg-go/config"
	"github.com/apache/iceberg-go/internal"
	iceio "github.com/apache/iceberg-go/io"
	"github.com/apache/iceberg-go/table/dv"
	tblutils "github.com/apache/iceberg-go/table/internal"
	"github.com/geoarrow/geoarrow-go"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

// constants to look for as Keys in Arrow field metadata
const (
	ArrowFieldDocKey = "doc"
	// Arrow schemas that are generated from the Parquet library will utilize
	// this key to identify the field id of the source Parquet field.
	// We use this when converting to Iceberg to provide field IDs
	ArrowParquetFieldIDKey = "PARQUET:field_id"

	defaultBinPackLookback = 20
)

// ArrowSchemaVisitor is an interface that can be implemented and used to
// call VisitArrowSchema for iterating
type ArrowSchemaVisitor[T any] interface {
	Schema(*arrow.Schema, T) T
	Struct(*arrow.StructType, []T) T
	Field(arrow.Field, T) T
	List(arrow.ListLikeType, T) T
	Map(mt *arrow.MapType, keyResult T, valueResult T) T
	Primitive(arrow.DataType) T
}

func VisitArrowSchema[T any](sc *arrow.Schema, visitor ArrowSchemaVisitor[T]) (res T, err error) {
	if sc == nil {
		err = fmt.Errorf("%w: cannot visit nil arrow schema", iceberg.ErrInvalidArgument)

		return res, err
	}

	defer internal.RecoverError(&err)

	return visitor.Schema(sc, visitArrowStruct(arrow.StructOf(sc.Fields()...), visitor)), err
}

func visitArrowField[T any](f arrow.Field, visitor ArrowSchemaVisitor[T]) T {
	switch typ := f.Type.(type) {
	case *arrow.StructType:
		return visitArrowStruct(typ, visitor)
	case *arrow.MapType:
		return visitArrowMap(typ, visitor)
	case arrow.ListLikeType:
		return visitArrowList(typ, visitor)
	default:
		return visitor.Primitive(typ)
	}
}

func visitArrowStruct[T any](dt *arrow.StructType, visitor ArrowSchemaVisitor[T]) T {
	type (
		beforeField interface {
			BeforeField(arrow.Field)
		}
		afterField interface {
			AfterField(arrow.Field)
		}
	)

	results := make([]T, dt.NumFields())
	bf, _ := visitor.(beforeField)
	af, _ := visitor.(afterField)

	for i, f := range dt.Fields() {
		if bf != nil {
			bf.BeforeField(f)
		}

		res := visitArrowField(f, visitor)

		if af != nil {
			af.AfterField(f)
		}

		results[i] = visitor.Field(f, res)
	}

	return visitor.Struct(dt, results)
}

func visitArrowMap[T any](dt *arrow.MapType, visitor ArrowSchemaVisitor[T]) T {
	type (
		beforeMapKey interface {
			BeforeMapKey(arrow.Field)
		}
		beforeMapValue interface {
			BeforeMapValue(arrow.Field)
		}
		afterMapKey interface {
			AfterMapKey(arrow.Field)
		}
		afterMapValue interface {
			AfterMapValue(arrow.Field)
		}
	)

	key, val := dt.KeyField(), dt.ItemField()

	if bmk, ok := visitor.(beforeMapKey); ok {
		bmk.BeforeMapKey(key)
	}

	keyResult := visitArrowField(key, visitor)

	if amk, ok := visitor.(afterMapKey); ok {
		amk.AfterMapKey(key)
	}

	if bmv, ok := visitor.(beforeMapValue); ok {
		bmv.BeforeMapValue(val)
	}

	valueResult := visitArrowField(val, visitor)

	if amv, ok := visitor.(afterMapValue); ok {
		amv.AfterMapValue(val)
	}

	return visitor.Map(dt, keyResult, valueResult)
}

func visitArrowList[T any](dt arrow.ListLikeType, visitor ArrowSchemaVisitor[T]) T {
	type (
		beforeListElem interface {
			BeforeListElement(arrow.Field)
		}
		afterListElem interface {
			AfterListElement(arrow.Field)
		}
	)

	elemField := dt.ElemField()

	if bl, ok := visitor.(beforeListElem); ok {
		bl.BeforeListElement(elemField)
	}

	res := visitArrowField(elemField, visitor)

	if al, ok := visitor.(afterListElem); ok {
		al.AfterListElement(elemField)
	}

	return visitor.List(dt, res)
}

func getFieldID(f arrow.Field) *int {
	if !f.HasMetadata() {
		return nil
	}

	fieldIDStr, ok := f.Metadata.GetValue(ArrowParquetFieldIDKey)
	if !ok {
		return nil
	}

	id, err := strconv.Atoi(fieldIDStr)
	if err != nil {
		return nil
	}

	if id > 0 {
		return &id
	}

	return nil
}

type hasIDs struct{}

func (hasIDs) Schema(sc *arrow.Schema, result bool) bool {
	return result
}

func (hasIDs) Struct(st *arrow.StructType, results []bool) bool {
	return !slices.Contains(results, false)
}

func (hasIDs) Field(f arrow.Field, result bool) bool {
	return getFieldID(f) != nil
}

func (hasIDs) List(dt arrow.ListLikeType, elem bool) bool {
	elemField := dt.ElemField()

	return elem && getFieldID(elemField) != nil
}

func (hasIDs) Map(m *arrow.MapType, key, val bool) bool {
	return key && val &&
		getFieldID(m.KeyField()) != nil && getFieldID(m.ItemField()) != nil
}

func (hasIDs) Primitive(arrow.DataType) bool { return true }

type convertToIceberg struct {
	downcastTimestamp bool
	// tableSchema and tableProperties provide table context that Arrow schema
	// metadata does not carry. They are used to recover things like projjson:<property>
	//  CRS references from embedded PROJJSON metadata.
	tableSchema     *iceberg.Schema
	tableProperties iceberg.Properties

	fieldID func(arrow.Field) int
	// fields tracks the current Arrow traversal path. The top field's ID is used
	// to match GeoArrow PROJJSON metadata back to the corresponding table field.
	fields []arrow.Field
}

func (convertToIceberg) Schema(_ *arrow.Schema, result iceberg.NestedField) iceberg.NestedField {
	return result
}

func (convertToIceberg) Struct(_ *arrow.StructType, results []iceberg.NestedField) iceberg.NestedField {
	return iceberg.NestedField{
		Type: &iceberg.StructType{FieldList: results},
	}
}

func (c convertToIceberg) Field(field arrow.Field, result iceberg.NestedField) iceberg.NestedField {
	result.ID = c.fieldID(field)
	if field.HasMetadata() {
		if doc, ok := field.Metadata.GetValue(ArrowFieldDocKey); ok {
			result.Doc = doc
		}
	}

	result.Required = !field.Nullable
	result.Name = field.Name

	return result
}

func (c convertToIceberg) List(dt arrow.ListLikeType, elemResult iceberg.NestedField) iceberg.NestedField {
	elemField := dt.ElemField()
	elemID := c.fieldID(elemField)

	return iceberg.NestedField{
		Type: &iceberg.ListType{
			ElementID:       elemID,
			Element:         elemResult.Type,
			ElementRequired: !elemField.Nullable,
		},
	}
}

func (c convertToIceberg) Map(m *arrow.MapType, keyResult, valueResult iceberg.NestedField) iceberg.NestedField {
	keyField, valField := m.KeyField(), m.ItemField()
	keyID, valID := c.fieldID(keyField), c.fieldID(valField)

	return iceberg.NestedField{
		Type: &iceberg.MapType{
			KeyID:         keyID,
			KeyType:       keyResult.Type,
			ValueID:       valID,
			ValueType:     valueResult.Type,
			ValueRequired: !valField.Nullable,
		},
	}
}

func (c *convertToIceberg) pushField(field arrow.Field) {
	c.fields = append(c.fields, field)
}

func (c *convertToIceberg) popField() {
	c.fields = c.fields[:len(c.fields)-1]
}

func (c *convertToIceberg) BeforeField(field arrow.Field) {
	c.pushField(field)
}

func (c *convertToIceberg) AfterField(arrow.Field) {
	c.popField()
}

func (c *convertToIceberg) BeforeListElement(field arrow.Field) {
	c.pushField(field)
}

func (c *convertToIceberg) AfterListElement(arrow.Field) {
	c.popField()
}

func (c *convertToIceberg) BeforeMapKey(field arrow.Field) {
	c.pushField(field)
}

func (c *convertToIceberg) AfterMapKey(arrow.Field) {
	c.popField()
}

func (c *convertToIceberg) BeforeMapValue(field arrow.Field) {
	c.pushField(field)
}

func (c *convertToIceberg) AfterMapValue(arrow.Field) {
	c.popField()
}

func (c convertToIceberg) currentFieldID() (int, bool) {
	if len(c.fields) == 0 {
		return 0, false
	}

	id := getFieldID(c.fields[len(c.fields)-1])
	if id == nil {
		return 0, false
	}

	return *id, true
}

var utcAliases = []string{"UTC", "+00:00", "Etc/UTC", "Z"}

func (c convertToIceberg) Primitive(dt arrow.DataType) (result iceberg.NestedField) {
	switch dt := dt.(type) {
	case *arrow.DictionaryType:
		if _, ok := dt.ValueType.(arrow.NestedType); ok {
			panic(fmt.Errorf("%w: unsupported arrow type for conversion - %s", iceberg.ErrInvalidSchema, dt))
		}

		return c.Primitive(dt.ValueType)
	case *arrow.RunEndEncodedType:
		if _, ok := dt.Encoded().(arrow.NestedType); ok {
			panic(fmt.Errorf("%w: unsupported arrow type for conversion - %s", iceberg.ErrInvalidSchema, dt))
		}

		return c.Primitive(dt.Encoded())
	case *arrow.BooleanType:
		result.Type = iceberg.PrimitiveTypes.Bool
	case *arrow.NullType:
		result.Type = iceberg.PrimitiveTypes.Unknown
	case *arrow.Uint8Type, *arrow.Uint16Type, *arrow.Uint32Type,
		*arrow.Int8Type, *arrow.Int16Type, *arrow.Int32Type:
		result.Type = iceberg.PrimitiveTypes.Int32
	case *arrow.Uint64Type, *arrow.Int64Type:
		result.Type = iceberg.PrimitiveTypes.Int64
	case *arrow.Float16Type, *arrow.Float32Type:
		result.Type = iceberg.PrimitiveTypes.Float32
	case *arrow.Float64Type:
		result.Type = iceberg.PrimitiveTypes.Float64
	case *arrow.Decimal32Type, *arrow.Decimal64Type, *arrow.Decimal128Type:
		dec := dt.(arrow.DecimalType)
		result.Type = iceberg.DecimalTypeOf(int(dec.GetPrecision()), int(dec.GetScale()))
	case *arrow.StringType, *arrow.LargeStringType:
		result.Type = iceberg.PrimitiveTypes.String
	case *arrow.BinaryType, *arrow.LargeBinaryType:
		result.Type = iceberg.PrimitiveTypes.Binary
	case *arrow.Date32Type:
		result.Type = iceberg.PrimitiveTypes.Date
	case *arrow.Time64Type:
		if dt.Unit == arrow.Microsecond {
			result.Type = iceberg.PrimitiveTypes.Time
		} else {
			panic(fmt.Errorf("%w: unsupported arrow type for conversion - %s", iceberg.ErrInvalidSchema, dt))
		}
	case *arrow.TimestampType:
		if dt.Unit == arrow.Nanosecond {
			if !c.downcastTimestamp {
				switch {
				case slices.Contains(utcAliases, dt.TimeZone):
					result.Type = iceberg.PrimitiveTypes.TimestampTzNs
				case dt.TimeZone == "":
					result.Type = iceberg.PrimitiveTypes.TimestampNs
				default:
					panic(fmt.Errorf("%w: unsupported arrow type for conversion - %s", iceberg.ErrInvalidSchema, dt))
				}

				return result
			}
			slog.Warn("downcasting nanosecond timestamp to microsecond, precision loss may occur")
		}

		if slices.Contains(utcAliases, dt.TimeZone) {
			result.Type = iceberg.PrimitiveTypes.TimestampTz
		} else if dt.TimeZone == "" {
			result.Type = iceberg.PrimitiveTypes.Timestamp
		} else {
			panic(fmt.Errorf("%w: unsupported arrow type for conversion - %s", iceberg.ErrInvalidSchema, dt))
		}
	case *arrow.FixedSizeBinaryType:
		result.Type = iceberg.FixedTypeOf(dt.ByteWidth)
	case arrow.ExtensionType:
		switch dt.ExtensionName() {
		case "arrow.uuid":
			result.Type = iceberg.PrimitiveTypes.UUID
		case extensions.VariantExtensionName, extensions.LegacyVariantExtensionName:
			result.Type = iceberg.VariantType{}
		case "geoarrow.wkb":
			wkb, ok := dt.(*geoarrow.WKBType)
			if !ok {
				panic(fmt.Errorf("%w: unsupported arrow type for conversion - %s", iceberg.ErrInvalidSchema, dt))
			}

			iceType, err := c.geoArrowMetadataToIcebergType(wkb.Metadata())
			if err != nil {
				panic(fmt.Errorf("%w: converting geoarrow metadata: %w", iceberg.ErrInvalidSchema, err))
			}
			result.Type = iceType
		default:
			panic(fmt.Errorf("%w: unsupported arrow type for conversion - %s", iceberg.ErrInvalidSchema, dt))
		}
	default:
		panic(fmt.Errorf("%w: unsupported arrow type for conversion - %s", iceberg.ErrInvalidSchema, dt))
	}

	return result
}

func ArrowTypeToIceberg(dt arrow.DataType, downcastNsTimestamp bool) (iceberg.Type, error) {
	sc := arrow.NewSchema([]arrow.Field{{
		Type:     dt,
		Metadata: arrow.NewMetadata([]string{ArrowParquetFieldIDKey}, []string{"1"}),
	}}, nil)

	out, err := VisitArrowSchema(sc, &convertToIceberg{
		downcastTimestamp: downcastNsTimestamp,
		fieldID: func(field arrow.Field) int {
			if id := getFieldID(field); id != nil {
				return *id
			}

			panic(fmt.Errorf("%w: cannot convert %s to Iceberg field, missing field_id",
				iceberg.ErrInvalidSchema, field))
		},
	})
	if err != nil {
		return nil, err
	}

	return out.Type.(*iceberg.StructType).FieldList[0].Type, nil
}

func ArrowSchemaToIceberg(sc *arrow.Schema, downcastNsTimestamp bool, nameMapping iceberg.NameMapping) (*iceberg.Schema, error) {
	return ArrowSchemaToIcebergWithOptions(sc, ArrowToIcebergOptions{
		DowncastNsTimestamp: downcastNsTimestamp,
		NameMapping:         nameMapping,
	})
}

// ArrowToIcebergOptions controls conversion from an Arrow schema to an Iceberg schema.
type ArrowToIcebergOptions struct {
	DowncastNsTimestamp bool
	NameMapping         iceberg.NameMapping
	TableSchema         *iceberg.Schema
	TableProperties     iceberg.Properties
}

// ArrowSchemaToIcebergWithOptions converts an Arrow schema to an Iceberg schema.
// Options carry conversion inputs that are not encoded in the Arrow schema
// itself, such as table metadata needed for exact CRS recovery.
// When TableSchema and TableProperties are provided, GeoArrow PROJJSON metadata
// can be matched back to the table schema's projjson:<property> CRS references.
func ArrowSchemaToIcebergWithOptions(sc *arrow.Schema, opts ArrowToIcebergOptions) (*iceberg.Schema, error) {
	hasIDs, _ := VisitArrowSchema(sc, hasIDs{})

	switch {
	case hasIDs:
		out, err := VisitArrowSchema(sc, &convertToIceberg{
			downcastTimestamp: opts.DowncastNsTimestamp,
			tableSchema:       opts.TableSchema,
			tableProperties:   opts.TableProperties,
			fieldID: func(field arrow.Field) int {
				if id := getFieldID(field); id != nil {
					return *id
				}

				panic(fmt.Errorf("%w: cannot convert %s to Iceberg field, missing field_id",
					iceberg.ErrInvalidSchema, field))
			},
		})
		if err != nil {
			return nil, err
		}

		return iceberg.NewSchema(0, out.Type.(*iceberg.StructType).FieldList...), nil
	case opts.NameMapping != nil:
		schemaWithoutIDs, err := arrowToSchemaWithoutIDs(sc, opts.DowncastNsTimestamp)
		if err != nil {
			return nil, err
		}

		return iceberg.ApplyNameMapping(schemaWithoutIDs, opts.NameMapping)
	default:
		return nil, fmt.Errorf("%w: arrow schema does not have field-ids and no name mapping provided",
			iceberg.ErrInvalidSchema)
	}
}

func ArrowSchemaToIcebergWithFreshIDs(sc *arrow.Schema, downcastNsTimestamp bool) (*iceberg.Schema, error) {
	schemaWithoutIDs, err := arrowToSchemaWithoutIDs(sc, downcastNsTimestamp)
	if err != nil {
		return nil, err
	}

	return iceberg.AssignFreshSchemaIDs(schemaWithoutIDs, nil)
}

func arrowToSchemaWithoutIDs(sc *arrow.Schema, downcastNsTimestamp bool) (*iceberg.Schema, error) {
	withoutIDs, err := VisitArrowSchema(sc, &convertToIceberg{
		downcastTimestamp: downcastNsTimestamp,
		fieldID:           func(_ arrow.Field) int { return -1 },
	})
	if err != nil {
		return nil, err
	}

	schemaWithoutIDs := iceberg.NewSchema(0, withoutIDs.Type.(*iceberg.StructType).FieldList...)

	return schemaWithoutIDs, nil
}

type convertToSmallTypes struct{}

func (convertToSmallTypes) Schema(_ *arrow.Schema, structResult arrow.Field) arrow.Field {
	return structResult
}

func (convertToSmallTypes) Struct(_ *arrow.StructType, results []arrow.Field) arrow.Field {
	return arrow.Field{Type: arrow.StructOf(results...)}
}

func (convertToSmallTypes) Field(field arrow.Field, fieldResult arrow.Field) arrow.Field {
	field.Type = fieldResult.Type

	return field
}

func (convertToSmallTypes) List(_ arrow.ListLikeType, elemResult arrow.Field) arrow.Field {
	return arrow.Field{Type: arrow.ListOfField(elemResult)}
}

func (convertToSmallTypes) Map(_ *arrow.MapType, keyResult, valueResult arrow.Field) arrow.Field {
	return arrow.Field{
		Type: arrow.MapOfWithMetadata(keyResult.Type, keyResult.Metadata,
			valueResult.Type, valueResult.Metadata),
	}
}

func (convertToSmallTypes) Primitive(dt arrow.DataType) arrow.Field {
	switch dt.ID() {
	case arrow.LARGE_STRING:
		dt = arrow.BinaryTypes.String
	case arrow.LARGE_BINARY:
		dt = arrow.BinaryTypes.Binary
	}

	return arrow.Field{Type: dt}
}

func ensureSmallArrowTypes(dt arrow.DataType) (arrow.DataType, error) {
	top, err := VisitArrowSchema(arrow.NewSchema([]arrow.Field{{Type: dt}}, nil), convertToSmallTypes{})
	if err != nil {
		return nil, err
	}

	return top.Type.(*arrow.StructType).Field(0).Type, nil
}

type convertToArrow struct {
	metadata        map[string]string
	includeFieldIDs bool
	useLargeTypes   bool
	tableProperties iceberg.Properties
}

func (c convertToArrow) Schema(_ *iceberg.Schema, result arrow.Field) arrow.Field {
	result.Metadata = arrow.MetadataFrom(c.metadata)

	return result
}

func (c convertToArrow) Struct(_ iceberg.StructType, results []arrow.Field) arrow.Field {
	return arrow.Field{Type: arrow.StructOf(results...)}
}

func (c convertToArrow) Field(field iceberg.NestedField, result arrow.Field) arrow.Field {
	meta := map[string]string{}
	if len(field.Doc) > 0 {
		meta[ArrowFieldDocKey] = field.Doc
	}

	if c.includeFieldIDs {
		meta[ArrowParquetFieldIDKey] = strconv.Itoa(field.ID)
	}

	if len(meta) > 0 {
		result.Metadata = arrow.MetadataFrom(meta)
	}

	result.Name, result.Nullable = field.Name, !field.Required

	return result
}

func (c convertToArrow) List(list iceberg.ListType, elemResult arrow.Field) arrow.Field {
	elemField := c.Field(list.ElementField(), elemResult)
	if c.useLargeTypes {
		return arrow.Field{Type: arrow.LargeListOfField(elemField)}
	}

	return arrow.Field{Type: arrow.ListOfField(elemField)}
}

func (c convertToArrow) Map(m iceberg.MapType, keyResult, valResult arrow.Field) arrow.Field {
	keyField := c.Field(m.KeyField(), keyResult)
	valField := c.Field(m.ValueField(), valResult)

	return arrow.Field{Type: arrow.MapOfFields(keyField, valField)}
}

func (c convertToArrow) Primitive(iceberg.PrimitiveType) arrow.Field { panic("shouldn't be called") }
func (c convertToArrow) Variant(iceberg.VariantType) arrow.Field     { panic("shouldn't be called") }

func (c convertToArrow) VisitFixed(f iceberg.FixedType) arrow.Field {
	return arrow.Field{Type: &arrow.FixedSizeBinaryType{ByteWidth: f.Len()}}
}

func (c convertToArrow) VisitDecimal(d iceberg.DecimalType) arrow.Field {
	return arrow.Field{Type: &arrow.Decimal128Type{
		Precision: int32(d.Precision()), Scale: int32(d.Scale()),
	}}
}

func (c convertToArrow) VisitBoolean() arrow.Field {
	return arrow.Field{Type: arrow.FixedWidthTypes.Boolean}
}

func (c convertToArrow) VisitInt32() arrow.Field {
	return arrow.Field{Type: arrow.PrimitiveTypes.Int32}
}

func (c convertToArrow) VisitInt64() arrow.Field {
	return arrow.Field{Type: arrow.PrimitiveTypes.Int64}
}

func (c convertToArrow) VisitFloat32() arrow.Field {
	return arrow.Field{Type: arrow.PrimitiveTypes.Float32}
}

func (c convertToArrow) VisitFloat64() arrow.Field {
	return arrow.Field{Type: arrow.PrimitiveTypes.Float64}
}

func (c convertToArrow) VisitDate() arrow.Field {
	return arrow.Field{Type: arrow.FixedWidthTypes.Date32}
}

func (c convertToArrow) VisitTime() arrow.Field {
	return arrow.Field{Type: arrow.FixedWidthTypes.Time64us}
}

func (c convertToArrow) VisitTimestampTz() arrow.Field {
	return arrow.Field{Type: arrow.FixedWidthTypes.Timestamp_us}
}

func (c convertToArrow) VisitTimestamp() arrow.Field {
	return arrow.Field{Type: &arrow.TimestampType{Unit: arrow.Microsecond}}
}

func (c convertToArrow) VisitTimestampNs() arrow.Field {
	return arrow.Field{Type: &arrow.TimestampType{Unit: arrow.Nanosecond}}
}

func (c convertToArrow) VisitTimestampNsTz() arrow.Field {
	return arrow.Field{Type: arrow.FixedWidthTypes.Timestamp_ns}
}

func (c convertToArrow) VisitString() arrow.Field {
	if c.useLargeTypes {
		return arrow.Field{Type: arrow.BinaryTypes.LargeString}
	}

	return arrow.Field{Type: arrow.BinaryTypes.String}
}

func (c convertToArrow) VisitBinary() arrow.Field {
	if c.useLargeTypes {
		return arrow.Field{Type: arrow.BinaryTypes.LargeBinary}
	}

	return arrow.Field{Type: arrow.BinaryTypes.Binary}
}

func (c convertToArrow) VisitUUID() arrow.Field {
	return arrow.Field{Type: extensions.NewUUIDType()}
}

func (c convertToArrow) VisitUnknown() arrow.Field {
	return arrow.Field{Type: arrow.Null}
}

func (c convertToArrow) VisitVariant() arrow.Field {
	if c.useLargeTypes {
		vt, _ := extensions.NewVariantType(arrow.StructOf(
			arrow.Field{Name: "metadata", Type: arrow.BinaryTypes.LargeBinary, Nullable: false},
			arrow.Field{Name: "value", Type: arrow.BinaryTypes.LargeBinary, Nullable: false},
		))

		return arrow.Field{Type: vt}
	}

	return arrow.Field{Type: extensions.NewDefaultVariantType()}
}

func (c convertToArrow) VisitGeometry(g iceberg.GeometryType) arrow.Field {
	meta, err := icebergCRSToGeoArrowMetadata(g.CRS(), c.tableProperties)
	if err != nil {
		// Panic to thread the error through iceberg.Visit's recover, matching the
		// convention used by the other visitor methods.
		panic(err)
	}

	if c.useLargeTypes {
		return arrow.Field{Type: geoarrow.NewWKBType(geoarrow.WKBWithLargeBinaryStorage(), geoarrow.WKBWithMetadata(meta))}
	}

	return arrow.Field{Type: geoarrow.NewWKBType(geoarrow.WKBWithBinaryStorage(), geoarrow.WKBWithMetadata(meta))}
}

func (c convertToArrow) VisitGeography(g iceberg.GeographyType) arrow.Field {
	meta, err := icebergCRSToGeoArrowMetadata(g.CRS(), c.tableProperties)
	if err != nil {
		// Panic to thread the error through iceberg.Visit's recover, matching the
		// convention used by the other visitor methods.
		panic(err)
	}

	// Always write an edge algorithm so iceberg-go-authored Arrow metadata can
	// distinguish Geography from Geometry on round trip. Arrow-only reads from
	// clients that omit edges for geography are lossy.
	meta.Edges = geoarrow.EdgeInterpolation(g.Algorithm())

	if c.useLargeTypes {
		return arrow.Field{Type: geoarrow.NewWKBType(geoarrow.WKBWithLargeBinaryStorage(), geoarrow.WKBWithMetadata(meta))}
	}

	return arrow.Field{Type: geoarrow.NewWKBType(geoarrow.WKBWithBinaryStorage(), geoarrow.WKBWithMetadata(meta))}
}

var _ iceberg.SchemaVisitorPerPrimitiveType[arrow.Field] = convertToArrow{}

// ArrowSchemaOptions controls conversion from an Iceberg schema or type to Arrow.
type ArrowSchemaOptions struct {
	Metadata        map[string]string
	IncludeFieldIDs bool
	UseLargeTypes   bool
	TableProperties iceberg.Properties
}

// SchemaToArrowSchema converts an Iceberg schema to an Arrow schema. If the metadata parameter
// is non-nil, it will be included as the top-level metadata in the schema. If includeFieldIDs
// is true, then each field of the schema will contain a metadata key PARQUET:field_id set to
// the field id from the iceberg schema.
func SchemaToArrowSchema(sc *iceberg.Schema, metadata map[string]string, includeFieldIDs, useLargeTypes bool) (*arrow.Schema, error) {
	return SchemaToArrowSchemaWithOptions(sc, ArrowSchemaOptions{
		Metadata:        metadata,
		IncludeFieldIDs: includeFieldIDs,
		UseLargeTypes:   useLargeTypes,
	})
}

// SchemaToArrowSchemaWithOptions converts an Iceberg schema to an Arrow schema.
// TableProperties are used to resolve references like projjson:<property> CRS which point to
// metadata in the table properties.
func SchemaToArrowSchemaWithOptions(sc *iceberg.Schema, opts ArrowSchemaOptions) (*arrow.Schema, error) {
	top, err := iceberg.Visit(sc, convertToArrow{
		metadata:        opts.Metadata,
		includeFieldIDs: opts.IncludeFieldIDs,
		useLargeTypes:   opts.UseLargeTypes,
		tableProperties: opts.TableProperties,
	})
	if err != nil {
		return nil, err
	}

	return arrow.NewSchema(top.Type.(*arrow.StructType).Fields(), &top.Metadata), nil
}

// TypeToArrowType converts a given iceberg type, into the equivalent Arrow data type.
// For dealing with nested fields (List, Struct, Map) if includeFieldIDs is true, then
// the child fields will contain a metadata key PARQUET:field_id set to the field id.
func TypeToArrowType(t iceberg.Type, includeFieldIDs bool, useLargeTypes bool) (arrow.DataType, error) {
	return TypeToArrowTypeWithOptions(t, ArrowSchemaOptions{
		IncludeFieldIDs: includeFieldIDs,
		UseLargeTypes:   useLargeTypes,
	})
}

// TypeToArrowTypeWithOptions converts a given Iceberg type into the equivalent Arrow data type.
// Options control representation details and provide table properties used to
// resolve metadata references such as CRS projjson:<property> which are stored as table properties.
func TypeToArrowTypeWithOptions(t iceberg.Type, opts ArrowSchemaOptions) (arrow.DataType, error) {
	top, err := iceberg.Visit(iceberg.NewSchema(0, iceberg.NestedField{Type: t}),
		convertToArrow{
			includeFieldIDs: opts.IncludeFieldIDs,
			useLargeTypes:   opts.UseLargeTypes,
			tableProperties: opts.TableProperties,
		})
	if err != nil {
		return nil, err
	}

	return top.Type.(*arrow.StructType).Field(0).Type, nil
}

type arrowAccessor struct {
	fileSchema *iceberg.Schema
}

func (a arrowAccessor) SchemaPartner(partner arrow.Array) arrow.Array {
	return partner
}

func (a arrowAccessor) FieldPartner(partnerStruct arrow.Array, fieldID int, _ string) arrow.Array {
	if partnerStruct == nil {
		return nil
	}

	field, ok := a.fileSchema.FindFieldByIDRef(fieldID, internal.SchemaRef{})
	if !ok {
		return nil
	}

	if st, ok := partnerStruct.(*array.Struct); ok {
		if idx, ok := st.DataType().(*arrow.StructType).FieldIdx(field.Name); ok {
			return st.Field(idx)
		}
	}

	panic(fmt.Errorf("cannot find %s in expected partner_struct type %s",
		field.Name, partnerStruct.DataType()))
}

func (a arrowAccessor) ListElementPartner(partnerList arrow.Array) arrow.Array {
	if l, ok := partnerList.(array.ListLike); ok {
		return l.ListValues()
	}

	return nil
}

func (a arrowAccessor) MapKeyPartner(partnerMap arrow.Array) arrow.Array {
	if m, ok := partnerMap.(*array.Map); ok {
		return m.Keys()
	}

	return nil
}

func (a arrowAccessor) MapValuePartner(partnerMap arrow.Array) arrow.Array {
	if m, ok := partnerMap.(*array.Map); ok {
		return m.Items()
	}

	return nil
}

func retOrPanic[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}

	return v
}

// numericDefault converts v to T, accepting the typed iceberg form, the
// float64 that encoding/json produces when deserializing into any, or the
// json.Number that a decoder configured with UseNumber() produces.
func numericDefault[T ~int32 | ~int64 | ~float32 | ~float64](v any) T {
	switch val := v.(type) {
	case T:
		return val
	case float64:
		return T(val)
	case json.Number:
		f, err := val.Float64()
		if err != nil {
			panic(fmt.Errorf("unsupported json.Number %q for numeric iceberg type: %w", val, err))
		}

		return T(f)
	}
	panic(fmt.Errorf("unsupported write-default value type %T for numeric iceberg type", v))
}

// defaultToScalar converts an Iceberg default value to an Arrow scalar.
func defaultToScalar(v any, t iceberg.Type, dt arrow.DataType) scalar.Scalar {
	switch typ := t.(type) {
	case iceberg.Float32Type:
		s, err := scalar.MakeScalarParam(numericDefault[float32](v), dt)
		if err != nil {
			panic(fmt.Errorf("write-default float32 (iceberg type %s, value %v %T): %w", t, v, v, err))
		}

		return s
	case iceberg.DateType:
		return scalar.NewDate32Scalar(arrow.Date32(numericDefault[iceberg.Date](v)))
	case iceberg.TimeType:
		return scalar.NewTime64Scalar(arrow.Time64(numericDefault[iceberg.Time](v)), dt)
	case iceberg.TimestampType, iceberg.TimestampTzType:
		return scalar.NewTimestampScalar(arrow.Timestamp(numericDefault[iceberg.Timestamp](v)), dt)
	case iceberg.TimestampNsType, iceberg.TimestampTzNsType:
		return scalar.NewTimestampScalar(arrow.Timestamp(numericDefault[iceberg.TimestampNano](v)), dt)
	case iceberg.UUIDType:
		switch val := v.(type) {
		case uuid.UUID:
			s, err := scalar.MakeScalarParam(val[:], &arrow.FixedSizeBinaryType{ByteWidth: 16})
			if err != nil {
				panic(fmt.Errorf("write-default uuid (value %v): %w", val, err))
			}

			return s
		case string:
			u, err := uuid.Parse(val)
			if err != nil {
				panic(fmt.Errorf("write-default uuid: cannot parse string %q: %w", val, err))
			}
			s, err := scalar.MakeScalarParam(u[:], &arrow.FixedSizeBinaryType{ByteWidth: 16})
			if err != nil {
				panic(fmt.Errorf("write-default uuid (value %v): %w", val, err))
			}

			return s
		}
		panic(fmt.Errorf("write-default uuid: unsupported value type %T (%v)", v, v))
	case iceberg.DecimalType:
		switch val := v.(type) {
		case iceberg.Decimal:
			return scalar.NewDecimal128Scalar(val.Val, dt)
		case string:
			n, err := decimal128.FromString(val, int32(typ.Precision()), int32(typ.Scale()))
			if err != nil {
				panic(fmt.Errorf("write-default decimal(p=%d, s=%d): cannot parse string %q: %w", typ.Precision(), typ.Scale(), val, err))
			}

			return scalar.NewDecimal128Scalar(n, dt)
		}
		panic(fmt.Errorf("write-default decimal: unsupported value type %T (%v)", v, v))
	case iceberg.BinaryType, iceberg.FixedType:
		switch val := v.(type) {
		case []byte:
			s, err := scalar.MakeScalarParam(val, dt)
			if err != nil {
				panic(fmt.Errorf("write-default binary/fixed (iceberg type %s, value %v): %w", t, val, err))
			}

			return s
		case string:
			fixedLen := -1
			if fixed, ok := t.(iceberg.FixedType); ok {
				fixedLen = fixed.Len()
			}
			b, err := internal.DecodeDefaultBytes(val, fixedLen)
			if err != nil {
				panic(fmt.Errorf("write-default binary/fixed (iceberg type %s): cannot decode string %q: %w", t, val, err))
			}
			s, err := scalar.MakeScalarParam(b, dt)
			if err != nil {
				panic(fmt.Errorf("write-default binary/fixed (iceberg type %s, value %v): %w", t, b, err))
			}

			return s
		}
		panic(fmt.Errorf("write-default binary/fixed: unsupported value type %T (%v)", v, v))
	// Float64, Bool, and String cast normally.
	// Int32 and Int64 arrive as float64 from JSON and are handled by MakeScalarParam.
	default:
		s, err := scalar.MakeScalarParam(v, dt)
		if err != nil {
			panic(fmt.Errorf("write-default (iceberg type %s, value %v %T): %w", t, v, v, err))
		}

		return s
	}
}

// defaultToArray creates an Arrow array of length n filled with the given default value v.
func defaultToArray(v any, t iceberg.Type, dt arrow.DataType, n int, alloc memory.Allocator) arrow.Array {
	sc := defaultToScalar(v, t, dt)
	out, err := scalar.MakeArrayFromScalar(sc, n, alloc)
	if err != nil {
		panic(fmt.Errorf("write-default (iceberg type %s, value %v %T): failed to create array: %w", t, v, v, err))
	}
	if _, ok := dt.(*extensions.UUIDType); ok {
		defer out.Release()

		data := array.NewData(dt, out.Len(), out.Data().Buffers(), nil, out.NullN(), 0)
		defer data.Release()

		return array.MakeFromData(data)
	}

	return out
}

type arrowProjectionVisitor struct {
	ctx                  context.Context
	fileSchema           *iceberg.Schema
	tableProperties      iceberg.Properties
	formatVersion        int
	includeFieldIDs      bool
	downcastNsTimestamp  bool
	useLargeTypes        bool
	useWriteDefault      bool
	allowMissingRequired bool
}

func (a *arrowProjectionVisitor) typeToArrowType(t iceberg.Type) arrow.DataType {
	return retOrPanic(TypeToArrowTypeWithOptions(t, ArrowSchemaOptions{
		IncludeFieldIDs: a.includeFieldIDs,
		UseLargeTypes:   a.useLargeTypes,
		TableProperties: a.tableProperties,
	}))
}

func (a *arrowProjectionVisitor) castIfNeeded(field iceberg.NestedField, vals arrow.Array) arrow.Array {
	fileField, ok := a.fileSchema.FindFieldByIDRef(field.ID, internal.SchemaRef{})
	if !ok {
		panic(fmt.Errorf("could not find field id %d in schema", field.ID))
	}

	typ, ok := fileField.Type.(iceberg.PrimitiveType)
	if !ok {
		vals.Retain()

		return vals
	}

	if !field.Type.Equals(typ) {
		if !canDowncastTimestampPrecision(fileField.Type, field.Type, a.downcastNsTimestamp) {
			var targetType arrow.DataType
			if a.formatVersion >= 3 && canPromoteDateToTimestamp(fileField.Type, field.Type) {
				targetType = a.typeToArrowType(field.Type)
			} else {
				promoted := retOrPanic(iceberg.PromoteType(fileField.Type, field.Type))
				targetType = a.typeToArrowType(promoted)
			}
			if !a.useLargeTypes {
				targetType = retOrPanic(ensureSmallArrowTypes(targetType))
			}

			return retOrPanic(compute.CastArray(a.ctx, vals,
				compute.SafeCastOptions(targetType)))
		}
	}

	targetType := a.typeToArrowType(field.Type)
	if !arrow.TypeEqual(targetType, vals.DataType()) {
		if out, ok := a.rewrapExtension(targetType, vals); ok {
			return out
		}

		switch field.Type.(type) {
		case iceberg.TimestampType:
			tt, tgtok := targetType.(*arrow.TimestampType)
			vt, valok := vals.DataType().(*arrow.TimestampType)

			if tgtok && valok && tt.TimeZone == "" && vt.TimeZone == "" && tt.Unit == arrow.Microsecond {
				if vt.Unit == arrow.Nanosecond && a.downcastNsTimestamp {
					return floorTimestampNanosecondsToMicroseconds(
						compute.GetAllocator(a.ctx), vals.(*array.Timestamp), tt)
				} else if vt.Unit == arrow.Second || vt.Unit == arrow.Millisecond {
					return retOrPanic(compute.CastArray(a.ctx, vals, compute.SafeCastOptions(tt)))
				}
			}

			panic(fmt.Errorf("unsupported schema projection from %s to %s",
				vals.DataType(), targetType))
		case iceberg.TimestampTzType:
			tt, tgtok := targetType.(*arrow.TimestampType)
			vt, valok := vals.DataType().(*arrow.TimestampType)

			if tgtok && valok && tt.TimeZone == "UTC" &&
				slices.Contains(utcAliases, vt.TimeZone) && tt.Unit == arrow.Microsecond {
				if vt.Unit == arrow.Nanosecond && a.downcastNsTimestamp {
					return floorTimestampNanosecondsToMicroseconds(
						compute.GetAllocator(a.ctx), vals.(*array.Timestamp), tt)
				} else if vt.Unit != arrow.Nanosecond {
					return retOrPanic(compute.CastArray(a.ctx, vals, compute.SafeCastOptions(tt)))
				}
			}

			panic(fmt.Errorf("unsupported schema projection from %s to %s",
				vals.DataType(), targetType))
		default:
			return retOrPanic(compute.CastArray(a.ctx, vals,
				compute.SafeCastOptions(targetType)))
		}
	}
	vals.Retain()

	return vals
}

// rewrapExtension re-wraps vals in targetType when both are extension types
// with the same extension name but differing extension parameters, returning
// false when the types are not such a pair. Arrow's compute layer has no
// extension-to-extension cast kernel, and the values are already in the
// extension's storage encoding, so only the storage array is cast and that only
// when the storage types differ. This is safe only because a shared extension
// name implies the storage payload is interpretation-compatible: parameters may
// differ, the encoded values do not.
func (a *arrowProjectionVisitor) rewrapExtension(targetType arrow.DataType, vals arrow.Array) (arrow.Array, bool) {
	tgtExt, ok := targetType.(arrow.ExtensionType)
	if !ok {
		return nil, false
	}

	srcExt, ok := vals.DataType().(arrow.ExtensionType)
	if !ok || srcExt.ExtensionName() != tgtExt.ExtensionName() {
		return nil, false
	}

	if arrow.TypeEqual(srcExt.StorageType(), tgtExt.StorageType()) {
		return array.NewExtensionArrayWithStorage(tgtExt,
			vals.(array.ExtensionArray).Storage()), true
	}

	storage := retOrPanic(compute.CastArray(a.ctx,
		vals.(array.ExtensionArray).Storage(),
		compute.SafeCastOptions(tgtExt.StorageType())))
	defer storage.Release()

	return array.NewExtensionArrayWithStorage(tgtExt, storage), true
}

func canDowncastTimestampPrecision(fileType, readType iceberg.Type, enabled bool) bool {
	if !enabled {
		return false
	}

	switch fileType.(type) {
	case iceberg.TimestampNsType:
		_, ok := readType.(iceberg.TimestampType)

		return ok
	case iceberg.TimestampTzNsType:
		_, ok := readType.(iceberg.TimestampTzType)

		return ok
	default:
		return false
	}
}

func canPromoteDateToTimestamp(fileType, readType iceberg.Type) bool {
	if _, ok := fileType.(iceberg.DateType); !ok {
		return false
	}

	switch readType.(type) {
	case iceberg.TimestampType, iceberg.TimestampNsType:
		return true
	default:
		return false
	}
}

func floorTimestampNanosecondsToMicroseconds(
	mem memory.Allocator,
	vals *array.Timestamp,
	targetType *arrow.TimestampType,
) arrow.Array {
	bldr := array.NewTimestampBuilder(mem, targetType)
	defer bldr.Release()

	for i := range vals.Len() {
		if vals.IsNull(i) {
			bldr.AppendNull()

			continue
		}

		bldr.Append(arrow.Timestamp(internal.FloorDiv(int64(vals.Value(i)), int64(1_000))))
	}

	return bldr.NewArray()
}

func (a *arrowProjectionVisitor) constructField(field iceberg.NestedField, arrowType arrow.DataType) arrow.Field {
	metadata := map[string]string{}
	if field.Doc != "" {
		metadata[ArrowFieldDocKey] = field.Doc
	}

	if a.includeFieldIDs {
		metadata[ArrowParquetFieldIDKey] = strconv.Itoa(field.ID)
	}

	return arrow.Field{
		Name:     field.Name,
		Type:     arrowType,
		Nullable: !field.Required,
		Metadata: arrow.MetadataFrom(metadata),
	}
}

func (a *arrowProjectionVisitor) Schema(_ *iceberg.Schema, _ arrow.Array, result arrow.Array) arrow.Array {
	return result
}

func ownsChildResult(arr arrow.Array) bool {
	if arr == nil {
		return false
	}

	_, nested := arr.DataType().(arrow.NestedType)

	return nested
}

func (a *arrowProjectionVisitor) Struct(st iceberg.StructType, structArr arrow.Array, fieldResults []arrow.Array) arrow.Array {
	if structArr == nil {
		return nil
	}

	fieldArrs := make([]arrow.Array, len(st.FieldList))
	fields := make([]arrow.Field, len(st.FieldList))
	for i, field := range st.FieldList {
		arr := fieldResults[i]
		if arr != nil {
			if ownsChildResult(arr) {
				defer arr.Release()
			}

			arr = a.castIfNeeded(field, arr)
			defer arr.Release()
			fieldArrs[i] = arr
			fields[i] = a.constructField(field, arr.DataType())
		} else {
			dt := a.typeToArrowType(field.Type)
			alloc := compute.GetAllocator(a.ctx)

			switch {
			case field.WriteDefault != nil && a.useWriteDefault:
				arr = defaultToArray(field.WriteDefault, field.Type, dt, structArr.Len(), alloc)
			case field.InitialDefault != nil && !a.useWriteDefault:
				arr = defaultToArray(field.InitialDefault, field.Type, dt, structArr.Len(), alloc)
			case !field.Required || a.allowMissingRequired:
				arr = array.MakeArrayOfNull(alloc, dt, structArr.Len())
			default:
				panic(fmt.Errorf("%w: required field is missing and has no default value: %s",
					iceberg.ErrInvalidSchema, field))
			}

			defer arr.Release()
			fieldArrs[i] = arr
			fields[i] = a.constructField(field, arr.DataType())
		}
	}

	var nullBitmap *memory.Buffer
	if structArr.NullN() > 0 {
		if structArr.Data().Offset() > 0 {
			// the children already accounted for any offset because we used the `Field` method
			// on the struct array in the FieldPartner accessor. So we just need to adjust the
			// bitmap to account for the offset.
			nullBitmap = memory.NewResizableBuffer(compute.GetAllocator(a.ctx))
			defer nullBitmap.Release()
			nullBitmap.Resize(int(bitutil.BytesForBits(int64(structArr.Len()))))

			bitutil.CopyBitmap(structArr.NullBitmapBytes(), structArr.Data().Offset(), structArr.Len(),
				nullBitmap.Bytes(), 0)

		} else {
			nullBitmap = structArr.Data().Buffers()[0]
		}
	}

	return retOrPanic(array.NewStructArrayWithFieldsAndNulls(fieldArrs, fields,
		nullBitmap, structArr.NullN(), 0))
}

func (a *arrowProjectionVisitor) Field(_ iceberg.NestedField, _ arrow.Array, fieldArr arrow.Array) arrow.Array {
	return fieldArr
}

func (a *arrowProjectionVisitor) List(listType iceberg.ListType, listArr arrow.Array, valArr arrow.Array) arrow.Array {
	if ownsChildResult(valArr) {
		defer valArr.Release()
	}

	arr, ok := listArr.(array.ListLike)
	if !ok || valArr == nil {
		return nil
	}

	valArr = a.castIfNeeded(listType.ElementField(), valArr)
	defer valArr.Release()

	var outType arrow.ListLikeType
	elemField := a.constructField(listType.ElementField(), valArr.DataType())
	switch arr.DataType().ID() {
	case arrow.LIST:
		outType = arrow.ListOfField(elemField)
	case arrow.LARGE_LIST:
		outType = arrow.LargeListOfField(elemField)
	case arrow.LIST_VIEW:
		outType = arrow.LargeListViewOfField(elemField)
	}

	data := array.NewData(outType, arr.Len(), arr.Data().Buffers(),
		[]arrow.ArrayData{valArr.Data()}, arr.NullN(), arr.Data().Offset())
	defer data.Release()

	out := array.MakeFromData(data)

	// The Parquet reader may return large_list offsets even when the schema
	// returned alongside the records (built with useLargeTypes) asks for small
	// list, or vice versa; coerce the offset width to match so the writer does
	// not reject the batch. A plain relabel would misread int64 offsets as int32.
	switch out.DataType().ID() {
	case arrow.LIST, arrow.LARGE_LIST:
		var want arrow.DataType = arrow.ListOfField(elemField)
		if a.useLargeTypes {
			want = arrow.LargeListOfField(elemField)
		}
		if !arrow.TypeEqual(out.DataType(), want) {
			defer out.Release()

			return retOrPanic(compute.CastArray(a.ctx, out, compute.SafeCastOptions(want)))
		}
	}

	return out
}

func (a *arrowProjectionVisitor) Map(m iceberg.MapType, mapArray, keyResult, valResult arrow.Array) arrow.Array {
	if ownsChildResult(keyResult) {
		defer keyResult.Release()
	}

	if ownsChildResult(valResult) {
		defer valResult.Release()
	}

	if keyResult == nil || valResult == nil {
		return nil
	}

	arr, ok := mapArray.(*array.Map)
	if !ok {
		return nil
	}

	keys := a.castIfNeeded(m.KeyField(), keyResult)
	defer keys.Release()
	vals := a.castIfNeeded(m.ValueField(), valResult)
	defer vals.Release()

	keyField := a.constructField(m.KeyField(), keys.DataType())
	valField := a.constructField(m.ValueField(), vals.DataType())

	mapType := arrow.MapOfWithMetadata(keyField.Type, keyField.Metadata, valField.Type, valField.Metadata)
	childData := array.NewData(mapType.Elem(), arr.Data().Children()[0].Len(), []*memory.Buffer{nil},
		[]arrow.ArrayData{keys.Data(), vals.Data()}, 0, 0)
	defer childData.Release()
	newData := array.NewData(mapType, arr.Len(), arr.Data().Buffers(),
		[]arrow.ArrayData{childData}, arr.NullN(), arr.Offset())
	defer newData.Release()

	return array.NewMapData(newData)
}

func (a *arrowProjectionVisitor) Primitive(_ iceberg.PrimitiveType, arr arrow.Array) arrow.Array {
	return arr
}

// Variant reassembles a shredded variant back into the unshredded layout so the
// projected schema matches; without it the scanner fails to assemble the table.
func (a *arrowProjectionVisitor) Variant(_ iceberg.VariantType, arr arrow.Array) arrow.Array {
	if arr == nil {
		return arr
	}
	varr, ok := arr.(*extensions.VariantArray)
	if !ok || !varr.IsShredded() {
		return arr
	}
	// UnshredVariant returns a caller-owned array whose ref is balanced by
	// castIfNeeded's Retain/Release. This holds only because VariantType is not
	// an arrow.NestedType; if it were, ownsChildResult would report true and the
	// parent Struct/List/Map would Release and double-free.
	// The checked-allocator leak tests guard this invariant.
	out, err := extensions.UnshredVariant(varr, compute.GetAllocator(a.ctx))
	if err != nil {
		panic(fmt.Errorf("variant projection: %w", err))
	}

	return out
}

// SchemaOptions controls the behaviour of ToRequestedSchema.
type SchemaOptions struct {
	DowncastTimestamp bool
	// FormatVersion enables format-version-specific read promotions.
	FormatVersion        int
	IncludeFieldIDs      bool
	UseLargeTypes        bool
	UseWriteDefault      bool
	AllowMissingRequired bool
	TableProperties      iceberg.Properties
}

// ToRequestedSchema will construct a new record batch matching the requested iceberg schema
// casting columns if necessary as appropriate.
func ToRequestedSchema(ctx context.Context, requested, fileSchema *iceberg.Schema, batch arrow.RecordBatch, opts SchemaOptions) (arrow.RecordBatch, error) {
	return toRequestedSchema(ctx, requested, fileSchema, batch, opts, nil)
}

// toRequestedSchema is the internal write-path variant of ToRequestedSchema.
// When targetArrowSchema is provided, an exact Arrow schema match can reuse the
// input batch without walking the Iceberg schema or rebuilding any arrays.
func toRequestedSchema(ctx context.Context, requested, fileSchema *iceberg.Schema, batch arrow.RecordBatch, opts SchemaOptions, targetArrowSchema *arrow.Schema) (arrow.RecordBatch, error) {
	if targetArrowSchema != nil && arrowSchemaEqual(batch.Schema(), targetArrowSchema) {
		batch.Retain()

		return batch, nil
	}

	st := array.RecordToStructArray(batch)
	defer st.Release()

	result, err := iceberg.VisitSchemaWithPartner[arrow.Array, arrow.Array](requested, st,
		&arrowProjectionVisitor{
			ctx:                  ctx,
			fileSchema:           fileSchema,
			tableProperties:      opts.TableProperties,
			formatVersion:        opts.FormatVersion,
			includeFieldIDs:      opts.IncludeFieldIDs,
			downcastNsTimestamp:  opts.DowncastTimestamp,
			useLargeTypes:        opts.UseLargeTypes,
			useWriteDefault:      opts.UseWriteDefault,
			allowMissingRequired: opts.AllowMissingRequired,
		}, arrowAccessor{fileSchema: fileSchema})
	if err != nil {
		return nil, err
	}
	out := array.RecordFromStructArray(result.(*array.Struct), nil)
	result.Release()

	return out, nil
}

func arrowSchemaEqual(left, right *arrow.Schema) bool {
	if left == nil || right == nil {
		return left == right
	}

	// Schema.Equal ignores top-level metadata. Full projection strips it, so
	// guard it here; otherwise the fast path would preserve metadata projection drops.
	return left.Equal(right) && left.Metadata().Equal(right.Metadata())
}

type schemaCompatVisitor struct {
	provided *iceberg.Schema

	errorData [][]string
}

func checkSchemaCompat(requested, provided *iceberg.Schema) error {
	sc := &schemaCompatVisitor{
		provided:  provided,
		errorData: [][]string{{"", "Table Field", "Requested Field"}},
	}

	_, compat := iceberg.PreOrderVisit(requested, sc)

	return compat
}

func checkArrowSchemaCompat(requested *iceberg.Schema, provided *arrow.Schema, downcastNanoToMicro bool) error {
	mapping := requested.NameMapping()
	providedSchema, err := ArrowSchemaToIceberg(provided, downcastNanoToMicro, mapping)
	if err != nil {
		return err
	}

	return checkSchemaCompat(requested, providedSchema)
}

func checkArrowSchemaCompatWithProperties(requested *iceberg.Schema, provided *arrow.Schema, downcastNanoToMicro bool, props iceberg.Properties) error {
	providedSchema, err := ArrowSchemaToIcebergWithOptions(provided, ArrowToIcebergOptions{
		DowncastNsTimestamp: downcastNanoToMicro,
		NameMapping:         requested.NameMapping(),
		TableSchema:         requested,
		TableProperties:     props,
	})
	if err != nil {
		return err
	}

	return checkSchemaCompat(requested, providedSchema)
}

func (sc *schemaCompatVisitor) isFieldCompat(lhs iceberg.NestedField) bool {
	rhs, ok := sc.provided.FindFieldByID(lhs.ID)
	if !ok {
		if lhs.Required && lhs.WriteDefault == nil && lhs.InitialDefault == nil {
			sc.errorData = append(sc.errorData,
				[]string{"❌", lhs.String(), "missing"})

			return false
		}
		sc.errorData = append(sc.errorData,
			[]string{"✅", lhs.String(), "missing"})

		return true
	}

	if lhs.Required && !rhs.Required {
		sc.errorData = append(sc.errorData,
			[]string{"❌", lhs.String(), rhs.String()})

		return false
	}

	if lhs.Type.Equals(rhs.Type) {
		sc.errorData = append(sc.errorData,
			[]string{"✅", lhs.String(), rhs.String()})

		return true
	}

	// we only check that parent node is also of the same type
	// we check the type of the child nodes as we traverse them later
	switch lhs.Type.(type) {
	case *iceberg.StructType:
		if rhs, ok := rhs.Type.(*iceberg.StructType); ok {
			sc.errorData = append(sc.errorData,
				[]string{"✅", lhs.String(), rhs.String()})

			return true
		}
	case *iceberg.ListType:
		if rhs, ok := rhs.Type.(*iceberg.ListType); ok {
			sc.errorData = append(sc.errorData,
				[]string{"✅", lhs.String(), rhs.String()})

			return true
		}
	case *iceberg.MapType:
		if rhs, ok := rhs.Type.(*iceberg.MapType); ok {
			sc.errorData = append(sc.errorData,
				[]string{"✅", lhs.String(), rhs.String()})

			return true
		}
	}

	if _, err := iceberg.PromoteType(rhs.Type, lhs.Type); err != nil {
		sc.errorData = append(sc.errorData,
			[]string{"❌", lhs.String(), rhs.String()})

		return false
	}

	sc.errorData = append(sc.errorData,
		[]string{"✅", lhs.String(), rhs.String()})

	return true
}

func (sc *schemaCompatVisitor) Schema(s *iceberg.Schema, v func() bool) bool {
	if !v() {
		// Render the collected mismatch rows into the error message with the
		// stdlib tabwriter. Unlike the former pterm renderer this writes only to
		// the returned error and has no stdout side effect. tabwriter measures
		// column width by rune count, so the leading status column (✅/❌) may be
		// off by one display cell in terminals that render those double-width;
		// the reported content is unaffected.
		var buf strings.Builder
		w := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
		for _, row := range sc.errorData {
			fmt.Fprintln(w, strings.Join(row, "\t"))
		}
		_ = w.Flush()

		panic("mismatch in fields:\n" + buf.String())
	}

	return true
}

func (sc *schemaCompatVisitor) Struct(st iceberg.StructType, v []func() bool) bool {
	out := true
	for _, res := range v {
		out = res() && out
	}

	return out
}

func (sc *schemaCompatVisitor) Field(n iceberg.NestedField, v func() bool) bool {
	return sc.isFieldCompat(n) && v()
}

func (sc *schemaCompatVisitor) List(l iceberg.ListType, v func() bool) bool {
	return sc.isFieldCompat(l.ElementField()) && v()
}

func (sc *schemaCompatVisitor) Map(m iceberg.MapType, vk, vv func() bool) bool {
	return sc.isFieldCompat(m.KeyField()) && sc.isFieldCompat(m.ValueField()) && vk() && vv()
}

func (sc *schemaCompatVisitor) Primitive(p iceberg.PrimitiveType) bool {
	return true
}

func (sc *schemaCompatVisitor) Variant(v iceberg.VariantType) bool {
	return true
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}

	return v
}

type arrowStatsCollector struct {
	fieldID          int
	schema           *iceberg.Schema
	defaultMode      tblutils.MetricsMode
	defaultModeError error
	columnModes      map[string]tblutils.MetricsMode
	columnModeErrors map[string]error
}

func (a *arrowStatsCollector) resolveColumnMetricsMode(colName string) tblutils.MetricsMode {
	if a.defaultModeError != nil {
		panic(a.defaultModeError)
	}

	if metMode, ok := a.columnModes[colName]; ok {
		return metMode
	}
	if err, ok := a.columnModeErrors[colName]; ok {
		panic(err)
	}

	return a.defaultMode
}

func (a *arrowStatsCollector) primitiveCollector(dt iceberg.PrimitiveType, isNested bool) (tblutils.StatisticsCollector, bool) {
	colName, ok := a.schema.FindColumnName(a.fieldID)
	if !ok {
		return tblutils.StatisticsCollector{}, false
	}

	metMode := a.resolveColumnMetricsMode(colName)

	switch dt.(type) {
	case iceberg.StringType:
	case iceberg.BinaryType:
	default:
		if metMode.Typ == tblutils.MetricModeTruncate {
			metMode = tblutils.MetricsMode{Typ: tblutils.MetricModeFull, Len: 0}
		}
	}

	if isNested && (metMode.Typ == tblutils.MetricModeTruncate || metMode.Typ == tblutils.MetricModeFull) {
		metMode = tblutils.MetricsMode{Typ: tblutils.MetricModeCounts}
	}

	return tblutils.StatisticsCollector{
		FieldID:    a.fieldID,
		IcebergTyp: dt,
		ColName:    colName,
		Mode:       metMode,
	}, true
}

func (a *arrowStatsCollector) variantCollector() (tblutils.StatisticsCollector, bool) {
	colName, ok := a.schema.FindColumnName(a.fieldID)
	if !ok {
		return tblutils.StatisticsCollector{}, false
	}

	return tblutils.StatisticsCollector{
		FieldID: a.fieldID,
		ColName: colName,
		Mode:    a.resolveColumnMetricsMode(colName),
	}, true
}

func statsPlanFieldCount(field iceberg.NestedField) int {
	switch typ := field.Type.(type) {
	case *iceberg.StructType:
		count := 0
		for _, nestedField := range typ.FieldList {
			count += statsPlanFieldCount(nestedField)
		}

		return count
	case *iceberg.ListType:
		return statsPlanFieldCount(typ.ElementField())
	case *iceberg.MapType:
		return statsPlanFieldCount(typ.KeyField()) + statsPlanFieldCount(typ.ValueField())
	default:
		return 1
	}
}

func collectStatsPlanField(visitor *arrowStatsCollector, result map[int]tblutils.StatisticsCollector, field iceberg.NestedField, isNested bool) {
	switch typ := field.Type.(type) {
	case *iceberg.StructType:
		for _, nestedField := range typ.FieldList {
			collectStatsPlanField(visitor, result, nestedField, true)
		}
	case *iceberg.ListType:
		collectStatsPlanField(visitor, result, typ.ElementField(), true)
	case *iceberg.MapType:
		collectStatsPlanField(visitor, result, typ.KeyField(), true)
		collectStatsPlanField(visitor, result, typ.ValueField(), true)
	case iceberg.VariantType:
		visitor.fieldID = field.ID
		if collector, ok := visitor.variantCollector(); ok {
			result[collector.FieldID] = collector
		}
	default:
		visitor.fieldID = field.ID
		collector, ok := visitor.primitiveCollector(field.Type.(iceberg.PrimitiveType), isNested)
		if ok {
			result[collector.FieldID] = collector
		}
	}
}

func computeStatsPlan(sc *iceberg.Schema, props iceberg.Properties) (result map[int]tblutils.StatisticsCollector, err error) {
	defer func() {
		if r := recover(); r != nil {
			result = nil
			switch e := r.(type) {
			case string:
				err = fmt.Errorf("%w: %s", iceberg.ErrInvalidSchema, e)
			case error:
				err = fmt.Errorf("error encountered during schema visitor: %w", e)
			}
		}
	}()

	if sc == nil {
		return nil, fmt.Errorf("%w: cannot visit nil schema", iceberg.ErrInvalidArgument)
	}

	defaultMode, defaultModeErr := tblutils.MatchMetricsMode(
		props.Get(DefaultWriteMetricsModeKey, DefaultWriteMetricsModeDefault))
	overrideCount := 0
	for key := range props {
		if strings.HasPrefix(key, MetricsModeColumnConfPrefix+".") {
			overrideCount++
		}
	}

	var columnModes map[string]tblutils.MetricsMode
	var columnModeErrors map[string]error
	for key, rawMode := range props {
		colName, ok := strings.CutPrefix(key, MetricsModeColumnConfPrefix+".")
		if !ok {
			continue
		}

		mode, err := tblutils.MatchMetricsMode(rawMode)
		if err != nil {
			if columnModeErrors == nil {
				columnModeErrors = make(map[string]error, overrideCount)
			}

			columnModeErrors[colName] = err

			continue
		}
		if columnModes == nil {
			columnModes = make(map[string]tblutils.MetricsMode, overrideCount)
		}
		columnModes[colName] = mode
	}

	visitor := &arrowStatsCollector{
		schema:           sc,
		defaultMode:      defaultMode,
		defaultModeError: defaultModeErr,
		columnModes:      columnModes,
		columnModeErrors: columnModeErrors,
	}

	fields := sc.FieldsRef(internal.SchemaRef{})
	resultCount := 0
	for _, field := range fields {
		resultCount += statsPlanFieldCount(field)
	}

	result = make(map[int]tblutils.StatisticsCollector, resultCount)
	for _, field := range fields {
		collectStatsPlanField(visitor, result, field, strings.Contains(field.Name, "."))
	}

	return result, nil
}

func filesToDataFiles(ctx context.Context, fileIO iceio.IO, meta *MetadataBuilder, filePaths []string, concurrency int) (_ []iceberg.DataFile, err error) {
	partitionSpec, err := meta.CurrentSpec()
	if err != nil || partitionSpec == nil {
		return nil, fmt.Errorf("%w: cannot add files without a current spec", err)
	}

	if err := checkNoUnknownTransform(partitionSpec); err != nil {
		return nil, err
	}

	currentSchema, currentSpec := meta.CurrentSchema(), *partitionSpec

	dataFiles := make([]iceberg.DataFile, len(filePaths))
	eg, ctx := errgroup.WithContext(ctx)
	eg.SetLimit(concurrency)
	for i, filePath := range filePaths {
		eg.Go(func() (err error) {
			defer func() {
				if r := recover(); r != nil {
					switch e := r.(type) {
					case error:
						err = fmt.Errorf("error encountered during file conversion: %w", e)
					default:
						err = fmt.Errorf("error encountered during file conversion: %v", e)
					}
				}
			}()

			// Pass 0 so AddFiles does not claim sort_order_id on files
			// it did not write (#1184). Callers that know the file's
			// sort layout use FileToDataFile with an explicit id.
			dataFiles[i] = fileToDataFile(ctx, fileIO, filePath, currentSchema, currentSpec, 0, meta.props)

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	return dataFiles, nil
}

// FileToDataFile builds a DataFile for an existing parquet file at
// filePath, reading its footer to populate record count, file size,
// column sizes, value/null counts and lower/upper bounds, and to infer
// partition values for order-preserving transforms. sortOrderID is
// stamped onto the DataFile so callers converting foreign parquet
// footers can convey the file's sort layout (spec data-file field
// sort_order_id); pass 0 to make no claim. Panics from the unexported
// conversion are recovered into an error.
func FileToDataFile(ctx context.Context, fileIO iceio.IO, filePath string, currentSchema *iceberg.Schema, currentSpec iceberg.PartitionSpec, sortOrderID int, props iceberg.Properties) (df iceberg.DataFile, err error) {
	defer func() {
		if r := recover(); r != nil {
			switch e := r.(type) {
			case error:
				err = fmt.Errorf("error encountered during file conversion: %w", e)
			default:
				err = fmt.Errorf("error encountered during file conversion: %v", e)
			}
		}
	}()

	return fileToDataFile(ctx, fileIO, filePath, currentSchema, currentSpec, sortOrderID, props), nil
}

func fileToDataFile(ctx context.Context, fileIO iceio.IO, filePath string, currentSchema *iceberg.Schema, currentSpec iceberg.PartitionSpec, sortOrderID int, props iceberg.Properties) iceberg.DataFile {
	format := tblutils.FormatFromFileName(filePath)
	rdr := must(format.Open(ctx, fileIO, filePath))
	defer rdr.Close()

	arrSchema := must(rdr.Schema())
	if err := checkArrowSchemaCompatWithProperties(currentSchema, arrSchema, false, props); err != nil {
		panic(err)
	}

	pathToIDSchema := currentSchema
	if fileHasIDs := must(VisitArrowSchema(arrSchema, hasIDs{})); fileHasIDs {
		pathToIDSchema = must(ArrowSchemaToIcebergWithOptions(arrSchema, ArrowToIcebergOptions{
			TableSchema:     currentSchema,
			TableProperties: props,
		}))
	}
	statistics := format.DataFileStatsFromMeta(
		rdr.Metadata(),
		must(computeStatsPlan(currentSchema, props)),
		must(format.PathToIDMapping(pathToIDSchema)),
		tblutils.VariantFieldIDsFromSchema(currentSchema),
		arrSchema,
	)

	partitionValues := make(map[int]any)
	if !currentSpec.Equals(*iceberg.UnpartitionedSpec) {
		for _, field := range currentSpec.Fields() {
			if !field.Transform.PreservesOrder() {
				panic(fmt.Errorf("cannot infer partition value from parquet metadata for a non-linear partition field: %s with transform %s", field.Name, field.Transform))
			}

			partitionVal := statistics.PartitionValue(field, currentSchema)
			if partitionVal != nil {
				partitionValues[field.FieldID] = partitionVal
			}
		}
	}

	return statistics.ToDataFile(tblutils.DataFileOpts{
		Schema:          currentSchema,
		Spec:            currentSpec,
		Path:            filePath,
		Format:          iceberg.ParquetFile,
		Content:         iceberg.EntryContentData,
		FileSize:        rdr.SourceFileSize(),
		PartitionValues: partitionValues,
		SortOrderID:     sortOrderID,
	})
}

func recordNBytes(rec arrow.RecordBatch) (total int64) {
	for _, c := range rec.Columns() {
		total += int64(c.Data().SizeInBytes())
	}

	return total
}

func binPackRecords(itr iter.Seq2[arrow.RecordBatch, error], recordLookback int, targetFileSize int64) iter.Seq[[]arrow.RecordBatch] {
	return internal.PackingIterator(func(yield func(arrow.RecordBatch) bool) {
		for rec, err := range itr {
			if err != nil {
				panic(err)
			}

			rec.Retain()
			if !yield(rec) {
				return
			}
		}
	}, targetFileSize, recordLookback, recordNBytes, false)
}

type recordWritingArgs struct {
	sc              *arrow.Schema
	itr             iter.Seq2[arrow.RecordBatch, error]
	fs              iceio.WriteFileIO
	writeUUID       *uuid.UUID
	counter         iter.Seq[int]
	maxWriteWorkers int
	clustered       bool
	factoryOpts     []writerFactoryOption

	// recordBatchBufferSize overrides the rolling writers' record
	// channel capacity; non-positive uses the default (see
	// rollingDataWriterQueueCapacity).
	recordBatchBufferSize int
	// existingDVs maps a data file path to the positions already recorded in
	// its current deletion vector. On the v3 DV write path these are folded
	// into the newly written DV so a data file that already had a DV ends up
	// with a single merged vector (the caller removes the superseded one).
	existingDVs map[string]*dv.RoaringPositionBitmap
}

// checkNoUnknownTransform rejects writes against a spec containing an unknown
// partition transform. We can't compute partition values for it, so writing
// would produce null values under a field=<transform-name> dir that Java and
// PyIceberg can't read. The spec forbids committing against such a spec.
func checkNoUnknownTransform(spec *iceberg.PartitionSpec) error {
	if spec == nil {
		return nil
	}
	for _, f := range spec.Fields() {
		if _, ok := f.Transform.(iceberg.UnknownTransform); ok {
			return fmt.Errorf("%w: cannot write to a partition spec with unknown transform: %s", iceberg.ErrInvalidTransform, f.Transform)
		}
	}

	return nil
}

// checkNoUnknownTransformInSpecs applies checkNoUnknownTransform to every spec
// the given data files were written under. Delete writes target existing files,
// so the relevant spec is each file's own, which may be an older one.
func checkNoUnknownTransformInSpecs(meta Metadata, partitionContextByFilePath map[string]partitionContext) error {
	seen := make(map[int32]struct{}, len(partitionContextByFilePath))
	for _, pCtx := range partitionContextByFilePath {
		if _, ok := seen[pCtx.specID]; ok {
			continue
		}
		seen[pCtx.specID] = struct{}{}
		if err := checkNoUnknownTransform(meta.PartitionSpecByID(int(pCtx.specID))); err != nil {
			return err
		}
	}

	currentSpec := meta.PartitionSpec()

	return checkNoUnknownTransform(&currentSpec)
}

func recordsToDataFiles(ctx context.Context, rootLocation string, meta *MetadataBuilder, args recordWritingArgs) (ret iter.Seq2[iceberg.DataFile, error]) {
	spec, err := meta.CurrentSpec()
	if err == nil {
		err = checkNoUnknownTransform(spec)
	}
	if err != nil {
		return func(yield func(iceberg.DataFile, error) bool) {
			yield(nil, err)
		}
	}

	if args.counter == nil {
		args.counter = internal.Counter(0)
	}

	defer func() {
		if r := recover(); r != nil {
			var err error
			switch e := r.(type) {
			case error:
				err = fmt.Errorf("error encountered during file writing: %w", e)
			default:
				err = fmt.Errorf("error encountered during position delete file writing: %v", e)
			}
			ret = func(yield func(iceberg.DataFile, error) bool) {
				yield(nil, err)
			}
		}
	}()

	if args.writeUUID == nil {
		u := uuid.Must(uuid.NewRandom())
		args.writeUUID = &u
	}

	targetFileSize := int64(meta.props.GetInt(WriteTargetFileSizeBytesKey,
		WriteTargetFileSizeBytesDefault))

	nameMapping := meta.CurrentSchema().NameMapping()
	taskSchema, err := ArrowSchemaToIcebergWithOptions(args.sc, ArrowToIcebergOptions{
		NameMapping:     nameMapping,
		TableSchema:     meta.CurrentSchema(),
		TableProperties: meta.props,
	})
	if err != nil {
		return func(yield func(iceberg.DataFile, error) bool) {
			yield(nil, err)
		}
	}

	factory, err := newWriterFactory(rootLocation, args, meta, taskSchema, targetFileSize, args.factoryOpts...)
	if err != nil {
		panic(err)
	}

	if factory.currentSpec.IsUnpartitioned() {
		return unpartitionedWrite(ctx, factory, args.itr)
	}

	if args.clustered {
		return clusteredPartitionedWrite(ctx, factory.currentSpec, meta.CurrentSchema(), factory, args.itr)
	}

	partitionWriter := newPartitionedFanoutWriter(factory.currentSpec, meta.CurrentSchema(), args.itr, factory)
	workers := config.EnvConfig.MaxWorkers
	if args.maxWriteWorkers > 0 {
		workers = args.maxWriteWorkers
	}

	return partitionWriter.Write(ctx, workers)
}

func unpartitionedWrite(ctx context.Context, factory *writerFactory, records iter.Seq2[arrow.RecordBatch, error]) iter.Seq2[iceberg.DataFile, error] {
	outputCh := make(chan iceberg.DataFile, 1)
	errCh := make(chan error, 1)
	writerCtx, cancel := context.WithCancel(ctx)

	go func() {
		defer close(outputCh)
		defer factory.stopCount()

		writer := factory.newRollingDataWriter(writerCtx, "", nil, outputCh)
		for rec, err := range records {
			if err != nil {
				errCh <- err
				close(errCh)
				writer.abortAndWait()

				return
			}

			if err := writer.Add(rec); err != nil {
				errCh <- err
				close(errCh)
				writer.abortAndWait()

				return
			}
		}
		if err := writer.closeAndWait(); err != nil {
			errCh <- err
		}
		close(errCh)
	}()

	return func(yield func(iceberg.DataFile, error) bool) {
		defer func() {
			for range outputCh {
			}
		}()
		defer cancel()
		for df := range outputCh {
			if !yield(df, nil) {
				return
			}
		}
		if err := <-errCh; err != nil {
			yield(nil, err)
		}
	}
}

type partitionContext struct {
	partitionData map[int]any
	specID        int32
}

func positionDeleteRecordsToDataFiles(ctx context.Context, rootLocation string, meta *MetadataBuilder, partitionContextByFilePath map[string]partitionContext, args recordWritingArgs) (ret iter.Seq2[iceberg.DataFile, error]) {
	if args.counter == nil {
		args.counter = internal.Counter(0)
	}

	defer func() {
		if r := recover(); r != nil {
			var err error
			switch e := r.(type) {
			case error:
				err = fmt.Errorf("error encountered during position delete file writing: %w", e)
			default:
				err = fmt.Errorf("error encountered during position delete file writing: %v", e)
			}
			ret = func(yield func(iceberg.DataFile, error) bool) {
				yield(nil, err)
			}
		}
	}()

	latestMetadata, err := meta.Build()
	if err != nil {
		return func(yield func(iceberg.DataFile, error) bool) {
			yield(nil, err)
		}
	}

	// Check the specs the targeted data files were written under, not just the
	// current one: a table that has since evolved past the unknown transform
	// still reaches this path through its historic specs.
	if err := checkNoUnknownTransformInSpecs(latestMetadata, partitionContextByFilePath); err != nil {
		return func(yield func(iceberg.DataFile, error) bool) {
			yield(nil, err)
		}
	}

	// V3+ tables write deletion vectors via Puffin regardless of partitioning,
	// matching the Java reference implementation, which hardcodes DV for v3 and
	// does not expose a property to opt out. The DV path threads each data
	// file's spec id and partition record through DVWriter so partitioned
	// outputs carry the correct manifest-entry partition.
	if latestMetadata.Version() >= 3 {
		return positionDeleteRecordsToDataFilesDV(ctx, rootLocation, args, latestMetadata, partitionContextByFilePath)
	}

	if args.writeUUID == nil {
		u := uuid.Must(uuid.NewRandom())
		args.writeUUID = &u
	}

	targetFileSize := int64(meta.props.GetInt(WriteTargetFileSizeBytesKey,
		WriteTargetFileSizeBytesDefault))

	if latestMetadata.PartitionSpec().IsUnpartitioned() {
		cw := newConcurrentDataFileWriter(func(rootLocation string, fs iceio.WriteFileIO, meta *MetadataBuilder, props iceberg.Properties, opts ...dataFileWriterOption) (dataFileWriter, error) {
			return newPositionDeleteWriter(rootLocation, fs, meta, props, opts...)
		}, withSchemaSanitization(false))
		nextCount, stopCount := iter.Pull(args.counter)
		tasks := func(yield func(WriteTask) bool) {
			defer stopCount()

			fileCount := 0
			for batch := range binPackRecords(args.itr, defaultBinPackLookback, targetFileSize) {
				cnt, _ := nextCount()
				fileCount++
				t := WriteTask{
					Uuid:        *args.writeUUID,
					ID:          cnt,
					PartitionID: iceberg.UnpartitionedSpec.ID(),
					FileCount:   fileCount,
					Schema:      iceberg.PositionalDeleteSchema,
					Batches:     batch,
				}
				if !yield(t) {
					return
				}
			}
		}

		return cw.writeFiles(ctx, rootLocation, args.fs, meta, meta.props, nil, tasks)
	}
	factory, err := newWriterFactory(rootLocation, args, meta, iceberg.PositionalDeleteSchema, targetFileSize,
		withContentType(iceberg.EntryContentPosDeletes),
		withFactoryFileSchema(iceberg.PositionalDeleteSchema))
	if err != nil {
		panic(err)
	}

	partitionWriter := newPositionDeletePartitionedFanoutWriter(latestMetadata, partitionContextByFilePath, args.itr, factory)
	workers := config.EnvConfig.MaxWorkers

	return partitionWriter.Write(ctx, workers)
}

// positionDeleteRecordsToDataFilesDV produces deletion-vector Puffin output
// for v3+ tables. Each row's data file path must have an entry in
// partitionContextByFilePath: the (specID, partitionData) is captured on
// DVWriter.Load/AddPosition and lands on the output DataFile manifest entry, mirroring
// Java's BaseDVFileWriter, which keys deletes by (path, spec, partition).
// A missing entry is a programming error (the caller failed to pre-register
// every data file targeted by the delete) and surfaces as an error rather
// than silently producing a default-partition entry.
func positionDeleteRecordsToDataFilesDV(ctx context.Context, rootLocation string, args recordWritingArgs, metadata Metadata, partitionContextByFilePath map[string]partitionContext) iter.Seq2[iceberg.DataFile, error] {
	return func(yield func(iceberg.DataFile, error) bool) {
		writer := dv.NewDVWriter(args.fs, func(id int32) *iceberg.PartitionSpec {
			return metadata.PartitionSpecByID(int(id))
		})

		// Seed the writer with any deletion vector the referenced data file
		// already carries so its previously deleted positions survive into the
		// merged DV. The scan that produced args.itr does NOT apply the existing
		// DVs (makePositionDeleteRecordsForFilter builds its FileScanTasks
		// without DeleteFiles), so the incoming batches may re-report positions
		// already in the old DV; re-Setting them on top of the seed is a no-op
		// because the bitmap is idempotent. The seed is what guarantees the old
		// positions survive — do not remove it, or a second delete would drop
		// the earlier deletes and resurrect those rows. The spec allows at most
		// one DV per data file, so the caller supersedes the old DV once this
		// merged one is written.
		hasEntries := false
		for filePath, bitmap := range args.existingDVs {
			pCtx, ok := partitionContextByFilePath[filePath]
			if !ok {
				yield(nil, fmt.Errorf("unexpected missing partition context for existing deletion vector path %s", filePath))

				return
			}
			writer.Load(filePath, bitmap, pCtx.specID, pCtx.partitionData)
			hasEntries = true
		}

		for batch, err := range args.itr {
			if err != nil {
				yield(nil, err)

				return
			}

			filePaths := batch.Column(0).(*array.String)
			positions := batch.Column(1).(*array.Int64)

			for i := range batch.NumRows() {
				filePath := filePaths.Value(int(i))
				pCtx, ok := partitionContextByFilePath[filePath]
				if !ok {
					yield(nil, fmt.Errorf("unexpected missing partition context for path %s", filePath))

					return
				}
				if err := writer.AddPosition(filePath, positions.Value(int(i)), pCtx.specID, pCtx.partitionData); err != nil {
					yield(nil, err)

					return
				}
				hasEntries = true
			}
		}

		if !hasEntries {
			return
		}

		if args.writeUUID == nil {
			u := uuid.Must(uuid.NewRandom())
			args.writeUUID = &u
		}
		location := rootLocation + "/data/" + fmt.Sprintf("00000-0-%s-deletes.puffin", *args.writeUUID)

		dataFiles, err := writer.Flush(ctx, location)
		if err != nil {
			yield(nil, err)

			return
		}

		for _, df := range dataFiles {
			if !yield(df, nil) {
				return
			}
		}
	}
}

func checkCRSString(rawCrs json.RawMessage) bool {
	b := bytes.TrimSpace(rawCrs)

	return len(b) > 0 && b[0] == '"'
}

func checkCRSJSON(rawCrs json.RawMessage) bool {
	b := bytes.TrimSpace(rawCrs)

	return len(b) > 0 && b[0] == '{'
}

// errWKT2CRSNotSupported is returned for WKT2:2019 CRS definitions, which this
// package does not support yet.
var errWKT2CRSNotSupported = errors.New("CRS type wkt2:2019 not supported")

// errPROJJSONStringCRSNotSupported is returned for projjson-typed string CRS
// values, which this package does not support yet.
var errPROJJSONStringCRSNotSupported = errors.New("CRS type projjson not supported for string CRS")

// isWKT2CRSString reports whether crs looks like a WKT2 (ISO 19162) CRS
// definition. It is a best-effort heuristic used only to surface the clearer
// errWKT2CRSNotSupported message. The list covers the WKT2:2019 CRS keywords
// and their long forms.
func isWKT2CRSString(crs string) bool {
	upper := strings.ToUpper(strings.TrimSpace(crs))

	for _, prefix := range []string{
		"BOUNDCRS[",
		"COMPOUNDCRS[",
		"COORDINATEMETADATA[",
		"COORDINATEOPERATION[",
		"CRS[",
		"DERIVEDPROJCRS[",
		"ENGCRS[",
		"ENGINEERINGCRS[",
		"GEODCRS[",
		"GEODETICCRS[",
		"GEOGCRS[",
		"GEOGRAPHICCRS[",
		"PARAMETRICCRS[",
		"PROJCRS[",
		"PROJECTEDCRS[",
		"TIMECRS[",
		"VERTCRS[",
		"VERTICALCRS[",
	} {
		if strings.HasPrefix(upper, prefix) {
			return true
		}
	}

	return false
}

// geoArrowCRSToIcebergCRS maps GeoArrow CRS metadata to an Iceberg CRS string.
// Absent CRS metadata means the default CRS OGC:CRS84, matching the Parquet
// geospatial spec and Iceberg's default geometry/geography types.
func geoArrowCRSToIcebergCRS(meta geoarrow.Metadata) (string, error) {
	if len(meta.CRS) == 0 {
		return iceberg.DefaultGeoCRS, nil
	}

	switch {
	case checkCRSString(meta.CRS):
		var crs string

		if err := json.Unmarshal(meta.CRS, &crs); err != nil {
			return "", fmt.Errorf("invalid geoarrow CRS metadata: %w", err)
		}
		if crs == "" {
			return "", errors.New("unsupported CRS: empty string CRS")
		}

		if strings.EqualFold(crs, iceberg.DefaultGeoCRS) || strings.EqualFold(crs, "EPSG:4326") {
			return iceberg.DefaultGeoCRS, nil
		}

		switch meta.CRSType {
		case geoarrow.CRSTypePROJJSON:
			return "", errPROJJSONStringCRSNotSupported
		case geoarrow.CRSTypeWKT22019:
			return "", errWKT2CRSNotSupported
		default:
			// Unset, AuthorityCode, SRID, and future CRSType values continue to
			// the string-shape checks below.
		}

		if isWKT2CRSString(crs) {
			return "", errWKT2CRSNotSupported
		}
		if meta.CRSType == geoarrow.CRSTypeSRID {
			// Some writers store the identifier already prefixed; don't prefix twice.
			if strings.HasPrefix(strings.ToLower(crs), "srid:") {
				return crs, nil
			}

			return "srid:" + crs, nil
		}

		// Bare or unrecognized string CRS values are opaque identifiers per the
		// Iceberg type contract; preserve them for read/write compatibility.
		return crs, nil
	case checkCRSJSON(meta.CRS):
		if meta.CRSType == geoarrow.CRSTypePROJJSON {
			return "", errors.New("unsupported CRS: projjson metadata requires table schema and properties to recover the Iceberg CRS reference")
		}

		var crs map[string]json.RawMessage

		if err := json.Unmarshal(meta.CRS, &crs); err != nil {
			return "", fmt.Errorf("invalid geoarrow CRS metadata: %w", err)
		}

		idRaw, ok := crs["id"]
		if !ok || len(idRaw) == 0 {
			return "", errors.New("unsupported CRS: missing id")
		}

		var id map[string]json.RawMessage
		if err := json.Unmarshal(idRaw, &id); err != nil {
			return "", fmt.Errorf("unsupported CRS: invalid id: %w", err)
		}

		var authority string
		authorityRaw, ok := id["authority"]
		if !ok || len(authorityRaw) == 0 {
			return "", errors.New("unsupported CRS: missing id.authority")
		}
		if err := json.Unmarshal(authorityRaw, &authority); err != nil {
			return "", fmt.Errorf("unsupported CRS: invalid id.authority: %w", err)
		}
		if authority == "" {
			return "", errors.New("unsupported CRS: empty id.authority")
		}

		codeRaw := id["code"]
		if len(codeRaw) == 0 {
			return "", errors.New("unsupported CRS: missing id.code")
		}

		var code string
		if err := json.Unmarshal(codeRaw, &code); err != nil {
			var codeNum json.Number
			if err := json.Unmarshal(codeRaw, &codeNum); err != nil {
				return "", fmt.Errorf("unsupported CRS: invalid id.code: %w", err)
			}
			code = codeNum.String()
		}
		if code == "" {
			return "", errors.New("unsupported CRS: empty id.code")
		}

		authorityCode := authority + ":" + code
		if strings.EqualFold(authorityCode, iceberg.DefaultGeoCRS) || strings.EqualFold(authorityCode, "EPSG:4326") {
			return iceberg.DefaultGeoCRS, nil
		}

		return authorityCode, nil
	default:
		return "", errors.New("unsupported CRS: CRS must either be omitted, a string, or a JSON object")
	}
}

func (c convertToIceberg) geoArrowMetadataToIcebergType(meta geoarrow.Metadata) (iceberg.Type, error) {
	crs, err := c.geoArrowCRSToIcebergCRS(meta)
	if err != nil {
		return nil, err
	}

	return geoArrowMetadataToIcebergTypeWithCRS(meta, crs)
}

func (c convertToIceberg) geoArrowCRSToIcebergCRS(meta geoarrow.Metadata) (string, error) {
	if meta.CRSType != geoarrow.CRSTypePROJJSON || !checkCRSJSON(meta.CRS) {
		return geoArrowCRSToIcebergCRS(meta)
	}

	fieldID, ok := c.currentFieldID()
	if !ok {
		return "", errors.New("unsupported CRS: projjson metadata requires field IDs to recover the Iceberg CRS reference")
	}

	return projJSONCRSFromTable(fieldID, meta.CRS, c.tableSchema, c.tableProperties)
}

func geoArrowMetadataToIcebergType(meta geoarrow.Metadata) (iceberg.Type, error) {
	crs, err := geoArrowCRSToIcebergCRS(meta)
	if err != nil {
		return nil, err
	}

	return geoArrowMetadataToIcebergTypeWithCRS(meta, crs)
}

func geoArrowMetadataToIcebergTypeWithCRS(meta geoarrow.Metadata, crs string) (iceberg.Type, error) {
	// Missing GeoArrow edges use the planar default, matching arrow-rs
	// geoarrow.wkb reads. PyIceberg-authored geography files may omit edges,
	// which is an interop gap: Arrow metadata alone cannot distinguish those
	// geography columns from Geometry here.
	switch meta.Edges {
	case geoarrow.EdgePlanar:
		return iceberg.GeometryTypeOf(crs)
	case geoarrow.EdgeVincenty, geoarrow.EdgeKarney, geoarrow.EdgeThomas, geoarrow.EdgeAndoyer, geoarrow.EdgeSpherical:
		return iceberg.GeographyTypeOf(crs, string(meta.Edges))
	default:
		return nil, fmt.Errorf("unsupported geoarrow edges %q", meta.Edges)
	}
}

var authorityCodeCRS = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*:[A-Za-z0-9_.-]+$`)

// icebergCRSToGeoArrowMetadata converts an Iceberg CRS string to GeoArrow
// metadata. props is used only for projjson:<property> references, where the
// property value supplies the PROJJSON document embedded into GeoArrow metadata.
func icebergCRSToGeoArrowMetadata(crs string, props iceberg.Properties) (geoarrow.Metadata, error) {
	lowerCRS := strings.ToLower(crs)
	if strings.HasPrefix(lowerCRS, "srid:") {
		id := crs[len("srid:"):]

		raw, _ := json.Marshal(id) //nolint:errcheck // Marshalling a string can't fail

		return geoarrow.Metadata{
			CRS:     raw,
			CRSType: geoarrow.CRSTypeSRID,
		}, nil
	}

	if strings.HasPrefix(lowerCRS, "projjson:") {
		// The scheme is case-insensitive, but the table property key is not.
		propKey := strings.TrimSpace(crs[len("projjson:"):])
		if propKey == "" {
			return geoarrow.Metadata{}, fmt.Errorf("%w: projjson CRS missing table property key", iceberg.ErrInvalidSchema)
		}
		if props == nil {
			return geoarrow.Metadata{}, fmt.Errorf("%w: projjson CRS %q could not be resolved from table properties", iceberg.ErrInvalidSchema, crs)
		}

		projjson, ok := props[propKey]
		if !ok || strings.TrimSpace(projjson) == "" {
			return geoarrow.Metadata{}, fmt.Errorf("%w: projjson CRS %q references missing table property %q", iceberg.ErrInvalidSchema, crs, propKey)
		}

		raw := json.RawMessage(projjson)
		if !json.Valid(raw) {
			return geoarrow.Metadata{}, fmt.Errorf("%w: projjson CRS table property %q is not valid JSON", iceberg.ErrInvalidSchema, propKey)
		}

		return geoarrow.Metadata{
			CRS:     raw,
			CRSType: geoarrow.CRSTypePROJJSON,
		}, nil
	}

	var raw []byte

	if lowerCRS == "epsg:4326" {
		// collapse EPSG:4326 to OGC:CRS84
		raw, _ = json.Marshal(iceberg.DefaultGeoCRS) //nolint:errcheck // Marshalling a string can't fail
	} else {
		raw, _ = json.Marshal(crs) //nolint:errcheck // Marshalling a string can't fail
	}

	meta := geoarrow.Metadata{CRS: raw}
	if authorityCodeCRS.MatchString(crs) {
		meta.CRSType = geoarrow.CRSTypeAuthorityCode
	}

	return meta, nil
}

func projJSONCRSFromTable(fieldID int, rawCRS json.RawMessage, tableSchema *iceberg.Schema, props iceberg.Properties) (string, error) {
	if tableSchema == nil || props == nil {
		return "", errors.New("unsupported CRS: projjson metadata requires table schema and properties to recover the Iceberg CRS reference")
	}

	field, ok := tableSchema.FindFieldByID(fieldID)
	if !ok {
		return "", fmt.Errorf("unsupported CRS: projjson field id %d not found in table schema", fieldID)
	}

	var crs string
	switch typ := field.Type.(type) {
	case iceberg.GeometryType:
		crs = typ.CRS()
	case iceberg.GeographyType:
		crs = typ.CRS()
	default:
		return "", fmt.Errorf("unsupported CRS: table field id %d is %s, not a geo type", fieldID, field.Type)
	}

	if !strings.HasPrefix(strings.ToLower(crs), "projjson:") {
		return "", fmt.Errorf("unsupported CRS: table field id %d has CRS %q, not a projjson reference", fieldID, crs)
	}

	propKey := strings.TrimSpace(crs[len("projjson:"):])
	if propKey == "" {
		return "", fmt.Errorf("%w: projjson CRS missing table property key", iceberg.ErrInvalidSchema)
	}

	projjson, ok := props[propKey]
	if !ok || strings.TrimSpace(projjson) == "" {
		return "", fmt.Errorf("%w: projjson CRS %q references missing table property %q", iceberg.ErrInvalidSchema, crs, propKey)
	}

	if !json.Valid([]byte(projjson)) {
		return "", fmt.Errorf("%w: projjson CRS table property %q is not valid JSON", iceberg.ErrInvalidSchema, propKey)
	}

	if !jsonEqual(rawCRS, []byte(projjson)) {
		return "", fmt.Errorf("unsupported CRS: projjson metadata for field id %d does not match table property %q", fieldID, propKey)
	}

	return crs, nil
}

func jsonEqual(lhs, rhs []byte) bool {
	var lhsCompact, rhsCompact bytes.Buffer
	if err := json.Compact(&lhsCompact, lhs); err != nil {
		return false
	}
	if err := json.Compact(&rhsCompact, rhs); err != nil {
		return false
	}

	return bytes.Equal(lhsCompact.Bytes(), rhsCompact.Bytes())
}
