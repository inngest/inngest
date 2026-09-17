package peg

import (
	"fmt"
	"strings"
)

type ExprKind int

const (
	ExprLiteral ExprKind = iota
	ExprClass
	ExprRef
	ExprCall
	ExprSeq
	ExprChoice
	ExprOptional
	ExprStar
	ExprPlus
	ExprNot
	ExprToken
)

// Expr is one node of a parsed rule's right-hand side.
type Expr struct {
	Kind     ExprKind
	Literal  string // ExprLiteral: the literal text, unescaped.
	Class    Class  // ExprClass.
	Ref      string // ExprRef / ExprCall: the referenced rule or bound parameter name.
	Children []Expr // ExprSeq/ExprChoice: each element/alternative.
	// ExprCall: exactly one child, the call's argument expression.
	// ExprOptional/ExprStar/ExprPlus/ExprNot/ExprToken: exactly one child.
}

type ClassRange struct{ Lo, Hi rune }

type Class struct {
	Negated bool
	Ranges  []ClassRange
}

func (c Class) Matches(r rune) bool {
	in := false
	for _, rg := range c.Ranges {
		if r >= rg.Lo && r <= rg.Hi {
			in = true
			break
		}
	}
	if c.Negated {
		return !in
	}
	return in
}

// Rule is one named production. Param is "" unless the rule is
// parameterized (only List/Parens in the vendored closure).
type Rule struct {
	Name  string
	Param string
	Expr  Expr
}

type Grammar struct {
	Rules map[string]*Rule
}

// ParseGrammar parses the concatenation of one or more `.gram` sources into
// a single rule table. Rule names must be unique across all sources passed
// together — the vendored files are parsed as one call for this reason.
func ParseGrammar(sources ...string) (*Grammar, error) {
	g := &Grammar{Rules: map[string]*Rule{}}
	for _, src := range sources {
		p := &dslParser{src: src}
		p.skipTrivia()
		for !p.atEnd() {
			rule, err := p.parseRule()
			if err != nil {
				return nil, err
			}
			if _, exists := g.Rules[rule.Name]; exists {
				return nil, fmt.Errorf("duplicate rule %q", rule.Name)
			}
			g.Rules[rule.Name] = rule
			p.skipTrivia()
		}
	}
	return g, nil
}

type dslParser struct {
	src string
	pos int
}

func (p *dslParser) atEnd() bool { return p.pos >= len(p.src) }

func (p *dslParser) peek() byte {
	if p.atEnd() {
		return 0
	}
	return p.src[p.pos]
}

func (p *dslParser) hasPrefix(s string) bool {
	return strings.HasPrefix(p.src[p.pos:], s)
}

// skipTrivia skips whitespace and '#'-to-end-of-line comments between DSL
// tokens. Distinct from SQL trivia (spaces/--/​/* */), which the packrat
// evaluator (Task 3) skips while parsing the *target* SQL text, not this
// meta-grammar.
func (p *dslParser) skipTrivia() {
	for !p.atEnd() {
		switch p.src[p.pos] {
		case ' ', '\t', '\r', '\n':
			p.pos++
		case '#':
			for !p.atEnd() && p.src[p.pos] != '\n' {
				p.pos++
			}
		default:
			return
		}
	}
}

// '%' is accepted only as a leading character so a directive like
// `%whitespace <- [ \t\n\r]*` (common.gram) parses as an ordinary rule
// (literally named "%whitespace") rather than failing to parse at all —
// nothing ever references that name, so it's harmless to just carry it in
// the rule table unused.
func dslIsIdentStart(c byte) bool {
	return c == '_' || c == '%' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func dslIsIdentCont(c byte) bool {
	return dslIsIdentStart(c) || (c >= '0' && c <= '9')
}

func (p *dslParser) parseIdentRaw() (string, bool) {
	start := p.pos
	if p.atEnd() || !dslIsIdentStart(p.src[p.pos]) {
		return "", false
	}
	p.pos++
	for !p.atEnd() && dslIsIdentCont(p.src[p.pos]) {
		p.pos++
	}
	return p.src[start:p.pos], true
}

func (p *dslParser) parseRule() (*Rule, error) {
	name, ok := p.parseIdentRaw()
	if !ok {
		return nil, fmt.Errorf("offset %d: expected rule name", p.pos)
	}
	rule := &Rule{Name: name}
	p.skipTrivia()
	if p.peek() == '(' {
		p.pos++
		p.skipTrivia()
		param, ok := p.parseIdentRaw()
		if !ok {
			return nil, fmt.Errorf("offset %d: expected parameter name in %q's definition", p.pos, name)
		}
		rule.Param = param
		p.skipTrivia()
		if p.peek() != ')' {
			return nil, fmt.Errorf("offset %d: expected ')' after parameter list in %q's definition", p.pos, name)
		}
		p.pos++
		p.skipTrivia()
	}
	if !p.hasPrefix("<-") {
		return nil, fmt.Errorf("offset %d: expected '<-' in %q's definition", p.pos, name)
	}
	p.pos += 2
	expr, err := p.parseChoice()
	if err != nil {
		return nil, err
	}
	rule.Expr = expr
	return rule, nil
}

func (p *dslParser) parseChoice() (Expr, error) {
	first, err := p.parseSeq()
	if err != nil {
		return Expr{}, err
	}
	alts := []Expr{first}
	for {
		p.skipTrivia()
		if p.peek() != '/' {
			break
		}
		p.pos++
		next, err := p.parseSeq()
		if err != nil {
			return Expr{}, err
		}
		alts = append(alts, next)
	}
	if len(alts) == 1 {
		return alts[0], nil
	}
	return Expr{Kind: ExprChoice, Children: alts}, nil
}

func (p *dslParser) parseSeq() (Expr, error) {
	var elems []Expr
	for {
		p.skipTrivia()
		elem, ok, err := p.tryParsePostfix()
		if err != nil {
			return Expr{}, err
		}
		if !ok {
			break
		}
		elems = append(elems, elem)
	}
	if len(elems) == 0 {
		return Expr{}, fmt.Errorf("offset %d: expected an expression", p.pos)
	}
	if len(elems) == 1 {
		return elems[0], nil
	}
	return Expr{Kind: ExprSeq, Children: elems}, nil
}

func (p *dslParser) tryParsePostfix() (Expr, bool, error) {
	p.skipTrivia()
	if p.peek() == '!' {
		p.pos++
		inner, ok, err := p.tryParsePostfix()
		if err != nil {
			return Expr{}, false, err
		}
		if !ok {
			return Expr{}, false, fmt.Errorf("offset %d: expected an expression after '!'", p.pos)
		}
		return Expr{Kind: ExprNot, Children: []Expr{inner}}, true, nil
	}
	atom, ok, err := p.tryParseAtom()
	if err != nil || !ok {
		return Expr{}, ok, err
	}
	switch p.peek() {
	case '?':
		p.pos++
		return Expr{Kind: ExprOptional, Children: []Expr{atom}}, true, nil
	case '*':
		p.pos++
		return Expr{Kind: ExprStar, Children: []Expr{atom}}, true, nil
	case '+':
		p.pos++
		return Expr{Kind: ExprPlus, Children: []Expr{atom}}, true, nil
	}
	return atom, true, nil
}

func (p *dslParser) tryParseAtom() (Expr, bool, error) {
	p.skipTrivia()
	switch {
	case p.peek() == '\'':
		lit, err := p.parseQuotedLiteral()
		if err != nil {
			return Expr{}, false, err
		}
		return Expr{Kind: ExprLiteral, Literal: lit}, true, nil
	case p.peek() == '[':
		cls, err := p.parseClass()
		if err != nil {
			return Expr{}, false, err
		}
		return Expr{Kind: ExprClass, Class: cls}, true, nil
	case p.peek() == '<':
		p.pos++
		inner, err := p.parseChoice()
		if err != nil {
			return Expr{}, false, err
		}
		p.skipTrivia()
		if p.peek() != '>' {
			return Expr{}, false, fmt.Errorf("offset %d: expected '>' to close token group", p.pos)
		}
		p.pos++
		return Expr{Kind: ExprToken, Children: []Expr{inner}}, true, nil
	case p.peek() == '(':
		p.pos++
		inner, err := p.parseChoice()
		if err != nil {
			return Expr{}, false, err
		}
		p.skipTrivia()
		if p.peek() != ')' {
			return Expr{}, false, fmt.Errorf("offset %d: expected ')' to close group", p.pos)
		}
		p.pos++
		return inner, true, nil
	case dslIsIdentStart(p.peek()):
		start := p.pos
		name, _ := p.parseIdentRaw()
		if p.peek() == '(' {
			p.pos++
			arg, err := p.parseChoice()
			if err != nil {
				return Expr{}, false, err
			}
			p.skipTrivia()
			if p.peek() != ')' {
				return Expr{}, false, fmt.Errorf("offset %d: expected ')' to close call to %q", p.pos, name)
			}
			p.pos++
			// Disambiguate "Name(Param) <- ..." (a new rule header) from a
			// genuine call "Name(Arg)" — both parse identically up to the
			// closing paren, so only what follows it tells them apart.
			save := p.pos
			p.skipTrivia()
			isNewRule := p.hasPrefix("<-")
			p.pos = save
			if isNewRule {
				p.pos = start
				return Expr{}, false, nil
			}
			return Expr{Kind: ExprCall, Ref: name, Children: []Expr{arg}}, true, nil
		}
		if p.startsNextRule() {
			p.pos = start
			return Expr{}, false, nil
		}
		return Expr{Kind: ExprRef, Ref: name}, true, nil
	default:
		return Expr{}, false, nil
	}
}

// startsNextRule reports whether, from the current position (immediately
// after an identifier atom we just tentatively parsed), the upcoming
// tokens form a new rule header (`<-` or `(Param) <-`) rather than a
// continuation of the expression we're inside. Never advances p.pos.
func (p *dslParser) startsNextRule() bool {
	save := p.pos
	defer func() { p.pos = save }()
	p.skipTrivia()
	if p.peek() == '(' {
		p.pos++
		p.skipTrivia()
		if _, ok := p.parseIdentRaw(); !ok {
			return false
		}
		p.skipTrivia()
		if p.peek() != ')' {
			return false
		}
		p.pos++
		p.skipTrivia()
	}
	return p.hasPrefix("<-")
}

func (p *dslParser) parseQuotedLiteral() (string, error) {
	p.pos++ // opening '\''
	var b strings.Builder
	for {
		if p.atEnd() {
			return "", fmt.Errorf("offset %d: unterminated literal", p.pos)
		}
		c := p.src[p.pos]
		if c == '\\' && p.pos+1 < len(p.src) {
			b.WriteByte(p.src[p.pos+1])
			p.pos += 2
			continue
		}
		if c == '\'' {
			p.pos++
			return b.String(), nil
		}
		b.WriteByte(c)
		p.pos++
	}
}

func (p *dslParser) parseClass() (Class, error) {
	p.pos++ // opening '['
	var cls Class
	if p.peek() == '^' {
		cls.Negated = true
		p.pos++
	}
	readRune := func() (rune, bool) {
		if p.atEnd() {
			return 0, false
		}
		if p.src[p.pos] == '\\' && p.pos+1 < len(p.src) {
			r := rune(p.src[p.pos+1])
			p.pos += 2
			return r, true
		}
		r := rune(p.src[p.pos])
		p.pos++
		return r, true
	}
	for {
		if p.atEnd() {
			return Class{}, fmt.Errorf("offset %d: unterminated character class", p.pos)
		}
		if p.src[p.pos] == ']' {
			p.pos++
			return cls, nil
		}
		lo, ok := readRune()
		if !ok {
			return Class{}, fmt.Errorf("offset %d: unterminated character class", p.pos)
		}
		hi := lo
		if !p.atEnd() && p.src[p.pos] == '-' && p.pos+1 < len(p.src) && p.src[p.pos+1] != ']' {
			p.pos++
			r, ok := readRune()
			if !ok {
				return Class{}, fmt.Errorf("offset %d: unterminated character class", p.pos)
			}
			hi = r
		}
		cls.Ranges = append(cls.Ranges, ClassRange{Lo: lo, Hi: hi})
	}
}
