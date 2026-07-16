package devserver

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	apiv2 "github.com/inngest/inngest/pkg/api/v2"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestRunListItemFromCQRSUsesTraceRunOutput(t *testing.T) {
	runID := ulid.Make()
	eventID := ulid.Make()
	startedAt := time.Now().UTC()
	finishedAt := startedAt.Add(time.Second)
	appID := uuid.New()
	functionID := uuid.New()

	result := runListItemFromCQRS(&cqrs.TraceRun{
		RunID:        runID.String(),
		StartedAt:    startedAt,
		EndedAt:      finishedAt,
		Status:       enums.RunStatusCompleted,
		AppID:        appID,
		AppName:      "app",
		FunctionID:   functionID,
		FunctionSlug: "app-test-fn",
		FunctionName: "Test function",
		TriggerIDs:   []string{eventID.String()},
		Output:       []byte(`{"data":{"ok":true}}`),
	}, true)

	require.NotNil(t, result.Output)
	var output map[string]bool
	require.NoError(t, json.Unmarshal(result.Output, &output))
	require.True(t, output["ok"])
}

func TestRunListItemFromCQRSUsesBareFunctionID(t *testing.T) {
	t.Run("uses distinct configured slug", func(t *testing.T) {
		result := mappedRunListItem(t, "app-app-test-fn", "app-test-fn")

		require.Equal(t, "app-test-fn", result.FunctionID)
	})

	t.Run("trims composite configured slug", func(t *testing.T) {
		result := mappedRunListItem(t, "app-test-fn", "app-test-fn")

		require.Equal(t, "test-fn", result.FunctionID)
	})

	t.Run("trims stored slug without configured slug", func(t *testing.T) {
		result := mappedRunListItem(t, "app-test-fn", "")

		require.Equal(t, "test-fn", result.FunctionID)
	})
}

func TestRunListItemFromCQRSUnwrapsRunCompleteOpcodeOutput(t *testing.T) {
	runID := ulid.Make()
	eventID := ulid.Make()
	startedAt := time.Now().UTC()
	finishedAt := startedAt.Add(time.Second)

	result := runListItemFromCQRS(&cqrs.TraceRun{
		RunID:      runID.String(),
		StartedAt:  startedAt,
		EndedAt:    finishedAt,
		Status:     enums.RunStatusCompleted,
		TriggerIDs: []string{eventID.String()},
		Output:     []byte(`{"data":[{"data":{"body":"Hello, World!"},"id":"step-1","op":"RunComplete"}]}`),
	}, true)

	require.NotNil(t, result.Output)
	var output map[string]string
	require.NoError(t, json.Unmarshal(result.Output, &output))
	require.Equal(t, "Hello, World!", output["body"])
}

func mappedRunListItem(t *testing.T, storedSlug, configuredSlug string) *apiv2.RunListItem {
	t.Helper()
	appID := uuid.New()
	functionID := uuid.New()
	return runListItemFromCQRS(&cqrs.TraceRun{
		RunID:        ulid.Make().String(),
		AppID:        appID,
		AppName:      "app",
		FunctionID:   functionID,
		FunctionSlug: apiv2.PublicFunctionID("app", storedSlug, configuredSlug),
	}, false)
}
