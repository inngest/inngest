package metadata

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKind_ValidateAllowed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		kind    Kind
		wantErr error
	}{
		{
			name:    "inngest.experiment is allowed",
			kind:    "inngest.experiment",
			wantErr: nil,
		},
		{
			name:    "inngest.ai is allowed",
			kind:    "inngest.ai",
			wantErr: nil,
		},
		{
			name:    "inngest.http is allowed",
			kind:    "inngest.http",
			wantErr: nil,
		},
		{
			name:    "inngest.http.timing is allowed",
			kind:    "inngest.http.timing",
			wantErr: nil,
		},
		{
			name:    "inngest.response_headers is allowed",
			kind:    "inngest.response_headers",
			wantErr: nil,
		},
		{
			name:    "inngest.warnings is allowed",
			kind:    "inngest.warnings",
			wantErr: nil,
		},
		{
			name:    "inngest.unknown is rejected",
			kind:    "inngest.unknown",
			wantErr: ErrKindNotAllowed,
		},
		{
			name:    "inngest.internal is rejected",
			kind:    "inngest.internal",
			wantErr: ErrKindNotAllowed,
		},
		{
			name:    "userland.anything passes",
			kind:    "userland.anything",
			wantErr: nil,
		},
		{
			name:    "userland.custom.deep.kind passes",
			kind:    "userland.custom.deep.kind",
			wantErr: nil,
		},
		{
			name:    "empty kind passes",
			kind:    "",
			wantErr: nil,
		},
		{
			name:    "kind exceeding max length is rejected",
			kind:    Kind(strings.Repeat("a", MaxKindLength+1)),
			wantErr: ErrKindTooLong,
		},
		{
			name:    "inngest-prefixed kind at max length is rejected (not in allowlist)",
			kind:    Kind("inngest." + strings.Repeat("x", MaxKindLength-len("inngest."))),
			wantErr: ErrKindNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.kind.ValidateAllowed()
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestKind_Validate(t *testing.T) {
	t.Parallel()

	t.Run("valid length", func(t *testing.T) {
		t.Parallel()
		k := Kind(strings.Repeat("a", MaxKindLength))
		assert.NoError(t, k.Validate())
	})

	t.Run("exceeds max length", func(t *testing.T) {
		t.Parallel()
		k := Kind(strings.Repeat("a", MaxKindLength+1))
		require.ErrorIs(t, k.Validate(), ErrKindTooLong)
	})
}

func TestKind_IsInngest(t *testing.T) {
	t.Parallel()

	assert.True(t, Kind("inngest.ai").IsInngest())
	assert.True(t, Kind("inngest.experiment").IsInngest())
	assert.False(t, Kind("userland.foo").IsInngest())
	assert.False(t, Kind("").IsInngest())
}

func TestKind_IsUser(t *testing.T) {
	t.Parallel()

	assert.True(t, Kind("userland.foo").IsUser())
	assert.True(t, Kind("userland.custom.nested").IsUser())
	assert.False(t, Kind("inngest.ai").IsUser())
	assert.False(t, Kind("").IsUser())
}
