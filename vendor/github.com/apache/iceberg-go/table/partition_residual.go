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
	"fmt"
	"math"

	"github.com/apache/iceberg-go"
)

// partitionResidualEvaluator partially evaluates a bound row filter using a
// file's partition values. It is built once per partition spec and can be
// reused for every data file in that spec.
type partitionResidualEvaluator struct {
	root          *partitionResidualNode
	partitionType *iceberg.StructType
}

type partitionResidualNodeKind uint8

const (
	partitionResidualOpaque partitionResidualNodeKind = iota
	partitionResidualTrue
	partitionResidualFalse
	partitionResidualPredicateKind
	partitionResidualNot
	partitionResidualAnd
	partitionResidualOr
)

type partitionResidualNode struct {
	kind      partitionResidualNodeKind
	expr      iceberg.BooleanExpression
	predicate *partitionResidualPredicate
	left      *partitionResidualNode
	right     *partitionResidualNode
	child     *partitionResidualNode
}

type partitionResidualPredicate struct {
	original    iceberg.BoundPredicate
	projections []partitionResidualProjection
}

type partitionResidualProjection struct {
	fieldID int

	// strictComplement evaluates the inclusive projection of the negated
	// predicate. If it is false, the original predicate is true for every
	// value in this partition.
	strictComplement func(_ iceberg.StructLike) (bool, error)
	inclusive        func(_ iceberg.StructLike) (bool, error)
}

type partitionResidualBuilder struct {
	spec            iceberg.PartitionSpec
	partitionSchema *iceberg.Schema
	caseSensitive   bool
	hasProjection   bool
	err             error
}

// newPartitionResidualEvaluator creates a Java-style residual evaluator for
// one partition spec. The filter must already be bound to the scan schema.
// Unsupported transforms and predicates simply keep the original predicate.
func newPartitionResidualEvaluator(
	schema *iceberg.Schema,
	spec *iceberg.PartitionSpec,
	filter iceberg.BooleanExpression,
	caseSensitive bool,
) (*partitionResidualEvaluator, error) {
	if schema == nil || spec == nil || filter == nil {
		return nil, nil
	}

	partitionType := spec.PartitionType(schema)
	builder := &partitionResidualBuilder{
		spec:            *spec,
		partitionSchema: iceberg.NewSchema(0, partitionType.FieldList...),
		caseSensitive:   caseSensitive,
	}
	root, err := iceberg.VisitExpr(filter, builder)
	if err != nil {
		return nil, err
	}
	if builder.err != nil {
		return nil, builder.err
	}
	if !builder.hasProjection {
		return nil, nil
	}

	return &partitionResidualEvaluator{
		root:          root,
		partitionType: partitionType,
	}, nil
}

func (b *partitionResidualBuilder) VisitTrue() *partitionResidualNode {
	return &partitionResidualNode{kind: partitionResidualTrue, expr: iceberg.AlwaysTrue{}}
}

func (b *partitionResidualBuilder) VisitFalse() *partitionResidualNode {
	return &partitionResidualNode{kind: partitionResidualFalse, expr: iceberg.AlwaysFalse{}}
}

func (b *partitionResidualBuilder) VisitNot(child *partitionResidualNode) *partitionResidualNode {
	return &partitionResidualNode{
		kind:  partitionResidualNot,
		expr:  iceberg.NewNot(child.expr),
		child: child,
	}
}

func (b *partitionResidualBuilder) VisitAnd(left, right *partitionResidualNode) *partitionResidualNode {
	return &partitionResidualNode{
		kind:  partitionResidualAnd,
		expr:  iceberg.NewAnd(left.expr, right.expr),
		left:  left,
		right: right,
	}
}

func (b *partitionResidualBuilder) VisitOr(left, right *partitionResidualNode) *partitionResidualNode {
	return &partitionResidualNode{
		kind:  partitionResidualOr,
		expr:  iceberg.NewOr(left.expr, right.expr),
		left:  left,
		right: right,
	}
}

func (b *partitionResidualBuilder) VisitUnbound(pred iceberg.UnboundPredicate) *partitionResidualNode {
	return &partitionResidualNode{kind: partitionResidualOpaque, expr: pred}
}

func (b *partitionResidualBuilder) VisitBound(pred iceberg.BoundPredicate) *partitionResidualNode {
	if literal, ok := pred.(iceberg.BoundLiteralPredicate); ok && partitionResidualValueIsNaN(literal.Literal()) {
		return &partitionResidualNode{kind: partitionResidualOpaque, expr: pred}
	}

	parts := b.spec.FieldsBySourceID(pred.Ref().Field().ID)
	if len(parts) == 0 {
		return &partitionResidualNode{kind: partitionResidualOpaque, expr: pred}
	}

	projections := make([]partitionResidualProjection, 0, len(parts))
	for _, part := range parts {
		projection, err := newPartitionResidualProjection(
			b.partitionSchema, part, pred, b.caseSensitive)
		if err != nil {
			if b.err == nil {
				b.err = fmt.Errorf("build partition residual for %s: %w", pred, err)
			}

			return &partitionResidualNode{kind: partitionResidualOpaque, expr: pred}
		}
		if projection.strictComplement == nil && projection.inclusive == nil {
			continue
		}

		projections = append(projections, projection)
	}
	if len(projections) == 0 {
		return &partitionResidualNode{kind: partitionResidualOpaque, expr: pred}
	}

	b.hasProjection = true

	return &partitionResidualNode{
		kind: partitionResidualPredicateKind,
		expr: pred,
		predicate: &partitionResidualPredicate{
			original:    pred,
			projections: projections,
		},
	}
}

func newPartitionResidualProjection(
	partitionSchema *iceberg.Schema,
	part iceberg.PartitionField,
	pred iceberg.BoundPredicate,
	caseSensitive bool,
) (partitionResidualProjection, error) {
	projection := partitionResidualProjection{fieldID: part.FieldID}

	negated, ok := pred.Negate().(iceberg.BoundPredicate)
	if ok {
		strictComplement, err := bindPartitionProjection(
			partitionSchema, part, negated, caseSensitive)
		if err != nil {
			return projection, err
		}
		projection.strictComplement = strictComplement
	}

	inclusive, err := bindPartitionProjection(partitionSchema, part, pred, caseSensitive)
	if err != nil {
		return projection, err
	}
	projection.inclusive = inclusive

	return projection, nil
}

func bindPartitionProjection(
	partitionSchema *iceberg.Schema,
	part iceberg.PartitionField,
	pred iceberg.BoundPredicate,
	caseSensitive bool,
) (func(_ iceberg.StructLike) (bool, error), error) {
	projected, err := part.Transform.Project(part.Name, pred)
	if err != nil || projected == nil {
		return nil, err
	}

	return iceberg.ExpressionEvaluator(partitionSchema, projected, caseSensitive)
}

// residual returns the portion of the original filter that still needs to be
// evaluated for a file. The changed result is false when no partition value
// simplified the filter, allowing callers to retain the nil-residual fallback.
func (p *partitionResidualEvaluator) residual(
	partition map[int]any,
) (iceberg.BooleanExpression, bool, error) {
	residual, changed, err := p.root.residual(
		borrowedPartitionRecord{partition: partition, partitionType: p.partitionType},
	)
	if !changed {
		return nil, false, err
	}

	return residual, true, err
}

func (n *partitionResidualNode) residual(
	partition borrowedPartitionRecord,
) (iceberg.BooleanExpression, bool, error) {
	switch n.kind {
	case partitionResidualOpaque:
		return n.expr, false, nil
	case partitionResidualTrue, partitionResidualFalse:
		return n.expr, false, nil
	case partitionResidualPredicateKind:
		for _, projection := range n.predicate.projections {
			value, known := partition.partition[projection.fieldID]
			if !known {
				continue
			}
			// Scalar comparisons order nulls and NaNs, while Arrow's row
			// filters propagate nulls and use IEEE floating-point comparisons.
			// Keep those predicates intact, including beneath NOT.
			op := n.predicate.original.Op()
			if op != iceberg.OpIsNull && op != iceberg.OpNotNull &&
				(value == nil || partitionResidualValueIsNaN(value)) {
				continue
			}

			if projection.strictComplement != nil {
				matches, err := projection.strictComplement(partition)
				if err != nil {
					return nil, false, err
				}
				if !matches {
					return iceberg.AlwaysTrue{}, true, nil
				}
			}

			if projection.inclusive != nil {
				matches, err := projection.inclusive(partition)
				if err != nil {
					return nil, false, err
				}
				if !matches {
					return iceberg.AlwaysFalse{}, true, nil
				}
			}
		}

		return n.predicate.original, false, nil
	case partitionResidualNot:
		child, changed, err := n.child.residual(partition)
		if err != nil {
			return nil, false, err
		}

		return iceberg.NewNot(child), changed, nil
	case partitionResidualAnd:
		left, leftChanged, err := n.left.residual(partition)
		if err != nil {
			return nil, false, err
		}
		if _, alwaysFalse := left.(iceberg.AlwaysFalse); alwaysFalse {
			return left, leftChanged, nil
		}

		right, rightChanged, err := n.right.residual(partition)
		if err != nil {
			return nil, false, err
		}

		return iceberg.NewAnd(left, right), leftChanged || rightChanged, nil
	case partitionResidualOr:
		left, leftChanged, err := n.left.residual(partition)
		if err != nil {
			return nil, false, err
		}
		if _, alwaysTrue := left.(iceberg.AlwaysTrue); alwaysTrue {
			return left, leftChanged, nil
		}

		right, rightChanged, err := n.right.residual(partition)
		if err != nil {
			return nil, false, err
		}

		return iceberg.NewOr(left, right), leftChanged || rightChanged, nil
	}

	return n.expr, false, nil
}

func partitionResidualValueIsNaN(value any) bool {
	if literal, ok := value.(iceberg.Literal); ok {
		value = literal.Any()
	}
	switch value := value.(type) {
	case float32:
		return math.IsNaN(float64(value))
	case float64:
		return math.IsNaN(value)
	default:
		return false
	}
}
