package fakedata

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/token"
	"cuelang.org/go/encoding/openapi"
)

var (
	// generatorFunc allows us to swap out the generator during testing, without creating
	// interfaces for a single implementation.
	generatorFunc = Generate

	// ctxPath lets us store the paths we've walked to figure out the constraints.
	ctxPath = "path"
)

func NewRand() *rand.Rand {
	return rand.New(rand.NewSource(time.Now().UnixNano()))
}

var DefaultOptions = Options{
	Rand:           NewRand(),
	FloatPrecision: 2,
	NumericBound:   1 << 16,
}

type Options struct {
	Rand *rand.Rand
	// FloatPrecision is the max number of decimal places to generate for
	// number or float datatypes.
	FloatPrecision int
	// NumericBound is the upper and lower numeric bound for random numbers
	NumericBound int
}

// Fake generates fake data for a given cue definition.  The returning cue.Value
// can be decoded into the corresponding Go type.  For example, if the cue definition
// represents a string, the returning value can be decoded into a Go string.  Likewise,
// for maps, we can decode into a map[string]interface{}.
func Fake(ctx context.Context, v cue.Value) (cue.Value, error) {
	s := &Output{StructLit: ast.NewStruct()}

	o := DefaultOptions
	if o.Rand == nil {
		o.Rand = NewRand()
	}

	// Iterate through the value, adding each field to the struct output.
	if err := walk(ctx, v, s.StructLit, DefaultOptions); err != nil {
		return cue.Value{}, err
	}

	// Format the output.
	byt, err := format.Node(
		s.StructLit,
		format.TabIndent(false),
		format.UseSpaces(2),
	)
	if err != nil {
		return cue.Value{}, fmt.Errorf("error formatting value: %w", err)
	}

	r := &cue.Runtime{}
	inst, err := r.Compile(".", byt)
	if err != nil {
		return cue.Value{}, fmt.Errorf("error compiling: %w", err)
	}

	return inst.Value(), nil
}

func walk(ctx context.Context, v cue.Value, to *ast.StructLit, o Options) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("error generating fake data: %v", r)
		}
	}()

	// This could be an enum across structs.  If that's the case, use one of the
	// structs as the value.
	op, exprVals := v.Expr()
	if op == cue.OrOp {
		// Choose one of the epressions to use randomly.
		i := o.Rand.Intn(len(exprVals))
		v = exprVals[i]
	}

	it, err := v.Fields()
	if err != nil {
		return err
	}

	for it.Next() {
		if it.IsHidden() {
			continue
		}

		// An optional field has a 50% chance of being generated.
		if it.IsOptional() && o.Rand.Intn(2) == 1 {
			continue
		}

		nestedCtx := withPath(ctx, it.Label())

		val := it.Value()
		label := it.Label()

		switch val.IncompleteKind() {
		case cue.BoolKind:
			t := "false"
			if generatorFunc(ctx, KindBool, o) == true {
				t = "true"
			}
			lit := ast.NewLit(token.STRING, t)
			set(to, label, lit)
		case cue.StringKind:
			lit := genString(nestedCtx, val, o)
			set(to, label, lit)
		case cue.NumberKind, cue.FloatKind:
			lit := genNumber(nestedCtx, KindFloat, val, o)
			set(to, label, lit)
		case cue.IntKind:
			lit := genNumber(nestedCtx, KindInt, val, o)
			set(to, label, lit)
		case cue.StructKind:
			// Create a new struct and set the field to the new struct
			inner := ast.NewStruct()
			set(to, label, inner)
			// Iterate into the struct and walk through those fields,
			// setting where necessary
			if err := walk(nestedCtx, val, inner, o); err != nil {
				return err
			}
		default:
			// Can't do this one, homie.
		}
	}

	return err
}

// genString returns cue AST representing a string
func genString(ctx context.Context, val cue.Value, o Options) *ast.BasicLit {
	constraints := constraints(ctx, KindString, val)
	f := generatorFunc(ctx, KindString, o, constraints...)
	return ast.NewLit(token.STRING, strconv.Quote(f.(string)))
}

// genNumber returns cue AST representing a float or int, depending on the
// kind passed as an argument.
func genNumber(ctx context.Context, k Kind, val cue.Value, o Options) *ast.BasicLit {
	if k != KindInt && k != KindFloat {
		return nil
	}
	f := generatorFunc(ctx, k, o, constraints(ctx, k, val)...)

	if k == KindFloat {
		byt, _ := json.Marshal(f)
		return ast.NewLit(token.FLOAT, fmt.Sprintf("%s", string(byt)))
	}

	return ast.NewLit(token.INT, fmt.Sprintf("%d", f.(int)))
}

// constraints returns data generation constraints given a cue value
// field definition.  This looks at the constraints by the type and also
// any additional constraints added via types to generate all constraints,
// eg. `uint8 & >= 5`.
//
// This turns nested cue operators to constraints for our generator.
func constraints(ctx context.Context, kind Kind, vals ...cue.Value) []Constraint {
	c := []Constraint{}

	for _, val := range vals {
		op, exprVals := val.Expr()

		switch op {
		case cue.NoOp:
			// There's no operator to this expression.  This may happen when we recurse
			// into a type definition such as `foo: uint & 1`.  This may be a type definition
			// or a value.
			node := val.Source()
			for node != nil {
				switch n := node.(type) {
				case *ast.Ident:
					// This is a type definition.  Do nothing.
					node = nil
				case *ast.BasicLit:
					// This is a value, which indicates a constraint.
					value, err := decode(kind, val)
					if err != nil {
						panic(err.Error())
					}
					c = append(c, Constraint{Rule: RuleEq, Value: value})
					node = nil
				case *ast.Field:
					// Recurse into the field's value.
					node = n.Value
				default:
					// break.
					node = nil
				}
			}

		case cue.OrOp:
			// Iterate through each element in the union and figure out constraints from there.
			next := constraints(ctx, kind, exprVals...)
			// Next are constraints with "RuleEq" values.  The value is an interface representing
			// the element we must match.
			value := []interface{}{}
			for _, constraint := range next {
				if constraint.Rule != RuleEq {
					// TODO (tonyhb): remove panic.
					panic(fmt.Errorf("unknown value"))
				}
				value = append(value, constraint.Value)
			}
			c = append(c, Constraint{Rule: RuleOneOf, Value: value})
		case cue.AndOp:
			// If there's > 1 value within this constraint, we know that it's combined
			// with more constraints (eg. uint& & >= 5).  In this example,  we want
			// to recurse and add the constraints from these to our collecton.
			c = append(c, constraints(ctx, kind, exprVals...)...)

		case cue.EqualOp:
			decoded := mustDecode(kind, exprVals[0])
			c = append(c, Constraint{Rule: RuleEq, Value: decoded})

		case cue.NotEqualOp, cue.NotOp:
			decoded := mustDecode(kind, exprVals[0])
			c = append(c, Constraint{Rule: RuleNEq, Value: decoded})

		case cue.LessThanOp:
			decoded := mustDecode(kind, exprVals[0])
			c = append(c, Constraint{Rule: RuleLT, Value: decoded})

		case cue.LessThanEqualOp:
			decoded := mustDecode(kind, exprVals[0])
			c = append(c, Constraint{Rule: RuleLTE, Value: decoded})

		case cue.GreaterThanEqualOp:
			decoded := mustDecode(kind, exprVals[0])
			c = append(c, Constraint{Rule: RuleGTE, Value: decoded})

		case cue.GreaterThanOp:
			decoded := mustDecode(kind, exprVals[0])
			c = append(c, Constraint{Rule: RuleGT, Value: decoded})
		}
	}

	// If this is of kind string, attempt to ascertain the expected output for
	// fake data generation based off of the path.
	if kind == KindString {
		c = append(c, predictStringFormats(ctx)...)
	}

	return c
}

func predictStringFormats(ctx context.Context) []Constraint {
	// Get the path from context.
	parts := strings.Split(path(ctx), ".")
	if len(parts) == 0 {
		return nil
	}

	field := strings.ToLower(parts[len(parts)-1])

	if field == "email" {
		return []Constraint{{Rule: RuleFormat, Value: FormatEmail}}
	}
	if strings.Contains(field, "phone") {
		return []Constraint{{Rule: RuleFormat, Value: FormatPhone}}
	}
	if strings.Contains(field, "url") || strings.Contains(field, "uri") {
		return []Constraint{{Rule: RuleFormat, Value: FormatURL}}
	}
	if strings.Contains(field, "ipv6") {
		return []Constraint{{Rule: RuleFormat, Value: FormatURL}}
	}
	if strings.Contains(field, "_ip") || strings.Contains(field, "ip_") {
		return []Constraint{{Rule: RuleFormat, Value: FormatIPv4}}
	}
	if strings.Contains(field, "date") {
		return []Constraint{{Rule: RuleFormat, Value: FormatDate}}
	}
	if strings.Contains(field, "name") {
		return []Constraint{{Rule: RuleFormat, Value: FormatName}}
	}
	if strings.Contains(field, "time") || strings.HasSuffix(field, "at") {
		return []Constraint{{Rule: RuleFormat, Value: FormatTime}}
	}
	if strings.Contains(field, "title") {
		return []Constraint{{Rule: RuleFormat, Value: FormatTitle}}
	}
	if field == "id" || strings.Contains(field, "uuid") || strings.HasSuffix(field, "_id") {
		return []Constraint{{Rule: RuleFormat, Value: FormatUUID}}
	}

	return nil
}

func mustDecode(k Kind, v cue.Value) interface{} {
	d, err := decode(k, v)
	if err != nil {
		panic(err)
	}
	return d
}

// decode decodes a cue value with a given kind, allowing us to type convert
// native go types.
func decode(k Kind, v cue.Value) (interface{}, error) {
	switch k {
	case KindString:
		var s string
		err := v.Decode(&s)
		if err != nil {
			return s, fmt.Errorf("error decoding to string (%v): %w", v, err)
		}
		return s, nil
	case KindInt:
		var i int
		err := v.Decode(&i)
		if err != nil {
			return i, fmt.Errorf("error decoding to int (%v): %w", v, err)
		}
		return i, nil
	case KindFloat:
		var f float64
		err := v.Decode(&f)
		return f, err
	}
	return nil, fmt.Errorf("not implemented")
}

// Output wraps a StructLit for easy marshalling.
type Output struct {
	*ast.StructLit
}

func (o *Output) MarshalJSON() ([]byte, error) {
	// openapi.OrderedMap uses cue's internal encoding to
	// generate valid JSON given our struct.
	if o.StructLit == nil {
		return nil, nil
	}
	return json.Marshal((*openapi.OrderedMap)(o.StructLit))
}

// set sets a field within a cue Struct
func set(s *ast.StructLit, key string, expr ast.Expr) {
	s.Elts = append(s.Elts, &ast.Field{
		Label: ast.NewString(key),
		Value: expr,
	})
}

func path(ctx context.Context) string {
	path, _ := ctx.Value(ctxPath).(string)
	return path
}

func withPath(ctx context.Context, path string) context.Context {
	existing, _ := ctx.Value(ctxPath).(string)
	if existing == "" {
		return context.WithValue(ctx, ctxPath, path)
	}
	return context.WithValue(ctx, ctxPath, existing+"."+path)
}
