package queue

import (
	"fmt"

	"github.com/google/uuid"
)

// Scope identifies the tenant/function namespace for queue-owned state.
type Scope struct {
	IsSystem   bool
	AccountID  uuid.UUID
	EnvID      uuid.UUID
	FunctionID uuid.UUID
}

// Validate checks whether the scope is valid for general queue operations.
// System scopes are queue-name scoped and may omit tenant/function IDs; non-system
// scopes must include account, environment, and function IDs.
func (s Scope) Validate() error {
	if s.IsSystem {
		return nil
	}
	return s.ValidateIDs()
}

// ValidateIDs checks that account, environment, and function IDs are all set.
// Use this for APIs that always require tenant/function context, even when the
// scope is marked as system.
func (s Scope) ValidateIDs() error {
	if s.AccountID == uuid.Nil {
		return fmt.Errorf("missing account ID")
	}
	if s.EnvID == uuid.Nil {
		return fmt.Errorf("missing env ID")
	}
	if s.FunctionID == uuid.Nil {
		return fmt.Errorf("missing function ID")
	}
	return nil
}

func ScopeFromQueueItem(i QueueItem) Scope {
	scope := Scope{
		AccountID:  i.Data.Identifier.AccountID,
		EnvID:      i.Data.Identifier.WorkspaceID,
		FunctionID: i.FunctionID,
	}
	if i.QueueName != nil {
		scope.IsSystem = true
	}
	return scope
}

func ScopeFromQueuePartition(partition *QueuePartition) Scope {
	scope := Scope{AccountID: partition.AccountID}
	if partition.EnvID != nil {
		scope.EnvID = *partition.EnvID
	}
	if partition.FunctionID != nil {
		scope.FunctionID = *partition.FunctionID
	}
	if partition.QueueName != nil {
		scope.IsSystem = true
	}
	return scope
}
