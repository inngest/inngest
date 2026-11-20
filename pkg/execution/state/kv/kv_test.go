package kv

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/domain"
	"github.com/stretchr/testify/require"
)

func testKey() Key {
	return Key{
		Tenant:    domain.NewTenant(uuid.New(), uuid.New()),
		Namespace: "steps",
		Key:       "step-1",
	}
}

func TestKeyText(t *testing.T) {
	k := testKey()
	want := k.Tenant.String() + "/steps/step-1"

	require.Equal(t, want, k.String())
	require.Len(t, want, k.textLen())

	text, err := k.MarshalText()
	require.NoError(t, err)
	require.Equal(t, want, string(text))

	appended, err := k.AppendText([]byte("x:"))
	require.NoError(t, err)
	require.Equal(t, "x:"+want, string(appended))
}

func TestKeyTextAllocs(t *testing.T) {
	k := testKey()

	require.Equal(t, 1.0, testing.AllocsPerRun(100, func() {
		_ = k.String()
	}))
	require.Equal(t, 1.0, testing.AllocsPerRun(100, func() {
		_, _ = k.MarshalText()
	}))

	buf := make([]byte, 0, k.textLen())
	require.Equal(t, 0.0, testing.AllocsPerRun(100, func() {
		buf, _ = k.AppendText(buf[:0])
	}))
}

func BenchmarkKeyString(b *testing.B) {
	k := testKey()
	b.ReportAllocs()
	for b.Loop() {
		_ = k.String()
	}
}

func BenchmarkKeyAppendText(b *testing.B) {
	k := testKey()
	buf := make([]byte, 0, k.textLen())
	b.ReportAllocs()
	for b.Loop() {
		buf, _ = k.AppendText(buf[:0])
	}
}

func TestKeyValidate(t *testing.T) {
	tenant := domain.NewTenant(uuid.New(), uuid.New())
	cases := []struct {
		name      string
		namespace string
		key       string
		err       error
	}{
		{name: "ok", namespace: "steps", key: "step-1"},
		{name: "ok mixed", namespace: "Steps_v2", key: "A-b_9"},
		{name: "empty namespace", namespace: "", key: "k", err: ErrEmptyNamespace},
		{name: "empty key", namespace: "ns", key: "", err: ErrEmptyKey},
		{name: "slash in namespace", namespace: "a/b", key: "k", err: ErrInvalidNamespace},
		{name: "space in key", namespace: "ns", key: "a b", err: ErrInvalidKey},
		{name: "dot in key", namespace: "ns", key: "a.b", err: ErrInvalidKey},
		{name: "unicode in key", namespace: "ns", key: "é", err: ErrInvalidKey},
		{name: "max len", namespace: strings.Repeat("n", MaxPartLen), key: strings.Repeat("k", MaxPartLen)},
		{name: "namespace too long", namespace: strings.Repeat("n", MaxPartLen+1), key: "k", err: ErrNamespaceTooLong},
		{name: "key too long", namespace: "ns", key: strings.Repeat("k", MaxPartLen+1), err: ErrKeyTooLong},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			k := Key{
				Tenant:    tenant,
				Namespace: c.namespace,
				Key:       c.key,
			}
			err := k.Validate()
			if c.err == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, c.err)
		})
	}

	k := testKey()
	require.Equal(t, 0.0, testing.AllocsPerRun(100, func() {
		_ = k.Validate()
	}))
}

func TestApplyOptions(t *testing.T) {
	o := ApplyOptions()
	require.Nil(t, o.Expected)
	require.Zero(t, o.TTL)

	o = ApplyOptions(WithExpected(0), WithTTL(time.Minute))
	require.NotNil(t, o.Expected)
	require.Equal(t, int64(0), *o.Expected)
	require.Equal(t, time.Minute, o.TTL)

	o = ApplyOptions(WithExpected(3), WithExpected(7))
	require.Equal(t, int64(7), *o.Expected)
}
