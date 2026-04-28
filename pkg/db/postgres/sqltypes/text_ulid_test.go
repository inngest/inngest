package sqltypes

import (
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTextULIDFromULIDRoundTrip(t *testing.T) {
	t.Parallel()

	want := ulid.Make()
	got := FromULID(want)

	assert.Equal(t, want, got.ULID())
	assert.Equal(t, want.String(), got.String())
}

func TestTextULIDValueReturnsCanonicalText(t *testing.T) {
	t.Parallel()

	id := FromULID(ulid.Make())

	value, err := id.Value()
	require.NoError(t, err)
	assert.Equal(t, id.String(), value)
}

func TestTextULIDScanString(t *testing.T) {
	t.Parallel()

	want := ulid.Make()

	var got TextULID
	err := (&got).Scan(want.String())
	require.NoError(t, err)

	assert.Equal(t, want, got.ULID())
}

func TestTextULIDScanNilLeavesZeroValue(t *testing.T) {
	t.Parallel()

	var got TextULID
	err := (&got).Scan(nil)
	require.NoError(t, err)

	assert.Equal(t, ulid.ULID{}, got.ULID())
}

func TestTextULIDScanInvalidValue(t *testing.T) {
	t.Parallel()

	var got TextULID
	err := (&got).Scan(123)
	require.Error(t, err)
	assert.ErrorIs(t, err, ulid.ErrScanValue)
}
