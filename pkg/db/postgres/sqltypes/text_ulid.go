package sqltypes

import (
	"database/sql/driver"

	"github.com/oklog/ulid/v2"
)

//
// NOTE
// This file exists because some of the ID columns in the tables that are
// supposed to be storing ULID data are not in binary types but in text instead.
// So unfortunately, we need a wrapper to handle the conversion since the ULID
// package assume the data is in binary format already.
//

// TextULID stores a ULID in its canonical 26-character text form.
// Use it only for text-backed ULID columns.
type TextULID ulid.ULID

func FromULID(id ulid.ULID) TextULID {
	return TextULID(id)
}

func (id TextULID) ULID() ulid.ULID {
	return ulid.ULID(id)
}

func (id TextULID) String() string {
	return ulid.ULID(id).String()
}

func (id TextULID) Value() (driver.Value, error) {
	return id.String(), nil
}

func (id *TextULID) Scan(src interface{}) error {
	var parsed ulid.ULID
	if err := (&parsed).Scan(src); err != nil {
		return err
	}
	*id = TextULID(parsed)
	return nil
}
