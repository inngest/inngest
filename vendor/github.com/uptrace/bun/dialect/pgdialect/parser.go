package pgdialect

import (
	"bytes"
	"encoding/hex"

	"github.com/uptrace/bun/internal/parser"
)

type pgparser struct {
	parser.Parser
	buf []byte
}

func newParser(b []byte) *pgparser {
	p := new(pgparser)
	p.Reset(b)
	return p
}

func (p *pgparser) ReadLiteral(ch byte) []byte {
	p.Unread()
	lit, _ := p.ReadSep(',')
	return lit
}

func (p *pgparser) ReadUnescapedSubstring(ch byte) ([]byte, error) {
	return p.readSubstring(ch, false)
}

func (p *pgparser) ReadSubstring(ch byte) ([]byte, error) {
	return p.readSubstring(ch, true)
}

func (p *pgparser) readSubstring(ch byte, escaped bool) ([]byte, error) {
	ch, err := p.ReadByte()
	if err != nil {
		return nil, err
	}

	p.buf = p.buf[:0]
	for {
		if ch == '"' {
			break
		}

		next, err := p.ReadByte()
		if err != nil {
			return nil, err
		}

		if ch == '\\' {
			switch next {
			case '\\', '"':
				p.buf = append(p.buf, next)

				ch, err = p.ReadByte()
				if err != nil {
					return nil, err
				}
			default:
				p.buf = append(p.buf, '\\')
				ch = next
			}
			continue
		}

		if escaped && ch == '\'' && next == '\'' {
			p.buf = append(p.buf, next)
			ch, err = p.ReadByte()
			if err != nil {
				return nil, err
			}
			continue
		}

		p.buf = append(p.buf, ch)
		ch = next
	}

	if bytes.HasPrefix(p.buf, []byte("\\x")) && len(p.buf)%2 == 0 {
		data := p.buf[2:]
		buf := make([]byte, hex.DecodedLen(len(data)))
		n, err := hex.Decode(buf, data)
		if err != nil {
			return nil, err
		}
		return buf[:n], nil
	}

	return p.buf, nil
}

func (p *pgparser) ReadRange(ch byte) ([]byte, error) {
	p.buf = p.buf[:0]
	p.buf = append(p.buf, ch)

	for p.Valid() {
		ch = p.Read()
		p.buf = append(p.buf, ch)
		if ch == ']' || ch == ')' {
			break
		}
	}

	return p.buf, nil
}

func (p *pgparser) ReadJSON() ([]byte, error) {
	p.Unread()

	c, err := p.ReadByte()
	if err != nil {
		return nil, err
	}

	p.buf = p.buf[:0]

	depth := 0
	for {
		switch c {
		case '{':
			depth++
		case '}':
			depth--
		}

		p.buf = append(p.buf, c)

		if depth == 0 {
			break
		}

		next, err := p.ReadByte()
		if err != nil {
			return nil, err
		}

		c = next
	}

	return p.buf, nil
}
