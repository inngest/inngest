package pgdialect

import (
	"bytes"
	"fmt"
	"io"
)

type hstoreParser struct {
	p pgparser

	key   string
	value string
	err   error
}

func newHStoreParser(b []byte) *hstoreParser {
	p := new(hstoreParser)
	if len(b) != 0 && (len(b) < 6 || b[0] != '"') {
		p.err = fmt.Errorf("pgdialect: can't parse hstore: %q", b)
		return p
	}
	p.p.Reset(b)
	return p
}

func (p *hstoreParser) Next() bool {
	if p.err != nil {
		return false
	}
	p.err = p.readNext()
	return p.err == nil
}

func (p *hstoreParser) Err() error {
	if p.err != io.EOF {
		return p.err
	}
	return nil
}

func (p *hstoreParser) Key() string {
	return p.key
}

func (p *hstoreParser) Value() string {
	return p.value
}

func (p *hstoreParser) readNext() error {
	if !p.p.Valid() {
		return io.EOF
	}

	if err := p.p.Skip('"'); err != nil {
		return err
	}

	key, err := p.p.ReadUnescapedSubstring('"')
	if err != nil {
		return err
	}
	p.key = string(key)

	if err := p.p.SkipPrefix([]byte("=>")); err != nil {
		return err
	}

	ch, err := p.p.ReadByte()
	if err != nil {
		return err
	}

	switch ch {
	case '"':
		value, err := p.p.ReadUnescapedSubstring(ch)
		if err != nil {
			return err
		}
		p.skipComma()
		p.value = string(value)
		return nil
	default:
		value := p.p.ReadLiteral(ch)
		if bytes.Equal(value, []byte("NULL")) {
			p.value = ""
		}
		p.skipComma()
		return nil
	}
}

func (p *hstoreParser) skipComma() {
	if p.p.Peek() == ',' {
		p.p.Advance()
	}
	if p.p.Peek() == ' ' {
		p.p.Advance()
	}
}
