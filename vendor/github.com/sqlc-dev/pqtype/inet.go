package pqtype

import (
	"database/sql/driver"
	"fmt"
	"net"
)

func decodeIPNet(src []byte) (*net.IPNet, error) {
	if ip := net.ParseIP(string(src)); ip != nil {
		if ipv4 := ip.To4(); ipv4 != nil {
			ip = ipv4
		}
		bitCount := len(ip) * 8
		mask := net.CIDRMask(bitCount, bitCount)
		return &net.IPNet{Mask: mask, IP: ip}, nil
	} else {
		ip, ipnet, err := net.ParseCIDR(string(src))
		if err != nil {
			return nil, err
		}
		if ipv4 := ip.To4(); ipv4 != nil {
			ip = ipv4
		}
		ones, _ := ipnet.Mask.Size()
		return &net.IPNet{IP: ip, Mask: net.CIDRMask(ones, len(ip)*8)}, nil
	}
}

// Inet represents both inet and cidr PostgreSQL types.
type Inet struct {
	IPNet net.IPNet
	Valid bool
}

func (dst *Inet) Scan(src interface{}) error {
	if src == nil {
		*dst = Inet{}
		return nil
	}

	switch src := src.(type) {
	case string:
		ipnet, err := decodeIPNet([]byte(src))
		if err != nil {
			return err
		}
		*dst = Inet{IPNet: *ipnet, Valid: true}
		return nil
	case []byte:
		srcCopy := make([]byte, len(src))
		copy(srcCopy, src)
		ipnet, err := decodeIPNet(srcCopy)
		if err != nil {
			return err
		}
		*dst = Inet{IPNet: *ipnet, Valid: true}
		return nil
	}

	return fmt.Errorf("cannot scan %T", src)
}

func (src Inet) encodeText(buf []byte) ([]byte, error) {
	if src.Valid {
		return append(buf, src.IPNet.String()...), nil
	} else {
		return nil, nil
	}
}

// Value implements the database/sql/driver Valuer interface.
func (src Inet) Value() (driver.Value, error) {
	buf, err := src.encodeText(make([]byte, 0, 32))
	if err != nil {
		return nil, err
	}
	if buf == nil {
		return nil, nil
	}
	return string(buf), err
}
