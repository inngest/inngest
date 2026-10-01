package queue

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestScopeValidate(t *testing.T) {
	tests := []struct {
		name    string
		scope   Scope
		wantErr string
	}{
		{
			name: "valid user scope",
			scope: Scope{
				AccountID:  uuid.New(),
				EnvID:      uuid.New(),
				FunctionID: uuid.New(),
			},
		},
		{
			name:  "system scope allows missing ids",
			scope: Scope{IsSystem: true},
		},
		{
			name: "system scope allows missing accountID",
			scope: Scope{
				IsSystem:   true,
				EnvID:      uuid.New(),
				FunctionID: uuid.New(),
			},
		},
		{
			name: "system scope allows missing envID",
			scope: Scope{
				IsSystem:   true,
				AccountID:  uuid.New(),
				FunctionID: uuid.New(),
			},
		},
		{
			name: "system scope allows missing functionID",
			scope: Scope{
				IsSystem:  true,
				AccountID: uuid.New(),
				EnvID:     uuid.New(),
			},
		},
		{
			name:    "missing account id",
			scope:   Scope{EnvID: uuid.New(), FunctionID: uuid.New()},
			wantErr: "missing account ID",
		},
		{
			name:    "missing env id",
			scope:   Scope{AccountID: uuid.New(), FunctionID: uuid.New()},
			wantErr: "missing env ID",
		},
		{
			name:    "missing function id",
			scope:   Scope{AccountID: uuid.New(), EnvID: uuid.New()},
			wantErr: "missing function ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.scope.Validate()
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestScopeValidateIDs(t *testing.T) {
	tests := []struct {
		name    string
		scope   Scope
		wantErr string
	}{
		{
			name: "valid ids",
			scope: Scope{
				AccountID:  uuid.New(),
				EnvID:      uuid.New(),
				FunctionID: uuid.New(),
			},
		},
		{
			name:    "missing account id",
			scope:   Scope{EnvID: uuid.New(), FunctionID: uuid.New()},
			wantErr: "missing account ID",
		},
		{
			name:    "missing env id",
			scope:   Scope{AccountID: uuid.New(), FunctionID: uuid.New()},
			wantErr: "missing env ID",
		},
		{
			name:    "missing function id",
			scope:   Scope{AccountID: uuid.New(), EnvID: uuid.New()},
			wantErr: "missing function ID",
		},
	}

	for _, tt := range tests {
		for _, isSystem := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/system=%t", tt.name, isSystem), func(t *testing.T) {
				scope := tt.scope
				scope.IsSystem = isSystem

				err := scope.ValidateIDs()
				if tt.wantErr != "" {
					require.ErrorContains(t, err, tt.wantErr)
					return
				}
				require.NoError(t, err)
			})
		}
	}
}

func TestScopeFromQueueItem(t *testing.T) {
	accountID, functionID := uuid.New(), uuid.New()
	identifierWorkspaceID, itemWorkspaceID := uuid.New(), uuid.New()
	tests := []struct {
		name                  string
		identifierWorkspaceID uuid.UUID
		itemWorkspaceID       uuid.UUID
		wantEnvID             uuid.UUID
	}{
		{
			name:                  "identifier workspace only",
			identifierWorkspaceID: identifierWorkspaceID,
			wantEnvID:             identifierWorkspaceID,
		},
		{
			name:            "falls back to item workspace",
			itemWorkspaceID: itemWorkspaceID,
			wantEnvID:       itemWorkspaceID,
		},
		{
			name:                  "identifier workspace takes precedence",
			identifierWorkspaceID: identifierWorkspaceID,
			itemWorkspaceID:       itemWorkspaceID,
			wantEnvID:             identifierWorkspaceID,
		},
		{
			name: "both workspaces missing",
		},
	}

	for _, tt := range tests {
		for _, isSystem := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/system=%t", tt.name, isSystem), func(t *testing.T) {
				item := QueueItem{
					WorkspaceID: tt.itemWorkspaceID,
					FunctionID:  functionID,
				}
				item.Data.Identifier.AccountID = accountID
				item.Data.Identifier.WorkspaceID = tt.identifierWorkspaceID
				if isSystem {
					queueName := KindCronHealthCheck
					item.QueueName = &queueName
				}

				require.Equal(t, Scope{
					IsSystem:   isSystem,
					AccountID:  accountID,
					EnvID:      tt.wantEnvID,
					FunctionID: functionID,
				}, ScopeFromQueueItem(item))
			})
		}
	}
}
