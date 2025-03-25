package types

import (
	"encoding/json"
	"fmt"
)

func StructToMap(v any) (map[string]any, error) {
	byt, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal: %w", err)
	}

	var out = make(map[string]any)
	err = json.Unmarshal(byt, &out)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %w", err)
	}

	return out, nil
}
