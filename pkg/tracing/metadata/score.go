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

// validateScoreName checks the user-supplied suffix of an inngest.score.<name>
// kind for characters that downstream consumers can't safely round-trip.
// Mirrors the SDK validation and the monorepo MetricKeyRegex: rejects control
// characters (0x00-0x1F, 0x7F) and the single quote (which would silently
// drop in cloud variant aggregation because MetricKeyRegex excludes it for
// SQL-injection defense). Overall length is bounded by Kind.Validate.
func validateScoreName(name string) error {
	for _, r := range name {
		if r < 0x20 || r == 0x7f || r == '\'' {
			return fmt.Errorf("invalid score name %q: %w", name, ErrScoreNameInvalid)
		}
	}
	return nil
}

// validateNamedScoreValue applies the value-shape rules for the
// inngest.score.<name> kind family. The user-supplied name lives in the kind
// suffix (analogous to userland.<name>), so values carries exactly one entry
// keyed "value" containing a finite number or boolean.
func validateNamedScoreValue(values Values) error {
	for name, raw := range values {
		var valueHolder struct {
			Value any `json:"value"`
		}

		dec := json.NewDecoder(strings.NewReader(string(raw)))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&valueHolder); err != nil {
			return fmt.Errorf("invalid score value: %w", ErrScoreValueInvalid)
		}

		if err := validateScoreName(name); err != nil {
			return fmt.Errorf("invalid score value: %w", err)
		}

		switch v := valueHolder.Value.(type) {
		case bool:
			continue
		case float64:
			if math.IsNaN(v) || math.IsInf(v, 0) {
				return fmt.Errorf("invalid score value: %w", ErrScoreValueInvalid)
			}
		default:
			return fmt.Errorf("invalid score value: %w", ErrScoreValueInvalid)
		}
	}

	return nil
}
