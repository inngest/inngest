package metadata

import (
	"encoding/json"
	"errors"
	"maps"
	"slices"
)

//tygo:generate
const (
	KindInngestWarnings Kind = "inngest.warnings"
)

// WarningKind returns the per code warning kind, IE inngest.warning.<code>.
func WarningKind(code string) Kind {
	return Kind(KindPrefixInngestWarning + code)
}

// WarningCode returns the warning code of a per code warning kind. It's false
// for the bare inngest.warnings kind, which keys its warnings by code in the
// values.
func (k Kind) WarningCode() (string, bool) {
	return k.trimPrefix(KindPrefixInngestWarning)
}

type WarningError struct {
	Key string
	Err error
}

func (e *WarningError) Error() string {
	return e.Err.Error()
}

//tygo:generate
type Warnings map[string]error

// Structured returns one metadata write per warning, ordered by code.
func (wm Warnings) Structured() []Structured {
	ret := make([]Structured, 0, len(wm))
	for _, code := range slices.Sorted(maps.Keys(wm)) {
		ret = append(ret, Warning{Code: code, Err: wm[code]})
	}
	return ret
}

// Warning is a single warning, written under its own inngest.warning.<code>
// kind w/ {"<code>": message} so it doesn't replace other warnings.
type Warning struct {
	Code string
	Err  error
}

func (w Warning) Kind() Kind {
	return WarningKind(w.Code)
}

func (w Warning) Serialize() (Values, error) {
	msg, err := json.Marshal(w.Err.Error())
	if err != nil {
		return nil, err
	}
	return Values{w.Code: msg}, nil
}

func ExtractWarnings(err error) Warnings {
	warnings := extractWarnings(err)

	md := make(Warnings)
	for _, warnings := range warnings {
		md[warnings.Key] = warnings.Err
	}

	return md
}

func extractWarnings(err error) []*WarningError {
	var warning *WarningError
	var joinedErr interface{ Unwrap() []error }
	switch {
	case errors.As(err, &joinedErr):
		var ret []*WarningError
		for _, err := range joinedErr.Unwrap() {
			ret = append(ret, extractWarnings(err)...)
		}

		return ret
	case errors.As(err, &warning):
		return []*WarningError{warning}
	default:
		return nil
	}
}

func WithWarnings(md []Structured, err error) []Structured {
	return append(md, ExtractWarnings(err).Structured()...)
}
