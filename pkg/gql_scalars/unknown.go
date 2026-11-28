package types

import (
	"encoding/json"
	"io"

	"github.com/99designs/gqlgen/graphql"
)

// MarshalUnknown writes v as JSON. A value encoding/json rejects (NaN/Inf,
// a channel, a map w/ unsupported keys, ...) is written as null instead of
// panicking mid-response: callers that can produce such values (e.g. the
// Insights resolver, via insights.JSONSafeValue) should convert them first,
// this is just the last line of defense.
func MarshalUnknown(v interface{}) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		raw, err := json.Marshal(v)
		if err != nil {
			raw = []byte("null")
		}
		_, _ = w.Write(raw)
	})
}

func UnmarshalUnknown(v interface{}) (interface{}, error) {
	return v, nil
}
