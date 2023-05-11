package httpdriver

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRedirect(t *testing.T) {
	input := []byte(`{"event":{"name":"hi","data":{}}}`)

	count := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch count {
		case 8:
			require.Equal(t, http.MethodPost, r.Method)
			byt, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			require.Equal(t, input, byt)
			require.Equal(t, "application/json", r.Header.Get("content-type"))
			_, _ = w.Write([]byte("ok"))
		default:
			w.Header().Add("location", "/redirected/")
			w.WriteHeader(301)
		}
		count++
	}))
	defer ts.Close()

	res, status, _, err := DefaultExecutor.do(context.Background(), ts.URL, input)
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.Equal(t, []byte("ok"), res)
}
