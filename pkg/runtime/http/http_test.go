package http

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/inngest/inngestctl/inngest"
	"github.com/stretchr/testify/require"
)

func TestExecute(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer s.Close()

	a := inngest.ActionVersion{
		DSN: "foo/bar",
		Runtime: inngest.RuntimeWrapper{
			Runtime: inngest.RuntimeHTTP{
				URL: s.URL,
			},
		},
	}

	state, err := Execute(context.Background(), a, nil)
	require.NoError(t, err)
	require.EqualValues(t, map[string]interface{}{"ok": true}, state)
}

func TestExecuteWithDockerRuntime(t *testing.T) {
	a := inngest.ActionVersion{
		DSN: "foo/bar",
		Runtime: inngest.RuntimeWrapper{
			Runtime: inngest.RuntimeDocker{
				Image: "hello-world",
			},
		},
	}

	state, err := Execute(context.Background(), a, nil)
	require.Nil(t, state)
	require.Error(t, fmt.Errorf("Unable to use HTTP executor for non-HTTP runtime"), err)
}
