// Copyright (c) 2020, Peter Ohler, All rights reserved.

package jp

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"unicode/utf8"

	"github.com/ohler55/ojg"
)

const (
	//   0123456789abcdef0123456789abcdef
	tokenMap = "" +
		"................................" + // 0x00
		"...o.o..........oooooooooooo...o" + // 0x20
		".oooooooooooooooooooooooooo...oo" + // 0x40
		".oooooooooooooooooooooooooooooo." + // 0x60
		"oooooooooooooooooooooooooooooooo" + // 0x80
		"oooooooooooooooooooooooooooooooo" + // 0xa0
		"oooooooooooooooooooooooooooooooo" + // 0xc0
		"oooooooooooooooooooooooooooooooo" //   0xe0

	// o for an operatio
	// v for a value start character
	//   0123456789abcdef0123456789abcdef
	eqMap = "" +
		"................................" + // 0x00
		".ov.v.ovv.oo.o.ovvvvvvvvvv..ooo." + // 0x20
		"v..............................." + // 0x40
		".....ov.oo....v.....v.......o.o." + // 0x60
		"................................" + // 0x80
		"................................" + // 0xa0
		"................................" + // 0xc0
		"................................" //   0xe0
)

// Performance is less a concern with Expr parsing as it is usually done just
// once if performance is important. Alternatively, an Expr can be built using
// function calls or bare structs. Parsing is more for convenience. Using this
// approach over modes only adds 10% so a reasonable penalty for
// maintainability.
type parser struct {
	buf []byte
	pos int
}

// ParseString parses a string into an Expr.
func ParseString(s string) (x Expr, err error) {
	return Parse([]byte(s))
}

// MustParseString parses a string into an Expr and panics on error.
func MustParseString(s string) (x Expr) {
	return MustParse([]byte(s))
}

// Parse parses a []byte into an Expr.
func Parse(buf []byte) (x Expr, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = ojg.NewError(r)
		}
	}()
	x = MustParse(buf)

	return
}

// MustParse parses a []byte into an Expr and panics on error.
func MustParse(buf []byte) (x Expr) {
	p := &parser{buf: buf}
	x = p.readExpr()
	if p.pos < len(buf) {
		p.raise("parse error")
	}
	return
}

func (p *parser) readExpr() (x Expr) {
	x = Expr{}
	var f Frag
	first := true
	lastDescent := false
	for {
		if f = p.nextFrag(first, lastDescent); f == nil {
			return
		}
		first = false
		if _, ok := f.(Descent); ok {
			lastDescent = true
		} else {
			lastDescent = false
		}
		x = append(x, f)
	}
}

func (p *parser) nextFrag(first, lastDescent bool) (f Frag) {
	if p.pos < len(p.buf) {
		b := p.buf[p.pos]
		p.pos++
		switch b {
		case '$':
			if first {
				f = Root('$')
			}
		case '@':
			if first {
				f = At('@')
			}
		case '.':
			f = p.afterDot()
		case '*':
			return Wildcard('*')
		case '[':
			f = p.afterBracket()
		case ']':
			p.pos--
			// done
		default:
			p.pos--
			if tokenMap[b] == 'o' {
				if first {
					f = p.afterDot()
				} else if lastDescent {
					f = p.afterDotDot()
				}
			}
		}
		// Any other character is the end of the Expr, figure out later if
		// that is an error.
	}
	return
}

func (p *parser) afterDot() Frag {
	if len(p.buf) <= p.pos {
		p.raise("not terminated")
	}
	var token []byte
	b := p.buf[p.pos]
	p.pos++
	switch b {
	case '*':
		return Wildcard('*')
	case '.':
		return Descent('.')
	default:
		if tokenMap[b] == '.' {
			p.raise("an expression fragment can not start with a '%c'", b)
		}
		token = append(token, b)
	}
	for p.pos < len(p.buf) {
		b := p.buf[p.pos]
		p.pos++
		if tokenMap[b] == '.' {
			p.pos--
			break
		}
		token = append(token, b)
	}
	return Child(token)
}

func (p *parser) afterDotDot() Frag {
	var token []byte
	b := p.buf[p.pos]
	p.pos++
	token = append(token, b)
	for p.pos < len(p.buf) {
		b := p.buf[p.pos]
		p.pos++
		if tokenMap[b] == '.' {
			p.pos--
			break
		}
		token = append(token, b)
	}
	return Child(token)
}

func (p *parser) afterBracket() Frag {
	if len(p.buf) <= p.pos {
		p.raise("not terminated")
	}
	b := p.skipSpace()
	switch b {
	case '*':
		// expect ]
		b := p.skipSpace()
		if b != ']' {
			p.raise("not terminated")
		}
		return Wildcard('#')
	case '\'', '"':
		s := p.readStr(b)
		b = p.skipSpace()
		switch b {
		case ']':
			return Child(s)
		case ',':
			return p.readUnion(s, b)
		default:
			p.raise("invalid bracket fragment")
		}
	case ':':
		return p.readSlice(0)
	case '?':
		return p.readFilter()
	case '(':
		p.raise("scripts not implemented yet")
	case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		var i int
		i, b = p.readInt(b)
	Next:
		switch b {
		case ' ':
			b = p.skipSpace()
			goto Next
		case ']':
			return Nth(i)
		case ',':
			return p.readUnion(int64(i), b)
		case ':':
			return p.readSlice(i)
		default:
			p.raise("invalid bracket fragment")
		}
	}
	p.pos--
	// Kind of ugly but needed to attain full cod coverage as the cover tool
	// and the compilier don't know about panics in functions so get the
	// return and raise on the same line.
	return func() Frag { p.raise("parse error"); return nil }()
}

func (p *parser) readInt(b byte) (int, byte) {
	// Allow numbers to begin with a zero.
	/*
		if b == '0' {
			if p.pos < len(p.buf) {
				b = p.buf[p.pos]
				p.pos++
			}
			return 0, b, nil
		}
	*/
	neg := b == '-'
	if neg {
		if len(p.buf) <= p.pos {
			p.raise("expected a number")
		}
		b = p.buf[p.pos]
		p.pos++
	}
	start := p.pos
	var i int
	for {
		if b < '0' || '9' < b {
			break
		}
		i = i*10 + int(b-'0')
		if len(p.buf) <= p.pos {
			break
		}
		b = p.buf[p.pos]
		p.pos++
	}
	if p.pos == start {
		p.raise("expected a number")
	}
	if neg {
		i = -i
	}
	return i, b
}

func (p *parser) readNum(b byte) any {
	var num []byte

	num = append(num, b)
	// Read digits first
	for p.pos < len(p.buf) {
		b = p.buf[p.pos]
		if b < '0' || '9' < b {
			break
		}
		num = append(num, b)
		p.pos++
	}
	switch b {
	case '.':
		num = append(num, b)
		p.pos++
		for p.pos < len(p.buf) {
			b = p.buf[p.pos]
			if b < '0' || '9' < b {
				break
			}
			num = append(num, b)
			p.pos++
		}
		if b == 'e' || b == 'E' {
			p.pos++
			num = append(num, b)
			if len(p.buf) <= p.pos {
				p.raise("expected a number")
			}
			b = p.buf[p.pos]
		} else {
			f, _ := strconv.ParseFloat(string(num), 64)
			return f
		}
	case 'e', 'E':
		p.pos++
		if len(p.buf) <= p.pos {
			p.raise("expected a number")
		}
		num = append(num, b)
		b = p.buf[p.pos]
	default:
		i, err := strconv.ParseInt(string(num), 10, 64)
		if err != nil {
			p.raise(err.Error())
		}
		return i
	}
	if b == '+' || b == '-' {
		num = append(num, b)
		p.pos++
		if len(p.buf) <= p.pos {
			p.raise("expected a number")
		}
	}
	for p.pos < len(p.buf) {
		b = p.buf[p.pos]
		if b < '0' || '9' < b {
			break
		}
		num = append(num, b)
		p.pos++
	}
	f, _ := strconv.ParseFloat(string(num), 64)
	return f
}

func (p *parser) readSlice(i int) Frag {
	if len(p.buf) <= p.pos {
		p.raise("not terminated")
	}
	f := Slice{i}
	b := p.buf[p.pos]
	if b == ']' {
		f = append(f, maxEnd)
		p.pos++
		return f
	}
	b = p.skipSpace()
	// read the end
	if b == ':' {
		f = append(f, maxEnd)
		if len(p.buf) <= p.pos {
			p.raise("not terminated")
		}
		b = p.buf[p.pos]
		p.pos++
		if b != ']' {
			i, b = p.readInt(b)
			f = append(f, i)
		}
	} else {
		i, b = p.readInt(b)
		f = append(f, i)
		if b == ':' {
			if len(p.buf) <= p.pos {
				p.raise("not terminated")
			}
			b = p.buf[p.pos]
			p.pos++
			if b != ']' {
				i, b = p.readInt(b)
				f = append(f, i)
			}
		}
	}
	if b != ']' {
		p.raise("invalid slice syntax")
	}
	return f
}

func (p *parser) readUnion(v any, b byte) Frag {
	if len(p.buf) <= p.pos {
		p.raise("not terminated")
	}
	f := Union{v}
	for {
		switch b {
		case ',':
			// next union member
		case ']':
			return f
		default:
			p.raise("invalid union syntax")
		}
		b = p.skipSpace()
		switch b {
		case '\'', '"':
			var s string
			s = p.readStr(b)
			b = p.skipSpace()
			f = append(f, s)
		case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			var i int
			i, b = p.readInt(b)
			f = append(f, int64(i))
			if b == ' ' {
				b = p.skipSpace()
			}
		default:
			p.raise("invalid union syntax")
		}
	}
}

func (p *parser) readHex(b byte) (i byte) {
	switch b {
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		i = b - '0'
	case 'a', 'b', 'c', 'd', 'e', 'f':
		i = b - 'a' + 10
	case 'A', 'B', 'C', 'D', 'E', 'F':
		i = b - 'A' + 10
	default:
		panic(fmt.Sprintf("0x%02x (%c) is not a valid hexadecimal character", b, b))
	}
	return
}

func (p *parser) readEscStr(start int, term byte) string {
	buf := make([]byte, 0, p.pos-start+16)
	p.pos--
	buf = append(buf, p.buf[start:p.pos]...)
	var b byte
top:
	for p.pos < len(p.buf) {
		b = p.buf[p.pos]
		p.pos++
		switch b {
		case '\\':
			b = p.buf[p.pos]
			p.pos++
			switch b {
			case 'b':
				buf = append(buf, '\b')
			case 't':
				buf = append(buf, '\t')
			case 'n':
				buf = append(buf, '\n')
			case 'f':
				buf = append(buf, '\f')
			case 'r':
				buf = append(buf, '\r')
			case '"':
				buf = append(buf, '"')
			case '\'':
				buf = append(buf, '\'')
			case '\\':
				buf = append(buf, '\\')
			case 'x':
				if len(p.buf) <= p.pos {
					goto fail
				}
				b = p.buf[p.pos]
				p.pos++
				if len(p.buf) <= p.pos {
					goto fail
				}
				i := p.readHex(b)
				b = p.buf[p.pos]
				p.pos++
				i = (i << 4) | p.readHex(b)
				buf = append(buf, i)
			case 'u', 'U':
				var r rune
				for i := 4; 0 < i; i-- {
					if len(p.buf) <= p.pos {
						goto fail
					}
					b = p.buf[p.pos]
					p.pos++
					r = (r << 4) | rune(p.readHex(b))
				}
				buf = utf8.AppendRune(buf, r)
			default:
				goto fail
			}
		case term:
			break top
		default:
			buf = append(buf, b)
		}
	}
	return string(buf)
fail:
	panic(fmt.Sprintf("0x%02x (%c) is not a valid escaped character", b, b))
}

func (p *parser) readStr(term byte) string {
	start := p.pos
	for p.pos < len(p.buf) {
		b := p.buf[p.pos]
		p.pos++
		if b == term {
			break
		}
		if b == '\\' {
			return p.readEscStr(start, term)
		}
	}
	return string(p.buf[start : p.pos-1])
}

func (p *parser) readRegex() *regexp.Regexp {
	start := p.pos
out:
	for p.pos < len(p.buf) {
		b := p.buf[p.pos]
		p.pos++
		switch b {
		case '/':
			break out
		case '\\':
			p.pos++ // skip and then continue
		}
	}
	rx, err := regexp.Compile(string(p.buf[start : p.pos-1]))
	if err != nil {
		p.raise(err.Error())
	}
	return rx
}

func (p *parser) readFilter() *Filter {
	if len(p.buf) <= p.pos {
		p.raise("not terminated")
	}
	eq := precedentCorrect(p.readEq())
	eq = reduceGroups(eq, nil)
	b := p.nextNonSpace()
	if len(p.buf) <= p.pos || b != ']' {
		p.raise("not terminated")
	}
	p.pos++

	return eq.Filter()
}

// Reads an equation by reading the left value first and then seeing if there
// is an operation after that. If so it reads the next equation and decides
// based on precedent which is contained in the other.
func (p *parser) readEq() (eq *Equation) {
	b := p.nextNonSpace()
	switch b {
	case '!':
		p.pos++
		eq = &Equation{o: not, left: p.readEq()}
	case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		p.pos++
		eq = &Equation{result: p.readNum(b)}
	case '\'', '"':
		p.pos++
		s := p.readStr(b)
		eq = &Equation{result: s}
	case '@', '$':
		x := p.readExpr()
		eq = &Equation{result: x}
	case '(':
		p.pos++
		eq = &Equation{left: p.readEq(), o: group}
		b = p.nextNonSpace()
		if b != ')' {
			p.raise("not terminated")
		}
		p.pos++
	case '[':
		eq = &Equation{result: p.readEqList()}
	case '/':
		p.pos++
		eq = &Equation{result: p.readRegex()}
	case 'N':
		p.readEqToken([]byte("Nothing"))
		eq = &Equation{result: Nothing}
	default:
		before := p.pos
		token := p.readToken()
		switch {
		case bytes.Equal([]byte("true"), token):
			eq = &Equation{result: true}
		case bytes.Equal([]byte("false"), token):
			eq = &Equation{result: false}
		case bytes.Equal([]byte("null"), token):
			eq = &Equation{result: nil}
		default:
			if o := opMap[string(token)]; o != nil {
				eq = p.readOpArgs(o)
			} else {
				p.pos = before
				p.raise("'%s' is not a value or function", token)
			}
		}
	}
	for p.pos < len(p.buf) {
		b := p.nextNonSpace()
		switch b {
		case ',': // probably reading array elements or function arguments
			return
		case ')':
			return
		case ']', 0:
			return
		}
		eq = &Equation{left: eq, o: p.readEqOp(), right: p.readEq()}
	}
	return
}

// reads just lower case alpha characters (a-z)
func (p *parser) readToken() []byte {
	start := p.pos
	for ; p.pos < len(p.buf); p.pos++ {
		b := p.buf[p.pos]
		if b < 'a' || 'z' < b {
			break
		}
	}
	return p.buf[start:p.pos]
}

func (p *parser) readOpArgs(o *op) (eq *Equation) {
	if p.buf[p.pos] != '(' {
		p.raise("expected a %s function", o.name)
	}
	eq = &Equation{o: o}
	p.pos++
	eq.left = p.readEq()
	b := p.nextNonSpace()
	if b == ',' {
		p.pos++
		eq.right = p.readEq()
		b = p.nextNonSpace()
	}
	if b != ')' {
		p.raise("not terminated")
	}
	p.pos++
	return
}

func (p *parser) readEqToken(token []byte) {
	for _, t := range token {
		if len(p.buf) <= p.pos || p.buf[p.pos] != t {
			p.raise("expected %s", token)
		}
		p.pos++
	}
}

func (p *parser) readEqList() (list []any) {
	p.pos++
List:
	for p.pos < len(p.buf) {
		eq := p.readEq()
		list = append(list, eq.result)
		b := p.skipSpace()
		switch b {
		case ',':
		case ']':
			break List
		default:
			p.raise("expected a comma or an array close")
		}
	}
	return
}

func partialOp(token []byte, b byte) bool {
	for k := range opMap {
		if len(token) < len(k) && bytes.HasPrefix([]byte(k), token) && b == k[len(token)] {
			return true
		}
	}
	return false
}

func (p *parser) readEqOp() (o *op) {
	var token []byte
	b := p.nextNonSpace()
	for {
		if eqMap[b] != 'o' {
			if len(token) == 0 || !partialOp(token, b) {
				break
			}
		}
		token = append(token, b)
		if b == '-' && 1 < len(token) {
			p.raise("'%s' is not a valid operation", token)
			return
		}
		p.pos++
		if len(p.buf) <= p.pos {
			p.raise("equation not terminated")
		}
		b = p.buf[p.pos]
	}
	o = opMap[string(token)]
	if o == nil {
		p.raise("'%s' is not a valid operation", token)
	}
	return
}

func (p *parser) skipSpace() (b byte) {
	for p.pos < len(p.buf) {
		b = p.buf[p.pos]
		p.pos++
		if b != ' ' {
			break
		}
	}
	return
}

func (p *parser) nextNonSpace() (b byte) {
	for p.pos < len(p.buf) {
		b2 := p.buf[p.pos]
		if b2 != ' ' {
			b = b2
			break
		}
		p.pos++
	}
	return
}

func (p *parser) raise(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	panic(fmt.Errorf("%s at %d in %s", msg, p.pos+1, p.buf))
}
