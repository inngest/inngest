package metadata

import (
	"encoding/json"
	"testing"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/stretchr/testify/require"
)

func TestValuesSize(t *testing.T) {
	tests := []struct {
		name     string
		values   Values
		expected int
	}{
		{
			name:     "empty values",
			values:   Values{},
			expected: 0,
		},
		{
			name: "single key-value",
			values: Values{
				"key": json.RawMessage(`"value"`),
			},
			expected: len("key") + len(`"value"`),
		},
		{
			name: "multiple key-values",
			values: Values{
				"alpha": json.RawMessage(`"one"`),
				"beta":  json.RawMessage(`{"nested":true}`),
			},
			expected: len("alpha") + len(`"one"`) + len("beta") + len(`{"nested":true}`),
		},
		{
			name: "nil json value",
			values: Values{
				"key": json.RawMessage(nil),
			},
			expected: len("key"),
		},
		{
			name: "empty json value",
			values: Values{
				"key": json.RawMessage{},
			},
			expected: len("key"),
		},
		{
			name: "realistic metadata payload",
			values: Values{
				"model":       json.RawMessage(`"gpt-4"`),
				"prompt":      json.RawMessage(`"Tell me about Go programming"`),
				"completion":  json.RawMessage(`"Go is a statically typed language designed at Google."`),
				"tokens_used": json.RawMessage(`150`),
				"latency_ms":  json.RawMessage(`432`),
			},
			expected: len("model") + len(`"gpt-4"`) +
				len("prompt") + len(`"Tell me about Go programming"`) +
				len("completion") + len(`"Go is a statically typed language designed at Google."`) +
				len("tokens_used") + len(`150`) +
				len("latency_ms") + len(`432`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.values.Size()
			if got != tt.expected {
				t.Errorf("Values.Size() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestValuesSizeNilMap(t *testing.T) {
	var v Values
	if got := v.Size(); got != 0 {
		t.Errorf("nil Values.Size() = %d, want 0", got)
	}
}

func TestUpdateValidateAllowedNamedScoreValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		update  Update
		wantErr error
	}{
		{
			name: "finite numeric value is valid",
			update: Update{RawUpdate: RawUpdate{
				Kind:   "inngest.score.accuracy",
				Op:     enums.MetadataOpcodeMerge,
				Values: Values{"value": json.RawMessage(`0.95`)},
			}},
		},
		{
			name: "arbitrary name in kind is accepted",
			update: Update{RawUpdate: RawUpdate{
				Kind:   "inngest.score.click-through rate (variant A)",
				Op:     enums.MetadataOpcodeMerge,
				Values: Values{"value": json.RawMessage(`0.23`)},
			}},
		},
		{
			name: "missing value key is rejected",
			update: Update{RawUpdate: RawUpdate{
				Kind:   "inngest.score.accuracy",
				Op:     enums.MetadataOpcodeMerge,
				Values: Values{"score": json.RawMessage(`1`)},
			}},
			wantErr: ErrScoreValueInvalid,
		},
		{
			name: "extra keys alongside value are rejected",
			update: Update{RawUpdate: RawUpdate{
				Kind:   "inngest.score.accuracy",
				Op:     enums.MetadataOpcodeMerge,
				Values: Values{"value": json.RawMessage(`1`), "extra": json.RawMessage(`2`)},
			}},
			wantErr: ErrScoreValueInvalid,
		},
		{
			name: "empty values map is rejected",
			update: Update{RawUpdate: RawUpdate{
				Kind:   "inngest.score.accuracy",
				Op:     enums.MetadataOpcodeMerge,
				Values: Values{},
			}},
			wantErr: ErrScoreValueInvalid,
		},
		{
			name: "null value is rejected",
			update: Update{RawUpdate: RawUpdate{
				Kind:   "inngest.score.accuracy",
				Op:     enums.MetadataOpcodeMerge,
				Values: Values{"value": json.RawMessage(`null`)},
			}},
			wantErr: ErrScoreValueInvalid,
		},
		{
			name: "string value is rejected",
			update: Update{RawUpdate: RawUpdate{
				Kind:   "inngest.score.accuracy",
				Op:     enums.MetadataOpcodeMerge,
				Values: Values{"value": json.RawMessage(`"high"`)},
			}},
			wantErr: ErrScoreValueInvalid,
		},
		{
			name: "object value is rejected",
			update: Update{RawUpdate: RawUpdate{
				Kind:   "inngest.score.accuracy",
				Op:     enums.MetadataOpcodeMerge,
				Values: Values{"value": json.RawMessage(`{"nested":1}`)},
			}},
			wantErr: ErrScoreValueInvalid,
		},
		{
			name: "non-score metadata keeps generic shape",
			update: Update{RawUpdate: RawUpdate{
				Kind:   "userland.score",
				Op:     enums.MetadataOpcodeMerge,
				Values: Values{"score": json.RawMessage(`{"value":1}`)},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.update.ValidateAllowed()
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
		})
	}
}
