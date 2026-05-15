package sync

import "github.com/google/uuid"

// Reply is used for sync response
type Reply struct {
	OK       bool       `json:"ok"`
	Modified bool       `json:"modified"`
	Message  *string    `json:"message,omitempty"`
	Error    *string    `json:"error,omitempty"`
	SyncID   *uuid.UUID `json:"sync_id,omitempty"`
	AppID    *uuid.UUID `json:"app_id,omitempty"`
}

func (sr *Reply) IsSuccess() bool {
	return sr.OK && sr.SyncID != nil && sr.AppID != nil
}
