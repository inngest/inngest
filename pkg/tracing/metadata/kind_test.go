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
			name:    "inngest.sandbox is allowed",
			kind:    KindInngestSandbox,
			wantErr: nil,
		},
		{
			name:    "bare inngest.score is allowed",
			kind:    KindInngestScore,
			wantErr: nil,
		},
		{
			name:    "inngest.score.<name> is allowed",
			kind:    "inngest.score.accuracy",
			wantErr: nil,
		},
		{
			name:    "inngest.score.<name> w/ dots in the name is allowed",
			kind:    "inngest.score.latency.p99",
			wantErr: nil,
		},
		{
			name:    "inngest.score. w/ empty name is rejected",
			kind:    KindPrefixInngestScore,
			wantErr: ErrKindNotAllowed,
		},
		{
			name:    "inngest.score.<name> w/ a max length name fits",
			kind:    ScoreKind(strings.Repeat("a", MaxScoreNameByteLength)),
			wantErr: nil,
		},
		{
			name:    "inngest.score.<name> over the max name length is too long",
			kind:    ScoreKind(strings.Repeat("a", MaxScoreNameByteLength+1)),
			wantErr: ErrKindTooLong,
		},
		{
			name:    "inngest.warning.<code> is allowed",
			kind:    "inngest.warning.metadata_size_exceeded",
			wantErr: nil,
		},
		{
			name:    "inngest.warning. w/ empty code is rejected",
			kind:    KindPrefixInngestWarning,
			wantErr: ErrKindNotAllowed,
		},
		{
			name:    "inngest.warning w/o the trailing dot is rejected",
			kind:    "inngest.warning",
			wantErr: ErrKindNotAllowed,
		},
		{
			name:    "inngest.warning.<code> over the max kind length is too long",
			kind:    WarningKind(strings.Repeat("a", MaxKindLength-len(KindPrefixInngestWarning)+1)),
			wantErr: ErrKindTooLong,
		},
		{
			name:    "inngest.scores.<name> is rejected",
			kind:    "inngest.scores.accuracy",
			wantErr: ErrKindNotAllowed,
		},
		{
			name:    "inngest.warnings.<code> is rejected",
			kind:    "inngest.warnings.code",
			wantErr: ErrKindNotAllowed,
		},
		{
			name:    "inngest.unknown is rejected",
			kind:    "inngest.unknown",
			wantErr: ErrKindNotAllowed,
		},
		{
			name:    "inngest.ai.suffix is rejected (only score allows suffixes)",
			kind:    "inngest.ai.gpt4",
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

func TestKind_ScoreName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		kind     Kind
		wantName string
		wantOK   bool
	}{
		{name: "per name score kind", kind: "inngest.score.accuracy", wantName: "accuracy", wantOK: true},
		{name: "name w/ dots", kind: "inngest.score.a.b", wantName: "a.b", wantOK: true},
		{name: "bare inngest.score", kind: KindInngestScore, wantOK: false},
		{name: "empty name", kind: "inngest.score.", wantOK: false},
		{name: "warning kind", kind: "inngest.warning.code", wantOK: false},
		{name: "userland kind", kind: "userland.score.accuracy", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			name, ok := tt.kind.ScoreName()
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantName, name)
		})
	}

	t.Run("round trips w/ ScoreKind", func(t *testing.T) {
		t.Parallel()
		name, ok := ScoreKind("click-through rate").ScoreName()
		assert.True(t, ok)
		assert.Equal(t, "click-through rate", name)
	})
}

func TestKind_WarningCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		kind     Kind
		wantCode string
		wantOK   bool
	}{
		{name: "per code warning kind", kind: "inngest.warning.size", wantCode: "size", wantOK: true},
		{name: "bare inngest.warnings", kind: KindInngestWarnings, wantOK: false},
		{name: "empty code", kind: "inngest.warning.", wantOK: false},
		{name: "score kind", kind: "inngest.score.accuracy", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			code, ok := tt.kind.WarningCode()
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantCode, code)
		})
	}

	t.Run("round trips w/ WarningKind", func(t *testing.T) {
		t.Parallel()
		code, ok := WarningKind("size").WarningCode()
		assert.True(t, ok)
		assert.Equal(t, "size", code)
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
