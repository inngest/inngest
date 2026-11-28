package pgdialect

import (
	"database/sql/driver"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/uptrace/bun/dialect"
)

func appendElem(buf []byte, val any) []byte {
	switch val := val.(type) {
	case int64:
		return strconv.AppendInt(buf, val, 10)
	case float64:
		return arrayAppendFloat64(buf, val)
	case bool:
		return dialect.AppendBool(buf, val)
	case []byte:
		return appendBytesElem(buf, val)
	case string:
		return appendStringElem(buf, val)
	case time.Time:
		buf = append(buf, '"')
		buf = appendTime(buf, val)
		buf = append(buf, '"')
		return buf
	case driver.Valuer:
		val2, err := val.Value()
		if err != nil {
			err := fmt.Errorf("pgdialect: can't append elem value: %w", err)
			return dialect.AppendError(buf, err)
		}
		return appendElem(buf, val2)
	default:
		err := fmt.Errorf("pgdialect: can't append elem %T", val)
		return dialect.AppendError(buf, err)
	}
}

func appendBytesElem(b []byte, bs []byte) []byte {
	if bs == nil {
		return dialect.AppendNull(b)
	}

	b = append(b, `"\\x`...)

	s := len(b)
	b = append(b, make([]byte, hex.EncodedLen(len(bs)))...)
	hex.Encode(b[s:], bs)

	b = append(b, '"')

	return b
}

func appendStringElem(b []byte, s string) []byte {
	b = append(b, '"')
	for _, r := range s {
		switch r {
		case 0:
			// ignore
		case '\'':
			b = append(b, "''"...)
		case '"':
			b = append(b, '\\', '"')
		case '\\':
			b = append(b, '\\', '\\')
		default:
			if r < utf8.RuneSelf {
				b = append(b, byte(r))
				break
			}
			l := len(b)
			if cap(b)-l < utf8.UTFMax {
				b = append(b, make([]byte, utf8.UTFMax)...)
			}
			n := utf8.EncodeRune(b[l:l+utf8.UTFMax], r)
			b = b[:l+n]
		}
	}
	b = append(b, '"')
	return b
}
