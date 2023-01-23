package httpdriver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRedirect(t *testing.T) {
	count := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch count {
		case 8:
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte("ok"))
		default:
			w.Header().Add("location", "/redirected/")
			w.WriteHeader(301)
		}
		count++
	}))
	defer ts.Close()

	res, status, err := DefaultExecutor.do(context.Background(), ts.URL, nil)
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.Equal(t, []byte("ok"), res)
}
