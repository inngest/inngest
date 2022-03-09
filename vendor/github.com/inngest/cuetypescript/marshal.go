package cuetypescript

import (
	"cuelang.org/go/cue"
	"github.com/inngest/cuetypescript/generation"
)

// MarshalString marshals a Cue string into a Typescript type string,
// returning an error.
func MarshalString(cuestr string) (string, error) {
	return generation.MarshalString(cuestr)
}

// MarshalCueValue returns a typescript type given a cue value.
func MarshalCueValue(v cue.Value) (string, error) {
	return generation.MarshalCueValue(v)
}
