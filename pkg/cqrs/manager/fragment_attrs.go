package manager

import (
	"encoding/json"
	"fmt"
)

// extractFragmentAttrs extracts the "attributes" field from a span fragment
// as map[string]any. In SQLite the column is TEXT so the value arrives as a
// JSON string; in PostgreSQL it is jsonb so json_build_object embeds it as an
// already-decoded object.
func extractFragmentAttrs(fragment map[string]any) (map[string]any, error) {
	switch v := fragment["attributes"].(type) {
	case string:
		m := map[string]any{}
		if err := json.Unmarshal([]byte(v), &m); err != nil {
			return nil, fmt.Errorf("unmarshal fragment attributes string: %w", err)
		}
		return m, nil
	case map[string]any:
		return v, nil
	case nil:
		return nil, fmt.Errorf("fragment attributes is nil")
	default:
		return nil, fmt.Errorf("unexpected fragment attributes type: %T", v)
	}
}
