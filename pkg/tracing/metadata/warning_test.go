package metadata

import (
	"encoding/json"
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
				fmt.Errorf("wrapped: %w", &WarningError{Key: "key1", Err: io.EOF}),
			),
		)
		require.Len(t, warnings, 1)
	})

}

func TestWarningsStructured(t *testing.T) {
	warnings := ExtractWarnings(errors.Join(
		&WarningError{Key: "size", Err: errors.New("too big")},
		&WarningError{Key: "auth", Err: errors.New("nope")},
	))

	md := warnings.Structured()
	require.Len(t, md, 2)

	require.Equal(t, Kind("inngest.warning.auth"), md[0].Kind())
	values, err := md[0].Serialize()
	require.NoError(t, err)
	require.Equal(t, Values{"auth": json.RawMessage(`"nope"`)}, values)

	require.Equal(t, Kind("inngest.warning.size"), md[1].Kind())
	values, err = md[1].Serialize()
	require.NoError(t, err)
	require.Equal(t, Values{"size": json.RawMessage(`"too big"`)}, values)

	require.Empty(t, Warnings(nil).Structured())
}

func TestWithWarnings(t *testing.T) {
	existing := []Structured{Warning{Code: "other", Err: io.EOF}}

	require.Equal(t, existing, WithWarnings(existing, nil))

	md := WithWarnings(existing, &WarningError{Key: "size", Err: io.EOF})
	require.Len(t, md, 2)
	require.Equal(t, Kind("inngest.warning.size"), md[1].Kind())
}
