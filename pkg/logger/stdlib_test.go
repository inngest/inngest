package logger

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestLoggerMergeAttrWithTags(t *testing.T) {
	ctx := context.Background()

	type dummy struct {
		ID    string `json:"id"`
		Value int64  `json:"v"`
	}

	testcases := []struct {
		name     string
		attrs    []any
		tags     map[string]string
		expected map[string]string
	}{
		{
			name:  "no merge",
			attrs: []any{"hello", "world", "foo", "bar"},
			expected: map[string]string{
				"hello": "world",
				"foo":   "bar",
			},
		},
		{
			name: "with stringers",
			attrs: []any{
				"key1", ulid.MustParse("01K0T21HZW9DHDZ5P5TQKBN1E6"),
				"key2", uuid.MustParse("2bbf4f69-6e0d-466f-9cdd-2cfa2f684c8b"),
			},
			expected: map[string]string{
				"key1": "01K0T21HZW9DHDZ5P5TQKBN1E6",
				"key2": "2bbf4f69-6e0d-466f-9cdd-2cfa2f684c8b",
			},
		},
		{
			name: "mix with objects",
			attrs: []any{
				"foo", "bar",
				"ulid", ulid.MustParse("01K0T21HZW9DHDZ5P5TQKBN1E6"),
				"dummy", dummy{},
			},
			expected: map[string]string{
				"foo":  "bar",
				"ulid": "01K0T21HZW9DHDZ5P5TQKBN1E6",
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			tags := map[string]string{}
			if tc.tags != nil {
				tags = tc.tags
			}

			l := StdlibLogger(ctx).With(tc.attrs...).(*logger)
			l.mergeAttrsWithErrorTags(tags)

			require.Equal(t, tc.expected, tags)
		})
	}
}
