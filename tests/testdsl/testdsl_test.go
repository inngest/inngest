package testdsl

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRequireLogFields(t *testing.T) {
	output := `{"some":"val","status":200}
{"ok":false,"error":"what error","status":500}
{"ok":false,"error":"what error","status":501}`

	data := &TestData{
		Out: bytes.NewBuffer([]byte(output)),
	}

	tests := []struct {
		fields map[string]any
		ok     bool
	}{
		{
			fields: map[string]any{
				"some": "nope",
			},
			ok: false,
		},
		{
			// all must match
			fields: map[string]any{
				"some":   "val",
				"status": 400,
			},
			ok: false,
		},
		{
			fields: map[string]any{
				"some": "val",
			},
			ok: true,
		},
		{
			fields: map[string]any{
				"some":   "val",
				"status": 200,
			},
			ok: true,
		},
		{
			fields: map[string]any{
				"ok":     false,
				"error":  "what error",
				"status": 200,
			},
			ok: false,
		},
		{
			fields: map[string]any{
				"ok":     false,
				"error":  "what error",
				"status": 501,
			},
			ok: true,
		},
	}

	ctx := context.Background()

	for _, test := range tests {
		err := RequireLogFields(test.fields)(ctx, data)
		if test.ok {
			require.NoError(t, err)
		} else {
			require.NotNil(t, err)
		}
	}
}

func TestRequireNoLogFieldsWithin(t *testing.T) {
	ctx := context.Background()
	buf := bytes.NewBuffer(nil)

	// Send "fail" within 1 second
	go func() {
		<-time.After(time.Second)
		_, _ = buf.WriteString(`{"fail":true}`)
	}()

	// Ensure that fail is found within ~500ms
	now := time.Now()
	data := &TestData{Out: buf}
	proc := RequireNoLogFieldsWithin(
		map[string]any{
			"fail": true,
		},
		2*time.Second,
	)
	err := proc(ctx, data)
	require.NotNil(t, err)
	require.Equal(t, 1, int(time.Since(now).Seconds()))

	// Ensure that success isn't found within 5 seconds.
	now = time.Now()
	proc = RequireNoLogFieldsWithin(
		map[string]any{
			"success": true,
		},
		5*time.Second,
	)
	err = proc(ctx, data)
	require.Nil(t, err)
	require.Equal(t, 5, int(time.Since(now).Seconds()))
}
