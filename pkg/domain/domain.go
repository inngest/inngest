// Package domain maps type aliases and domain-specific types for reuse
// across packages.
package domain

import (
	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
)

type (
	AccountID  = uuid.UUID
	EnvID      = uuid.UUID
	FunctionID = uuid.UUID

	RunID   = ulid.ULID
	EventID = ulid.ULID
)
