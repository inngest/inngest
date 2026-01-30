package debugapi

import "encoding/json"

// DebugResponse shows the response structure for debug API calls
type DebugResponse struct {
	Data any `json:"data,omitempty"`
}

// BatchInfoRequest is used to query the current batch for a function and batch key.
type BatchInfoRequest struct {
	// FunctionID is the UUID of the function.
	FunctionID string `json:"function_id"`
	// BatchKey is the optional batch key expression result. If empty, uses "default".
	BatchKey string `json:"batch_key"`
	// AccountID is required for shard selection.
	AccountID string `json:"account_id"`
}

// BatchInfoResponse contains information about the current batch.
type BatchInfoResponse struct {
	// BatchID is the current batch ULID if one exists.
	BatchID string `json:"batch_id"`
	// ItemCount is the number of events currently in the batch.
	ItemCount int32 `json:"item_count"`
	// Items contains the batch items with event data.
	Items []*BatchEventItem `json:"items"`
	// Status is the current batch status (pending, started, etc.).
	Status string `json:"status"`
}

// BatchEventItem represents a single event in a batch.
type BatchEventItem struct {
	EventID         string          `json:"event_id"`
	AccountID       string          `json:"account_id"`
	WorkspaceID     string          `json:"workspace_id"`
	AppID           string          `json:"app_id"`
	FunctionID      string          `json:"function_id"`
	FunctionVersion int             `json:"function_version"`
	EventData       json.RawMessage `json:"event_data"`
}

// SingletonInfoRequest is used to query the current singleton lock.
type SingletonInfoRequest struct {
	// SingletonKey is the evaluated singleton key (function_id-hash or just function_id).
	SingletonKey string `json:"singleton_key"`
	// AccountID is required for shard selection.
	AccountID string `json:"account_id"`
}

// SingletonInfoResponse contains information about the current singleton lock.
type SingletonInfoResponse struct {
	// HasLock indicates whether there is currently an active singleton lock.
	HasLock bool `json:"has_lock"`
	// CurrentRunID is the ULID of the run that holds the lock, if any.
	CurrentRunID string `json:"current_run_id"`
}

// DebounceInfoRequest is used to query the currently debounced event.
type DebounceInfoRequest struct {
	// FunctionID is the UUID of the function.
	FunctionID string `json:"function_id"`
	// DebounceKey is the evaluated debounce key (from the key expression or function_id).
	DebounceKey string `json:"debounce_key"`
	// AccountID is required for shard selection.
	AccountID string `json:"account_id"`
}

// DebounceInfoResponse contains information about the current debounce.
type DebounceInfoResponse struct {
	// HasDebounce indicates whether there is a currently pending debounce.
	HasDebounce bool `json:"has_debounce"`
	// DebounceID is the ULID of the current debounce.
	DebounceID string `json:"debounce_id"`
	// EventID is the ULID of the currently debounced event.
	EventID string `json:"event_id"`
	// EventData is the JSON-encoded event payload.
	EventData json.RawMessage `json:"event_data"`
	// Timeout is the maximum timeout for the debounce, as unix milliseconds.
	Timeout int64 `json:"timeout"`
	// AccountID from the debounce item.
	AccountID string `json:"account_id"`
	// WorkspaceID from the debounce item.
	WorkspaceID string `json:"workspace_id"`
	// FunctionID from the debounce item.
	FunctionID string `json:"function_id"`
}
