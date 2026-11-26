package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/domain"
	"github.com/inngest/inngest/pkg/util/errs"
	"github.com/stretchr/testify/require"
)

func TestExternalZeroValue(t *testing.T) {
	var number domain.External[int]
	require.True(t, number.Valid())
	require.False(t, number.HasError())
	require.NoError(t, number.Err())
	require.Zero(t, number.Value())
	require.Zero(t, number.Or(42))

	value, err := number.Unpack()
	require.Zero(t, value)
	require.NoError(t, err)

	var pointer domain.External[*int]
	require.True(t, pointer.Valid())
	require.False(t, pointer.HasError())
	require.NoError(t, pointer.Err())
	require.Nil(t, pointer.Value())
	require.Nil(t, pointer.Or(new(int)))
}

func TestExternalSuccess(t *testing.T) {
	type response struct {
		Body []byte
	}
	want := &response{Body: []byte("SDK response")}
	result := domain.ExternalValue(want)

	require.True(t, result.Valid())
	require.False(t, result.HasError())
	require.NoError(t, result.Err())
	require.Same(t, want, result.Value())
	require.Same(t, want, result.Or(&response{}))

	value, err := result.Unpack()
	require.Same(t, want, value)
	require.NoError(t, err)
}

func TestExternalValueZero(t *testing.T) {
	result := domain.ExternalValue(0)

	require.True(t, result.Valid())
	require.False(t, result.HasError())
	require.NoError(t, result.Err())
	require.Zero(t, result.Value())
	require.Zero(t, result.Or(42))
}

func TestExternalErrorNil(t *testing.T) {
	result := domain.ExternalError[int](nil)

	require.Equal(t, domain.External[int]{}, result)
	require.True(t, result.Valid())
	require.False(t, result.HasError())
	require.NoError(t, result.Err())
	require.Zero(t, result.Value())
	require.Zero(t, result.Or(42))
}

func TestExternalError(t *testing.T) {
	cause := errors.New("SDK request failed")
	raw := []byte(`{"message":"SDK request failed"}`)

	tests := []struct {
		name       string
		err        errs.UserError
		code       int
		retryable  bool
		retryAfter time.Duration
		raw        []byte
	}{
		{
			name: "no retry",
			err:  errs.WrapUser(400, false, "SDK error: %w", cause),
			code: 400,
		},
		{
			name:      "retry",
			err:       errs.WrapUser(500, true, "SDK error: %w", cause),
			code:      500,
			retryable: true,
		},
		{
			name:       "retry after",
			err:        errs.WrapAfterUser(429, 5*time.Second, "SDK error: %w", cause),
			code:       429,
			retryable:  true,
			retryAfter: 5 * time.Second,
		},
		{
			name:      "raw response",
			err:       errs.WrapResponseAsUser(503, true, raw, "SDK error: %w", cause),
			code:      503,
			retryable: true,
			raw:       raw,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := domain.ExternalError[string](tt.err)

			require.False(t, result.Valid())
			require.True(t, result.HasError())
			require.Same(t, tt.err, result.Err())
			require.Empty(t, result.Value())
			require.Equal(t, "fallback", result.Or("fallback"))
			require.ErrorIs(t, result.Err(), cause)
			require.Equal(t, tt.code, result.Err().ErrorCode())
			require.Equal(t, tt.retryable, result.Err().Retryable())
			require.Equal(t, tt.retryAfter, result.Err().RetryAfter())
			require.Equal(t, tt.raw, result.Err().Raw())

			value, err := result.Unpack()
			require.Empty(t, value)
			require.Same(t, tt.err, err)
		})
	}
}
