package util

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestIsGRPCMessageSizeError(t *testing.T) {
	for _, tc := range []struct {
		message string
		want    bool
	}{
		{"grpc: received message larger than max (8979575 vs. 4194304)", true},
		{"grpc: received message larger than max length allowed on current machine (8979575 vs. 4194304)", true},
		{"grpc: received message after decompression larger than max 4194304", true},
		{"trying to send message larger than max (8979575 vs. 4194304)", true},
		{"grpc: trying to send message larger than max (8979575 vs. 4194304)", true},
		{"quota exceeded", false},
		{"gateway capacity exhausted", false},
	} {
		t.Run(tc.message, func(t *testing.T) {
			err := status.Error(codes.ResourceExhausted, tc.message)
			require.Equal(t, tc.want, IsGRPCMessageSizeError(err))
			require.Equal(t, tc.want, IsGRPCMessageSizeError(fmt.Errorf("forward: %w", err)))
			require.False(t, IsGRPCMessageSizeError(status.Error(codes.Unavailable, tc.message)))
			require.False(t, IsGRPCMessageSizeError(errors.New(tc.message)))
		})
	}
	require.False(t, IsGRPCMessageSizeError(nil))
}
