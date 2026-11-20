package domain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestTenantRoundTrip(t *testing.T) {
	acc := uuid.New()
	env := uuid.New()
	tn := NewTenant(acc, env)

	require.Equal(t, acc, tn.AccountID())
	require.Equal(t, env, tn.EnvID())

	want := acc.String() + "/" + env.String()
	require.Equal(t, want, tn.String())

	text, err := tn.MarshalText()
	require.NoError(t, err)
	require.Equal(t, want, string(text))

	appended, err := tn.AppendText([]byte("x:"))
	require.NoError(t, err)
	require.Equal(t, "x:"+want, string(appended))

	var fromText Tenant
	require.NoError(t, fromText.UnmarshalText(text))
	require.Equal(t, tn, fromText)

	bin, err := tn.MarshalBinary()
	require.NoError(t, err)
	require.Len(t, bin, 32)

	var fromBin Tenant
	require.NoError(t, fromBin.UnmarshalBinary(bin))
	require.Equal(t, tn, fromBin)
}

func TestTenantUnmarshalTextErrors(t *testing.T) {
	var tn Tenant
	valid := NewTenant(uuid.New(), uuid.New()).String()

	require.ErrorIs(t, tn.UnmarshalText([]byte(valid[:72])), ErrInvalidTenantTextLen)

	bad := []byte(valid)
	bad[36] = '-'
	require.ErrorIs(t, tn.UnmarshalText(bad), ErrInvalidTenantSep)

	bad = []byte(valid)
	bad[0] = 'z'
	require.Error(t, tn.UnmarshalText(bad))

	require.ErrorIs(t, tn.UnmarshalBinary(make([]byte, 31)), ErrInvalidTenantBinLen)
}

func BenchmarkTenantString(b *testing.B) {
	tn := NewTenant(uuid.New(), uuid.New())
	b.ReportAllocs()
	for b.Loop() {
		_ = tn.String()
	}
}

func BenchmarkTenantAppendText(b *testing.B) {
	tn := NewTenant(uuid.New(), uuid.New())
	buf := make([]byte, 0, TenantTextLen)
	b.ReportAllocs()
	for b.Loop() {
		buf, _ = tn.AppendText(buf[:0])
	}
}
