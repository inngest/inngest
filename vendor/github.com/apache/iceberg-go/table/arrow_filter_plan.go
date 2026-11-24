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
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/apache/arrow-go/v18/arrow/compute/exprs"
	"github.com/apache/arrow-go/v18/parquet/metadata"
	"github.com/apache/iceberg-go"
	iceinternal "github.com/apache/iceberg-go/internal"
	tblutils "github.com/apache/iceberg-go/table/internal"
	"github.com/apache/iceberg-go/table/substrait"
	"github.com/substrait-io/substrait-go/v8/expr"
)

// compiledFileFilterPlan contains immutable work derived from one Iceberg
// filter and one physical file schema. Row-group stats evaluators are created
// from statsFilter for each file because inclusiveMetricsEval stores mutable
// per-row-group maps.
type compiledFileFilterPlan struct {
	statsFilter     iceberg.BooleanExpression
	bloomPreds      []tblutils.RowGroupBloomPred
	dictionaryPreds []tblutils.RowGroupDictionaryPred

	recordFilter      expr.Expression
	extensionRegistry *expr.ExtensionRegistry
	dropFile          bool

	// extracts are variant sub-path terms materialized as synthetic derived
	// columns for the record filter; applied per-batch by extractResidualFilter.
	extracts []iceberg.VariantExtractColumn
}

func (p *compiledFileFilterPlan) recordProcessor(ctx context.Context) recProcessFn {
	if p == nil || p.recordFilter == nil {
		return nil
	}

	ctx = exprs.WithExtensionIDSet(ctx, exprs.NewExtensionSetDefault(*p.extensionRegistry))

	return filterRecords(ctx, p.recordFilter)
}

func (p *compiledFileFilterPlan) statsEvaluator() func(*metadata.RowGroupMetaData, []int) (bool, error) {
	// Passing an AlwaysTrue evaluator still allocates fresh metric maps for
	// every row group. A nil evaluator has the same keep-all semantics and lets
	// the Parquet reader take its no-pruning path.
	if p == nil || p.statsFilter == nil || p.statsFilter.Equals(iceberg.AlwaysTrue{}) {
		return nil
	}

	return newParquetRowGroupStatsEvaluatorFromRewritten(p.statsFilter, false)
}

type compiledFileFilterPlans struct {
	record  *compiledFileFilterPlan
	pruning *compiledFileFilterPlan
}

type compiledFileFilterPlanCache struct {
	mu    sync.RWMutex
	plans map[string]*compiledFileFilterPlans
}

// cachedFileFilterPlans returns plans for the scan's fixed filter(s). The
// schema string is a structural key: the schema ID is intentionally omitted
// because files from different schema versions can share the same physical
// layout and use the same compiled plan. Pruning plans are only built when the
// caller is reading a format that supports Parquet row-group pruning.
func (as *arrowScan) cachedFileFilterPlans(fileSchema *iceberg.Schema, includePruning bool) (*compiledFileFilterPlans, error) {
	if fileSchema == nil {
		return nil, fmt.Errorf("%w: cannot compile a filter plan for a nil file schema", iceberg.ErrInvalidArgument)
	}

	key, err := physicalSchemaKey(fileSchema)
	if err != nil {
		return nil, err
	}

	as.filterPlanCache.mu.RLock()
	plans, ok := as.filterPlanCache.plans[key]
	if ok && (!includePruning || plans.pruning != nil) {
		as.filterPlanCache.mu.RUnlock()

		return plans, nil
	}
	as.filterPlanCache.mu.RUnlock()

	as.filterPlanCache.mu.Lock()
	defer as.filterPlanCache.mu.Unlock()

	plans, ok = as.filterPlanCache.plans[key]
	if as.filterPlanCache.plans == nil {
		as.filterPlanCache.plans = make(map[string]*compiledFileFilterPlans)
	}

	if !ok {
		sharePruning := includePruning && as.rowGroupFilter == nil
		record, err := compileFileFilterPlan(fileSchema, as.boundRowFilter, as.caseSensitive, true, sharePruning)
		if err != nil {
			return nil, err
		}
		plans = &compiledFileFilterPlans{record: record}
		if sharePruning {
			plans.pruning = record
		}
		as.filterPlanCache.plans[key] = plans
	}
	if !includePruning || plans.pruning != nil {
		return plans, nil
	}

	pruningFilter := as.rowGroupFilter
	if pruningFilter == nil {
		pruningFilter = as.boundRowFilter
	}

	logicalSchema := as.filterSchema
	if logicalSchema == nil {
		logicalSchema = as.projectedSchema
	}
	hasMissingDefault, err := pruningFilterHasMissingInitialDefault(
		pruningFilter, logicalSchema, fileSchema)
	if err != nil {
		return nil, err
	}
	if hasMissingDefault {
		// TranslateColumnNames treats fields missing from the physical file as
		// null. That is unsafe for an initial default, so keep pruning disabled
		// just as the uncached path does.
		pruningFilter = iceberg.AlwaysTrue{}
	}

	pruning, err := compileFileFilterPlan(fileSchema, pruningFilter, as.caseSensitive, false, true)
	if err != nil {
		return nil, err
	}

	plans.pruning = pruning

	return plans, nil
}

// physicalSchemaKeyPanic marks invalid input, not unexpected implementation panics.
type physicalSchemaKeyPanic string

func physicalSchemaKey(fileSchema *iceberg.Schema) (key string, err error) {
	defer func() {
		if r := recover(); r != nil {
			invalid, ok := r.(physicalSchemaKeyPanic)
			if !ok {
				panic(r)
			}
			err = fmt.Errorf("%w: cannot encode physical schema key: %s", iceberg.ErrInvalidSchema, invalid)
		}
	}()

	if fileSchema == nil {
		panic(physicalSchemaKeyPanic("nil schema"))
	}

	// The key includes field names, IDs, requiredness, and type details. Schema
	// IDs, docs, and defaults do not affect filter translation or compilation.
	var builder strings.Builder
	for _, field := range fileSchema.FieldsRef(iceinternal.SchemaRef{}) {
		writePhysicalFieldKey(&builder, field)
	}

	return builder.String(), nil
}

func writePhysicalFieldKey(builder *strings.Builder, field iceberg.NestedField) {
	builder.WriteByte('f')
	writePhysicalInt(builder, field.ID)
	writePhysicalString(builder, field.Name)
	if field.Required {
		builder.WriteByte('1')
	} else {
		builder.WriteByte('0')
	}
	writePhysicalTypeKey(builder, field.Type)
}

func writePhysicalTypeKey(builder *strings.Builder, typ iceberg.Type) {
	// Type tags are internal to the cache key. Parameterized types append their
	// parameters so equal physical layouts still produce equal keys.
	switch t := typ.(type) {
	case *iceberg.StructType:
		if t == nil {
			panic(physicalSchemaKeyPanic("nil struct type"))
		}
		builder.WriteByte('s')
		builder.WriteByte('{')
		for _, field := range t.FieldList {
			writePhysicalFieldKey(builder, field)
		}
		builder.WriteByte('}')
	case *iceberg.ListType:
		if t == nil {
			panic(physicalSchemaKeyPanic("nil list type"))
		}
		builder.WriteByte('l')
		writePhysicalInt(builder, t.ElementID)
		if t.ElementRequired {
			builder.WriteByte('1')
		} else {
			builder.WriteByte('0')
		}
		writePhysicalTypeKey(builder, t.Element)
	case *iceberg.MapType:
		if t == nil {
			panic(physicalSchemaKeyPanic("nil map type"))
		}
		builder.WriteByte('m')
		writePhysicalInt(builder, t.KeyID)
		writePhysicalTypeKey(builder, t.KeyType)
		writePhysicalInt(builder, t.ValueID)
		if t.ValueRequired {
			builder.WriteByte('1')
		} else {
			builder.WriteByte('0')
		}
		writePhysicalTypeKey(builder, t.ValueType)
	case iceberg.BooleanType:
		builder.WriteByte('b')
	case iceberg.Int32Type:
		builder.WriteByte('i')
	case iceberg.Int64Type:
		builder.WriteByte('j')
	case iceberg.Float32Type:
		builder.WriteByte('k')
	case iceberg.Float64Type:
		builder.WriteByte('d')
	case iceberg.DateType:
		builder.WriteByte('D')
	case iceberg.TimeType:
		builder.WriteByte('T')
	case iceberg.TimestampType:
		builder.WriteByte('t')
	case iceberg.TimestampTzType:
		builder.WriteByte('z')
	case iceberg.StringType:
		builder.WriteByte('S')
	case iceberg.UUIDType:
		builder.WriteByte('u')
	case iceberg.BinaryType:
		builder.WriteByte('B')
	case iceberg.TimestampNsType:
		builder.WriteByte('n')
	case iceberg.TimestampTzNsType:
		builder.WriteByte('N')
	case iceberg.UnknownType:
		builder.WriteByte('U')
	case iceberg.VariantType:
		builder.WriteByte('v')
	case iceberg.FixedType:
		builder.WriteByte('F')
		writePhysicalInt(builder, t.Len())
	case iceberg.DecimalType:
		builder.WriteByte('q')
		writePhysicalInt(builder, t.Precision())
		writePhysicalInt(builder, t.Scale())
	case iceberg.PrimitiveType:
		if value := reflect.ValueOf(t); value.Kind() == reflect.Pointer && value.IsNil() {
			panic(physicalSchemaKeyPanic("nil primitive type"))
		}
		// Keep custom and parameterized primitive types (for example geometry)
		// compatible with the previous structural key encoding.
		builder.WriteByte('p')
		writePhysicalString(builder, typ.String())
	default:
		panic(physicalSchemaKeyPanic(fmt.Sprintf("unsupported physical type: %T", typ)))
	}
}

func writePhysicalInt(builder *strings.Builder, value int) {
	var buf [20]byte
	builder.Write(strconv.AppendInt(buf[:0], int64(value), 10))
	builder.WriteByte(':')
}

func writePhysicalString(builder *strings.Builder, value string) {
	writePhysicalInt(builder, len(value))
	builder.WriteString(value)
	builder.WriteByte(':')
}

func compileFileFilterPlan(
	fileSchema *iceberg.Schema,
	rowFilter iceberg.BooleanExpression,
	caseSensitive, includeRecordFilter, includePruning bool,
) (*compiledFileFilterPlan, error) {
	if rowFilter == nil {
		rowFilter = iceberg.AlwaysTrue{}
	}

	translatedRecord, extracts, err := iceberg.TranslateColumnNamesForScan(rowFilter, fileSchema)
	if err != nil {
		return nil, err
	}

	// Variant extract terms become synthetic derived columns bound against an
	// augmented schema for the record filter and residual.
	recordSchema := fileSchema
	if len(extracts) > 0 {
		recordSchema = augmentSchemaWithExtracts(fileSchema, extracts)
	}

	boundFilter := translatedRecord
	if !translatedRecord.Equals(iceberg.AlwaysFalse{}) {
		boundFilter, err = iceberg.BindExpr(recordSchema, translatedRecord, caseSensitive)
		if err != nil {
			return nil, err
		}
	}

	plan := &compiledFileFilterPlan{extracts: extracts}
	if includePruning {
		// Variant extract terms have no row-group statistics. The spec prunes them
		// at the file level via variant bounds (format/spec.md "Bounds for Variant"),
		// so they are excluded from the row-group stats/dictionary/bloom filter here.
		pruneFilter := boundFilter
		if len(extracts) > 0 {
			stripped, err := stripExtractPredicates(rowFilter)
			if err != nil {
				return nil, err
			}
			pruneFilter, err = iceberg.TranslateColumnNames(stripped, fileSchema)
			if err != nil {
				return nil, err
			}
			if !pruneFilter.Equals(iceberg.AlwaysFalse{}) && !pruneFilter.Equals(iceberg.AlwaysTrue{}) {
				pruneFilter, err = iceberg.BindExpr(fileSchema, pruneFilter, caseSensitive)
				if err != nil {
					return nil, err
				}
			}
		}

		statsFilter, err := iceberg.RewriteNotExpr(pruneFilter)
		if err != nil {
			return nil, err
		}
		bloomPreds, err := newBloomFilterPredicatesFromRewritten(statsFilter)
		if err != nil {
			return nil, err
		}
		dictionaryPreds, err := newDictionaryPredicatesFromRewritten(statsFilter)
		if err != nil {
			return nil, err
		}

		plan.statsFilter = statsFilter
		plan.bloomPreds = bloomPreds
		plan.dictionaryPreds = dictionaryPreds
	}
	if !includeRecordFilter {
		return plan, nil
	}
	if boundFilter.Equals(iceberg.AlwaysFalse{}) {
		plan.dropFile = true

		return plan, nil
	}
	if boundFilter.Equals(iceberg.AlwaysTrue{}) {
		return plan, nil
	}

	extSet, recordFilter, err := substrait.ConvertExpr(recordSchema, boundFilter, caseSensitive)
	if err != nil {
		return nil, err
	}
	plan.recordFilter = recordFilter
	plan.extensionRegistry = extSet

	return plan, nil
}

// stripExtractPredicates replaces variant extract predicates, and any NOT over them, with AlwaysTrue for stats/bloom pruning.
func stripExtractPredicates(expr iceberg.BooleanExpression) (iceberg.BooleanExpression, error) {
	res, err := iceberg.VisitExpr(expr, extractStripper{})
	if err != nil {
		return nil, err
	}

	return res.expr, nil
}

// strippedResult is the rewritten expression plus whether its subtree referenced an extract.
type strippedResult struct {
	expr       iceberg.BooleanExpression
	hasExtract bool
}

type extractStripper struct{}

func (extractStripper) VisitTrue() strippedResult {
	return strippedResult{expr: iceberg.AlwaysTrue{}}
}

func (extractStripper) VisitFalse() strippedResult {
	return strippedResult{expr: iceberg.AlwaysFalse{}}
}

func (extractStripper) VisitNot(child strippedResult) strippedResult {
	if child.hasExtract {
		return strippedResult{expr: iceberg.AlwaysTrue{}, hasExtract: true}
	}

	return strippedResult{expr: iceberg.NewNot(child.expr)}
}

func (extractStripper) VisitAnd(left, right strippedResult) strippedResult {
	return strippedResult{expr: iceberg.NewAnd(left.expr, right.expr), hasExtract: left.hasExtract || right.hasExtract}
}

func (extractStripper) VisitOr(left, right strippedResult) strippedResult {
	return strippedResult{expr: iceberg.NewOr(left.expr, right.expr), hasExtract: left.hasExtract || right.hasExtract}
}

func (extractStripper) VisitUnbound(pred iceberg.UnboundPredicate) strippedResult {
	return strippedResult{expr: pred}
}

func (extractStripper) VisitBound(pred iceberg.BoundPredicate) strippedResult {
	if _, ok := pred.Term().(iceberg.BoundExtract); ok {
		return strippedResult{expr: iceberg.AlwaysTrue{}, hasExtract: true}
	}

	return strippedResult{expr: pred}
}
