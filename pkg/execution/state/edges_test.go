package state

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestEdgeExpressionData(t *testing.T) {
	ctrl := gomock.NewController(t)
	state := NewMockState(ctrl)

	event := map[string]any{
		"data": map[string]any{
			"ok": true,
		},
	}
	state.EXPECT().Event().Return(event)

	actions := map[string]any{
		"first": map[string]any{
			"result": "yep",
		},
	}
	state.EXPECT().Actions().Return(actions)

	first := map[string]any{
		"result": "yep",
	}
	state.EXPECT().ActionID("first").Return(first, nil)

	result := EdgeExpressionData(context.Background(), state, "first")
	require.EqualValues(t, map[string]any{
		"event":    event,
		"steps":    actions,
		"response": first,
	}, result)
}
