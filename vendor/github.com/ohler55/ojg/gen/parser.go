// Copyright (c) 2020, Peter Ohler, All rights reserved.

package gen

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"unicode/utf8"
)

const (
	stackInitSize = 32 // for container stack { or [
	tmpInitSize   = 32 // for tokens and numbers
	mapInitSize   = 8
	readBufSize   = 4096
)

// Parser is a reusable JSON parser. It can be reused for multiple parsings
// which allows buffer reuse for a performance advantage.
type Parser struct {
	tmp        []byte // used for numbers and strings
	runeBytes  []byte
	stack      []Node
	starts     []int
	maps       []Object
	cb         func(Node)
	resultChan chan Node
	line       int
	noff       int // Offset of last newline from start of buf. Can be negative when using a reader.
	ri         int // read index for null, false, and true
	mi         int
	num        Number
	rn         rune
	result     Node
	mode       string
	nextMode   string

	// OnlyOne returns an error if more than one JSON is in the string or stream.
	OnlyOne bool

	// Reuse maps. Previously returned maps will no longer be valid or rather
	// could be modified during parsing.
	Reuse bool
}

// Parse a JSON string in to simple types. An error is returned if not valid JSON.
func (p *Parser) Parse(buf []byte, args ...any) (Node, error) {
	p.cb = nil
	p.resultChan = nil
	p.OnlyOne = true
	for _, a := range args {
		switch ta := a.(type) {
		case func(Node) bool:
			p.cb = func(x Node) { _ = ta(x) }
			p.OnlyOne = false
		case func(Node):
			p.cb = ta
			p.OnlyOne = false
		case chan Node:
			p.resultChan = ta
			p.OnlyOne = false
			p.Reuse = false
		default:
			return nil, fmt.Errorf("a %T is not a valid option type", a)
		}
	}
	if p.stack == nil {
		p.stack = make([]Node, 0, stackInitSize)
		p.tmp = make([]byte, 0, tmpInitSize)
		p.starts = make([]int, 0, 16)
		p.maps = make([]Object, 0, 16)
	} else {
		p.stack = p.stack[:0]
		p.tmp = p.tmp[:0]
		p.starts = p.starts[:0]
	}
	p.result = nil
	p.noff = -1
	p.line = 1
	p.mode = valueMap
	p.mi = 0
	var err error
	// Skip BOM if present.
	if 3 < len(buf) && buf[0] == 0xEF {
		if buf[1] == 0xBB && buf[2] == 0xBF {
			err = p.parseBuffer(buf[3:], true)
		} else {
			return nil, fmt.Errorf("expected BOM at 1:3")
		}
	} else {
		err = p.parseBuffer(buf, true)
	}
	p.stack = p.stack[:cap(p.stack)]
	for i := len(p.stack) - 1; 0 <= i; i-- {
		p.stack[i] = nil
	}
	p.stack = p.stack[:0]

	return p.result, err
}

// ParseReader a JSON io.Reader. An error is returned if not valid JSON.
func (p *Parser) ParseReader(r io.Reader, args ...any) (data Node, err error) {
	p.cb = nil
	p.resultChan = nil
	p.OnlyOne = true
	for _, a := range args {
		switch ta := a.(type) {
		case func(Node) bool:
			p.cb = func(x Node) { _ = ta(x) }
			p.OnlyOne = false
		case func(Node):
			p.cb = ta
			p.OnlyOne = false
		case chan Node:
			p.resultChan = ta
			p.OnlyOne = false
			p.Reuse = false
		default:
			return nil, fmt.Errorf("a %T is not a valid option type", a)
		}
	}
	if p.stack == nil {
		p.stack = make([]Node, 0, stackInitSize)
		p.tmp = make([]byte, 0, tmpInitSize)
		p.starts = make([]int, 0, 16)
		p.maps = make([]Object, 0, 16)
	} else {
		p.stack = p.stack[:0]
		p.tmp = p.tmp[:0]
		p.starts = p.starts[:0]
	}
	p.result = nil
	p.noff = -1
	p.line = 1
	p.mi = 0
	buf := make([]byte, readBufSize)
	eof := false
	var cnt int
	cnt, err = r.Read(buf)
	buf = buf[:cnt]
	p.mode = valueMap
	if err != nil {
		if !errors.Is(err, io.EOF) {
			return
		}
		eof = true
	}
	var skip int
	// Skip BOM if present.
	if 3 < len(buf) && buf[0] == 0xEF && buf[1] == 0xBB && buf[2] == 0xBF {
		skip = 3
	}
	for {
		if 0 < skip {
			err = p.parseBuffer(buf[skip:], eof)
		} else {
			err = p.parseBuffer(buf, eof)
		}
		if err != nil {
			p.stack = p.stack[:cap(p.stack)]
			for i := len(p.stack) - 1; 0 <= i; i-- {
				p.stack[i] = nil
			}
			p.stack = p.stack[:0]

			return
		}
		skip = 0
		if eof {
			break
		}
		buf = buf[:cap(buf)]
		cnt, err = r.Read(buf)
		buf = buf[:cnt]
		if err != nil {
			if !errors.Is(err, io.EOF) {
				return
			}
			eof = true
		}
	}
	data = p.result

	return
}

func (p *Parser) parseBuffer(buf []byte, last bool) error {
	var b byte
	var i int
	var off int
	depth := len(p.starts)
	for off = 0; off < len(buf); off++ {
		b = buf[off]
		switch p.mode[b] {
		case skipNewline:
			p.line++
			p.noff = off
			for i, b = range buf[off+1:] {
				if spaceMap[b] != skipChar {
					break
				}
			}
			off += i
			continue
		case colonColon:
			p.mode = valueMap
			continue
		case skipChar: // skip and continue
			continue
		case strOk:
			p.tmp = append(p.tmp, b)
		case keyQuote:
			start := off + 1
			if len(buf) <= start {
				p.tmp = p.tmp[:0]
				p.mode = stringMap
				p.nextMode = colonMap
				continue
			}
			for i, b = range buf[off+1:] {
				if stringMap[b] != strOk {
					break
				}
			}
			off += i
			if b == '"' {
				off++
				p.stack = append(p.stack, Key(buf[start:off]))
				p.mode = colonMap
			} else {
				p.tmp = p.tmp[:0]
				p.tmp = append(p.tmp, buf[start:off+1]...)
				p.mode = stringMap
				p.nextMode = colonMap
			}
			continue
		case afterComma:
			if 0 < len(p.starts) && p.starts[len(p.starts)-1] == -1 {
				p.mode = keyMap
			} else {
				p.mode = commaMap
			}
			continue
		case valQuote:
			start := off + 1
			if len(buf) <= start {
				p.tmp = p.tmp[:0]
				p.mode = stringMap
				p.nextMode = afterMap
				continue
			}
			for i, b = range buf[off+1:] {
				if stringMap[b] != strOk {
					break
				}
			}
			off += i
			if b == '"' {
				off++
				p.add(String(buf[start:off]))
				p.mode = afterMap
			} else {
				p.tmp = p.tmp[:0]
				p.tmp = append(p.tmp, buf[start:off+1]...)
				p.mode = stringMap
				p.nextMode = afterMap
				continue
			}
		case numComma:
			p.add(p.num.AsNode())
			if 0 < len(p.starts) && p.starts[len(p.starts)-1] == -1 {
				p.mode = keyMap
			} else {
				p.mode = commaMap
			}
		case strSlash:
			p.mode = escMap
			continue
		case escOk:
			p.tmp = append(p.tmp, escByteMap[b])
			p.mode = stringMap
			continue
		case openObject:
			p.starts = append(p.starts, -1)
			p.mode = key1Map
			var m Object
			if p.Reuse {
				if p.mi < len(p.maps) {
					m = p.maps[p.mi]
					for k := range m {
						delete(m, k)
					}
				} else {
					m = make(Object, mapInitSize)
					p.maps = append(p.maps, m)
				}
				p.mi++
			} else {
				m = make(Object, mapInitSize)
			}
			p.stack = append(p.stack, m)
			depth++
			continue
		case closeObject:
			depth--
			if depth < 0 || 0 <= p.starts[depth] {
				return p.newError(off, "unexpected object close")
			}
			if 256 < len(p.mode) && p.mode[256] == 'n' {
				p.add(p.num.AsNode())
			}
			p.starts = p.starts[0:depth]
			n := p.stack[len(p.stack)-1]
			p.stack = p.stack[:len(p.stack)-1]
			p.add(n)
			p.mode = afterMap
		case val0:
			p.mode = zeroMap
			p.num.Reset()
		case valDigit:
			p.num.Reset()
			p.mode = digitMap
			p.num.I = uint64(b - '0')
			for i, b = range buf[off+1:] {
				if digitMap[b] != numDigit {
					break
				}
				if BigLimit <= p.num.I {
					p.num.FillBig()
					p.num.AddDigit(b)
					break
				}
				p.num.I = p.num.I*10 + uint64(b-'0')
			}
			if digitMap[b] == numDigit {
				off++
			}
			off += i
		case valNeg:
			p.mode = negMap
			p.num.Reset()
			p.num.Neg = true
			continue
		case escU:
			p.mode = uMap
			p.rn = 0
			p.ri = 0
			continue
		case openArray:
			p.starts = append(p.starts, len(p.stack))
			p.stack = append(p.stack, EmptyArray)
			p.mode = valueMap
			depth++
			continue
		case closeArray:
			depth--
			if depth < 0 || p.starts[depth] < 0 {
				return p.newError(off, "unexpected array close")
			}
			// Only modes with a close array are value, after, and numbers
			// which are all over 256 long.
			if p.mode[256] == 'n' {
				p.add(p.num.AsNode())
			}
			start := p.starts[len(p.starts)-1] + 1
			p.starts = p.starts[:len(p.starts)-1]
			size := len(p.stack) - start
			n := make(Array, size)
			copy(n, p.stack[start:len(p.stack)])
			p.stack = p.stack[0 : start-1]
			p.add(n)
			p.mode = afterMap
		case valNull:
			if off+4 <= len(buf) && string(buf[off:off+4]) == "null" {
				off += 3
				p.mode = afterMap
				p.add(nil)
			} else {
				p.mode = nullMap
				p.ri = 0
			}
		case valTrue:
			if off+4 <= len(buf) && string(buf[off:off+4]) == "true" {
				off += 3
				p.mode = afterMap
				p.add(True)
			} else {
				p.mode = trueMap
				p.ri = 0
			}
		case valFalse:
			if off+5 <= len(buf) && string(buf[off:off+5]) == "false" {
				off += 4
				p.mode = afterMap
				p.add(False)
			} else {
				p.mode = falseMap
				p.ri = 0
			}
		case numDot:
			if 0 < len(p.num.BigBuf) {
				p.num.BigBuf = append(p.num.BigBuf, b)
				p.mode = dotMap
				continue
			}
			for i, b = range buf[off+1:] {
				if digitMap[b] != numDigit {
					break
				}
				p.num.Frac = p.num.Frac*10 + uint64(b-'0')
				p.num.Div *= 10.0
				if BigLimit <= p.num.Div {
					p.num.FillBig()
					break
				}
			}
			off += i
			if digitMap[b] == numDigit {
				off++
			}
			p.mode = fracMap
		case numFrac:
			p.num.AddFrac(b)
			p.mode = fracMap
		case fracE:
			if 0 < len(p.num.BigBuf) {
				p.num.BigBuf = append(p.num.BigBuf, b)
			}
			p.mode = expSignMap
			continue
		case strQuote:
			p.mode = p.nextMode
			if p.mode[':'] == colonColon {
				p.stack = append(p.stack, Key(p.tmp))
			} else {
				p.add(String(p.tmp))
			}
		case numZero:
			p.mode = zeroMap
		case numDigit:
			p.num.AddDigit(b)
		case negDigit:
			p.num.AddDigit(b)
			p.mode = digitMap
		case numSpc:
			p.add(p.num.AsNode())
			p.mode = afterMap
		case numNewline:
			p.add(p.num.AsNode())
			p.line++
			p.noff = off
			p.mode = afterMap
			for i, b = range buf[off+1:] {
				if spaceMap[b] != skipChar {
					break
				}
			}
			off += i
		case expSign:
			p.mode = expZeroMap
			if b == '-' {
				p.num.NegExp = true
			}
			continue
		case expDigit:
			p.num.AddExp(b)
			p.mode = expMap
		case uOk:
			p.ri++
			switch b {
			case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
				p.rn = p.rn<<4 | rune(b-'0')
			case 'a', 'b', 'c', 'd', 'e', 'f':
				p.rn = p.rn<<4 | rune(b-'a'+10)
			case 'A', 'B', 'C', 'D', 'E', 'F':
				p.rn = p.rn<<4 | rune(b-'A'+10)
			}
			if p.ri == 4 {
				if len(p.runeBytes) < 6 {
					p.runeBytes = make([]byte, 6)
				}
				n := utf8.EncodeRune(p.runeBytes, p.rn)
				p.tmp = append(p.tmp, p.runeBytes[:n]...)
				p.mode = stringMap
			}
			continue
		case tokenOk:
			switch {
			case p.mode['r'] == tokenOk:
				p.ri++
				if "true"[p.ri] != b {
					return p.newError(off, "expected true")
				}
				if 3 <= p.ri {
					p.add(True)
					p.mode = afterMap
				}
			case p.mode['a'] == tokenOk:
				p.ri++
				if "false"[p.ri] != b {
					return p.newError(off, "expected false")
				}
				if 4 <= p.ri {
					p.add(False)
					p.mode = afterMap
				}
			case p.mode['u'] == tokenOk && p.mode['l'] == tokenOk:
				p.ri++
				if "null"[p.ri] != b {
					return p.newError(off, "expected null")
				}
				if 3 <= p.ri {
					p.add(nil)
					p.mode = afterMap
				}
			}
		case charErr:
			return p.byteError(off, p.mode, b, bytes.Runes(buf[off:])[0])
		}
		if depth == 0 && 256 < len(p.mode) && p.mode[256] == 'a' {
			if p.cb == nil && p.resultChan == nil {
				p.result = p.stack[0]
			} else {
				if p.cb != nil {
					p.cb(p.stack[0])
				}
				if p.resultChan != nil {
					p.resultChan <- p.stack[0]
				}
			}
			p.stack = p.stack[:0]
			p.mi = 0
			if p.OnlyOne {
				p.mode = spaceMap
			} else {
				p.mode = valueMap
			}
		}
	}
	if last {
		if len(p.mode) == 256 { // valid finishing maps are one byte longer
			return p.newError(off, "incomplete JSON")
		}
		if p.mode[256] == 'n' {
			p.add(p.num.AsNode())
			if p.cb == nil && p.resultChan == nil {
				p.result = p.stack[0]
			} else {
				if p.cb != nil {
					p.cb(p.stack[0])
				}
				if p.resultChan != nil {
					p.resultChan <- p.stack[0]
				}
			}
		}
	}
	return nil
}

func (p *Parser) add(n Node) {
	if 2 <= len(p.stack) {
		if k, ok := p.stack[len(p.stack)-1].(Key); ok {
			obj, _ := p.stack[len(p.stack)-2].(Object)
			obj[string(k)] = n
			p.stack = p.stack[0 : len(p.stack)-1]

			return
		}
	}
	p.stack = append(p.stack, n)
}

func (p *Parser) newError(off int, format string, args ...any) error {
	return &ParseError{
		Message: fmt.Sprintf(format, args...),
		Line:    p.line,
		Column:  off - p.noff,
	}
}

func (p *Parser) byteError(off int, mode string, b byte, r rune) error {
	err := &ParseError{
		Line:   p.line,
		Column: off - p.noff,
	}
	switch mode {
	case nullMap:
		err.Message = "expected null"
	case trueMap:
		err.Message = "expected true"
	case falseMap:
		err.Message = "expected false"
	case afterMap:
		err.Message = fmt.Sprintf("expected a comma or close, not '%c'", r)
	case key1Map:
		err.Message = fmt.Sprintf("expected a string start or object close, not '%c'", r)
	case keyMap:
		err.Message = fmt.Sprintf("expected a string start, not '%c'", r)
	case colonMap:
		err.Message = fmt.Sprintf("expected a colon, not '%c'", r)
	case negMap, zeroMap, digitMap, dotMap, fracMap, expSignMap, expZeroMap, expMap:
		err.Message = "invalid number"
	case stringMap:
		err.Message = fmt.Sprintf("invalid JSON character 0x%02x", b)
	case escMap:
		err.Message = fmt.Sprintf("invalid JSON escape character '\\%c'", r)
	case uMap:
		err.Message = fmt.Sprintf("invalid JSON unicode character '%c'", r)
	case spaceMap:
		err.Message = fmt.Sprintf("extra characters after close, '%c'", r)
	default:
		err.Message = fmt.Sprintf("unexpected character '%c'", r)
	}
	return err
}
