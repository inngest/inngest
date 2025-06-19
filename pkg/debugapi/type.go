package debugapi

// DebugResponse shows the response structure for debug API calls
type DebugResponse struct {
	Data  any   `json:"data,omitempty"`
	Error error `json:"error,omitempty"`
}
