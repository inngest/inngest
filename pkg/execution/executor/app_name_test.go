package executor

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/stretchr/testify/assert"
)

func TestResolveAppName(t *testing.T) {
	appID := uuid.New()
	loader := func(_ context.Context, id uuid.UUID) (string, error) {
		if id != appID {
			return "", errors.New("app not found")
		}
		return "loaded-app", nil
	}

	tests := []struct {
		name   string
		loader AppNameLoader
		req    execution.ScheduleRequest
		want   string
	}{
		{name: "request name wins", loader: loader, req: execution.ScheduleRequest{AppID: appID, AppName: "my-app"}, want: "my-app"},
		{name: "loaded when missing", loader: loader, req: execution.ScheduleRequest{AppID: appID}, want: "loaded-app"},
		{name: "lookup error", loader: loader, req: execution.ScheduleRequest{AppID: uuid.New()}, want: ""},
		{name: "no app id", loader: loader, req: execution.ScheduleRequest{}, want: ""},
		{name: "no loader", req: execution.ScheduleRequest{AppID: appID}, want: ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := &executor{log: logger.VoidLogger(), appNameLoader: tc.loader}
			assert.Equal(t, tc.want, e.resolveAppName(context.Background(), tc.req))
		})
	}
}
