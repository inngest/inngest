package headers

import (
	"net/http"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/stretchr/testify/require"
)

func TestRetryAfterZero(t *testing.T) {
	before := time.Now().UTC().Truncate(time.Second).Add(consts.MinRetryDuration)
	at := RetryAfter(http.Header{"Retry-After": {"0"}})
	after := time.Now().UTC().Truncate(time.Second).Add(consts.MinRetryDuration)

	require.NotNil(t, at)
	require.False(t, at.Before(before))
	require.False(t, at.After(after))
}

func TestRetryAfterInvalid(t *testing.T) {
	for _, value := range []string{"", "invalid", "-1", "-0", "+0", "0.5", " 0 "} {
		t.Run(value, func(t *testing.T) {
			require.Nil(t, RetryAfter(http.Header{"Retry-After": {value}}))
		})
	}
}

func TestRetryAfterPositiveSecondsAndDates(t *testing.T) {
	for _, value := range []string{"60", "+60", time.Now().UTC().Add(time.Minute).Format(time.RFC3339), time.Now().UTC().Add(time.Minute).Format(time.RFC1123)} {
		t.Run(value, func(t *testing.T) {
			before := time.Now().UTC().Truncate(time.Second)
			at := RetryAfter(http.Header{"Retry-After": {value}})
			require.NotNil(t, at)
			require.WithinDuration(t, before.Add(time.Minute), *at, 2*time.Second)
		})
	}
}
