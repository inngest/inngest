package metadata

import (
	"errors"
	"strings"
)

type Kind string

var (
	ErrKindTooLong = errors.New("kind exceeds maximum length")
)

const (
	MaxKindLength = 128

	KindPrefixInngest  = "inngest."
	KindPrefixUserland = "userland."

	KindInngestAI       Kind = "inngest.ai"
	KindInngestHTTP     Kind = "inngest.http"
	KindInngestWarnings Kind = "inngest.warnings"
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
