package pqtype

import (
	"database/sql/driver"
	"fmt"
	"net"
)

type Macaddr struct {
	Addr  net.HardwareAddr
	Valid bool
}

func (dst *Macaddr) decodeText(src []byte) error {
	if src == nil {
		*dst = Macaddr{}
		return nil
	}

	addr, err := net.ParseMAC(string(src))
	if err != nil {
		return err
	}

	*dst = Macaddr{Addr: addr, Valid: true}
	return nil
}

func (src Macaddr) encodeText(buf []byte) ([]byte, error) {
	if src.Valid {
		return append(buf, src.Addr.String()...), nil
	} else {
		return nil, nil
	}
}

// Scan implements the database/sql Scanner interface.
func (dst *Macaddr) Scan(src interface{}) error {
	if src == nil {
		*dst = Macaddr{}
		return nil
	}

	switch src := src.(type) {
	case string:
		return dst.decodeText([]byte(src))
	case []byte:
		srcCopy := make([]byte, len(src))
		copy(srcCopy, src)
		return dst.decodeText(srcCopy)
	}

	return fmt.Errorf("cannot scan %T", src)
}

// Value implements the database/sql/driver Valuer interface.
func (src Macaddr) Value() (driver.Value, error) {
	buf, err := src.encodeText(make([]byte, 0, 32))
	if err != nil {
		return nil, err
	}
	if buf == nil {
		return nil, nil
	}
	return string(buf), err
}
