package logger

import (
	"bytes"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestConditionalCheck(t *testing.T) {
	// Clean up after test
	defer func() {
		conditionalCheckMu.Lock()
		conditionalCheckFn = nil
		conditionalCheckMu.Unlock()
	}()

	t.Run("no conditional check registered", func(t *testing.T) {
		buf := &bytes.Buffer{}
		l := From(context.Background(),
			WithLoggerLevel(LevelDebug),
			WithLoggerWriter(buf),
			WithHandler(TextHandler),
		)
		ctx := WithStdlib(context.Background(), l)

		From(ctx).Debug("test message")
		require.Contains(t, buf.String(), "test message")
	})

	t.Run("conditional check returns true", func(t *testing.T) {
		buf := &bytes.Buffer{}
		l := From(context.Background(),
			WithLoggerLevel(LevelDebug),
			WithLoggerWriter(buf),
			WithHandler(TextHandler),
		)
		ctx := WithStdlib(context.Background(), l)

		RegisterConditionalCheck(func(ctx context.Context) bool {
			return true
		})

		From(ctx).Debug("allowed message")
		require.Contains(t, buf.String(), "allowed message")
	})

	t.Run("conditional check returns false", func(t *testing.T) {
		buf := &bytes.Buffer{}
		l := From(context.Background(),
			WithLoggerLevel(LevelDebug),
			WithLoggerWriter(buf),
			WithHandler(TextHandler),
		)
		ctx := WithStdlib(context.Background(), l)

		RegisterConditionalCheck(func(ctx context.Context) bool {
			return false
		})

		From(ctx).Debug("should not appear")
		require.Empty(t, buf.String())
	})
}

func TestVoidLogger(t *testing.T) {
	buf := &bytes.Buffer{}

	// VoidLogger should not write anything
	l := VoidLogger()
	l.Debug("debug message")
	l.Info("info message")
	l.Warn("warn message")
	l.Error("error message")

	// The buffer should be empty since VoidLogger discards output
	// Note: VoidLogger uses io.Discard, not our buffer, so this is just
	// verifying the function exists and doesn't panic
	require.Empty(t, buf.String())
}

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
