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

// Scores & warnings get one kind per score name / warning code so each can be
// replaced on its own (IE `inngest.score.<name>`, `inngest.warning.<code>`).
//
//tygo:generate
const KindPrefixInngestScore = "inngest.score."

//tygo:generate
const KindPrefixInngestWarning = "inngest.warning."

const (
	KindInngestExperiment Kind = "inngest.experiment"
)

func (k Kind) String() string {
	return string(k)
}

func (k Kind) Suffix() string {
	switch {
	case strings.HasPrefix(string(k), KindPrefixUserland):
		return string(k[len(KindPrefixUserland):])
	case strings.HasPrefix(string(k), KindPrefixInngest):
		return string(k[len(KindPrefixInngest):])
	default:
		return string(k)
	}
}

func (k Kind) IsInngest() bool {
	return strings.HasPrefix(string(k), KindPrefixInngest)
}

func (k Kind) IsUser() bool {
	return strings.HasPrefix(string(k), KindPrefixUserland)
}

// MaxScoreKindLength fits a max length score name after the score prefix, so
// a name the SDK accepts never makes its kind too long.
const MaxScoreKindLength = len(KindPrefixInngestScore) + MaxScoreNameByteLength

func (k Kind) Validate() error {
	maxLength := MaxKindLength
	if strings.HasPrefix(string(k), KindPrefixInngestScore) {
		maxLength = MaxScoreKindLength
	}
	if len(k) > maxLength {
		return ErrKindTooLong
	}

	return nil
}

// allowedInngestKinds is the set of inngest-prefixed metadata kinds that SDK
// clients are permitted to set. Any inngest.* kind not in this set is rejected
// to prevent spoofing of internal metadata. Per name score & warning kinds
// (inngest.score.<name>, inngest.warning.<code>) are accepted by prefix in
// ValidateAllowed. The bare inngest.score & inngest.warnings kinds hold several
// names/codes in their values map and stay allowed for older SDKs.
//
// Synthetic read-time kinds (inngest.usage, inngest.ai.summary) stay off the
// allowlist so no stored entry of either kind can exist: the AI summary read
// path attaches its computed entry without stripping stored ones first and
// relies on this rejection to stay the only entry of its kind.
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
// restriction. Per name score & warning kinds pass w/ any non empty suffix;
// the score name suffix is checked by validateScoreName along w/ the values.
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
	if _, ok := k.ScoreName(); ok {
		return nil
	}
	if _, ok := k.WarningCode(); ok {
		return nil
	}
	return ErrKindNotAllowed
}

// trimPrefix returns the kind w/o prefix, or false if the kind doesn't have
// the prefix or nothing follows it.
func (k Kind) trimPrefix(prefix string) (string, bool) {
	suffix, ok := strings.CutPrefix(string(k), prefix)
	if !ok || suffix == "" {
		return "", false
	}
	return suffix, true
}
