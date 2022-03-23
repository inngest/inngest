package fakedata

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/bxcodec/faker/v3"
	"github.com/google/uuid"
	lorem "github.com/inngest/golorem"
)

const (
	maxf64 = math.MaxFloat64
	minf64 = math.MaxFloat64 * -1
)

type Kind int
type Rule int
type Format int

const (
	KindFloat Kind = iota
	KindInt
	KindString
	KindBool
)

const (
	RuleLT Rule = iota
	RuleLTE
	RuleGT
	RuleGTE
	RuleEq
	RuleNEq
	RuleOneOf
	RuleFormat // format represents that the string should be in the specified format
)

const (
	FormatEmail Format = iota
	FormatURL
	FormatPhone
	FormatName
	FormatIPv6
	FormatIPv4
	FormatDate
	FormatTime
	FormatTitle
	FormatUUID
)

var (
	kindStrings = []string{"float", "int", "string", "email", "bool"}
	ruleStrings = []string{"<", "<=", ">", ">=", "==", "!=", "|"}
)

func (k Kind) String() string {
	return kindStrings[k]
}

func (r Rule) String() string {
	return ruleStrings[r]
}

// Constraint represents a single constraint for the generator.
type Constraint struct {
	Rule  Rule
	Value interface{}
}

func (c Constraint) String() string {
	return fmt.Sprintf("%s %v", c.Rule.String(), c.Value)
}

// Generate generates constraints given a kind, options, and values.
func Generate(ctx context.Context, kind Kind, o Options, constraints ...Constraint) interface{} {
	g := generator{kind: kind, o: o, constraints: constraints}
	return g.Generate()
}

type generator struct {
	o           Options
	kind        Kind
	constraints []Constraint
}

func (g generator) Generate() interface{} {
	switch g.kind {
	case KindInt, KindFloat:
		return g.Number()
	case KindString:
		return g.String()
	case KindBool:
		return g.o.Rand.Int31n(2) == 1
	}
	return nil
}

func (g generator) String() string {

	faker.SetRandomSource(g.o.Rand)

	for _, c := range g.constraints {
		switch c.Rule {
		case RuleOneOf:
			choices := []string{}
			for _, val := range c.Value.([]interface{}) {
				s, _ := val.(string)
				choices = append(choices, s)
			}
			i := g.o.Rand.Intn(len(choices))
			return choices[i]
		case RuleEq:
			return c.Value.(string)

		case RuleFormat:
			f := c.Value.(Format)
			switch f {
			case FormatUUID:
				byt := make([]byte, 16)
				g.o.Rand.Read(byt)
				id, _ := uuid.NewRandomFromReader(bytes.NewBuffer(byt))
				return id.String()
			case FormatTitle:
				parts := strings.Split(faker.Sentence(), " ")
				if len(parts) > 4 {
					return strings.Join(parts[:4], " ")
				}
				return strings.Join(parts, " ")
			case FormatEmail:
				return faker.Email()
			case FormatURL:
				return faker.URL()
			case FormatName:
				return faker.Name()
			case FormatIPv4:
				return faker.IPv4()
			case FormatIPv6:
				return faker.IPv6()
			case FormatDate:
				return faker.Date()
			case FormatPhone:
				return faker.Phonenumber()
			case FormatTime:
				return time.Unix(faker.UnixTime(), 0).Format(time.RFC3339)
			}
		default:
			// Other rules are not implemented
		}
	}

	return lorem.New(g.o.Rand).Sentence(1, 20)
}

func (g generator) Number() interface{} {
	ne := []float64{}

	// Iterate through each rule and add a constraint
	r := rng{o: g.o}
	for _, c := range g.constraints {
		switch c.Rule {
		case RuleLT:
			val := num(c.Value) - 1
			r.max = &val
		case RuleLTE:
			val := num(c.Value)
			r.max = &val
		case RuleGT:
			val := num(c.Value) + 1
			r.min = &val
		case RuleGTE:
			val := num(c.Value)
			r.min = &val
		case RuleEq:
			val := num(c.Value)
			r.min = &val
			r.max = &val
		case RuleNEq:
			ne = append(ne, num(c.Value))
		case RuleOneOf:
			r.choices = []float64{}
			for _, val := range c.Value.([]interface{}) {
				r.choices = append(r.choices, num(val))
			}
		}
	}

	// Generate a number that's not in this range.
	var f64 float64
	ok := false
	for !ok {
		f64 = r.generate()
		ok = true
		for _, item := range ne {
			if g.kind == KindInt && int(item) == int(f64) {
				ok = false
				break
			}
			if item == f64 {
				ok = false
				break
			}
		}
	}

	if g.kind == KindInt {
		return int(f64)
	}

	return f64
}

// rng generates a number between [min, max] (inclusive).
type rng struct {
	o Options
	// choices represent an enum of values, if specified.
	choices []float64
	// min is inclusive.
	min *float64
	// max is inclusive.
	max *float64
}

// generate a random number usiong the given constraints.
func (r rng) generate() float64 {
	if len(r.choices) > 0 {
		// Pick one of the choices at random.
		i := r.o.Rand.Intn(len(r.choices))
		return r.choices[i]
	}

	min := float64(r.o.NumericBound) * -1
	max := float64(r.o.NumericBound)
	if r.min != nil && *r.min > min {
		min = *r.min
	}
	if r.max != nil && *r.max < max {
		max = *r.max
	}

	f := min + r.o.Rand.Float64()*(max-min)
	if r.o.FloatPrecision == 0 {
		// return the whole number with 0 precision.
		return math.Round(f)
	}

	// Round to a specific decimal precision
	precision := math.Pow10(r.o.FloatPrecision)
	return math.Round(f*precision) / precision
}

func num(i interface{}) float64 {
	switch v := i.(type) {
	case uint:
		return float64(v)
	case int:
		return float64(v)
	case int8:
		return float64(v)
	case int16:
		return float64(v)
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	case float32:
		return float64(v)
	case float64:
		return float64(v)
	}
	return 0.0
}
