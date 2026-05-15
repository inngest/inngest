package debugapi

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution/queue"
)

func debugScope(functionID, accountID, envID string) (queue.Scope, error) {
	fnID, err := uuid.Parse(functionID)
	if err != nil {
		return queue.Scope{}, fmt.Errorf("invalid function_id: %w", err)
	}

	acctID, err := uuid.Parse(accountID)
	if err != nil {
		return queue.Scope{}, fmt.Errorf("invalid account_id: %w", err)
	}

	environmentID, err := uuid.Parse(envID)
	if err != nil {
		return queue.Scope{}, fmt.Errorf("invalid env_id: %w", err)
	}

	scope := queue.Scope{
		AccountID:  acctID,
		EnvID:      environmentID,
		FunctionID: fnID,
	}
	if err := scope.Validate(); err != nil {
		return queue.Scope{}, err
	}

	return scope, nil
}
