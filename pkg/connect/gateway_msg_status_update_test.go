package connect

import (
	"errors"
	"testing"

	"github.com/coder/websocket"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/syscode"
	"github.com/stretchr/testify/require"
)

func TestHandleConnStatusUpdateResultNilErrorResetsFailureCount(t *testing.T) {
	ch := &connectionHandler{
		log: logger.StdlibLogger(t.Context(), logger.WithLoggerLevel(logger.LevelEmergency)),
	}
	ch.consecutiveConnStatusUpdateFailures.Store(maxConsecutiveConnStatusUpdateFailures - 1)

	serr := ch.handleConnStatusUpdateResult(nil, "status update")
	require.Nil(t, serr)
	require.Zero(t, ch.consecutiveConnStatusUpdateFailures.Load())
}

func TestHandleConnStatusUpdateResultFailsAfterThreshold(t *testing.T) {
	ch := &connectionHandler{
		log: logger.StdlibLogger(t.Context(), logger.WithLoggerLevel(logger.LevelEmergency)),
	}

	for range maxConsecutiveConnStatusUpdateFailures - 1 {
		serr := ch.handleConnStatusUpdateResult(errors.New("upsert connection failed"), "status update")
		require.Nil(t, serr)
	}

	serr := ch.handleConnStatusUpdateResult(errors.New("upsert connection failed"), "status update")
	require.NotNil(t, serr)
	require.Equal(t, syscode.CodeConnectInternal, serr.SysCode)
	require.Equal(t, websocket.StatusInternalError, serr.StatusCode)
	require.Contains(t, serr.Msg, "could not update connection status")
	require.EqualValues(t, maxConsecutiveConnStatusUpdateFailures, ch.consecutiveConnStatusUpdateFailures.Load())
}

func TestHandleConnStatusUpdateResultSuccessBreaksFailureStreak(t *testing.T) {
	ch := &connectionHandler{
		log: logger.StdlibLogger(t.Context(), logger.WithLoggerLevel(logger.LevelEmergency)),
	}

	for range maxConsecutiveConnStatusUpdateFailures - 1 {
		serr := ch.handleConnStatusUpdateResult(errors.New("upsert connection failed"), "status update")
		require.Nil(t, serr)
	}

	serr := ch.handleConnStatusUpdateResult(nil, "status update")
	require.Nil(t, serr)

	for range maxConsecutiveConnStatusUpdateFailures - 1 {
		serr = ch.handleConnStatusUpdateResult(errors.New("upsert connection failed"), "status update")
		require.Nil(t, serr)
	}
}
