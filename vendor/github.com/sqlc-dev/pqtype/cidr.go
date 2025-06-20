package pqtype

import (
	"database/sql/driver"
	"fmt"
)

type CIDR Inet

func (dst *CIDR) Scan(src interface{}) error {
	if src == nil {
		*dst = CIDR{}
		return nil
	}

	switch src := src.(type) {
	case string:
		ipnet, err := decodeIPNet([]byte(src))
		if err != nil {
			return err
		}
		*dst = CIDR{IPNet: *ipnet, Valid: true}
		return nil
	case []byte:
		srcCopy := make([]byte, len(src))
		copy(srcCopy, src)
		ipnet, err := decodeIPNet(srcCopy)
		if err != nil {
			return err
		}
		*dst = CIDR{IPNet: *ipnet, Valid: true}
		return nil
	}

	return fmt.Errorf("cannot scan %T", src)
}

func (src CIDR) Value() (driver.Value, error) {
	return Inet(src).Value()
}
