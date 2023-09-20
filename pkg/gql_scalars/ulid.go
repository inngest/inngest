package types

import (
	"io"
	"strconv"

	"github.com/99designs/gqlgen/graphql"
	"github.com/oklog/ulid/v2"
	"github.com/pkg/errors"
)

func MarshalULID(id ulid.ULID) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		_, _ = io.WriteString(w, strconv.Quote(id.String()))
	})
}

func UnmarshalULID(v interface{}) (ulid.ULID, error) {
	switch v := v.(type) {
	case string:
		return ulid.Parse(v)
	default:
		return ulid.ULID{}, errors.Errorf("%T is not a ULID", v)
	}
}
