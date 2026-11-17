package pgdialect

import (
	"bytes"
	"fmt"
	"io"
)

type arrayParser struct {
	p pgparser

	elem []byte
	err  error

	isJson bool
}

func newArrayParser(b []byte) *arrayParser {
	p := new(arrayParser)

	if b[0] == 'n' {
		p.p.Reset(nil)
		return p
	}

	if len(b) < 2 || (b[0] != '{' && b[0] != '[') || (b[len(b)-1] != '}' && b[len(b)-1] != ']') {
		p.err = fmt.Errorf("pgdialect: can't parse array: %q", b)
		return p
	}
	p.isJson = b[0] == '['

	p.p.Reset(b[1 : len(b)-1])
	return p
}

func (p *arrayParser) Next() bool {
	if p.err != nil {
		return false
	}
	p.err = p.readNext()
	return p.err == nil
}

func (p *arrayParser) Err() error {
	if p.err != io.EOF {
		return p.err
	}
	return nil
}

func (p *arrayParser) Elem() []byte {
	return p.elem
}

func (p *arrayParser) readNext() error {
	ch := p.p.Read()
	if ch == 0 {
		return io.EOF
	}

	switch ch {
	case '}', ']':
		return io.EOF
	case '"':
		b, err := p.p.ReadSubstring(ch)
		if err != nil {
			return err
		}

		if p.p.Peek() == ',' {
			p.p.Advance()
		}

		p.elem = b
		return nil
	case '[', '(':
		rng, err := p.p.ReadRange(ch)
		if err != nil {
			return err
		}

		if p.p.Peek() == ',' {
			p.p.Advance()
		}

		p.elem = rng
		return nil
	default:
		if ch == '{' && p.isJson {
			json, err := p.p.ReadJSON()
			if err != nil {
				return err
			}

			for {
				if p.p.Peek() == ',' || p.p.Peek() == ' ' {
					p.p.Advance()
				} else {
					break
				}
			}

			p.elem = json
			return nil
		} else {
			lit := p.p.ReadLiteral(ch)
			if bytes.Equal(lit, []byte("NULL")) {
				lit = nil
			}

			if p.p.Peek() == ',' {
				p.p.Advance()
			}

			p.elem = lit
			return nil
		}
	}
}
