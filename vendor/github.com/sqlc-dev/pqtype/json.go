package pqtype

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// NullRawMessage represents a json.RawMessage that may be null.
// NullRawMessage implements the Scanner interface so
// it can be used as a scan destination, similar to NullString.
type NullRawMessage struct {
	RawMessage json.RawMessage
	Valid      bool // Valid is true if RawMessage is not NULL
}

// Scan implements the Scanner interface.
func (n *NullRawMessage) Scan(src interface{}) error {
	if src == nil {
		n.Valid = false
		return nil
	}
	switch src := src.(type) {
	case []byte:
		srcCopy := make([]byte, len(src))
		copy(srcCopy, src)
		n.RawMessage, n.Valid = srcCopy, true
	default:
		return fmt.Errorf("unsupported Scan, storing driver.Value type %T into type %T", src, []byte{})
	}
	return nil
}

// Value implements the driver Valuer interface.
func (n NullRawMessage) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return []byte(n.RawMessage), nil
}
