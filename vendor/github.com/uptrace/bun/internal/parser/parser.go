package parser

import (
	"bytes"
	"fmt"
	"io"
	"strconv"

	"github.com/uptrace/bun/internal"
)

type Parser struct {
	b []byte
	i int
}

func New(b []byte) *Parser {
	return &Parser{
		b: b,
	}
}

func NewString(s string) *Parser {
	return New(internal.Bytes(s))
}

func (p *Parser) Reset(b []byte) {
	p.b = b
	p.i = 0
}

func (p *Parser) Valid() bool {
	return p.i < len(p.b)
}

func (p *Parser) Remaining() []byte {
	return p.b[p.i:]
}

func (p *Parser) ReadByte() (byte, error) {
	if p.Valid() {
		ch := p.b[p.i]
		p.Advance()
		return ch, nil
	}
	return 0, io.ErrUnexpectedEOF
}

func (p *Parser) Read() byte {
	if p.Valid() {
		ch := p.b[p.i]
		p.Advance()
		return ch
	}
	return 0
}

func (p *Parser) Unread() {
	if p.i > 0 {
		p.i--
	}
}

func (p *Parser) Peek() byte {
	if p.Valid() {
		return p.b[p.i]
	}
	return 0
}

func (p *Parser) Advance() {
	p.i++
}

func (p *Parser) Skip(skip byte) error {
	ch := p.Peek()
	if ch == skip {
		p.Advance()
		return nil
	}
	return fmt.Errorf("got %q, wanted %q", ch, skip)
}

func (p *Parser) SkipPrefix(skip []byte) error {
	if !bytes.HasPrefix(p.b[p.i:], skip) {
		return fmt.Errorf("got %q, wanted prefix %q", p.b, skip)
	}
	p.i += len(skip)
	return nil
}

func (p *Parser) CutPrefix(skip []byte) bool {
	if !bytes.HasPrefix(p.b[p.i:], skip) {
		return false
	}
	p.i += len(skip)
	return true
}

func (p *Parser) ReadSep(sep byte) ([]byte, bool) {
	ind := bytes.IndexByte(p.b[p.i:], sep)
	if ind == -1 {
		b := p.b[p.i:]
		p.i = len(p.b)
		return b, false
	}

	b := p.b[p.i : p.i+ind]
	p.i += ind + 1
	return b, true
}

func (p *Parser) ReadIdentifier() (string, bool) {
	if p.i < len(p.b) && p.b[p.i] == '(' {
		s := p.i + 1
		if ind := bytes.IndexByte(p.b[s:], ')'); ind != -1 {
			b := p.b[s : s+ind]
			if isIdent(b) {
				p.i = s + ind + 1
				return internal.String(b), false
			}
		}
	}

	ind := len(p.b) - p.i
	var alpha bool
	for i, c := range p.b[p.i:] {
		if isNum(c) {
			continue
		}
		if isAlpha(c) || (i > 0 && alpha && c == '_') {
			alpha = true
			continue
		}
		ind = i
		break
	}
	if ind == 0 {
		return "", false
	}
	b := p.b[p.i : p.i+ind]
	p.i += ind
	return internal.String(b), !alpha
}

func (p *Parser) ReadNumber() int {
	ind := len(p.b) - p.i
	for i, c := range p.b[p.i:] {
		if !isNum(c) {
			ind = i
			break
		}
	}
	if ind == 0 {
		return 0
	}
	n, err := strconv.Atoi(string(p.b[p.i : p.i+ind]))
	if err != nil {
		panic(err)
	}
	p.i += ind
	return n
}

func isNum(c byte) bool {
	return c >= '0' && c <= '9'
}

func isAlpha(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// isIdent reports whether b is a valid identifier consisting of
// letters, digits, and underscores, with at least one letter.
func isIdent(b []byte) bool {
	if len(b) == 0 {
		return false
	}
	hasAlpha := false
	for i, c := range b {
		if isAlpha(c) {
			hasAlpha = true
			continue
		}
		if isNum(c) {
			continue
		}
		if c == '_' && i > 0 {
			continue
		}
		return false
	}
	return hasAlpha
}
