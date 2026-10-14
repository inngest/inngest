package metadata

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

//tygo:generate
const (
	KindInngestScore Kind = "inngest.score"
)

const (
	MaxScoreNameByteLength = 128
)

// ScoreKind returns the per name score kind, IE inngest.score.<name>.
func ScoreKind(name string) Kind {
	return Kind(KindPrefixInngestScore + name)
}

// ScoreName returns the score name of a per name score kind. It's false for
// the bare inngest.score kind, which keys its scores by name in the values.
func (k Kind) ScoreName() (string, bool) {
	return k.trimPrefix(KindPrefixInngestScore)
}

// validateScoreName checks a user-supplied score name (an inngest.score.<name>
// kind suffix, or a key in the inngest.score values map) for characters that downstream consumers can't
// safely round-trip. Mirrors the SDK validation and the monorepo
// MetricKeyRegex: rejects control characters (0x00-0x1F, 0x7F) and the single
// quote (which would silently drop in cloud variant aggregation because
// MetricKeyRegex excludes it for SQL-injection defense).
func validateScoreName(name string) error {
	if len(name) > MaxScoreNameByteLength {
		return fmt.Errorf("invalid score name %q: %w of %d UTF-8 bytes", name, ErrScoreNameTooLong, MaxScoreNameByteLength)
	}

	for _, r := range name {
		if r < 0x20 || r == 0x7f || r == '\'' {
			return fmt.Errorf("invalid score name %q: %w", name, ErrScoreNameInvalid)
		}
	}
	return nil
}

// validateScoreKindValues applies the value-shape rules for a per name
// inngest.score.<name> kind: the values hold a single "value" field w/ a finite
// number or boolean.
func validateScoreKindValues(name string, values Values) error {
	if err := validateScoreName(name); err != nil {
		return fmt.Errorf("invalid score value: %w", err)
	}

	raw, ok := values["value"]
	if !ok || len(values) != 1 {
		return fmt.Errorf("invalid score value: %w", ErrScoreValueInvalid)
	}

	return validateScoreValue(raw)
}

// validateNamedScoreValue applies the value-shape rules for the legacy
// inngest.score kind. Each entry in the values map is keyed by a user-supplied
// score name (analogous to userland.<name>) and carries a single "value" field
// holding a finite number or boolean.
func validateNamedScoreValue(values Values) error {
	for name, raw := range values {
		var valueHolder struct {
			Value json.RawMessage `json:"value"`
		}

		dec := json.NewDecoder(strings.NewReader(string(raw)))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&valueHolder); err != nil {
			return fmt.Errorf("invalid score value: %w", ErrScoreValueInvalid)
		}

		if err := validateScoreName(name); err != nil {
			return fmt.Errorf("invalid score value: %w", err)
		}

		if err := validateScoreValue(valueHolder.Value); err != nil {
			return err
		}
	}

	return nil
}

// validateScoreValue checks that a raw score value is a finite number or
// boolean.
func validateScoreValue(raw json.RawMessage) error {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return fmt.Errorf("invalid score value: %w", ErrScoreValueInvalid)
	}

	switch v := value.(type) {
	case bool:
		return nil
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return fmt.Errorf("invalid score value: %w", ErrScoreValueInvalid)
		}
		return nil
	default:
		return fmt.Errorf("invalid score value: %w", ErrScoreValueInvalid)
	}
}
