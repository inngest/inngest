package util

import (
	"encoding/json"
	"fmt"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/require"
)

func TestParallelDecode(t *testing.T) {
	items := []any{
		`{"hi":"ok"}`,
		`{"a":true}`,
		`{"arr":[1, 2, 3, 4, 5]}`,
		`{"who": "it be"}`,
		`{"hi":"ok"}`,
		`{"a":true}`,
		`{"arr":[1, 2, 3, 4, 5]}`,
		`{"who": "it be"}`,
		`{"hi":"ok"}`,
		`{"a":true}`,
		`{"arr":[1, 2, 3, 4, 5]}`,
		`{"who": "it be"}`,
		`{"hi":"ok"}`,
		`{"a":true}`,
		`{"arr":[1, 2, 3, 4, 5]}`,
		`{"who": "it be"}`,
		`{"hi":"ok"}`,
		`{"a":true}`,
		`{"arr":[1, 2, 3, 4, 5]}`,
		`{"who": "it be"}`,
		`{"hi":"ok"}`,
		`{"a":true}`,
		`{"arr":[1, 2, 3, 4, 5]}`,
		`{"who": "it be"}`,
		`{"hi":"ok"}`,
		`{"a":true}`,
		`{"arr":[1, 2, 3, 4, 5]}`,
		`{"who": "it be"}`,
		`{"hi":"ok"}`,
		`{"a":true}`,
		`{"arr":[1, 2, 3, 4, 5]}`,
		`{"who": "it be"}`,
		`{"hi":"ok"}`,
		`{"a":true}`,
		`{"arr":[1, 2, 3, 4, 5]}`,
		`{"who": "it be"}`,
		`{"hi":"ok"}`,
		`{"a":true}`,
		`{"arr":[1, 2, 3, 4, 5]}`,
		`{"who": "it be"}`,
		`{"hi":"ok"}`,
		`{"a":true}`,
		`{"arr":[1, 2, 3, 4, 5]}`,
		`{"who": "it be"}`,
		`{"hi":"ok"}`,
		`{"a":true}`,
		`{"arr":[1, 2, 3, 4, 5]}`,
		`{"who": "it be"}`,
		`{"hi":"ok"}`,
		`{"a":true}`,
		`{"arr":[1, 2, 3, 4, 5]}`,
		`{"who": "it be"}`,
		`{"hi":"ok"}`,
		`{"a":true}`,
		`{"arr":[1, 2, 3, 4, 5]}`,
		`{"who": "it be"}`,
		`{"hi":"ok"}`,
		`{"a":true}`,
		`{"arr":[1, 2, 3, 4, 5]}`,
		`{"who": "it be"}`,
	}

	vals, err := ParallelDecode(items, func(val any) (map[string]any, error) {
		str, _ := val.(string)
		item := map[string]any{}
		if err := json.Unmarshal(unsafe.Slice(unsafe.StringData(str), len(str)), &item); err != nil {
			return nil, fmt.Errorf("error reading partition item: %w", err)
		}
		return item, nil

	})
	require.NoError(t, err)
	require.EqualValues(t, vals[0], map[string]any{"hi": "ok"})
}
