package util

import (
	"encoding/json"
	"fmt"
	"sync/atomic"
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

	vals, err := ParallelDecode(items, func(val any, _ int) (map[string]any, bool, error) {
		str, _ := val.(string)
		item := map[string]any{}
		if err := json.Unmarshal(unsafe.Slice(unsafe.StringData(str), len(str)), &item); err != nil {
			return nil, false, fmt.Errorf("error reading partition item: %w", err)
		}
		return item, false, nil
	})
	require.NoError(t, err)
	require.EqualValues(t, vals[0], map[string]any{"hi": "ok"})
}

func TestParallelDecodeWithNils(t *testing.T) {
	items := []any{
		`{"val":0}`,
		`{"val":0}`,
		`{"val":1}`,
		`{"val":1}`,
		`{"val":1}`,
	}

	var nilCtr int64

	type itemTyp struct {
		Val int `json:"val"`
	}
	vals, err := ParallelDecode(items, func(val any, _ int) (*itemTyp, bool, error) {
		str, _ := val.(string)
		item := itemTyp{}
		if err := json.Unmarshal(unsafe.Slice(unsafe.StringData(str), len(str)), &item); err != nil {
			return nil, false, fmt.Errorf("error reading partition item: %w", err)
		}
		if item.Val == 0 {
			atomic.AddInt64(&nilCtr, 1)
			return nil, true, nil
		}
		return &item, false, nil
	})
	require.NoError(t, err)
	require.Len(t, vals, 3)
	require.EqualValues(t, 1, vals[0].Val)
	require.EqualValues(t, 1, vals[1].Val)
	require.EqualValues(t, 1, vals[2].Val)
	require.Equal(t, 2, int(nilCtr))
}
