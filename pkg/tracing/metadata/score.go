package metadata

import (
	"encoding/json"
	"fmt"
	"math"
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
func validateNamedScoreValue(kind Kind, values Values) error {
	if len(values) != 1 {
		return fmt.Errorf("score %q must have exactly one entry keyed \"value\": %w", kind, ErrScoreValueInvalid)
	}

	raw, ok := values["value"]
	if !ok {
		return fmt.Errorf("score %q value key must be \"value\": %w", kind, ErrScoreValueInvalid)
	}

	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return fmt.Errorf("invalid score value for kind %q: %w", kind, ErrScoreValueInvalid)
	}

	switch v := value.(type) {
	case bool:
		return nil
	case float64:
		if !math.IsNaN(v) && !math.IsInf(v, 0) {
			return nil
		}
	}

	return fmt.Errorf("invalid score value for kind %q: %w", kind, ErrScoreValueInvalid)
}
