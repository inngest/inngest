package rueidis

import (
	"bufio"
	"errors"
	"io"
	"strconv"
	"strings"
)

var errChunked = errors.New("unbounded redis message")
var errOldNull = errors.New("RESP2 null")

type reader func(i *bufio.Reader) (RedisMessage, error)

var readers = [256]reader{}

func init() {
	readers['$'] = readBlobString
	readers['+'] = readSimpleString
	readers['-'] = readSimpleString
	readers[':'] = readInteger
	readers['_'] = readNull
	readers[','] = readSimpleString
	readers['#'] = readBoolean
	readers['!'] = readBlobString
	readers['='] = readBlobString
	readers['('] = readSimpleString
	readers['*'] = readArray
	readers['%'] = readMap
	readers['~'] = readArray
	readers['|'] = readMap
	readers['>'] = readArray
	readers['.'] = readNull
}

func readSimpleString(i *bufio.Reader) (m RedisMessage, err error) {
	m.string, err = readS(i)
	return
}

func readBlobString(i *bufio.Reader) (m RedisMessage, err error) {
	m.string, err = readB(i)
	if err == errChunked {
		sb := strings.Builder{}
		for {
			if _, err = i.Discard(1); err != nil { // discard the ';'
				return RedisMessage{}, err
			}
			length, err := readI(i)
			if err != nil {
				return RedisMessage{}, err
			}
			if length == 0 {
				return RedisMessage{string: sb.String()}, nil
			}
			sb.Grow(int(length))
			if _, err = io.CopyN(&sb, i, length); err != nil {
				return RedisMessage{}, err
			}
			if _, err = i.Discard(2); err != nil {
				return RedisMessage{}, err
			}
		}
	}
	return
}

func readInteger(i *bufio.Reader) (m RedisMessage, err error) {
	m.integer, err = readI(i)
	return
}

func readBoolean(i *bufio.Reader) (m RedisMessage, err error) {
	b, err := i.ReadByte()
	if err != nil {
		return RedisMessage{}, err
	}
	if b == 't' {
		m.integer = 1
	}
	_, err = i.Discard(2)
	return
}

func readNull(i *bufio.Reader) (m RedisMessage, err error) {
	_, err = i.Discard(2)
	return
}

func readArray(i *bufio.Reader) (m RedisMessage, err error) {
	length, err := readI(i)
	if err == nil {
		if length == -1 {
			return m, errOldNull
		}
		m.values, err = readA(i, length)
	} else if err == errChunked {
		m.values, err = readE(i)
	}
	return m, err
}

func readMap(i *bufio.Reader) (m RedisMessage, err error) {
	length, err := readI(i)
	if err == nil {
		m.values, err = readA(i, length*2)
	} else if err == errChunked {
		m.values, err = readE(i)
	}
	return m, err
}

func readS(i *bufio.Reader) (string, error) {
	bs, err := i.ReadBytes('\n')
	if err != nil {
		return "", err
	}
	if trim := len(bs) - 2; trim < 0 {
		return "", errors.New(unexpectedNoCRLF)
	} else {
		bs = bs[:trim]
	}
	return BinaryString(bs), nil
}

func readI(i *bufio.Reader) (int64, error) {
	var v int64
	var neg bool
	for {
		c, err := i.ReadByte()
		if err != nil {
			return 0, err
		}
		switch {
		case c >= '0' && c <= '9':
			v = v*10 + int64(c-'0')
		case c == '\r':
			_, err = i.Discard(1)
			if neg {
				return v * -1, err
			}
			return v, err
		case c == '-':
			neg = true
		case c == '?':
			if _, err = i.Discard(2); err == nil {
				err = errChunked
			}
			return 0, err
		default:
			return 0, errors.New(unexpectedNumByte + strconv.Itoa(int(c)))
		}
	}
}

func readB(i *bufio.Reader) (string, error) {
	length, err := readI(i)
	if err != nil {
		return "", err
	}
	if length == -1 {
		return "", errOldNull
	}
	bs := make([]byte, length)
	if _, err = io.ReadFull(i, bs); err != nil {
		return "", err
	}
	if _, err = i.Discard(2); err != nil {
		return "", err
	}
	return BinaryString(bs), nil
}

func readE(i *bufio.Reader) ([]RedisMessage, error) {
	v := make([]RedisMessage, 0)
	for {
		n, err := readNextMessage(i)
		if err != nil {
			return nil, err
		}
		if n.typ == '.' {
			return v, err
		}
		v = append(v, n)
	}
}

func readA(i *bufio.Reader, length int64) (v []RedisMessage, err error) {
	v = make([]RedisMessage, length)
	for n := int64(0); n < length; n++ {
		if v[n], err = readNextMessage(i); err != nil {
			return nil, err
		}
	}
	return v, nil
}

func writeB(o *bufio.Writer, id byte, str string) (err error) {
	_ = writeS(o, id, strconv.Itoa(len(str)))
	_, _ = o.WriteString(str)
	_, err = o.WriteString("\r\n")
	return err
}

func writeS(o *bufio.Writer, id byte, str string) (err error) {
	_ = o.WriteByte(id)
	_, _ = o.WriteString(str)
	_, err = o.WriteString("\r\n")
	return err
}

func readNextMessage(i *bufio.Reader) (m RedisMessage, err error) {
	var attrs *RedisMessage
	var typ byte
	for {
		if typ, err = i.ReadByte(); err != nil {
			return RedisMessage{}, err
		}
		fn := readers[typ]
		if fn == nil {
			return RedisMessage{}, errors.New(unknownMessageType + strconv.Itoa(int(typ)))
		}
		if m, err = fn(i); err != nil {
			if err == errOldNull {
				return RedisMessage{typ: '_'}, nil
			}
			return RedisMessage{}, err
		}
		m.typ = typ
		if m.typ == '|' { // handle the attributes
			a := m     // clone the original m first, and then take address of the clone
			attrs = &a // to avoid go compiler allocating the m on heap which causing worse performance.
			m = RedisMessage{}
			continue
		}
		m.attrs = attrs
		return m, nil
	}
}

func writeCmd(o *bufio.Writer, cmd []string) (err error) {
	err = writeS(o, '*', strconv.Itoa(len(cmd)))
	for _, m := range cmd {
		err = writeB(o, '$', m)
	}
	return err
}

const (
	unexpectedNoCRLF   = "received unexpected simple string message ending without CRLF"
	unexpectedNumByte  = "received unexpected number byte: "
	unknownMessageType = "received unknown message type: "
)
