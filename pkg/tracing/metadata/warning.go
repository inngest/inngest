package metadata

import (
	"encoding/json"
	"errors"

	"github.com/inngest/inngest/pkg/enums"
)

//tygo:generate
const (
	KindInngestWarnings Kind = "inngest.warnings"
)

type WarningError struct {
	Key string
	Err error
}

func (e *WarningError) Error() string {
	return e.Err.Error()
}

//tygo:generate
type Warnings map[string]error

func (wm Warnings) Kind() Kind {
	return KindInngestWarnings
}

func (wm Warnings) Op() enums.MetadataOpcode {
	return enums.MetadataOpcodeMerge
}

func (wm Warnings) Serialize() (Values, error) {
	ret := make(Values)
	for key, warning := range wm {
		ret[key], _ = json.Marshal(warning.Error())
	}

	return ret, nil
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
	warnings := ExtractWarnings(err)
	if len(warnings) != 0 {
		md = append(md, warnings)
	}

	return md
}
