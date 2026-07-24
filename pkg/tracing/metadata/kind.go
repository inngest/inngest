package metadata

import (
	"errors"
	"strings"
)

type Kind string

var (
	ErrKindTooLong    = errors.New("kind exceeds maximum length")
	ErrKindNotAllowed = errors.New("inngest-prefixed kind is not in the allowlist")
)

const (
	MaxKindLength = 128

	KindPrefixInngest  = "inngest."
	KindPrefixUserland = "userland."
)

const (
	KindInngestExperiment Kind = "inngest.experiment"
)

func (k Kind) String() string {
	return string(k)
}

func (k Kind) IsInngest() bool {
	return strings.HasPrefix(string(k), KindPrefixInngest)
}

func (k Kind) IsUser() bool {
	return strings.HasPrefix(string(k), KindPrefixUserland)
}

func (k Kind) Validate() error {
	if len(k) > MaxKindLength {
		return ErrKindTooLong
	}

	return nil
}

// allowedInngestKinds is the set of inngest-prefixed metadata kinds that SDK
// clients are permitted to set. Any inngest.* kind not in this set is rejected
// to prevent spoofing of internal metadata. The score kind is the bare
// constant inngest.score; the user-supplied score name is a key in the values
// map, not a kind suffix.
var allowedInngestKinds = map[Kind]bool{
	"inngest.ai":               true,
	"inngest.http":             true,
	"inngest.http.timing":      true,
	"inngest.response_headers": true,
	"inngest.warnings":         true,
	KindInngestExperiment:      true,
	KindInngestScore:           true,
}

// ValidateAllowed checks that the kind is valid and, if it uses the inngest.*
// prefix, that it belongs to the allowlist. Userland kinds pass without
// restriction. The score kind inngest.score is allowlisted like the others;
// the user-supplied score name is a key in the values map (validated by
// validateScoreName), not a kind suffix.
func (k Kind) ValidateAllowed() error {
	if err := k.Validate(); err != nil {
		return err
	}
	if !k.IsInngest() {
		return nil
	}
	if allowedInngestKinds[k] {
		return nil
	}
	return ErrKindNotAllowed
}
