package metadata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"math"
	"regexp"

	"github.com/99designs/gqlgen/graphql"
	"github.com/inngest/inngest/pkg/enums"
)

var (
	ErrMetadataSpanTooLarge    = errors.New("metadata span exceeds maximum size")
	ErrRunMetadataSizeExceeded = errors.New("run cumulative metadata size exceeded")
	ErrScoreNameInvalid        = errors.New("score name is invalid")
	ErrScoreScopeInvalid       = errors.New("score metadata must target step scope")
	ErrScoreValueInvalid       = errors.New("score value must be a finite number")
)

var scoreNameRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]{0,63}$`)

type Opcode = enums.MetadataOpcode

type Scope = enums.MetadataScope

type Structured interface {
	Kind() Kind
	Serialize() (Values, error)

	Op() enums.MetadataOpcode
}

type Values map[string]json.RawMessage

var _ graphql.ContextMarshaler = Values(nil)
var _ graphql.ContextUnmarshaler = (*Values)(nil)

func (m Values) MarshalGQLContext(ctx context.Context, w io.Writer) error {
	return json.NewEncoder(w).Encode(m)
}

func (m *Values) UnmarshalGQLContext(ctx context.Context, v any) error {
	vm, ok := v.(map[string]any)
	if !ok {
		return fmt.Errorf("cannot unmarshal %T as RawMetadata", v)
	}

	clear(*m)
	for k, v := range vm {
		var err error
		(*m)[k], err = json.Marshal(v)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *Values) FromStruct(v any) error {
	// TODO: reflect stuff so we don't need to remarshal?
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, m)
}

func (m Values) Combine(o Values, op enums.MetadataOpcode) error {
	switch op {
	case enums.MetadataOpcodeMerge:
		maps.Copy(m, o)
		return nil
	case enums.MetadataOpcodeDelete:
		for k := range o {
			delete(m, k)
		}
		return nil
	case enums.MetadataOpcodeAdd:
		for k := range o {
			var a float64
			if err := json.Unmarshal(m[k], &a); err != nil {
				m[k] = o[k]
				continue
			}

			var b float64
			if err := json.Unmarshal(o[k], &b); err != nil {
				continue
			}

			m[k], _ = json.Marshal(a + b)
		}
		return nil
	case enums.MetadataOpcodeSet:
		clear(m)
		maps.Copy(m, o)
		return nil
	default:
		return fmt.Errorf("unrecognized op %q", op)
	}
}

// Size returns the sum of key lengths and raw JSON value byte lengths.
// Map overhead is excluded; this is intended as an approximate cost metric.
func (m Values) Size() int {
	total := 0
	for k, val := range m {
		total += len(k) + len(val)
	}
	return total
}

type RawUpdate struct {
	Kind   Kind   `json:"kind"`
	Op     Opcode `json:"op"`
	Values Values `json:"values"`
}

type ScopedUpdate struct {
	Scope Scope `json:"scope"`
	Update
}

// ValidateAllowed checks raw shape, reserved-kind allowlisting, and any
// kind-specific scope rules for metadata that already carries its scope.
func (m ScopedUpdate) ValidateAllowed() error {
	return m.Update.validateAllowedForScope(m.Scope)
}

type Update struct {
	RawUpdate
}

func (m Update) Kind() Kind {
	return m.RawUpdate.Kind
}

func (m Update) Op() Opcode {
	return m.RawUpdate.Op
}

func (m Update) Serialize() (Values, error) {
	return m.RawUpdate.Values, nil
}

func (m Update) Validate() error {
	if err := m.RawUpdate.Kind.Validate(); err != nil {
		return fmt.Errorf("invalid kind: %w", err)
	}

	return nil
}

func (m Update) validateAllowed() error {
	if err := m.Validate(); err != nil {
		return err
	}

	if err := m.Kind().ValidateAllowed(); err != nil {
		return err
	}

	if m.Kind() == KindInngestScore {
		return validateScoreValues(m.RawUpdate.Values)
	}

	return nil
}

func (m Update) validateAllowedForScope(scope Scope) error {
	if err := m.validateAllowed(); err != nil {
		return err
	}

	if m.Kind() == KindInngestScore && scope != enums.MetadataScopeStep {
		return fmt.Errorf("invalid score scope %q: %w", scope, ErrScoreScopeInvalid)
	}

	return nil
}

func validateScoreValues(values Values) error {
	for name, raw := range values {
		if !scoreNameRegex.MatchString(name) {
			return fmt.Errorf("invalid score name %q: %w", name, ErrScoreNameInvalid)
		}

		var value *float64
		if err := json.Unmarshal(raw, &value); err != nil {
			return fmt.Errorf("invalid score value for %q: %w", name, ErrScoreValueInvalid)
		}
		if value == nil || math.IsNaN(*value) || math.IsInf(*value, 0) {
			return fmt.Errorf("invalid score value for %q: %w", name, ErrScoreValueInvalid)
		}
	}

	return nil
}
