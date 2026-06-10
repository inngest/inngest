package devserver

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/db"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestRunListItemFromRowUsesTraceRunOutput(t *testing.T) {
	runID := ulid.Make()
	eventID := ulid.Make()
	startedAt := time.Now().UTC()
	finishedAt := startedAt.Add(time.Second)

	result := runListItemFromRow(&db.RunListItemRow{
		FunctionRun: db.FunctionRun{
			RunID:        runID,
			RunStartedAt: startedAt,
			EventID:      eventID,
		},
		FunctionFinish: db.FunctionFinish{
			RunID:     runID,
			Status:    sql.NullString{String: "completed", Valid: true},
			Output:    sql.NullString{String: "", Valid: true},
			CreatedAt: sql.NullTime{Time: finishedAt, Valid: true},
		},
		Output:         []byte(`{"data":{"ok":true}}`),
		FunctionSlug:   "app-test-fn",
		FunctionName:   "Test function",
		FunctionConfig: `{"name":"Test function","slug":"test-fn"}`,
		FunctionAppID:  uuid.New(),
		AppName:        "app",
	}, true)

	require.NotNil(t, result.Output)
	var output map[string]bool
	require.NoError(t, json.Unmarshal(result.Output, &output))
	require.True(t, output["ok"])
}

func TestRunListItemFromRowUsesBareFunctionID(t *testing.T) {
	t.Run("uses distinct configured slug", func(t *testing.T) {
		result := runListItemFromRow(&db.RunListItemRow{
			FunctionSlug:   "app-app-test-fn",
			FunctionConfig: `{"name":"Test function","slug":"app-test-fn"}`,
			AppName:        "app",
		}, false)

		require.Equal(t, "app-test-fn", result.FunctionID)
	})

	t.Run("trims composite configured slug", func(t *testing.T) {
		result := runListItemFromRow(&db.RunListItemRow{
			FunctionSlug:   "app-test-fn",
			FunctionConfig: `{"name":"Test function","slug":"app-test-fn"}`,
			AppName:        "app",
		}, false)

		require.Equal(t, "test-fn", result.FunctionID)
	})

	t.Run("trims stored slug without configured slug", func(t *testing.T) {
		result := runListItemFromRow(&db.RunListItemRow{
			FunctionSlug: "app-test-fn",
			AppName:      "app",
		}, false)

		require.Equal(t, "test-fn", result.FunctionID)
	})
}

func TestRunListItemFromRowUnwrapsRunCompleteOpcodeOutput(t *testing.T) {
	runID := ulid.Make()
	eventID := ulid.Make()
	startedAt := time.Now().UTC()
	finishedAt := startedAt.Add(time.Second)

	result := runListItemFromRow(&db.RunListItemRow{
		FunctionRun: db.FunctionRun{
			RunID:        runID,
			RunStartedAt: startedAt,
			EventID:      eventID,
		},
		FunctionFinish: db.FunctionFinish{
			RunID:     runID,
			Status:    sql.NullString{String: "completed", Valid: true},
			CreatedAt: sql.NullTime{Time: finishedAt, Valid: true},
		},
		Output: []byte(`{"data":[{"data":{"body":"Hello, World!"},"id":"step-1","op":"RunComplete"}]}`),
	}, true)

	require.NotNil(t, result.Output)
	var output map[string]string
	require.NoError(t, json.Unmarshal(result.Output, &output))
	require.Equal(t, "Hello, World!", output["body"])
}
