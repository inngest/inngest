package headers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/99designs/gqlgen/graphql"
)

// Compact is a wrapper around http.Header that marshals to JSON in a more compact format.
// Instead of marshaling to a map[string][]string, it marshals to a map[string]any where values
// with a single string are marshaled as a string instead of an array.
type Compact http.Header

func (h Compact) MarshalJSON() ([]byte, error) {
	compact := make(map[string]any)
	for key, values := range h {
		if len(values) == 1 {
			compact[key] = values[0]
		} else {
			compact[key] = values
		}
	}
	return json.Marshal(compact)
}

func (h *Compact) UnmarshalJSON(data []byte) error {
	var compact map[string]any
	if err := json.Unmarshal(data, &compact); err != nil {
		return err
	}

	*h = make(Compact)
	for key, value := range compact {
		switch v := value.(type) {
		case string:
			(*h)[key] = []string{v}
		case []any:
			(*h)[key] = make([]string, len(v))
			for i, item := range v {
				if str, ok := item.(string); ok {
					(*h)[key][i] = str
				}
			}
		}
	}

	return nil
}

var _ graphql.ContextMarshaler = Compact(nil)
var _ graphql.ContextUnmarshaler = (*Compact)(nil)

func (h Compact) MarshalGQLContext(ctx context.Context, w io.Writer) error {
	return json.NewEncoder(w).Encode(h)
}

func (h *Compact) UnmarshalGQLContext(ctx context.Context, v any) error {
	vm, ok := v.(map[string]any)
	if !ok {
		return fmt.Errorf("cannot unmarshal %T as RawMetadata", v)
	}
*h = make(Compact)
	for key, value := range vm {
		switch v := value.(type) {
		case string:
			(*h)[key] = []string{v}
		case []any:
			(*h)[key] = make([]string, len(v))
			for i, item := range v {
				if str, ok := item.(string); ok {
					(*h)[key][i] = str
				}
			}
		}
	}

	return nil
}
