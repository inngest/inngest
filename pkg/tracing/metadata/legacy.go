package metadata

import (
	"encoding/json"
	"maps"
	"slices"
)

// KindValues is the values written for a single metadata kind.
type KindValues struct {
	Kind   Kind
	Values Values
}

// SplitLegacy splits a legacy multi key write into one write per key, since
// every write is a full replace of its kind:
//
//   - inngest.score w/ {"<name>": {"value": v}} -> inngest.score.<name> w/ {"value": v}
//   - inngest.warnings w/ {"<code>": msg} -> inngest.warning.<code> w/ {"<code>": msg}
//
// Writes are ordered by key. ok is false for every other kind, which is
// written as is.
func SplitLegacy(kind Kind, values Values) (writes []KindValues, ok bool) {
	switch kind {
	case KindInngestScore:
		writes = make([]KindValues, 0, len(values))
		for _, name := range slices.Sorted(maps.Keys(values)) {
			writes = append(writes, KindValues{
				Kind:   ScoreKind(name),
				Values: legacyScoreValues(values[name]),
			})
		}
		return writes, true
	case KindInngestWarnings:
		writes = make([]KindValues, 0, len(values))
		for _, code := range slices.Sorted(maps.Keys(values)) {
			writes = append(writes, KindValues{
				Kind:   WarningKind(code),
				Values: Values{code: values[code]},
			})
		}
		return writes, true
	default:
		return nil, false
	}
}

// legacyScoreValues unwraps a legacy {"value": v} score entry into the values
// of its per name kind. Validated writes are always an object, anything else
// is kept whole under "value".
func legacyScoreValues(raw json.RawMessage) Values {
	var values Values
	if err := json.Unmarshal(raw, &values); err != nil || values == nil {
		return Values{"value": raw}
	}
	return values
}
