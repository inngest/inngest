package apiv1

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngestgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckpointNewRunRequestFnConfigIdentity(t *testing.T) {
	for _, slug := range []string{"", "custom-function"} {
		t.Run("slug="+slug, func(t *testing.T) {
			envID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
			request := CheckpointNewRunRequest{
				Event: inngestgo.GenericEvent[NewAPIRunData]{
					Data: NewAPIRunData{
						Domain: "https://example.com",
						Method: "POST",
						Path:   "/api/run",
						Fn:     slug,
					},
				},
				Retries: 3,
			}

			var config inngest.Function
			require.NoError(t, json.Unmarshal([]byte(request.FnConfig(envID)), &config))
			appID := request.AppID(envID)
			assert.Equal(t, request.FnID(appID), config.ID)
			expected, err := json.Marshal(request.Fn(appID))
			require.NoError(t, err)
			assert.JSONEq(t, string(expected), request.FnConfig(envID))

			// Equal slugs in different apps must retain different function identities.
			otherApp := request
			otherApp.Event.Data.Domain = "https://other.example.com"
			var otherConfig inngest.Function
			require.NoError(t, json.Unmarshal([]byte(otherApp.FnConfig(envID)), &otherConfig))
			assert.Equal(t, otherApp.FnID(otherApp.AppID(envID)), otherConfig.ID)
			assert.NotEqual(t, config.ID, otherConfig.ID)

			otherEnvID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
			require.NoError(t, json.Unmarshal([]byte(request.FnConfig(otherEnvID)), &otherConfig))
			assert.Equal(t, request.FnID(request.AppID(otherEnvID)), otherConfig.ID)
			assert.NotEqual(t, config.ID, otherConfig.ID)
		})
	}
}
