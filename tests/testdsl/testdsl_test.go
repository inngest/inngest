package testdsl

import (
	"bytes"
	"context"
	"testing"

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
