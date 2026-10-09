package metadata

import (
	"encoding/json"
	"strings"
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
		name       string
		update     Update
		wantErr    error
		wantErrMsg string
	}{
		{
			name: "single finite numeric score is valid",
			update: Update{RawUpdate: RawUpdate{
				Kind:   KindInngestScore,
				Op:     enums.MetadataOpcodeMerge,
				Values: Values{"accuracy": json.RawMessage(`{"value":0.95}`)},
			}},
		},
		{
			name: "boolean score value is valid",
			update: Update{RawUpdate: RawUpdate{
				Kind:   KindInngestScore,
				Op:     enums.MetadataOpcodeMerge,
				Values: Values{"passed": json.RawMessage(`{"value":true}`)},
			}},
		},
		{
			name: "multiple scores in one update are valid",
			update: Update{RawUpdate: RawUpdate{
				Kind: KindInngestScore,
				Op:   enums.MetadataOpcodeMerge,
				Values: Values{
					"accuracy":  json.RawMessage(`{"value":0.95}`),
					"relevance": json.RawMessage(`{"value":0.8}`),
				},
			}},
		},
		{
			name: "arbitrary score name is accepted",
			update: Update{RawUpdate: RawUpdate{
				Kind:   KindInngestScore,
				Op:     enums.MetadataOpcodeMerge,
				Values: Values{"click-through rate (variant A)": json.RawMessage(`{"value":0.23}`)},
			}},
		},
		{
			name: "empty values map is valid",
			update: Update{RawUpdate: RawUpdate{
				Kind:   KindInngestScore,
				Op:     enums.MetadataOpcodeMerge,
				Values: Values{},
			}},
		},
		{
			name: "score name with single quote is rejected",
			update: Update{RawUpdate: RawUpdate{
				Kind:   KindInngestScore,
				Op:     enums.MetadataOpcodeMerge,
				Values: Values{"name'with'quotes": json.RawMessage(`{"value":0.5}`)},
			}},
			wantErr:    ErrScoreNameInvalid,
			wantErrMsg: "contains invalid characters",
		},
		{
			name: "score name over byte limit is rejected",
			update: Update{RawUpdate: RawUpdate{
				Kind:   KindInngestScore,
				Op:     enums.MetadataOpcodeMerge,
				Values: Values{strings.Repeat("a", MaxScoreNameByteLength+1): json.RawMessage(`{"value":0.5}`)},
			}},
			wantErr:    ErrScoreNameTooLong,
			wantErrMsg: "score name exceeds maximum length",
		},
		{
			name: "extra keys alongside value are rejected",
			update: Update{RawUpdate: RawUpdate{
				Kind:   KindInngestScore,
				Op:     enums.MetadataOpcodeMerge,
				Values: Values{"accuracy": json.RawMessage(`{"value":0.95,"extra":2}`)},
			}},
			wantErr: ErrScoreValueInvalid,
		},
		{
			name: "missing nested value key is rejected",
			update: Update{RawUpdate: RawUpdate{
				Kind:   KindInngestScore,
				Op:     enums.MetadataOpcodeMerge,
				Values: Values{"accuracy": json.RawMessage(`{"score":0.95}`)},
			}},
			wantErr: ErrScoreValueInvalid,
		},
		{
			name: "bare scalar instead of value object is rejected",
			update: Update{RawUpdate: RawUpdate{
				Kind:   KindInngestScore,
				Op:     enums.MetadataOpcodeMerge,
				Values: Values{"accuracy": json.RawMessage(`0.95`)},
			}},
			wantErr: ErrScoreValueInvalid,
		},
		{
			name: "null score value is rejected",
			update: Update{RawUpdate: RawUpdate{
				Kind:   KindInngestScore,
				Op:     enums.MetadataOpcodeMerge,
				Values: Values{"accuracy": json.RawMessage(`{"value":null}`)},
			}},
			wantErr: ErrScoreValueInvalid,
		},
		{
			name: "string score value is rejected",
			update: Update{RawUpdate: RawUpdate{
				Kind:   KindInngestScore,
				Op:     enums.MetadataOpcodeMerge,
				Values: Values{"accuracy": json.RawMessage(`{"value":"high"}`)},
			}},
			wantErr: ErrScoreValueInvalid,
		},
		{
			name: "object score value is rejected",
			update: Update{RawUpdate: RawUpdate{
				Kind:   KindInngestScore,
				Op:     enums.MetadataOpcodeMerge,
				Values: Values{"accuracy": json.RawMessage(`{"value":{"nested":1}}`)},
			}},
			wantErr: ErrScoreValueInvalid,
		},
		{
			name: "per name score kind w/ numeric value is valid",
			update: Update{RawUpdate: RawUpdate{
				Kind:   ScoreKind("accuracy"),
				Op:     enums.MetadataOpcodeMerge,
				Values: Values{"value": json.RawMessage(`0.95`)},
			}},
		},
		{
			name: "per name score kind w/ boolean value is valid",
			update: Update{RawUpdate: RawUpdate{
				Kind:   ScoreKind("passed"),
				Op:     enums.MetadataOpcodeSet,
				Values: Values{"value": json.RawMessage(`false`)},
			}},
		},
		{
			name: "per name score kind w/ a max length name is valid",
			update: Update{RawUpdate: RawUpdate{
				Kind:   ScoreKind(strings.Repeat("a", MaxScoreNameByteLength)),
				Op:     enums.MetadataOpcodeSet,
				Values: Values{"value": json.RawMessage(`1`)},
			}},
		},
		{
			name: "per name score kind w/ arbitrary name is valid",
			update: Update{RawUpdate: RawUpdate{
				Kind:   ScoreKind("click-through rate (variant A)"),
				Op:     enums.MetadataOpcodeSet,
				Values: Values{"value": json.RawMessage(`0.23`)},
			}},
		},
		{
			name: "per name score kind w/ single quote in the name is rejected",
			update: Update{RawUpdate: RawUpdate{
				Kind:   ScoreKind("name'with'quotes"),
				Op:     enums.MetadataOpcodeSet,
				Values: Values{"value": json.RawMessage(`0.5`)},
			}},
			wantErr:    ErrScoreNameInvalid,
			wantErrMsg: "contains invalid characters",
		},
		{
			name: "per name score kind w/ control character in the name is rejected",
			update: Update{RawUpdate: RawUpdate{
				Kind:   ScoreKind("bad\nname"),
				Op:     enums.MetadataOpcodeSet,
				Values: Values{"value": json.RawMessage(`0.5`)},
			}},
			wantErr: ErrScoreNameInvalid,
		},
		{
			name: "per name score kind over the name limit is rejected",
			update: Update{RawUpdate: RawUpdate{
				Kind:   ScoreKind(strings.Repeat("a", MaxScoreNameByteLength+1)),
				Op:     enums.MetadataOpcodeSet,
				Values: Values{"value": json.RawMessage(`0.5`)},
			}},
			wantErr: ErrKindTooLong,
		},
		{
			name: "per name score kind w/ empty name is rejected",
			update: Update{RawUpdate: RawUpdate{
				Kind:   KindPrefixInngestScore,
				Op:     enums.MetadataOpcodeSet,
				Values: Values{"value": json.RawMessage(`0.5`)},
			}},
			wantErr: ErrKindNotAllowed,
		},
		{
			name: "per name score kind w/ legacy named shape is rejected",
			update: Update{RawUpdate: RawUpdate{
				Kind:   ScoreKind("accuracy"),
				Op:     enums.MetadataOpcodeSet,
				Values: Values{"accuracy": json.RawMessage(`{"value":0.95}`)},
			}},
			wantErr: ErrScoreValueInvalid,
		},
		{
			name: "per name score kind w/ value object is rejected",
			update: Update{RawUpdate: RawUpdate{
				Kind:   ScoreKind("accuracy"),
				Op:     enums.MetadataOpcodeSet,
				Values: Values{"value": json.RawMessage(`{"value":0.95}`)},
			}},
			wantErr: ErrScoreValueInvalid,
		},
		{
			name: "per name score kind w/ extra keys alongside value is rejected",
			update: Update{RawUpdate: RawUpdate{
				Kind: ScoreKind("accuracy"),
				Op:   enums.MetadataOpcodeSet,
				Values: Values{
					"value": json.RawMessage(`0.95`),
					"extra": json.RawMessage(`2`),
				},
			}},
			wantErr: ErrScoreValueInvalid,
		},
		{
			name: "per name score kind w/ empty values is rejected",
			update: Update{RawUpdate: RawUpdate{
				Kind:   ScoreKind("accuracy"),
				Op:     enums.MetadataOpcodeSet,
				Values: Values{},
			}},
			wantErr: ErrScoreValueInvalid,
		},
		{
			name: "per name score kind w/ null value is rejected",
			update: Update{RawUpdate: RawUpdate{
				Kind:   ScoreKind("accuracy"),
				Op:     enums.MetadataOpcodeSet,
				Values: Values{"value": json.RawMessage(`null`)},
			}},
			wantErr: ErrScoreValueInvalid,
		},
		{
			name: "per name score kind w/ string value is rejected",
			update: Update{RawUpdate: RawUpdate{
				Kind:   ScoreKind("accuracy"),
				Op:     enums.MetadataOpcodeSet,
				Values: Values{"value": json.RawMessage(`"high"`)},
			}},
			wantErr: ErrScoreValueInvalid,
		},
		{
			name: "per code warning kind is valid",
			update: Update{RawUpdate: RawUpdate{
				Kind:   WarningKind("metadata_size_exceeded"),
				Op:     enums.MetadataOpcodeSet,
				Values: Values{"message": json.RawMessage(`"too big"`)},
			}},
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
				if tt.wantErrMsg != "" {
					require.ErrorContains(t, err, tt.wantErrMsg)
				}
				return
			}

			require.NoError(t, err)
		})
	}
}
