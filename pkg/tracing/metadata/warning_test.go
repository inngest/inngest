package metadata

import (
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractWarnings(t *testing.T) {

	t.Run("empty", func(t *testing.T) {
		warnings := ExtractWarnings(nil)
		require.Len(t, warnings, 0)
	})

	t.Run("direct", func(t *testing.T) {
		warnings := ExtractWarnings(&WarningError{Key: "key1", Err: io.EOF})
		require.Len(t, warnings, 1)
	})

	t.Run("wrapped", func(t *testing.T) {
		warnings := ExtractWarnings(
			fmt.Errorf("wrapped: %w",
				&WarningError{Key: "key1", Err: io.EOF},
			),
		)
		require.Len(t, warnings, 1)
	})

	t.Run("joined", func(t *testing.T) {
		warnings := ExtractWarnings(
			errors.Join(
				&WarningError{Key: "key1", Err: io.EOF},
				&WarningError{Key: "key2", Err: io.EOF},
				&WarningError{Key: "key3", Err: io.EOF},
			),
		)
		require.Len(t, warnings, 3)
	})

	t.Run("multi-wrapped", func(t *testing.T) {
		warnings := ExtractWarnings(
			fmt.Errorf("wrapped: %w %w",
				&WarningError{Key: "key1", Err: io.EOF},
				&WarningError{Key: "key2", Err: io.EOF},
			),
		)
		require.Len(t, warnings, 2)
	})

	t.Run("complex", func(t *testing.T) {
		warnings := ExtractWarnings(
			errors.Join(
				fmt.Errorf("wrapped: %w %w",
					&WarningError{Key: "key1", Err: io.EOF},
					&WarningError{Key: "key2", Err: io.EOF},
				),
				&WarningError{Key: "key3", Err: io.EOF},
			),
		)
		require.Len(t, warnings, 3)
	})

	t.Run("overwrite", func(t *testing.T) {
		warnings := ExtractWarnings(
			errors.Join(
				fmt.Errorf("wrapped: %w %w",
					&WarningError{Key: "key1", Err: io.EOF},
					&WarningError{Key: "key1", Err: io.EOF},
				),
				&WarningError{Key: "key1", Err: io.EOF},
			),
		)
		require.Len(t, warnings, 1)
	})

}
