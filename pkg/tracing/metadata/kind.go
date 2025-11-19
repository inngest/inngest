package metadata

import "strings"

type Kind string

const (
	KindPrefixInngest = "inngest."
	KindPrefixUser    = "user."

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
	return strings.HasPrefix(string(k), KindPrefixUser)
}
