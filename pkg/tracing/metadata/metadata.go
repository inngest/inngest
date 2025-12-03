package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"

	"github.com/99designs/gqlgen/graphql"
	"github.com/inngest/inngest/pkg/enums"
)

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

type RawUpdate struct {
	Kind   Kind   `json:"kind"`
	Op     Opcode `json:"op"`
	Values Values `json:"values"`
}

type ScopedUpdate struct {
	Scope Scope `json:"scope"`
	Update
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
