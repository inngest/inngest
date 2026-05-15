package metadata

import (
	"errors"
	"strings"
)

type Kind string

var (
	ErrKindTooLong       = errors.New("kind exceeds maximum length")
	ErrKindNotAllowed    = errors.New("inngest-prefixed kind is not in the allowlist")
)

const (
	MaxKindLength = 128

	KindPrefixInngest  = "inngest."
	KindPrefixUserland = "userland."
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
// to prevent spoofing of internal metadata.
var allowedInngestKinds = map[Kind]bool{
	"inngest.ai":               true,
	"inngest.http":             true,
	"inngest.http.timing":      true,
	"inngest.response_headers": true,
	"inngest.warnings":         true,
	"inngest.experiment":       true,
}

// ValidateAllowed checks that the kind is valid and, if it uses the inngest.*
// prefix, that it belongs to the allowlist. Userland kinds pass without
// restriction.
func (k Kind) ValidateAllowed() error {
	if err := k.Validate(); err != nil {
		return err
	}
	if k.IsInngest() && !allowedInngestKinds[k] {
		return ErrKindNotAllowed
	}
	return nil
}
