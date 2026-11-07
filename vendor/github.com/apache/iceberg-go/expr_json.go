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

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// JSON for boolean expressions, used as the "filter" of a REST scan-planning
// request. Mirrors Java's ExpressionParser.

// exprKeyTransform is the "type" value identifying a transform term.
const exprKeyTransform = "transform"

// opToJSON maps an Operation to its wire string (OpLTEQ -> "lt-eq"). OpTrue and
// OpFalse are handled separately: they serialize as bare JSON booleans.
var opToJSON = map[Operation]string{
	OpIsNull:        "is-null",
	OpNotNull:       "not-null",
	OpIsNan:         "is-nan",
	OpNotNan:        "not-nan",
	OpLT:            "lt",
	OpLTEQ:          "lt-eq",
	OpGT:            "gt",
	OpGTEQ:          "gt-eq",
	OpEQ:            "eq",
	OpNEQ:           "not-eq",
	OpStartsWith:    "starts-with",
	OpNotStartsWith: "not-starts-with",
	OpIn:            "in",
	OpNotIn:         "not-in",
	OpNot:           "not",
	OpAnd:           "and",
	OpOr:            "or",
}

var jsonToOp = func() map[string]Operation {
	m := make(map[string]Operation, len(opToJSON))
	for op, s := range opToJSON {
		m[s] = op
	}

	return m
}()

// Predicate operations grouped by the shape they carry on the wire. Spelled out
// rather than keyed off iota ranges, so reordering the Operation constants can't
// silently reclassify a predicate.
var (
	unaryOps = map[Operation]bool{
		OpIsNull: true, OpNotNull: true, OpIsNan: true, OpNotNan: true,
	}
	literalOps = map[Operation]bool{
		OpLT: true, OpLTEQ: true, OpGT: true, OpGTEQ: true,
		OpEQ: true, OpNEQ: true, OpStartsWith: true, OpNotStartsWith: true,
	}
	setOps = map[Operation]bool{
		OpIn: true, OpNotIn: true,
	}
)

// exprNode is the wire form of an expression node. omitempty leaves only the
// keys relevant to a given node; field order matches Java's output.
type exprNode struct {
	Type   string            `json:"type"`
	Child  json.RawMessage   `json:"child,omitempty"`
	Left   json.RawMessage   `json:"left,omitempty"`
	Right  json.RawMessage   `json:"right,omitempty"`
	Term   json.RawMessage   `json:"term,omitempty"`
	Value  json.RawMessage   `json:"value,omitempty"`
	Values []json.RawMessage `json:"values,omitempty"`
}

// transformNode is the wire form of a transform term, e.g.
// {"type":"transform","transform":"bucket[16]","term":"id"}.
type transformNode struct {
	Type      string `json:"type"`
	Transform string `json:"transform"`
	Term      string `json:"term"`
}

// ParseExpr parses an expression from its REST JSON form (a request "filter" or
// a task's residual filter).
//
// With a schema, literals take the referenced field's type (e.g. "2022-08-14"
// on a date column becomes a DateLiteral) and field names are resolved
// case-sensitively, since REST field names are authoritative. Without a schema
// literals fall back to the base JSON kind: Int64Literal, Float64Literal,
// StringLiteral, or BoolLiteral. The result is unbound.
//
// Transform terms (e.g. {"type":"transform","transform":"bucket[16]","term":"id"})
// parse into an UnboundTransform. The term resolves its result type against the
// schema, and a full predicate over a transform term can be bound and evaluated
// against rows.
func ParseExpr(data []byte, schema *Schema) (BooleanExpression, error) {
	return parseExpr(json.RawMessage(data), schema, true)
}

func parseExpr(raw json.RawMessage, schema *Schema, caseSensitive bool) (BooleanExpression, error) {
	return decodeExpr(raw, schema, caseSensitive)
}

// MarshalJSON emits the REST form, so an expression can be used as a "filter"
// field directly. Tag such fields omitempty to drop a nil one.
func (AlwaysTrue) MarshalJSON() ([]byte, error)  { return []byte("true"), nil }
func (AlwaysFalse) MarshalJSON() ([]byte, error) { return []byte("false"), nil }

func (n NotExpr) MarshalJSON() ([]byte, error) {
	child, err := json.Marshal(n.child)
	if err != nil {
		return nil, err
	}

	return json.Marshal(exprNode{Type: opToJSON[OpNot], Child: child})
}

func (a AndExpr) MarshalJSON() ([]byte, error) {
	left, right, err := marshalChildren(a.left, a.right)
	if err != nil {
		return nil, err
	}

	return json.Marshal(exprNode{Type: opToJSON[OpAnd], Left: left, Right: right})
}

func (o OrExpr) MarshalJSON() ([]byte, error) {
	left, right, err := marshalChildren(o.left, o.right)
	if err != nil {
		return nil, err
	}

	return json.Marshal(exprNode{Type: opToJSON[OpOr], Left: left, Right: right})
}

func marshalChildren(left, right BooleanExpression) (json.RawMessage, json.RawMessage, error) {
	l, err := json.Marshal(left)
	if err != nil {
		return nil, nil, err
	}
	r, err := json.Marshal(right)
	if err != nil {
		return nil, nil, err
	}

	return l, r, nil
}

// Bound predicates need their own MarshalJSON too; without it json.Marshal would
// fall through to {} since their fields are unexported.
func (up *unboundUnaryPredicate) MarshalJSON() ([]byte, error) {
	return marshalUnaryPredicate(up.op, up.term)
}

func (bp *boundUnaryPredicate[T]) MarshalJSON() ([]byte, error) {
	return marshalUnaryPredicate(bp.op, bp.term)
}

func (ul *unboundLiteralPredicate) MarshalJSON() ([]byte, error) {
	return marshalLiteralPredicate(ul.op, ul.term, ul.lit)
}

func (blp *boundLiteralPredicate[T]) MarshalJSON() ([]byte, error) {
	return marshalLiteralPredicate(blp.op, blp.term, blp.lit)
}

func (usp *unboundSetPredicate) MarshalJSON() ([]byte, error) {
	return marshalSetPredicate(usp.op, usp.term, usp.lits.Members())
}

func (bsp *boundSetPredicate[T]) MarshalJSON() ([]byte, error) {
	return marshalSetPredicate(bsp.op, bsp.term, bsp.lits.Members())
}

// Geospatial bbox predicates have no representation in the REST expression JSON
// grammar; they are used only for local scan planning. Report an error rather
// than silently emitting an empty object. The sentinel lives in errors.go with
// the other wrapped sentinels (see ErrBBoxNotSerializable).
func (u *unboundBBoxPredicate) MarshalJSON() ([]byte, error) { return nil, ErrBBoxNotSerializable }
func (b *boundBBoxPredicate) MarshalJSON() ([]byte, error)   { return nil, ErrBBoxNotSerializable }

// predicateType returns the wire "type" string for a predicate operation.
func predicateType(op Operation) (string, error) {
	s, ok := opToJSON[op]
	if !ok {
		return "", fmt.Errorf("%w: cannot serialize expression with operation %s", ErrInvalidArgument, op)
	}

	return s, nil
}

func marshalUnaryPredicate(op Operation, term Term) ([]byte, error) {
	js, err := predicateType(op)
	if err != nil {
		return nil, err
	}
	t, err := json.Marshal(term)
	if err != nil {
		return nil, err
	}

	return json.Marshal(exprNode{Type: js, Term: t})
}

func marshalLiteralPredicate(op Operation, term Term, lit Literal) ([]byte, error) {
	js, err := predicateType(op)
	if err != nil {
		return nil, err
	}
	t, err := json.Marshal(term)
	if err != nil {
		return nil, err
	}
	v, err := json.Marshal(literalValue{lit, termType(term)})
	if err != nil {
		return nil, err
	}

	return json.Marshal(exprNode{Type: js, Term: t, Value: v})
}

// termType is the field type of a bound term, or nil for an unbound reference.
func termType(term Term) Type {
	if bt, ok := term.(BoundTerm); ok {
		return bt.Ref().Field().Type
	}

	return nil
}

func marshalSetPredicate(op Operation, term Term, lits []Literal) ([]byte, error) {
	js, err := predicateType(op)
	if err != nil {
		return nil, err
	}
	t, err := json.Marshal(term)
	if err != nil {
		return nil, err
	}

	typ := termType(term)
	values := make([]json.RawMessage, 0, len(lits))
	for _, l := range lits {
		v, err := json.Marshal(literalValue{l, typ})
		if err != nil {
			return nil, err
		}
		values = append(values, v)
	}
	// A set has no order; sort the encoded values for deterministic output.
	sort.Slice(values, func(i, j int) bool {
		return bytes.Compare(values[i], values[j]) < 0
	})

	return json.Marshal(exprNode{Type: js, Term: t, Values: values})
}

func (r Reference) MarshalJSON() ([]byte, error) { return json.Marshal(string(r)) }

func (b *boundRef[T]) MarshalJSON() ([]byte, error) { return json.Marshal(b.fullName) }

func (b *BoundTransform) MarshalJSON() ([]byte, error) {
	ref := b.term.Ref()
	name := ref.Field().Name
	if named, ok := ref.(interface{ referenceName() string }); ok {
		name = named.referenceName()
	}

	return json.Marshal(transformNode{
		Type:      exprKeyTransform,
		Transform: b.transform.String(),
		Term:      name,
	})
}

func (u *UnboundTransform) MarshalJSON() ([]byte, error) {
	if u == nil {
		return nil, fmt.Errorf("%w: cannot serialize a nil transform term", ErrInvalidArgument)
	}
	if isNilTransform(u.transform) {
		return nil, fmt.Errorf("%w: transform cannot be nil", ErrInvalidArgument)
	}

	ref, ok := u.term.(Reference)
	if !ok {
		return nil, fmt.Errorf("%w: cannot serialize transform over a non-reference term", ErrInvalidArgument)
	}

	return json.Marshal(transformNode{
		Type:      exprKeyTransform,
		Transform: u.transform.String(),
		Term:      string(ref),
	})
}

// Each literal writes its REST form (Java's SingleValueParser).
func (l BoolLiteral) MarshalJSON() ([]byte, error)   { return json.Marshal(bool(l)) }
func (l Int32Literal) MarshalJSON() ([]byte, error)  { return json.Marshal(int32(l)) }
func (l Int64Literal) MarshalJSON() ([]byte, error)  { return json.Marshal(int64(l)) }
func (l StringLiteral) MarshalJSON() ([]byte, error) { return json.Marshal(string(l)) }

func (l Float32Literal) MarshalJSON() ([]byte, error) {
	if f := float64(l); math.IsInf(f, 0) || math.IsNaN(f) {
		return nil, fmt.Errorf("%w: cannot serialize non-finite float %v", ErrInvalidArgument, f)
	}

	return json.Marshal(float32(l))
}

func (l Float64Literal) MarshalJSON() ([]byte, error) {
	if f := float64(l); math.IsInf(f, 0) || math.IsNaN(f) {
		return nil, fmt.Errorf("%w: cannot serialize non-finite float %v", ErrInvalidArgument, f)
	}

	return json.Marshal(float64(l))
}

func (l DateLiteral) MarshalJSON() ([]byte, error) {
	return json.Marshal(Date(l).ToTime().Format("2006-01-02"))
}

func (l TimeLiteral) MarshalJSON() ([]byte, error) {
	// "9"s trim trailing fractional zeros (and the point when zero), as Java does.
	return json.Marshal(time.UnixMicro(int64(l)).UTC().Format("15:04:05.999999"))
}

// A bare timestamp literal has no field type, so it can't pick a wire form;
// serialize through a predicate (see literalValue) instead.
func (TimestampLiteral) MarshalJSON() ([]byte, error) {
	return nil, fmt.Errorf("%w: serialize a timestamp literal through a predicate so the field type is known", ErrInvalidArgument)
}

func (TimestampNsLiteral) MarshalJSON() ([]byte, error) {
	return nil, fmt.Errorf("%w: serialize a timestamp literal through a predicate so the field type is known", ErrInvalidArgument)
}

func (l UUIDLiteral) MarshalJSON() ([]byte, error) {
	return json.Marshal(uuid.UUID(l).String())
}

func (l FixedLiteral) MarshalJSON() ([]byte, error) {
	return json.Marshal(strings.ToUpper(hex.EncodeToString([]byte(l))))
}

func (l BinaryLiteral) MarshalJSON() ([]byte, error) {
	return json.Marshal(strings.ToUpper(hex.EncodeToString([]byte(l))))
}

func (l DecimalLiteral) MarshalJSON() ([]byte, error) {
	return json.Marshal(Decimal(l).String())
}

// literalValue pairs a literal with its field type, like typeIFace does for
// types. timestamp and timestamptz share one literal type, so the field type is
// the only thing that decides the +00:00 offset on the wire.
type literalValue struct {
	Literal
	typ Type
}

func (v literalValue) MarshalJSON() ([]byte, error) {
	switch l := v.Literal.(type) {
	case TimestampLiteral:
		return marshalTimestamp(Timestamp(l).ToTime(), v.typ)
	case TimestampNsLiteral:
		return marshalTimestamp(TimestampNano(l).ToTime(), v.typ)
	default:
		return json.Marshal(v.Literal)
	}
}

func (v *literalValue) UnmarshalJSON(b []byte) error {
	base, err := asObject(b)
	if err != nil {
		return err
	}
	lit, err := convertValue(base, v.typ)
	if err != nil {
		return err
	}
	v.Literal = lit

	return nil
}

// marshalTimestamp emits the REST form: with the UTC offset for timestamptz,
// without it for timestamp.
func marshalTimestamp(tm time.Time, typ Type) ([]byte, error) {
	switch typ.(type) {
	case TimestampTzType:
		// "-07:00" prints +00:00 for a UTC instant, matching Java.
		return json.Marshal(tm.Format("2006-01-02T15:04:05.999999-07:00"))
	case TimestampTzNsType:
		// Nanosecond precision needs nine fractional digits.
		return json.Marshal(tm.Format("2006-01-02T15:04:05.999999999-07:00"))
	case TimestampType:
		return json.Marshal(tm.Format("2006-01-02T15:04:05.999999"))
	case TimestampNsType:
		return json.Marshal(tm.Format("2006-01-02T15:04:05.999999999"))
	default:
		return nil, fmt.Errorf("%w: serializing a timestamp filter needs the field type; bind the expression to a schema first", ErrInvalidArgument)
	}
}

func decodeExpr(raw json.RawMessage, schema *Schema, caseSensitive bool) (BooleanExpression, error) {
	// A bare boolean is AlwaysTrue/AlwaysFalse; anything else is a node object.
	tok, err := firstToken(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: cannot parse expression: %s", ErrInvalidArgument, err)
	}
	if b, ok := tok.(bool); ok {
		if b {
			return AlwaysTrue{}, nil
		}

		return AlwaysFalse{}, nil
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return nil, fmt.Errorf("%w: cannot parse expression from %s", ErrInvalidArgument, raw)
	}

	var node exprNode
	if err := json.Unmarshal(raw, &node); err != nil {
		return nil, fmt.Errorf("%w: cannot parse expression: %s", ErrInvalidArgument, err)
	}

	// {"type":"literal","value":<bool>} is an alternate spelling of a constant.
	if node.Type == "literal" {
		// A missing or null value would unmarshal to false and silently become
		// AlwaysFalse, turning a corrupt payload into "filter everything out".
		if len(node.Value) == 0 || bytes.Equal(node.Value, []byte("null")) {
			return nil, fmt.Errorf("%w: literal expression is missing a boolean value", ErrInvalidArgument)
		}
		var bv bool
		if err := json.Unmarshal(node.Value, &bv); err != nil {
			return nil, fmt.Errorf("%w: cannot parse literal expression: %s", ErrInvalidArgument, err)
		}
		if bv {
			return AlwaysTrue{}, nil
		}

		return AlwaysFalse{}, nil
	}

	op, ok := jsonToOp[node.Type]
	if !ok {
		return nil, fmt.Errorf("%w: unknown expression type %q", ErrInvalidArgument, node.Type)
	}

	switch op {
	case OpNot:
		child, err := decodeExpr(node.Child, schema, caseSensitive)
		if err != nil {
			return nil, err
		}

		return NewNot(child), nil
	case OpAnd, OpOr:
		left, err := decodeExpr(node.Left, schema, caseSensitive)
		if err != nil {
			return nil, err
		}
		right, err := decodeExpr(node.Right, schema, caseSensitive)
		if err != nil {
			return nil, err
		}
		if op == OpAnd {
			return NewAnd(left, right), nil
		}

		return NewOr(left, right), nil
	}

	return decodePredicate(op, node, schema, caseSensitive)
}

// firstToken returns the first JSON token of raw, used to classify a node
// without inspecting bytes by hand.
func firstToken(raw json.RawMessage) (json.Token, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if _, ok := tok.(bool); ok {
		if _, err := dec.Token(); !errors.Is(err, io.EOF) {
			if err == nil {
				return nil, errors.New("trailing data after boolean expression")
			}

			return nil, err
		}
	}

	return tok, nil
}

func decodePredicate(op Operation, node exprNode, schema *Schema, caseSensitive bool) (BooleanExpression, error) {
	term, err := decodeTerm(node.Term)
	if err != nil {
		return nil, err
	}

	// Resolve the field type once; nil means schema-less.
	var typ Type
	if schema != nil {
		bound, err := term.Bind(schema, caseSensitive)
		if err != nil {
			return nil, err
		}
		typ = bound.Type()
	}

	switch {
	case unaryOps[op]:
		if len(node.Value) != 0 || node.Values != nil {
			return nil, fmt.Errorf("%w: unary predicate %s must not have a value", ErrInvalidArgument, node.Type)
		}

		return UnaryPredicate(op, term), nil
	case literalOps[op]:
		if len(node.Value) == 0 {
			return nil, fmt.Errorf("%w: predicate %s is missing a value", ErrInvalidArgument, node.Type)
		}
		if node.Values != nil {
			return nil, fmt.Errorf("%w: predicate %s must not have values", ErrInvalidArgument, node.Type)
		}
		lit, err := decodeValue(node.Value, typ)
		if err != nil {
			return nil, err
		}

		return LiteralPredicate(op, term, lit), nil
	case setOps[op]:
		if node.Values == nil {
			return nil, fmt.Errorf("%w: predicate %s is missing values", ErrInvalidArgument, node.Type)
		}
		if len(node.Value) != 0 {
			return nil, fmt.Errorf("%w: predicate %s must not have a value", ErrInvalidArgument, node.Type)
		}
		lits := make([]Literal, 0, len(node.Values))
		for _, v := range node.Values {
			lit, err := decodeValue(v, typ)
			if err != nil {
				return nil, err
			}
			lits = append(lits, lit)
		}

		return SetPredicate(op, term, lits), nil
	default:
		return nil, fmt.Errorf("%w: unsupported predicate operation %s", ErrInvalidArgument, op)
	}
}

func decodeTerm(raw json.RawMessage) (UnboundTerm, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("%w: predicate is missing a term", ErrInvalidArgument)
	}

	tok, err := firstToken(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: cannot parse term: %s", ErrInvalidArgument, err)
	}

	// A bare string is a plain reference.
	if name, ok := tok.(string); ok {
		return Reference(name), nil
	}

	var t transformNode
	if err := json.Unmarshal(raw, &t); err != nil {
		return nil, fmt.Errorf("%w: cannot parse term: %s", ErrInvalidArgument, err)
	}
	switch t.Type {
	case "reference":
		// {"type":"reference","term":"name"}
		return Reference(t.Term), nil
	case exprKeyTransform:
		// {"type":"transform","transform":"bucket[16]","term":"id"}
		tf, err := ParseTransform(t.Transform)
		if err != nil {
			return nil, fmt.Errorf("%w: cannot parse transform term: %s", ErrInvalidArgument, err)
		}
		// Unknown transforms are tolerated in partition/sort metadata, but a
		// filter expression referencing one can't be evaluated.
		if isUnknownTransform(tf) {
			return nil, fmt.Errorf("%w: unknown transform in expression term: %s", ErrInvalidArgument, t.Transform)
		}

		return NewUnboundTransform(tf, Reference(t.Term)), nil
	default:
		return nil, fmt.Errorf("%w: cannot parse term with type %q", ErrInvalidArgument, t.Type)
	}
}

// decodeValue parses one literal value and converts it to typ.
func decodeValue(raw json.RawMessage, typ Type) (Literal, error) {
	v := literalValue{typ: typ}
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, err
	}

	return v.Literal, nil
}

// convertValue turns a base literal into typ. A nil type yields the base kind for
// the JSON token, as the reference does without a schema.
func convertValue(base Literal, typ Type) (Literal, error) {
	if typ == nil {
		return base, nil
	}

	// Binary and fixed arrive as hex strings; decode them rather than treating
	// the string as raw bytes.
	switch typ.(type) {
	case BinaryType, FixedType:
		s, ok := base.(StringLiteral)
		if !ok {
			return nil, fmt.Errorf("%w: expected a string for %s value", ErrInvalidArgument, typ)
		}
		decoded, err := hex.DecodeString(string(s))
		if err != nil {
			return nil, fmt.Errorf("%w: invalid hex for %s value: %s", ErrInvalidArgument, typ, err)
		}
		if ft, ok := typ.(FixedType); ok {
			if len(decoded) != ft.Len() {
				return nil, fmt.Errorf("%w: fixed[%d] value has %d bytes", ErrInvalidArgument, ft.Len(), len(decoded))
			}

			return FixedLiteral(decoded), nil
		}

		return BinaryLiteral(decoded), nil
	}

	return base.To(typ)
}

// asObject parses a JSON scalar to its base literal kind (Int64, Float64,
// String, or Bool), as the reference does without a schema.
func asObject(raw json.RawMessage) (Literal, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()

	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, fmt.Errorf("%w: cannot parse literal value: %s", ErrInvalidArgument, err)
	}

	switch x := v.(type) {
	case json.Number:
		if i, err := x.Int64(); err == nil {
			return Int64Literal(i), nil
		}
		f, err := x.Float64()
		if err != nil {
			return nil, fmt.Errorf("%w: cannot parse number literal %q: %s", ErrInvalidArgument, x.String(), err)
		}

		return Float64Literal(f), nil
	case string:
		return StringLiteral(x), nil
	case bool:
		return BoolLiteral(x), nil
	default:
		return nil, fmt.Errorf("%w: cannot parse literal value from %s", ErrInvalidArgument, raw)
	}
}
