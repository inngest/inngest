package state

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestExpressionData(t *testing.T) {
	ctrl := gomock.NewController(t)
	state := NewMockState(ctrl)

	event := map[string]any{
		"data": map[string]any{
			"ok": true,
		},
	}
	state.EXPECT().Event().Return(event)

	result := ExpressionData(context.Background(), state)
	require.EqualValues(t, map[string]any{
		"event": event,
	}, result)
}
