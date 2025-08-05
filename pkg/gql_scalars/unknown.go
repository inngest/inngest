package types

import (
	"encoding/json"
	"io"

	"github.com/99designs/gqlgen/graphql"
)

func MarshalUnknown(v interface{}) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		err := json.NewEncoder(w).Encode(v)
		if err != nil {
			panic(err)
		}
	})
}

func UnmarshalUnknown(v interface{}) (interface{}, error) {
	return v, nil
}
